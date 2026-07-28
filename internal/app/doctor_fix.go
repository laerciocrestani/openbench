package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

// Doctor fix step kinds (deterministic executor).
const (
	DoctorStepStashPush      = "stash_push"
	DoctorStepFetch          = "fetch"
	DoctorStepUpdateBase     = "update_base"
	DoctorStepCheckout       = "checkout"
	DoctorStepCreateBranch   = "create_branch"
	DoctorStepStashPop       = "stash_pop"
	DoctorStepPullFF         = "pull_ff"
	DoctorStepRebaseUpstream = "rebase_upstream"
	DoctorStepResetBase      = "reset_base"
	DoctorStepRebaseBase       = "rebase_base"
	DoctorStepPushBase         = "push_base"
	DoctorStepSoftReset        = "soft_reset"
	DoctorStepMixedReset       = "mixed_reset"
	DoctorStepUnstageBlockers  = "unstage_blockers"
)

// Post-merge destinations for work_on_merged_branch.
const (
	MergedActionContinue   = "continue"    // create a new feature branch from updated base
	MergedActionReturnBase = "return_base" // update base and check it out (finish / sync)
)

// Base actions for local/remote divergence on the configured base branch.
const (
	BaseActionUpdate  = "update"
	BaseActionRebase  = "rebase"
	BaseActionReset   = "reset"
	BaseActionPush    = "push"
	BaseActionBranch  = "branch"  // move local-ahead commits onto a feature branch
	BaseActionCleanup = "cleanup" // soft-reset + unstage large/junk files from unpushed history
)

// DoctorFixOptions configures plan execution.
type DoctorFixOptions struct {
	WorkDir            string
	Base               string
	NewBranchName      string
	BaseAction         string // update | rebase | reset | push | branch | cleanup
	MergedAction       string // continue | return_base
	ConfirmDestructive bool
}

// DoctorFixStep is one planned git action.
type DoctorFixStep struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Command    string   `json:"command"`
	Risk       string   `json:"risk"` // ok | warn | destructive
	FromRef    string   `json:"fromRef,omitempty"`
	Paths      []string `json:"paths,omitempty"`
	IssueCodes []string `json:"issueCodes,omitempty"`
}

// DoctorFixPlan is the preview shown before confirm.
type DoctorFixPlan struct {
	Steps                   []DoctorFixStep `json:"steps"`
	SuggestedBranch         string          `json:"suggestedBranch"`
	NeedsBranchName         bool            `json:"needsBranchName"`
	NeedsBaseAction         bool            `json:"needsBaseAction"`
	BaseActionOptions       []string        `json:"baseActionOptions,omitempty"`
	SuggestedBaseAction     string          `json:"suggestedBaseAction,omitempty"`
	NeedsMergedAction       bool            `json:"needsMergedAction"`
	MergedActionOptions     []string        `json:"mergedActionOptions,omitempty"`
	SuggestedMergedAction   string          `json:"suggestedMergedAction,omitempty"`
	NeedsDestructiveConfirm bool            `json:"needsDestructiveConfirm"`
	Summary                 string          `json:"summary"`
	Warnings                []string        `json:"warnings,omitempty"`
	CanAutoFix              bool            `json:"canAutoFix"`
	BlockReason             string          `json:"blockReason,omitempty"`
	IssueCodes              []string        `json:"issueCodes,omitempty"`
	Branch                  string          `json:"branch"`
	Base                    string          `json:"base"`
}

// DoctorFixStepResult is the outcome of one executed step.
type DoctorFixStepResult struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Command    string `json:"command"`
	Risk       string `json:"risk,omitempty"`
	Status     string `json:"status"` // pending | running | ok | skipped | error | manual
	Detail     string `json:"detail,omitempty"`
	ManualHint string `json:"manualHint,omitempty"`
}

// DoctorFixResult aggregates execution.
type DoctorFixResult struct {
	OK      bool                  `json:"ok"`
	Message string                `json:"message"`
	Steps   []DoctorFixStepResult `json:"steps"`
}

// DoctorFixRunner executes a prepared plan one step at a time (UI-friendly).
type DoctorFixRunner struct {
	repo    *gitpkg.Repo
	base    string
	opts    DoctorFixOptions
	steps   []DoctorFixStep
	stashed bool
	index   int
	failed  bool
}

// PlanDoctorFix builds a deterministic remediation plan from the current health snapshot.
// Replans (action/branch toggles) reuse a short-lived context cache to avoid repeated git/gh work.
func PlanDoctorFix(opts DoctorFixOptions) (*DoctorFixPlan, error) {
	repo, snap, issues, _, err := loadDoctorContextCached(opts.WorkDir, opts.Base)
	if err != nil {
		return nil, err
	}
	return buildDoctorFixPlan(repo, snap, issues, opts), nil
}

// NewDoctorFixRunner validates options and prepares an executable step runner.
func NewDoctorFixRunner(opts DoctorFixOptions) (*DoctorFixRunner, *DoctorFixPlan, error) {
	repo, snap, issues, _, err := loadDoctorContextFresh(opts.WorkDir, opts.Base)
	if err != nil {
		return nil, nil, err
	}
	plan := buildDoctorFixPlan(repo, snap, issues, opts)
	if !plan.CanAutoFix {
		return nil, plan, fmt.Errorf("%s", plan.BlockReason)
	}

	if plan.NeedsMergedAction {
		action := effectiveMergedAction(opts, plan)
		if action == "" {
			return nil, plan, fmt.Errorf("escolha o destino após o merge (%s)", strings.Join(plan.MergedActionOptions, "|"))
		}
		opts.MergedAction = action
		plan = buildDoctorFixPlan(repo, snap, issues, opts)
		if !plan.CanAutoFix {
			return nil, plan, fmt.Errorf("%s", plan.BlockReason)
		}
	}

	if plan.NeedsBranchName {
		name := strings.TrimSpace(opts.NewBranchName)
		if name == "" {
			name = plan.SuggestedBranch
		}
		if name == "" {
			return nil, plan, fmt.Errorf("informe o nome da nova branch")
		}
		if !isValidBranchName(name) {
			return nil, plan, fmt.Errorf("nome de branch inválido: %s", name)
		}
		if repo.LocalBranchExists(name) {
			return nil, plan, fmt.Errorf("branch %q já existe — escolha outro nome", name)
		}
		opts.NewBranchName = name
	}
	if plan.NeedsBaseAction {
		action := strings.TrimSpace(opts.BaseAction)
		if action == "" {
			action = plan.SuggestedBaseAction
		}
		if action == "" {
			return nil, plan, fmt.Errorf("escolha a ação para a base (%s)", strings.Join(plan.BaseActionOptions, "|"))
		}
		opts.BaseAction = action
	}
	if plan.NeedsDestructiveConfirm && !opts.ConfirmDestructive {
		return nil, plan, fmt.Errorf("confirme a ação destrutiva (reset da base) para continuar")
	}

	plan = buildDoctorFixPlan(repo, snap, issues, opts)
	return &DoctorFixRunner{
		repo:  repo,
		base:  snap.Base,
		opts:  opts,
		steps: append([]DoctorFixStep{}, plan.Steps...),
	}, plan, nil
}

// Remaining returns how many steps are left (including the current next step).
func (r *DoctorFixRunner) Remaining() int {
	if r == nil {
		return 0
	}
	n := len(r.steps) - r.index
	if n < 0 {
		return 0
	}
	return n
}

// Next runs the next step. done=true when there are no more steps (or runner failed previously).
func (r *DoctorFixRunner) Next() (sr DoctorFixStepResult, done bool, err error) {
	if r == nil {
		return DoctorFixStepResult{}, true, fmt.Errorf("runner inválido")
	}
	if r.failed {
		return DoctorFixStepResult{}, true, fmt.Errorf("execução anterior falhou")
	}
	if r.index >= len(r.steps) {
		return DoctorFixStepResult{}, true, nil
	}
	step := r.steps[r.index]
	sr = DoctorFixStepResult{
		ID:      step.ID,
		Kind:    step.Kind,
		Title:   step.Title,
		Command: step.Command,
		Risk:    step.Risk,
		Status:  "ok",
	}
	if execErr := executeDoctorStep(r.repo, r.base, r.opts, step, &r.stashed); execErr != nil {
		sr.Status = "error"
		sr.Detail = execErr.Error()
		sr.ManualHint = manualHintForStep(step.Kind, r.base, r.opts.NewBranchName, execErr)
		r.failed = true
		r.index++
		return sr, r.index >= len(r.steps), nil
	}
	if step.Kind == DoctorStepStashPush {
		r.stashed = true
	}
	r.index++
	return sr, r.index >= len(r.steps), nil
}

// RunDoctorFix executes the planned steps. onStep is optional (progress callback).
func RunDoctorFix(opts DoctorFixOptions, onStep func(DoctorFixStepResult)) (*DoctorFixResult, error) {
	runner, plan, err := NewDoctorFixRunner(opts)
	if err != nil {
		msg := ""
		if plan != nil {
			msg = plan.BlockReason
		}
		if msg == "" {
			msg = err.Error()
		}
		return &DoctorFixResult{OK: false, Message: msg}, err
	}
	if len(runner.steps) == 0 {
		return &DoctorFixResult{OK: true, Message: "Nada a fazer"}, nil
	}

	result := &DoctorFixResult{OK: true, Steps: make([]DoctorFixStepResult, 0, len(runner.steps))}
	for {
		if onStep != nil && runner.index < len(runner.steps) {
			step := runner.steps[runner.index]
			onStep(DoctorFixStepResult{
				ID: step.ID, Kind: step.Kind, Title: step.Title,
				Command: step.Command, Risk: step.Risk, Status: "running",
			})
		}
		sr, done, nextErr := runner.Next()
		if nextErr != nil {
			return result, nextErr
		}
		if sr.ID != "" {
			result.Steps = append(result.Steps, sr)
			if onStep != nil {
				onStep(sr)
			}
			if sr.Status == "error" {
				result.OK = false
				result.Message = fmt.Sprintf("Parou em: %s", sr.Title)
				return result, nil
			}
		}
		if done {
			break
		}
	}
	result.Message = "Ajuste concluído"
	return result, nil
}

func loadDoctorContext(workDir, base string) (*gitpkg.Repo, *gitpkg.HealthSnapshot, []healthIssue, *prpkg.PRView, error) {
	return loadDoctorContextFresh(workDir, base)
}

type doctorContextCache struct {
	workDir string
	base    string
	repo    *gitpkg.Repo
	snap    *gitpkg.HealthSnapshot
	issues  []healthIssue
	pr      *prpkg.PRView
	at      time.Time
}

var (
	doctorCtxMu    sync.Mutex
	doctorCtxCache doctorContextCache
)

const doctorContextCacheTTL = 45 * time.Second

func loadDoctorContextCached(workDir, base string) (*gitpkg.Repo, *gitpkg.HealthSnapshot, []healthIssue, *prpkg.PRView, error) {
	workDir = strings.TrimSpace(workDir)
	base = normalizeDoctorBase(base)

	doctorCtxMu.Lock()
	defer doctorCtxMu.Unlock()
	if doctorCtxCache.repo != nil &&
		doctorCtxCache.workDir == workDir &&
		doctorCtxCache.base == base &&
		time.Since(doctorCtxCache.at) < doctorContextCacheTTL {
		return doctorCtxCache.repo, doctorCtxCache.snap, doctorCtxCache.issues, doctorCtxCache.pr, nil
	}

	repo, snap, issues, pr, err := loadDoctorContextUncached(workDir, base)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	doctorCtxCache = doctorContextCache{
		workDir: workDir,
		base:    base,
		repo:    repo,
		snap:    snap,
		issues:  issues,
		pr:      pr,
		at:      time.Now(),
	}
	return repo, snap, issues, pr, nil
}

func loadDoctorContextFresh(workDir, base string) (*gitpkg.Repo, *gitpkg.HealthSnapshot, []healthIssue, *prpkg.PRView, error) {
	workDir = strings.TrimSpace(workDir)
	base = normalizeDoctorBase(base)
	repo, snap, issues, pr, err := loadDoctorContextUncached(workDir, base)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	doctorCtxMu.Lock()
	doctorCtxCache = doctorContextCache{
		workDir: workDir,
		base:    base,
		repo:    repo,
		snap:    snap,
		issues:  issues,
		pr:      pr,
		at:      time.Now(),
	}
	doctorCtxMu.Unlock()
	return repo, snap, issues, pr, nil
}

func normalizeDoctorBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "main"
	}
	return strings.TrimPrefix(base, "origin/")
}

func loadDoctorContextUncached(workDir, base string) (*gitpkg.Repo, *gitpkg.HealthSnapshot, []healthIssue, *prpkg.PRView, error) {
	repo, err := openRepo(workDir)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("diretório atual não é um repositório git")
	}
	base = normalizeDoctorBase(base)
	snap, err := repo.CollectHealthSnapshot(base)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var openPR *prpkg.PRView
	if client, err := openPRClient(workDir); err == nil {
		// Meta only: CI checks are slow and unused by the fix planner.
		openPR, _ = client.ViewCurrentMeta()
	}
	issues := analyzeHealthIssues(snap, openPR)
	return repo, snap, issues, openPR, nil
}

func buildDoctorFixPlan(repo *gitpkg.Repo, snap *gitpkg.HealthSnapshot, issues []healthIssue, opts DoctorFixOptions) *DoctorFixPlan {
	plan := &DoctorFixPlan{
		Branch:     snap.Branch,
		Base:       snap.Base,
		CanAutoFix: true,
	}
	codes := map[string]bool{}
	for _, issue := range issues {
		codes[issue.Code] = true
		plan.IssueCodes = append(plan.IssueCodes, issue.Code)
	}
	if len(codes) == 0 {
		plan.Summary = "Nenhum problema para ajustar"
		plan.CanAutoFix = false
		plan.BlockReason = "Repositório saudável — nada a ajustar"
		return plan
	}

	merged := codes["work_on_merged_branch"]
	dirty := codes["dirty_tree"] || snap.IsDirty
	behind := codes["behind_remote"]
	diverged := codes["branch_diverged"]
	baseDiv := codes["base_diverged"] || (snap.BaseDivergence != nil &&
		(snap.BaseDivergence.LocalAhead > 0 || snap.BaseDivergence.RemoteAhead > 0))
	commitsOnBase := codes["commits_on_base"]
	blockedHistory := codes["unpushed_blocked"] || (snap.UnpushedBlockers != nil && snap.UnpushedBlockers.HasBlockers())
	needsStructuralFix := merged || behind || diverged || baseDiv || commitsOnBase || blockedHistory

	exists := func(name string) bool {
		if repo == nil {
			return false
		}
		return repo.LocalBranchExists(name)
	}
	plan.SuggestedBranch = SuggestDoctorBranchName(snap.Branch, snap.Base, exists)
	if name := strings.TrimSpace(opts.NewBranchName); name != "" {
		plan.SuggestedBranch = name
	}

	// Dirty alone on the base: move WIP to a feature branch (checkout -b keeps the tree).
	if dirty && !needsStructuralFix && snap.OnBase {
		plan.NeedsBranchName = true
		branchName := plan.SuggestedBranch
		plan.Steps = []DoctorFixStep{{
			ID:         "1-" + DoctorStepCreateBranch,
			Kind:       DoctorStepCreateBranch,
			Risk:       "ok",
			Title:      "Mover WIP para feature branch",
			Command:    fmt.Sprintf("git checkout -b %s", branchName),
			FromRef:    "", // HEAD
			IssueCodes: []string{"dirty_tree"},
		}}
		plan.Summary = "Sair da base: criar feature branch com o WIP"
		plan.Warnings = append(plan.Warnings,
			"Depois: Commit na feature branch, push e abra a Pull Request — não versione direto em "+snap.Base,
		)
		return plan
	}

	// Dirty alone on a usable feature branch: Commit is the correct path (not stash/pop).
	if dirty && !needsStructuralFix {
		plan.CanAutoFix = false
		plan.Summary = "Versionar o WIP com Commit nesta branch"
		plan.BlockReason = "Working tree dirty — use Commit para versionar nesta branch. Stash/pop não é o próximo passo."
		plan.Warnings = append(plan.Warnings,
			"Depois do commit: push e abra (ou atualize) a Pull Request",
		)
		return plan
	}

	// Soft reset keeps index+WIP; push does not need a clean tree; branch move keeps WIP via checkout -b.
	// Cleanup must NOT stash: soft reset works with a dirty tree, and stash -u races with the desktop
	// refresh (index.lock / "could not write index").
	baseActionHint := previewBaseAction(snap, opts)
	willStash := dirty && needsStructuralFix &&
		baseActionHint != BaseActionBranch &&
		baseActionHint != BaseActionPush &&
		baseActionHint != BaseActionCleanup

	var steps []DoctorFixStep
	add := func(step DoctorFixStep) {
		step.ID = fmt.Sprintf("%d-%s", len(steps)+1, step.Kind)
		steps = append(steps, step)
	}

	if willStash {
		add(DoctorFixStep{
			Kind: DoctorStepStashPush, Risk: "warn",
			Title:   "Salvar alterações locais (stash)",
			Command: `git stash push -u -m "openbench-doctor-wip"`,
			IssueCodes: []string{"dirty_tree", "work_on_merged_branch"},
		})
	}

	needsFetch := merged || behind || diverged || baseDiv
	if needsFetch {
		add(DoctorFixStep{
			Kind: DoctorStepFetch, Risk: "ok",
			Title:   "Atualizar refs remotas",
			Command: "git fetch --prune origin",
			IssueCodes: []string{"work_on_merged_branch", "behind_remote", "branch_diverged", "base_diverged"},
		})
	}

	branchName := plan.SuggestedBranch
	mergedAction := ""

	// Base divergence handling (also when leaving a merged feature that needs a fresh base).
	if baseDiv && snap.BaseDivergence != nil {
		appendBaseDivergenceSteps(plan, snap, opts, add)
	} else if merged {
		add(DoctorFixStep{
			Kind: DoctorStepUpdateBase, Risk: "ok",
			Title:      fmt.Sprintf("Atualizar base local %s", snap.Base),
			Command:    fmt.Sprintf("git fetch origin %s:%s", snap.Base, snap.Base),
			IssueCodes: []string{"work_on_merged_branch"},
		})
	}

	if merged {
		plan.NeedsMergedAction = true
		plan.MergedActionOptions = []string{MergedActionReturnBase, MergedActionContinue}
		// Finished work (clean) → prefer sync back to base. WIP → prefer new feature branch.
		plan.SuggestedMergedAction = MergedActionReturnBase
		if dirty {
			plan.SuggestedMergedAction = MergedActionContinue
		}
		mergedAction = effectiveMergedAction(opts, plan)
		plan.SuggestedMergedAction = mergedAction

		switch mergedAction {
		case MergedActionReturnBase:
			plan.NeedsBranchName = false
			add(DoctorFixStep{
				Kind: DoctorStepCheckout, Risk: "ok",
				Title:      fmt.Sprintf("Voltar para %s", snap.Base),
				Command:    fmt.Sprintf("git checkout %s", snap.Base),
				FromRef:    snap.Base,
				IssueCodes: []string{"work_on_merged_branch"},
			})
			// Do not stash-pop onto the base: "terminar" should leave main clean.
			// WIP stays in stash until the user continues on a new feature branch.
			if willStash {
				plan.Warnings = append(plan.Warnings,
					"WIP fica no stash (não reaplicado em "+snap.Base+") — use \"Nova feature branch\" para continuar o trabalho",
				)
			}
			plan.Warnings = append(plan.Warnings,
				fmt.Sprintf("Não faça push/PR de novo em %s — a PR já foi mergeada", snap.Branch),
				"Depois: use Hygiene para limpar branches mergeadas antigas",
			)
			if willStash {
				plan.Summary = fmt.Sprintf("Salvar WIP no stash, atualizar %s e voltar para a base", snap.Base)
			} else {
				plan.Summary = fmt.Sprintf("Atualizar %s e voltar para a base (repo sincronizado)", snap.Base)
			}
		default: // continue
			plan.NeedsBranchName = true
			add(DoctorFixStep{
				Kind: DoctorStepCreateBranch, Risk: "ok",
				Title:      "Criar nova feature branch",
				Command:    fmt.Sprintf("git checkout -b %s %s", branchName, snap.Base),
				FromRef:    snap.Base,
				IssueCodes: []string{"work_on_merged_branch"},
			})
			if willStash {
				add(DoctorFixStep{
					Kind: DoctorStepStashPop, Risk: "warn",
					Title:      "Reaplicar alterações na branch nova",
					Command:    "git stash pop",
					IssueCodes: []string{"work_on_merged_branch"},
				})
			}
			plan.Warnings = append(plan.Warnings,
				fmt.Sprintf("Não faça push/PR de novo em %s — a PR já foi mergeada", snap.Branch))
			if willStash {
				plan.Summary = "Salvar WIP, atualizar base e continuar em uma branch nova"
			} else {
				plan.Summary = "Atualizar base e continuar em uma branch nova"
			}
		}
	} else {
		if commitsOnBase {
			plan.NeedsBranchName = true
			add(DoctorFixStep{
				Kind: DoctorStepCreateBranch, Risk: "ok",
				Title:      "Mover trabalho para feature branch",
				Command:    fmt.Sprintf("git checkout -b %s", branchName),
				FromRef:    "", // HEAD
				IssueCodes: []string{"commits_on_base"},
			})
		}

		if diverged {
			add(DoctorFixStep{
				Kind: DoctorStepRebaseUpstream, Risk: "warn",
				Title:      "Rebase da branch no upstream",
				Command:    "git rebase @{u}",
				IssueCodes: []string{"branch_diverged"},
			})
		} else if behind {
			add(DoctorFixStep{
				Kind: DoctorStepPullFF, Risk: "ok",
				Title:      "Fast-forward da branch atual",
				Command:    "git pull --ff-only",
				IssueCodes: []string{"behind_remote"},
			})
		}

		if willStash {
			add(DoctorFixStep{
				Kind: DoctorStepStashPop, Risk: "warn",
				Title:      "Reaplicar alterações (stash pop)",
				Command:    "git stash pop",
				IssueCodes: []string{"dirty_tree"},
			})
		}

		if plan.Summary == "" {
			plan.Summary = "Aplicar correções determinísticas aos problemas detectados"
		}
	}

	// Re-number IDs after helper appends
	for i := range steps {
		steps[i].ID = fmt.Sprintf("%d-%s", i+1, steps[i].Kind)
	}
	plan.Steps = steps

	if len(steps) == 0 {
		plan.CanAutoFix = false
		plan.BlockReason = "Não há passos seguros automáticos para este estado — use Commit ou ajuste manual"
		plan.Summary = "Ajuste automático indisponível"
	}
	if plan.NeedsBaseAction {
		plan.Warnings = appendUnique(plan.Warnings, "Confirme a ação da base antes de executar")
	}
	return plan
}

func appendBaseDivergenceSteps(
	plan *DoctorFixPlan,
	snap *gitpkg.HealthSnapshot,
	opts DoctorFixOptions,
	add func(DoctorFixStep),
) {
	div := snap.BaseDivergence
	if div == nil {
		return
	}
	allDiscardable := baseCommitsDiscardable(div)
	branchName := plan.SuggestedBranch

	switch {
	case div.RemoteAhead > 0 && div.LocalAhead == 0:
		plan.SuggestedBaseAction = BaseActionUpdate
		add(DoctorFixStep{
			Kind: DoctorStepUpdateBase, Risk: "ok",
			Title:      fmt.Sprintf("Atualizar base local %s", snap.Base),
			Command:    fmt.Sprintf("git fetch origin %s:%s", snap.Base, snap.Base),
			IssueCodes: []string{"base_diverged"},
		})
	case div.LocalAhead > 0 && div.RemoteAhead == 0 && allDiscardable:
		plan.SuggestedBaseAction = BaseActionReset
		plan.NeedsDestructiveConfirm = true
		add(DoctorFixStep{
			Kind: DoctorStepResetBase, Risk: "destructive",
			Title:      fmt.Sprintf("Descartar commits locais da base %s", snap.Base),
			Command:    fmt.Sprintf("git reset --hard origin/%s", snap.Base),
			IssueCodes: []string{"base_diverged", "build_artifacts"},
		})
		plan.Warnings = appendUnique(plan.Warnings, "Reset da base é destrutivo — commits locais da base serão descartados")
	case div.LocalAhead > 0 && div.RemoteAhead == 0:
		plan.NeedsBaseAction = true
		blocked := snap.UnpushedBlockers != nil && snap.UnpushedBlockers.HasBlockers()
		if blocked {
			plan.BaseActionOptions = []string{BaseActionCleanup, BaseActionBranch, BaseActionPush, BaseActionReset}
			plan.SuggestedBaseAction = BaseActionCleanup
		} else {
			plan.BaseActionOptions = []string{BaseActionBranch, BaseActionPush, BaseActionReset}
			plan.SuggestedBaseAction = BaseActionBranch
		}
		action := effectiveBaseAction(opts, plan)
		switch action {
		case BaseActionCleanup:
			appendCleanupHistorySteps(plan, snap, add)
		case BaseActionReset:
			plan.NeedsDestructiveConfirm = true
			plan.NeedsBranchName = false
			add(DoctorFixStep{
				Kind: DoctorStepResetBase, Risk: "destructive",
				Title:      fmt.Sprintf("Reset da base %s para origin", snap.Base),
				Command:    fmt.Sprintf("git reset --hard origin/%s", snap.Base),
				IssueCodes: []string{"base_diverged"},
			})
			plan.Summary = fmt.Sprintf("Descartar commits locais de %s e alinhar com origin", snap.Base)
		case BaseActionPush:
			plan.NeedsBranchName = false
			add(DoctorFixStep{
				Kind: DoctorStepPushBase, Risk: "warn",
				Title:      fmt.Sprintf("Push da base %s", snap.Base),
				Command:    fmt.Sprintf("git push -u origin %s", snap.Base),
				IssueCodes: []string{"base_diverged"},
			})
			plan.Summary = fmt.Sprintf("Publicar commits locais de %s no remoto", snap.Base)
			plan.Warnings = appendUnique(plan.Warnings,
				"Push direto na base — use só se a intenção for publicar sem Pull Request",
			)
			if blocked {
				plan.Warnings = appendUnique(plan.Warnings,
					"Histórico local tem arquivo grande ou lixo — o push provavelmente será rejeitado; prefira limpeza",
				)
			}
		default: // branch
			plan.NeedsBranchName = true
			add(DoctorFixStep{
				Kind: DoctorStepCreateBranch, Risk: "ok",
				Title:      "Mover commits para feature branch",
				Command:    fmt.Sprintf("git checkout -b %s", branchName),
				FromRef:    "", // HEAD (keeps dirty working tree)
				IssueCodes: []string{"base_diverged"},
			})
			add(DoctorFixStep{
				Kind: DoctorStepResetBase, Risk: "warn",
				Title:      fmt.Sprintf("Alinhar base local %s com origin", snap.Base),
				Command:    fmt.Sprintf("git branch -f %s origin/%s", snap.Base, snap.Base),
				IssueCodes: []string{"base_diverged"},
			})
			plan.Summary = fmt.Sprintf("Mover commits de %s para feature branch e alinhar a base com origin", snap.Base)
			plan.Warnings = appendUnique(plan.Warnings,
				"Depois: Commit do WIP na feature, push e abra a PR — não faça push direto de "+snap.Base,
			)
			if blocked {
				plan.Warnings = appendUnique(plan.Warnings,
					"Atenção: os commits movidos ainda têm arquivo grande/lixo — limpe o histórico antes do push da feature",
				)
			}
		}
	default:
		plan.NeedsBaseAction = true
		plan.BaseActionOptions = []string{BaseActionRebase, BaseActionReset}
		if allDiscardable {
			plan.SuggestedBaseAction = BaseActionReset
		} else {
			plan.SuggestedBaseAction = BaseActionRebase
		}
		action := effectiveBaseAction(opts, plan)
		if action == BaseActionReset {
			plan.NeedsDestructiveConfirm = true
			add(DoctorFixStep{
				Kind: DoctorStepResetBase, Risk: "destructive",
				Title:      fmt.Sprintf("Reset da base %s (descarta local)", snap.Base),
				Command:    fmt.Sprintf("git reset --hard origin/%s", snap.Base),
				IssueCodes: []string{"base_diverged", "build_artifacts"},
			})
		} else {
			add(DoctorFixStep{
				Kind: DoctorStepCheckout, Risk: "ok",
				Title:      fmt.Sprintf("Checkout %s", snap.Base),
				Command:    fmt.Sprintf("git checkout %s", snap.Base),
				FromRef:    snap.Base,
				IssueCodes: []string{"base_diverged"},
			})
			add(DoctorFixStep{
				Kind: DoctorStepRebaseBase, Risk: "warn",
				Title:      fmt.Sprintf("Rebase de %s em origin/%s", snap.Base, snap.Base),
				Command:    fmt.Sprintf("git rebase origin/%s", snap.Base),
				IssueCodes: []string{"base_diverged"},
			})
		}
	}
}

// previewBaseAction returns the base action that will be used (opts override or suggestion).
func previewBaseAction(snap *gitpkg.HealthSnapshot, opts DoctorFixOptions) string {
	if action := strings.TrimSpace(opts.BaseAction); action != "" {
		return action
	}
	div := snap.BaseDivergence
	if div == nil {
		return ""
	}
	switch {
	case div.RemoteAhead > 0 && div.LocalAhead == 0:
		return BaseActionUpdate
	case div.LocalAhead > 0 && div.RemoteAhead == 0 && baseCommitsDiscardable(div):
		return BaseActionReset
	case div.LocalAhead > 0 && div.RemoteAhead == 0:
		if snap.UnpushedBlockers != nil && snap.UnpushedBlockers.HasBlockers() {
			return BaseActionCleanup
		}
		return BaseActionBranch
	case baseCommitsDiscardable(div):
		return BaseActionReset
	default:
		return BaseActionRebase
	}
}

func appendCleanupHistorySteps(plan *DoctorFixPlan, snap *gitpkg.HealthSnapshot, add func(DoctorFixStep)) {
	plan.NeedsBranchName = false
	plan.NeedsDestructiveConfirm = true
	// Mixed reset (not soft): moves HEAD to origin/<base>, resets the index to match,
	// and keeps all files on disk unstaged. Soft reset was wrong — it staged ~entire trees
	// (node_modules/vendor/etc.) and selective rm --cached often failed mid-flight.
	add(DoctorFixStep{
		Kind: DoctorStepMixedReset, Risk: "warn",
		Title:      fmt.Sprintf("Desfazer commits locais (mixed reset → origin/%s)", snap.Base),
		Command:    fmt.Sprintf("git reset --mixed origin/%s", snap.Base),
		FromRef:    "origin/" + snap.Base,
		IssueCodes: []string{"unpushed_blocked", "base_diverged"},
	})
	plan.Summary = "Limpar histórico local bloqueador (mixed reset) e preparar Commit seletivo"
	plan.Warnings = appendUnique(plan.Warnings,
		"Os commits locais não publicados são desfeitos; os arquivos permanecem no disco, fora do stage",
	)
	plan.Warnings = appendUnique(plan.Warnings,
		"Depois: adicione só o código útil (não node_modules/.env/.dmg), Commit, depois push da feature/PR",
	)
	plan.Warnings = appendUnique(plan.Warnings,
		"Não faça git add . — isso recolocaria lixo no stage",
	)
}

func effectiveBaseAction(opts DoctorFixOptions, plan *DoctorFixPlan) string {
	action := strings.TrimSpace(opts.BaseAction)
	if action == "" {
		action = plan.SuggestedBaseAction
	}
	return action
}

func effectiveMergedAction(opts DoctorFixOptions, plan *DoctorFixPlan) string {
	action := strings.TrimSpace(opts.MergedAction)
	if action == "" {
		action = plan.SuggestedMergedAction
	}
	switch action {
	case MergedActionContinue, MergedActionReturnBase:
		return action
	default:
		return plan.SuggestedMergedAction
	}
}

func baseCommitsDiscardable(div *gitpkg.DivergenceReport) bool {
	if div == nil || div.LocalAhead == 0 || len(div.LocalAnalyses) == 0 {
		return false
	}
	for _, a := range div.LocalAnalyses {
		if !a.LikelyDiscardable {
			return false
		}
	}
	return true
}

func appendUnique(list []string, item string) []string {
	for _, s := range list {
		if s == item {
			return list
		}
	}
	return append(list, item)
}

func executeDoctorStep(repo *gitpkg.Repo, base string, opts DoctorFixOptions, step DoctorFixStep, stashed *bool) error {
	switch step.Kind {
	case DoctorStepStashPush, DoctorStepStashPop, DoctorStepSoftReset, DoctorStepMixedReset,
		DoctorStepUnstageBlockers, DoctorStepResetBase, DoctorStepCreateBranch, DoctorStepCheckout:
		if err := repo.EnsureWritableIndex(); err != nil {
			return err
		}
	}

	switch step.Kind {
	case DoctorStepStashPush:
		if err := repo.StashPushAll("openbench-doctor-wip"); err != nil {
			return err
		}
		*stashed = true
		return nil
	case DoctorStepFetch:
		return repo.FetchPrune()
	case DoctorStepUpdateBase:
		_, err := repo.UpdateLocalBranchFromOrigin(base)
		return err
	case DoctorStepCheckout:
		target := step.FromRef
		if target == "" {
			target = base
		}
		return repo.Checkout(target)
	case DoctorStepCreateBranch:
		name := strings.TrimSpace(opts.NewBranchName)
		if name == "" {
			return fmt.Errorf("nome da branch vazio")
		}
		from := strings.TrimSpace(step.FromRef)
		if from == "" {
			current, err := repo.CurrentBranch()
			if err != nil {
				return err
			}
			from = current
		}
		return repo.CreateBranch(name, from)
	case DoctorStepStashPop:
		if stashed != nil && !*stashed {
			return nil
		}
		return repo.StashPop()
	case DoctorStepPullFF:
		return repo.PullFFOnly()
	case DoctorStepRebaseUpstream:
		return repo.RebaseUpstream()
	case DoctorStepResetBase:
		return repo.ResetBranchToOrigin(base)
	case DoctorStepRebaseBase:
		if err := repo.Checkout(base); err != nil {
			return err
		}
		return repo.RebaseOnto("origin/" + base)
	case DoctorStepPushBase:
		return repo.PushBranch(base)
	case DoctorStepSoftReset:
		ref := strings.TrimSpace(step.FromRef)
		if ref == "" {
			ref = "origin/" + base
		}
		return repo.SoftReset(ref)
	case DoctorStepMixedReset:
		ref := strings.TrimSpace(step.FromRef)
		if ref == "" {
			ref = "origin/" + base
		}
		return repo.MixedReset(ref)
	case DoctorStepUnstageBlockers:
		return repo.RmCached(step.Paths)
	default:
		return fmt.Errorf("passo desconhecido: %s", step.Kind)
	}
}

func manualHintForStep(kind, base, newBranch string, err error) string {
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	lower := strings.ToLower(errText)
	conflict := strings.Contains(lower, "conflict") ||
		strings.Contains(lower, "conflito") ||
		strings.Contains(lower, "could not apply") ||
		strings.Contains(lower, "needs merge")

	switch kind {
	case DoctorStepStashPush:
		if strings.Contains(lower, "could not write index") || strings.Contains(lower, "index.lock") {
			return "Git não gravou o index (.git/index.lock). Feche outros clientes git, confirme que não há operação em andamento, remova .git/index.lock se estiver órfão e rode o Doctor de novo. O app também tenta limpar lock antigo automaticamente — reinicie se ainda falhar."
		}
		return "Stash falhou. Verifique git status e .git/index.lock; corrija o bloqueio e rode o Doctor novamente."
	case DoctorStepStashPop:
		if conflict {
			return "Conflito no stash pop. Resolva os arquivos, depois: git add <arquivos>. Se o stash ainda existir: git stash drop. Rode o Doctor de novo ao terminar."
		}
		if strings.Contains(lower, "could not write index") || strings.Contains(lower, "index.lock") {
			return "Não foi possível gravar o index no stash pop. Remova .git/index.lock se estiver órfão e tente git stash pop de novo."
		}
		return "O stash pode continuar em git stash list. Reaplique com git stash pop quando a branch estiver pronta."
	case DoctorStepRebaseUpstream, DoctorStepRebaseBase:
		if conflict {
			return fmt.Sprintf("Conflito de rebase. 1) Resolva arquivos 2) git add . 3) git rebase --continue. Abortar: git rebase --abort. Base: %s", base)
		}
		return "Verifique git status. Para abortar: git rebase --abort."
	case DoctorStepPullFF:
		return "Fast-forward indisponível. Caminho seguro: git fetch && git rebase @{u} (ou merge), resolvendo conflitos se aparecerem."
	case DoctorStepUpdateBase:
		return fmt.Sprintf("A base %s não pôde ser atualizada só com ff. Escolha rebase ou reset no Doctor, ou ajuste manualmente em %s.", base, base)
	case DoctorStepCreateBranch:
		return fmt.Sprintf("Escolha outro nome (ex.: %s) ou remova a branch local conflitante se for segura.", newBranch)
	case DoctorStepResetBase:
		return fmt.Sprintf("Reset não aplicado. Confirme que origin/%s existe (git fetch) e que não há operação git em andamento.", base)
	case DoctorStepPushBase:
		if strings.Contains(lower, "gh001") ||
			strings.Contains(lower, "file size limit") ||
			strings.Contains(lower, "large files detected") ||
			strings.Contains(lower, "git-lfs") {
			return "Push rejeitado por arquivo >100MB no histórico. No Doctor Fix, escolha a ação “limpeza” (stash → reset --soft → git rm --cached) e confirme. Depois Commit limpo + feature branch/PR."
		}
		if strings.Contains(lower, "pre-receive hook declined") || strings.Contains(lower, "remote rejected") {
			return "Remoto rejeitou o push. Se for arquivo grande/lixo no histórico, use a ação “limpeza” no Doctor. Caso contrário, verifique proteção de branch/permissões."
		}
		return fmt.Sprintf("Push de %s falhou. Verifique rede, permissões e proteção da branch; corrija e rode o Doctor de novo.", base)
	case DoctorStepSoftReset:
		return fmt.Sprintf("Soft reset falhou. Confirme que origin/%s existe (git fetch) e que não há rebase/merge em andamento.", base)
	case DoctorStepMixedReset:
		return fmt.Sprintf("Mixed reset falhou. Confirme que origin/%s existe (git fetch), remova .git/index.lock se órfão, e tente de novo.", base)
	case DoctorStepUnstageBlockers:
		return "Falha ao tirar paths do index. Rode git status; se .git/index.lock existir e estiver órfão, remova-o e tente de novo."
	default:
		return "Intervenha com git status, corrija o bloqueio e rode o Doctor novamente."
	}
}

// SuggestDoctorBranchName proposes an editable default feature branch name.
func SuggestDoctorBranchName(current, base string, exists func(string) bool) string {
	cur := strings.TrimSpace(current)
	if cur == "" || cur == "HEAD" || cur == base {
		cur = "feature/ajuste"
		if exists == nil || !exists(cur) {
			return cur
		}
	}
	candidate := nextBranchName(cur)
	if exists == nil {
		return candidate
	}
	for i := 0; i < 50 && exists(candidate); i++ {
		candidate = nextBranchName(candidate)
	}
	return candidate
}

var branchSuffixRe = regexp.MustCompile(`^(.*)-(\d+)$`)

func nextBranchName(name string) string {
	if m := branchSuffixRe.FindStringSubmatch(name); len(m) == 3 {
		n, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%s-%d", m[1], n+1)
	}
	return name + "-2"
}

func isValidBranchName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "-") || strings.Contains(name, " ") {
		return false
	}
	if strings.Contains(name, "..") || strings.HasSuffix(name, ".lock") {
		return false
	}
	return true
}
