package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/ai"
	"github.com/laerciocrestani/openbench/internal/config"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
	"github.com/laerciocrestani/openbench/internal/ui"
)

// DoctorOptions controla a execução do doctor.
type DoctorOptions struct {
	Explain  bool
	Base     string
	WorkDir  string // optional; when set, git/gh/docker run in this directory
	Progress Progress
}

// DoctorIssue is a structured health finding for CLI/TUI/desktop.
type DoctorIssue struct {
	Level  string `json:"level"` // ok|warn|critical
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// DoctorReport é o resultado formatado do doctor.
type DoctorReport struct {
	Overall         gitpkg.HealthLevel
	Issues          []DoctorIssue
	Recommendations []string
	Lines           []string
	Facts           string
	AI              *ai.HealthExplanation
	Usage           ai.UsageSummary
	Branch          string
	Base            string
}

// RunDoctor analisa a saúde do repositório e retorna um panorama.
func RunDoctor(ctx context.Context, opts DoctorOptions) (*DoctorReport, error) {
	prog := opts.Progress
	if prog == nil {
		sess := ui.New("doctor", false)
		sess.Header()
		prog = sess
	}

	var repo *gitpkg.Repo
	if err := prog.Step("Opening repository", func() error {
		r, err := openRepo(opts.WorkDir)
		if err != nil {
			return err
		}
		if err := r.IsRepo(); err != nil {
			return fmt.Errorf("diretório atual não é um repositório git")
		}
		repo = r
		return nil
	}); err != nil {
		return nil, err
	}

	base := opts.Base
	var cfg *config.Config
	if err := prog.Step("Loading configuration", func() error {
		var err error
		cfg, err = config.Load()
		if err == nil && base == "" {
			base = cfg.BaseBranch
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if base == "" {
		base = "main"
	}

	var snap *gitpkg.HealthSnapshot
	if err := prog.Step("Analyzing repository health", func() error {
		var err error
		snap, err = repo.CollectHealthSnapshot(base)
		return err
	}); err != nil {
		return nil, err
	}

	var openPR *prpkg.PRView
	if client, err := openPRClient(opts.WorkDir); err == nil {
		openPR, _ = client.ViewCurrent()
	}

	issues := analyzeHealthIssues(snap, openPR)
	recommendations := buildHealthRecommendations(snap, issues, openPR)

	var dockerOverview *dockerpkg.Overview
	if err := prog.Step("Checking Docker environment", func() error {
		dockerOverview = dockerpkg.LoadOverview(opts.WorkDir)
		dockerIssues, dockerRecs := analyzeDockerHealth(dockerOverview)
		issues = append(issues, dockerIssues...)
		recommendations = appendUniqueRecommendations(recommendations, dockerRecs)
		return nil
	}); err != nil {
		return nil, err
	}

	overall := overallHealth(issues, snap)

	report := &DoctorReport{
		Overall:         overall,
		Issues:          toDoctorIssues(issues),
		Recommendations: append([]string{}, recommendations...),
		Facts:           formatHealthFacts(snap, openPR, dockerOverview, issues, recommendations),
		Lines:           formatDoctorLines(snap, openPR, issues, recommendations, overall, nil),
		Branch:          snap.Branch,
		Base:            snap.Base,
	}

	if opts.Explain {
		if cfg == nil || cfg.APIKey == "" {
			return report, fmt.Errorf("API key não configurada — use ob config ou OB_API_KEY para --explain")
		}

		var provider ai.Provider
		if err := prog.Step("Consulting AI", func() error {
			var err error
			provider, err = ai.New(cfg)
			if err != nil {
				return err
			}
			explanation, err := provider.ExplainHealth(withAINotices(ctx, prog), report.Facts, cfg.Language)
			if err != nil {
				return err
			}
			report.AI = explanation
			report.Usage = provider.UsageStats()
			return nil
		}); err != nil {
			return report, err
		}

		report.Lines = formatDoctorLines(snap, openPR, issues, recommendations, overall, report.AI)
		if cfg != nil && len(report.Usage.Records) > 0 {
			recordAIUsage("doctor", cfg, report.Usage)
		}
	}

	return report, nil
}

// LoadDoctorReport coleta o doctor sem UI de progresso (TUI).
func LoadDoctorReport(ctx context.Context, explain bool) (*DoctorReport, error) {
	return RunDoctor(ctx, DoctorOptions{Explain: explain})
}

// PrintDoctor exibe o relatório na CLI.
func PrintDoctor(report *DoctorReport, prog Progress) {
	if report == nil {
		return
	}
	if prog == nil {
		for _, line := range report.Lines {
			fmt.Println(line)
		}
		return
	}

	prog.Info("Repository health")
	for _, line := range report.Lines {
		if strings.HasPrefix(line, "  ") {
			prog.Detail(strings.TrimPrefix(line, "  "))
			continue
		}
		if line == "" {
			fmt.Println()
			continue
		}
		prog.Info(line)
	}

	if len(report.Usage.Records) > 0 {
		cfg, _ := config.Load()
		if cfg != nil {
			for _, line := range report.Usage.FormatLines(cfg) {
				prog.Detail(line)
			}
		}
	}

	prog.Success("Diagnóstico concluído")
}

// DiagnoseSyncFailure imprime orientação quando o sync falha por divergência.
func DiagnoseSyncFailure(base string, syncErr error, prog Progress) {
	if syncErr == nil || !isFastForwardError(syncErr) {
		return
	}

	repo, err := gitpkg.New()
	if err != nil {
		return
	}

	snap, err := repo.CollectHealthSnapshot(base)
	if err != nil {
		return
	}

	issues := analyzeHealthIssues(snap, nil)
	recommendations := buildHealthRecommendations(snap, issues, nil)
	overall := overallHealth(issues, snap)

	if prog == nil {
		prog = ui.New("sync", false)
	}

	prog.Warn("Pull bloqueado — branches divergiram (fast-forward impossível)")
	prog.Info("Diagnóstico rápido")
	for _, line := range formatDoctorLines(snap, nil, issues, recommendations, overall, nil) {
		if strings.HasPrefix(line, "  ") {
			prog.Detail(strings.TrimPrefix(line, "  "))
		} else if line != "" {
			prog.Info(line)
		}
	}
		prog.Info("Para panorama completo: ob doctor")
}

func isFastForwardError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "fast-forward") ||
		strings.Contains(msg, "diverging branches")
}

type healthIssue struct {
	Level  gitpkg.HealthLevel
	Code   string
	Title  string
	Detail string
}

func analyzeHealthIssues(snap *gitpkg.HealthSnapshot, currentPR *prpkg.PRView) []healthIssue {
	if snap == nil {
		return nil
	}

	var issues []healthIssue

	if snap.IsDirty {
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthWarn,
			Code:   "dirty_tree",
			Title:  "Working tree com alterações",
			Detail: fmt.Sprintf("%d staged · %d modified · %d untracked", snap.Staged, snap.Modified, snap.Untracked),
		})
	}

	if currentPR != nil && strings.EqualFold(currentPR.State, "MERGED") && !snap.OnBase {
		detail := fmt.Sprintf("PR #%d já foi mergeada — não continue desenvolvendo nesta branch", currentPR.Number)
		if snap.IsDirty {
			detail += "; salve o work e abra uma branch nova a partir da base atualizada"
		}
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthWarn,
			Code:   "work_on_merged_branch",
			Title:  fmt.Sprintf("Branch %q já tem PR mergeada", snap.Branch),
			Detail: detail,
		})
	}

	if snap.Diverged {
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthCritical,
			Code:   "branch_diverged",
			Title:  fmt.Sprintf("Branch %q divergiu do upstream", snap.Branch),
			Detail: fmt.Sprintf("↑%d commit(s) local · ↓%d commit(s) remoto", snap.Ahead, snap.Behind),
		})
	} else if snap.Behind > 0 && !snap.IsDirty {
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthWarn,
			Code:   "behind_remote",
			Title:  "Branch atrás do remoto",
			Detail: fmt.Sprintf("%d commit(s) no remoto ainda não puxados", snap.Behind),
		})
	}

	if snap.BaseDivergence != nil && (snap.BaseDivergence.LocalAhead > 0 || snap.BaseDivergence.RemoteAhead > 0) {
		div := snap.BaseDivergence
		level := gitpkg.HealthCritical
		if div.LocalAhead > 0 && div.RemoteAhead > 0 {
			level = gitpkg.HealthCritical
		} else if div.LocalAhead > 0 {
			level = gitpkg.HealthWarn
		}
		issues = append(issues, healthIssue{
			Level: level,
			Code:  "base_diverged",
			Title: fmt.Sprintf("Base %q divergiu de %s", snap.Base, div.RemoteRef),
			Detail: fmt.Sprintf(
				"local ↑%d · remoto ↑%d · merge-base %s",
				div.LocalAhead, div.RemoteAhead, shortHash(div.MergeBase),
			),
		})

		for _, analysis := range div.LocalAnalyses {
			if analysis.LikelyDiscardable {
				issues = append(issues, healthIssue{
					Level: gitpkg.HealthWarn,
					Code:  "build_artifacts",
					Title: fmt.Sprintf("Commit %s parece conter artefatos de build", analysis.Hash),
					Detail: fmt.Sprintf(
						"%s — %d arquivo(s), %d artefato(s) de build",
						analysis.Subject, analysis.FileCount, analysis.BuildArtifactFiles,
					),
				})
			}
		}
	}

	if snap.OnBase && snap.CommitsAheadOfBase == 0 && !snap.IsDirty && snap.Ahead == 0 && snap.Behind == 0 {
		if snap.BaseDivergence == nil {
			// healthy base — no issue
		}
	}

	if snap.OnBase && snap.CommitsAheadOfBase > 0 && !snap.IsDirty {
		hasLocalDivergence := snap.BaseDivergence != nil && snap.BaseDivergence.LocalAhead > 0
		if !hasLocalDivergence {
			issues = append(issues, healthIssue{
				Level:  gitpkg.HealthWarn,
				Code:   "commits_on_base",
				Title:  fmt.Sprintf("Commits diretos na base %q", snap.Base),
				Detail: fmt.Sprintf("%d commit(s) à frente da base remota — prefira feature branches", snap.CommitsAheadOfBase),
			})
		}
	}

	return issues
}

func buildHealthRecommendations(snap *gitpkg.HealthSnapshot, issues []healthIssue, currentPR *prpkg.PRView) []string {
	if snap == nil {
		return nil
	}

	var recs []string
	seen := map[string]bool{}
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		recs = append(recs, s)
	}

	mergedBranch := currentPR != nil && strings.EqualFold(currentPR.State, "MERGED") && !snap.OnBase

	for _, issue := range issues {
		switch issue.Code {
		case "work_on_merged_branch":
			if snap.IsDirty {
				add("Salve o work: Commit na branch atual OU git stash push -m \"wip\"")
			}
			add(fmt.Sprintf("Atualize a base local %s (fetch)", snap.Base))
			add(fmt.Sprintf("Crie uma NOVA feature branch a partir de %s atualizado", snap.Base))
			if snap.IsDirty {
				add("Aplique o work na branch nova (stash pop / cherry-pick) e abra outra PR")
			} else {
				add("Continue o desenvolvimento na branch nova (abra PR quando houver commits)")
			}
			add(fmt.Sprintf("Evite push/PR de novo em %s — a PR já foi mergeada", snap.Branch))
		case "dirty_tree":
			if !mergedBranch {
				add("Commit as alterações nesta branch (botão Commit / ob commit)")
				add("Depois: push e abra (ou atualize) a Pull Request")
			}
		case "behind_remote":
			if !snap.IsDirty {
				add("ob sync")
			}
		case "base_diverged":
			if snap.BaseDivergence != nil {
				div := snap.BaseDivergence
				allDiscardable := div.LocalAhead > 0 && len(div.LocalAnalyses) > 0
				for _, a := range div.LocalAnalyses {
					if !a.LikelyDiscardable {
						allDiscardable = false
						break
					}
				}
				if allDiscardable && !snap.IsDirty {
					add(fmt.Sprintf("git fetch origin && git reset --hard origin/%s  # descarta commits locais de build", snap.Base))
				} else if div.LocalAhead > 0 && div.RemoteAhead > 0 {
					add(fmt.Sprintf("git fetch origin && git rebase origin/%s  # se os commits locais têm valor", snap.Base))
					add(fmt.Sprintf("git fetch origin && git reset --hard origin/%s  # se os commits locais são descartáveis", snap.Base))
				} else if div.LocalAhead > 0 {
					add(fmt.Sprintf("git push origin %s  # se os commits locais devem ir ao remoto", snap.Base))
				} else if div.RemoteAhead > 0 {
					add("ob sync")
				}
			}
		case "branch_diverged":
			add("git fetch origin")
			add("git rebase @{u}  # ou git merge @{u}")
		case "commits_on_base":
			add("git checkout -b feature/minha-alteracao")
		}
	}

	if len(recs) == 0 && !snap.IsDirty {
		add("Continuar desenvolvimento — repositório saudável")
	}

	return recs
}

func overallHealth(issues []healthIssue, snap *gitpkg.HealthSnapshot) gitpkg.HealthLevel {
	level := gitpkg.HealthOK
	for _, issue := range issues {
		if issue.Level == gitpkg.HealthCritical {
			return gitpkg.HealthCritical
		}
		if issue.Level == gitpkg.HealthWarn {
			level = gitpkg.HealthWarn
		}
	}
	if snap != nil && snap.IsDirty {
		level = gitpkg.HealthWarn
	}
	return level
}

func formatHealthFacts(
	snap *gitpkg.HealthSnapshot,
	openPR *prpkg.PRView,
	dockerOverview *dockerpkg.Overview,
	issues []healthIssue,
	recommendations []string,
) string {
	var b strings.Builder
	if snap == nil {
		return ""
	}

	fmt.Fprintf(&b, "Branch: %s\n", snap.Branch)
	fmt.Fprintf(&b, "Base: %s (on_base=%v)\n", snap.Base, snap.OnBase)
	fmt.Fprintf(&b, "Working tree: dirty=%v staged=%d modified=%d untracked=%d\n",
		snap.IsDirty, snap.Staged, snap.Modified, snap.Untracked)
	fmt.Fprintf(&b, "Upstream sync: ahead=%d behind=%d diverged=%v\n", snap.Ahead, snap.Behind, snap.Diverged)
	fmt.Fprintf(&b, "Commits ahead of base: %d\n", snap.CommitsAheadOfBase)

	if openPR != nil {
		label := "PR"
		if strings.EqualFold(openPR.State, "MERGED") {
			label = "PR (já mergeada — NÃO continue nesta branch)"
		} else if strings.EqualFold(openPR.State, "OPEN") {
			label = "Open PR"
		}
		fmt.Fprintf(&b, "%s: #%d %s (state=%s draft=%v)\n",
			label, openPR.Number, openPR.Title, openPR.State, openPR.IsDraft)
	}

	if dockerOverview != nil {
		fmt.Fprintf(&b, "\nDocker: available=%v daemon=%v compose=%q\n",
			dockerOverview.Available, dockerOverview.DaemonRunning, dockerOverview.ComposeFile)
		for _, c := range dockerOverview.Containers {
			fmt.Fprintf(&b, "  container %s: state=%s ports=%s health=%s\n",
				c.Service, c.State, c.Ports, c.Health)
		}
		if dockerOverview.Error != "" {
			fmt.Fprintf(&b, "  docker error: %s\n", dockerOverview.Error)
		}
	}

	appendDivergenceFacts(&b, "Base divergence", snap.BaseDivergence)
	appendDivergenceFacts(&b, "Branch divergence", snap.BranchDivergence)

	if len(issues) > 0 {
		b.WriteString("\nIssues:\n")
		for _, issue := range issues {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", issue.Level, issue.Title, issue.Detail)
		}
	}

	if len(recommendations) > 0 {
		b.WriteString("\nRecommendations:\n")
		for _, rec := range recommendations {
			fmt.Fprintf(&b, "- %s\n", rec)
		}
	}

	return b.String()
}

func appendDivergenceFacts(b *strings.Builder, label string, div *gitpkg.DivergenceReport) {
	if div == nil {
		return
	}
	fmt.Fprintf(b, "\n%s (%s vs %s):\n", label, div.LocalRef, div.RemoteRef)
	fmt.Fprintf(b, "  merge-base: %s\n", shortHash(div.MergeBase))
	fmt.Fprintf(b, "  local ahead: %d, remote ahead: %d\n", div.LocalAhead, div.RemoteAhead)
	for _, c := range div.LocalCommits {
		fmt.Fprintf(b, "  local commit: %s\n", c)
	}
	for _, c := range div.RemoteCommits {
		fmt.Fprintf(b, "  remote commit: %s\n", c)
	}
	for _, a := range div.LocalAnalyses {
		fmt.Fprintf(b, "  analysis %s: files=%d build=%d discardable=%v — %s\n",
			a.Hash, a.FileCount, a.BuildArtifactFiles, a.LikelyDiscardable, a.Subject)
	}
	if div.LocalDiffStat != "" {
		b.WriteString("  diff --stat (local only):\n")
		for _, line := range strings.Split(div.LocalDiffStat, "\n") {
			fmt.Fprintf(b, "    %s\n", line)
		}
	}
}

func formatDoctorLines(
	snap *gitpkg.HealthSnapshot,
	openPR *prpkg.PRView,
	issues []healthIssue,
	recommendations []string,
	overall gitpkg.HealthLevel,
	aiExplanation *ai.HealthExplanation,
) []string {
	var lines []string

	lines = append(lines, "Panorama de saúde")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Status geral: %s", healthLevelLabel(overall)))

	if snap != nil {
		lines = append(lines, fmt.Sprintf("  Branch: %s → base %s", snap.Branch, snap.Base))
		if snap.IsDirty {
			lines = append(lines, fmt.Sprintf("  Working tree: %d staged · %d modified · %d untracked",
				snap.Staged, snap.Modified, snap.Untracked))
		} else {
			lines = append(lines, "  Working tree: limpa")
		}

		switch {
		case snap.Diverged:
			lines = append(lines, fmt.Sprintf("  Sync upstream: divergiu (↑%d · ↓%d)", snap.Ahead, snap.Behind))
		case snap.Ahead > 0 && snap.Behind > 0:
			lines = append(lines, fmt.Sprintf("  Sync upstream: divergiu (↑%d · ↓%d)", snap.Ahead, snap.Behind))
		case snap.Ahead > 0:
			lines = append(lines, fmt.Sprintf("  Sync upstream: ↑ %d à frente do remoto", snap.Ahead))
		case snap.Behind > 0:
			lines = append(lines, fmt.Sprintf("  Sync upstream: ↓ %d atrás do remoto", snap.Behind))
		default:
			lines = append(lines, "  Sync upstream: em dia")
		}

		if !snap.OnBase && snap.CommitsAheadOfBase > 0 {
			lines = append(lines, fmt.Sprintf("  Desenvolvimento: %d commit(s) à frente de %s", snap.CommitsAheadOfBase, snap.Base))
		}
	}

	if openPR != nil {
		state := strings.ToLower(openPR.State)
		if openPR.IsDraft {
			state = "draft"
		}
		lines = append(lines, fmt.Sprintf("  Pull request: #%d %s (%s)", openPR.Number, openPR.Title, state))
	}

	if len(issues) > 0 {
		lines = append(lines, "", "Achados")
		for _, issue := range issues {
			prefix := "  •"
			switch issue.Level {
			case gitpkg.HealthCritical:
				prefix = "  ✗"
			case gitpkg.HealthWarn:
				prefix = "  !"
			}
			lines = append(lines, fmt.Sprintf("%s %s", prefix, issue.Title))
			if issue.Detail != "" {
				lines = append(lines, "      "+issue.Detail)
			}
		}
	} else {
		lines = append(lines, "", "Achados", "  ✓ Nenhum problema detectado")
	}

	appendDivergenceLines(&lines, snap)

	if len(recommendations) > 0 {
		lines = append(lines, "", "Recomendações")
		for _, rec := range recommendations {
			lines = append(lines, "  → "+rec)
		}
	}

	if aiExplanation != nil {
		lines = append(lines, "", "Análise IA")
		if aiExplanation.Summary != "" {
			lines = append(lines, "  "+aiExplanation.Summary)
		}
		if aiExplanation.Cause != "" {
			lines = append(lines, "  Causa: "+aiExplanation.Cause)
		}
		if aiExplanation.Risk != "" {
			lines = append(lines, "  Risco: "+aiExplanation.Risk)
		}
		if len(aiExplanation.Steps) > 0 {
			lines = append(lines, "", "  Passos sugeridos:")
			for _, step := range aiExplanation.Steps {
				lines = append(lines, "    → "+step)
			}
		}
		if len(aiExplanation.Warnings) > 0 {
			lines = append(lines, "", "  Alertas:")
			for _, w := range aiExplanation.Warnings {
				lines = append(lines, "    ⚠ "+w)
			}
		}
	}

	return lines
}

func appendDivergenceLines(lines *[]string, snap *gitpkg.HealthSnapshot) {
	if snap == nil {
		return
	}
	div := snap.BaseDivergence
	if div == nil {
		div = snap.BranchDivergence
	}
	if div == nil || (div.LocalAhead == 0 && div.RemoteAhead == 0) {
		return
	}

	*lines = append(*lines, "", "Divergência")
	*lines = append(*lines, fmt.Sprintf("  %s vs %s (ancestral %s)", div.LocalRef, div.RemoteRef, shortHash(div.MergeBase)))
	if div.LocalAhead > 0 {
		*lines = append(*lines, fmt.Sprintf("  Local: %d commit(s) exclusivo(s)", div.LocalAhead))
		for _, c := range div.LocalCommits {
			*lines = append(*lines, "      "+c)
		}
	}
	if div.RemoteAhead > 0 {
		*lines = append(*lines, fmt.Sprintf("  Remoto: %d commit(s) exclusivo(s)", div.RemoteAhead))
		for _, c := range div.RemoteCommits {
			*lines = append(*lines, "      "+c)
		}
	}
}

func healthLevelLabel(level gitpkg.HealthLevel) string {
	switch level {
	case gitpkg.HealthCritical:
		return "crítico — ação necessária"
	case gitpkg.HealthWarn:
		return "atenção — revisar recomendações"
	default:
		return "saudável"
	}
}

func shortHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func analyzeDockerHealth(ov *dockerpkg.Overview) ([]healthIssue, []string) {
	if ov == nil {
		return nil, nil
	}
	var issues []healthIssue
	var recs []string

	if !ov.Available {
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthWarn,
			Code:   "docker_missing",
			Title:  "Docker CLI não encontrado",
			Detail: "instale Docker para subir o ambiente local",
		})
		return issues, recs
	}
	if !ov.DaemonRunning {
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthWarn,
			Code:   "docker_daemon",
			Title:  "Docker daemon parado",
			Detail: "inicie o Docker Desktop ou o serviço docker",
		})
		recs = append(recs, "ob docker status")
		return issues, recs
	}
	if ov.ComposeFile == "" {
		return issues, recs
	}
	if !dockerpkg.HasRunningContainers(ov.Containers) {
		issues = append(issues, healthIssue{
			Level:  gitpkg.HealthWarn,
			Code:   "docker_stopped",
			Title:  "Compose detectado, containers parados",
			Detail: ov.ComposeFile,
		})
		recs = append(recs, "ob docker up")
	}
	return issues, recs
}

func toDoctorIssues(issues []healthIssue) []DoctorIssue {
	if len(issues) == 0 {
		return nil
	}
	out := make([]DoctorIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, DoctorIssue{
			Level:  string(issue.Level),
			Code:   issue.Code,
			Title:  issue.Title,
			Detail: issue.Detail,
		})
	}
	return out
}

func appendUniqueRecommendations(base, extra []string) []string {
	seen := map[string]bool{}
	for _, s := range base {
		seen[s] = true
	}
	for _, s := range extra {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		base = append(base, s)
	}
	return base
}

// FormatDoctorContent retorna o relatório como texto único (TUI).
func FormatDoctorContent(report *DoctorReport) string {
	if report == nil {
		return ""
	}
	return strings.Join(report.Lines, "\n")
}
