package desktop

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/laerciocrestani/openbench/internal/ai"
	"github.com/laerciocrestani/openbench/internal/aiskills"
	"github.com/laerciocrestani/openbench/internal/config"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
)

// DockerFixServiceView mirrors a service row from the action dialog.
type DockerFixServiceView struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// DockerFixFailureView is the snapshot sent from the UI to PlanDockerFix.
type DockerFixFailureView struct {
	Action   string                 `json:"action"`
	Message  string                 `json:"message"`
	Lines    []string               `json:"lines"`
	Services []DockerFixServiceView `json:"services"`
}

// DockerFixFileView is one proposed file change for preview.
type DockerFixFileView struct {
	Path    string `json:"path"`
	Reason  string `json:"reason"`
	Bytes   int    `json:"bytes"`
	Preview string `json:"preview"`
	Enabled bool   `json:"enabled"`
}

// DockerFixStepView is one planned/executed remediation step.
type DockerFixStepView struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Target  string `json:"target"`
	Risk    string `json:"risk"`
	Command string `json:"command,omitempty"`
	Status  string `json:"status,omitempty"` // pending|running|ok|error|skipped
	Detail  string `json:"detail,omitempty"`
	Enabled bool   `json:"enabled"`
}

// DockerFixPlanView is shown in the Corrigir com IA dialog.
type DockerFixPlanView struct {
	Problem    string               `json:"problem"`
	Resolution string               `json:"resolution"`
	Steps      []DockerFixStepView  `json:"steps"`
	Files      []DockerFixFileView  `json:"files"`
	Notes      []string             `json:"notes,omitempty"`
	Message    string               `json:"message,omitempty"`
	CanFix     bool                 `json:"canFix"`
	Action     string               `json:"action,omitempty"` // original docker action to re-run
}

// DockerFixAdvanceView is one step result from AdvanceDockerFix.
type DockerFixAdvanceView struct {
	Step    DockerFixStepView `json:"step"`
	Done    bool              `json:"done"`
	OK      bool              `json:"ok"`
	Message string            `json:"message,omitempty"`
}

type pendingDockerFix struct {
	action string
	sug    *ai.DockerFixSuggestion
}

// DockerFixSession holds an in-progress step-by-step docker fix.
type DockerFixSession struct {
	Path    string
	Action  string
	Runner  *dockerFixRunner
}

type dockerFixRunner struct {
	projectPath string
	composeFile string
	steps       []ai.DockerFixStep
	files       map[string]ai.DockerFixFile
	index       int
	failed      bool
}

var (
	dockerFixMu      sync.Mutex
	dockerFixPending = map[string]*pendingDockerFix{}
)

// PlanDockerFix asks the AI for a remediation plan from a failed docker action.
func PlanDockerFix(ctx context.Context, projectPath string, failure DockerFixFailureView) (*DockerFixPlanView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	ov := dockerpkg.LoadOverview(projectPath)
	composeRel := ""
	if ov.ComposeFile != "" {
		composeRel = strings.TrimPrefix(ov.ComposeFile, projectPath)
		composeRel = strings.TrimPrefix(composeRel, "/")
		if composeRel == "" {
			composeRel = ov.ComposeFile
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	provider, err := ai.New(cfg)
	if err != nil {
		return nil, err
	}
	lang := cfg.Language
	if strings.TrimSpace(lang) == "" {
		lang = "pt-BR"
	}

	skillBody := ""
	if sk, err := aiskills.Get("docker-debug"); err == nil && sk != nil {
		skillBody = sk.Body
	}

	services := make([]ai.DockerFixServiceStatus, 0, len(failure.Services))
	for _, s := range failure.Services {
		services = append(services, ai.DockerFixServiceStatus{
			Name:   s.Name,
			Status: s.Status,
			Detail: s.Detail,
		})
	}

	fc := ai.DockerFixContext{
		Action:      failure.Action,
		Message:     failure.Message,
		Lines:       append([]string{}, failure.Lines...),
		Services:    services,
		ComposeFile: composeRel,
		SkillBody:   skillBody,
	}

	sug, err := provider.SuggestDockerFix(ctx, fc, lang)
	if err != nil {
		return &DockerFixPlanView{
			Action:  failure.Action,
			Message: err.Error(),
			CanFix:  false,
		}, nil
	}

	view := mapDockerFixSuggestion(sug, failure.Action)
	if !view.CanFix {
		if view.Message == "" {
			view.Message = "IA não montou um plano executável — revise o log manualmente"
		}
	}

	dockerFixMu.Lock()
	dockerFixPending[projectPath] = &pendingDockerFix{
		action: failure.Action,
		sug:    sug,
	}
	dockerFixMu.Unlock()
	return view, nil
}

// BeginDockerFix prepares a step-by-step execution with the user's enabled selection.
func BeginDockerFix(projectPath string, enabledStepIDs, enabledFilePaths []string) (*DockerFixPlanView, *DockerFixSession, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, nil, fmt.Errorf("no project open")
	}
	dockerFixMu.Lock()
	pending := dockerFixPending[projectPath]
	dockerFixMu.Unlock()
	if pending == nil || pending.sug == nil {
		return nil, nil, fmt.Errorf("nenhuma correção pendente — rode PlanDockerFix antes")
	}

	enabledSteps := toSet(enabledStepIDs)
	enabledFiles := toSet(enabledFilePaths)

	var steps []ai.DockerFixStep
	for _, st := range pending.sug.Steps {
		if !enabledSteps[st.ID] {
			continue
		}
		if st.Kind == "write_file" && !enabledFiles[st.Target] {
			continue
		}
		steps = append(steps, st)
	}

	// Also run write_file for enabled files that have no explicit step.
	fileByPath := map[string]ai.DockerFixFile{}
	for _, f := range pending.sug.Files {
		fileByPath[f.Path] = f
	}
	hasWriteStep := map[string]bool{}
	for _, st := range steps {
		if st.Kind == "write_file" {
			hasWriteStep[st.Target] = true
		}
	}
	for path := range enabledFiles {
		if hasWriteStep[path] {
			continue
		}
		if _, ok := fileByPath[path]; !ok {
			continue
		}
		steps = append(steps, ai.DockerFixStep{
			ID:     "file-" + path,
			Kind:   "write_file",
			Title:  "Escrever " + path,
			Target: path,
			Risk:   "warn",
		})
	}

	if len(steps) == 0 {
		return mapDockerFixSuggestion(pending.sug, pending.action), nil, fmt.Errorf("nenhum passo habilitado para executar")
	}

	ov := dockerpkg.LoadOverview(projectPath)
	if !ov.Available {
		return nil, nil, fmt.Errorf("docker CLI não encontrado")
	}
	if !ov.DaemonRunning {
		return nil, nil, fmt.Errorf("Docker daemon não está rodando")
	}
	if ov.ComposeFile == "" {
		return nil, nil, fmt.Errorf("compose file não encontrado no projeto")
	}

	runner := &dockerFixRunner{
		projectPath: projectPath,
		composeFile: ov.ComposeFile,
		steps:       steps,
		files:       fileByPath,
	}
	plan := &DockerFixPlanView{
		Problem:    pending.sug.Problem,
		Resolution: pending.sug.Resolution,
		Notes:      append([]string{}, pending.sug.Notes...),
		Action:     pending.action,
		CanFix:     true,
		Steps:      make([]DockerFixStepView, 0, len(steps)),
		Files:      make([]DockerFixFileView, 0, len(pending.sug.Files)),
	}
	for _, st := range steps {
		plan.Steps = append(plan.Steps, DockerFixStepView{
			ID:      st.ID,
			Kind:    st.Kind,
			Title:   st.Title,
			Target:  st.Target,
			Risk:    st.Risk,
			Command: dockerFixCommand(st),
			Status:  "pending",
			Enabled: true,
		})
	}
	for _, f := range pending.sug.Files {
		plan.Files = append(plan.Files, DockerFixFileView{
			Path:    f.Path,
			Reason:  f.Reason,
			Bytes:   len(f.Content),
			Preview: truncateRunes(f.Content, 800),
			Enabled: enabledFiles[f.Path],
		})
	}
	return plan, &DockerFixSession{Path: projectPath, Action: pending.action, Runner: runner}, nil
}

// AdvanceDockerFixSession runs the next step of an active session.
func AdvanceDockerFixSession(sess *DockerFixSession) (*DockerFixAdvanceView, error) {
	if sess == nil || sess.Runner == nil {
		return nil, fmt.Errorf("nenhuma execução Docker Fix em andamento — confirme de novo")
	}
	sr, done, err := sess.Runner.Next()
	if err != nil {
		return nil, err
	}
	out := &DockerFixAdvanceView{
		Step: sr,
		Done: done,
		OK:   sr.Status != "error",
	}
	if sr.Status == "error" {
		out.OK = false
		out.Done = true
		out.Message = fmt.Sprintf("Parou em: %s", sr.Title)
		return out, nil
	}
	if done {
		out.Message = "Correção concluída"
		// Clear pending so a new Plan is required next time.
		dockerFixMu.Lock()
		delete(dockerFixPending, sess.Path)
		dockerFixMu.Unlock()
	}
	return out, nil
}

func (r *dockerFixRunner) Next() (DockerFixStepView, bool, error) {
	if r == nil {
		return DockerFixStepView{}, true, fmt.Errorf("runner inválido")
	}
	if r.failed {
		return DockerFixStepView{}, true, fmt.Errorf("execução anterior falhou")
	}
	if r.index >= len(r.steps) {
		return DockerFixStepView{}, true, nil
	}
	step := r.steps[r.index]
	r.index++
	view := DockerFixStepView{
		ID:      step.ID,
		Kind:    step.Kind,
		Title:   step.Title,
		Target:  step.Target,
		Risk:    step.Risk,
		Command: dockerFixCommand(step),
		Status:  "running",
		Enabled: true,
	}
	detail, err := r.execute(step)
	if err != nil {
		r.failed = true
		view.Status = "error"
		view.Detail = err.Error()
		return view, true, nil
	}
	view.Status = "ok"
	view.Detail = detail
	done := r.index >= len(r.steps)
	return view, done, nil
}

func (r *dockerFixRunner) execute(step ai.DockerFixStep) (string, error) {
	switch step.Kind {
	case "docker_stop":
		res, err := dockerpkg.StopResult(dockerpkg.ServiceOptions{
			ComposeFile: r.composeFile,
			Services:    []string{step.Target},
		})
		return strings.TrimSpace(res.Output), err
	case "docker_rm":
		res, err := dockerpkg.RmResult(dockerpkg.ServiceOptions{
			ComposeFile: r.composeFile,
			Services:    []string{step.Target},
		})
		return strings.TrimSpace(res.Output), err
	case "docker_up":
		res, err := dockerpkg.UpResult(dockerpkg.UpOptions{
			ComposeFile: r.composeFile,
			Services:    []string{step.Target},
		})
		return strings.TrimSpace(res.Output), err
	case "docker_recreate":
		res, err := dockerpkg.RecreateResult(r.composeFile, step.Target, false, nil)
		return strings.TrimSpace(res.Output), err
	case "write_file":
		f, ok := r.files[step.Target]
		if !ok {
			return "", fmt.Errorf("conteúdo ausente para %s", step.Target)
		}
		msg, err := toolWriteFile(r.projectPath, f.Path, f.Content)
		return msg, err
	default:
		return "", fmt.Errorf("kind não permitido: %s", step.Kind)
	}
}

func dockerFixCommand(step ai.DockerFixStep) string {
	switch step.Kind {
	case "docker_stop":
		return "docker compose stop " + step.Target
	case "docker_rm":
		return "docker compose rm -f " + step.Target
	case "docker_up":
		return "docker compose up -d " + step.Target
	case "docker_recreate":
		return "docker compose up -d --force-recreate --no-deps " + step.Target
	case "write_file":
		return "write " + step.Target
	default:
		return step.Kind
	}
}

func mapDockerFixSuggestion(sug *ai.DockerFixSuggestion, action string) *DockerFixPlanView {
	if sug == nil {
		return &DockerFixPlanView{Action: action, CanFix: false, Message: "sem sugestão"}
	}
	view := &DockerFixPlanView{
		Problem:    sug.Problem,
		Resolution: sug.Resolution,
		Notes:      append([]string{}, sug.Notes...),
		Action:     action,
		Steps:      make([]DockerFixStepView, 0, len(sug.Steps)),
		Files:      make([]DockerFixFileView, 0, len(sug.Files)),
	}
	for _, st := range sug.Steps {
		view.Steps = append(view.Steps, DockerFixStepView{
			ID:      st.ID,
			Kind:    st.Kind,
			Title:   st.Title,
			Target:  st.Target,
			Risk:    st.Risk,
			Command: dockerFixCommand(st),
			Status:  "pending",
			Enabled: true,
		})
	}
	for _, f := range sug.Files {
		view.Files = append(view.Files, DockerFixFileView{
			Path:    f.Path,
			Reason:  f.Reason,
			Bytes:   len(f.Content),
			Preview: truncateRunes(f.Content, 800),
			Enabled: true,
		})
	}
	view.CanFix = len(view.Steps) > 0 || len(view.Files) > 0
	return view
}

func toSet(ids []string) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = true
		}
	}
	return out
}
