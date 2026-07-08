package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/laerciocrestani/gitai/internal/ai"
	"github.com/laerciocrestani/gitai/internal/config"
	"github.com/laerciocrestani/gitai/internal/formatter"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	prpkg "github.com/laerciocrestani/gitai/internal/pr"
	"github.com/laerciocrestani/gitai/internal/ui"
)

type Options struct {
	NoAdd               bool
	DryRun              bool
	Draft               bool
	Base                string
	Verbose             bool
	UI                  *ui.Session
	Progress            Progress
	CachedCommitMessage string
	CachedPRSuggestion  *ai.PRSuggestion
	CachedPRBody        string
}

type Result struct {
	Suggestion   *ai.CommitSuggestion
	PRSuggestion *ai.PRSuggestion
	Message      string
	PRBody       string
	PRURL        string
	PRPreview    string
}

func (o Options) session(command string) *ui.Session {
	if o.UI != nil {
		return o.UI
	}
	return ui.New(command, o.DryRun)
}

func (o Options) reporter(command string) Progress {
	if o.Progress != nil {
		return o.Progress
	}
	return o.session(command)
}

func printUsage(prog Progress, cfg *config.Config, summary ai.UsageSummary) {
	if prog == nil || len(summary.Records) == 0 {
		return
	}
	for _, line := range summary.FormatLines(cfg) {
		prog.Detail(line)
	}
}

func RunCommit(ctx context.Context, opts Options) (*Result, error) {
	prog := opts.reporter("commit")
	if opts.Progress == nil {
		opts.session("commit").Header()
	}

	result, provider, err := commitFlow(ctx, opts, prog)
	if err != nil {
		return nil, err
	}

	if provider != nil {
		cfg, _ := config.Load()
		if cfg != nil {
			recordAIUsage("commit", cfg, provider.UsageStats())
		}
		printUsage(prog, cfg, provider.UsageStats())
	}

	if result != nil && result.Message != "" && !opts.DryRun {
		prog.Detail(formatter.TitleLine(result.Message))
	}
	prog.Success("Ready! 🚀")
	return result, nil
}

// PreviewCommit gera sugestão de commit sem gravar (DryRun).
func PreviewCommit(ctx context.Context, opts Options) (*Result, error) {
	opts.DryRun = true
	result, provider, err := commitFlow(ctx, opts, opts.reporter("commit"))
	if err != nil {
		return nil, err
	}
	if provider != nil {
		cfg, _ := config.Load()
		if cfg != nil {
			recordAIUsage("commit", cfg, provider.UsageStats())
		}
	}
	return result, nil
}

// ConfirmCommit grava mensagem já sugerida pela IA.
func ConfirmCommit(ctx context.Context, preview *Result, opts Options) (*Result, error) {
	if preview == nil || preview.Message == "" {
		return nil, fmt.Errorf("nenhuma mensagem de commit para confirmar")
	}
	opts.DryRun = false
	opts.CachedCommitMessage = preview.Message
	prog := opts.reporter("commit")
	result, _, err := commitFlow(ctx, opts, prog)
	if err != nil {
		return nil, err
	}
	prog.Success("Commit criado ✓")
	return result, nil
}

// PreviewPush simula commit (se necessário) + push sem gravar (DryRun).
func PreviewPush(ctx context.Context, opts Options) (*Result, error) {
	opts.DryRun = true
	return RunPush(ctx, opts)
}

// ConfirmPush executa commit (se necessário) + push após preview.
func ConfirmPush(ctx context.Context, preview *Result, opts Options) (*Result, error) {
	opts.DryRun = false
	if preview != nil && preview.Message != "" {
		opts.CachedCommitMessage = preview.Message
	}
	prog := opts.reporter("push")
	result, err := RunPush(ctx, opts)
	if err != nil {
		return nil, err
	}
	prog.Success("Push concluído ✓")
	return result, nil
}

// PreviewPR gera sugestão de PR sem push/create (DryRun).
func PreviewPR(ctx context.Context, opts Options) (*Result, error) {
	opts.DryRun = true
	return RunPR(ctx, opts)
}

// ConfirmPR cria o PR a partir de preview já gerado.
func ConfirmPR(ctx context.Context, preview *Result, draft bool, opts Options) (*Result, error) {
	if preview == nil || preview.PRSuggestion == nil {
		return nil, fmt.Errorf("nenhuma sugestão de PR para confirmar")
	}
	opts.DryRun = false
	opts.Draft = draft
	opts.CachedPRSuggestion = preview.PRSuggestion
	opts.CachedPRBody = preview.PRBody
	if preview.Message != "" {
		opts.CachedCommitMessage = preview.Message
	}
	return RunPR(ctx, opts)
}

func RunPush(ctx context.Context, opts Options) (*Result, error) {
	prog := opts.reporter("push")
	if opts.Progress == nil {
		opts.session("push").Header()
	}

	repo, err := gitpkg.New()
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	if !opts.NoAdd {
		if err := prog.Step("Staging changes", func() error {
			if opts.DryRun {
				prog.Detail("git add .")
				return nil
			}
			return repo.AddAll()
		}); err != nil {
			return nil, err
		}
	}

	var result *Result
	var provider ai.Provider

	diff, err := repo.DiffForCommit()
	if err != nil {
		return nil, err
	}

	if diff != "" {
		pushOpts := opts
		pushOpts.NoAdd = true
		pushOpts.Progress = prog
		result, provider, err = commitFlow(ctx, pushOpts, prog)
		if err != nil {
			return nil, err
		}
	} else {
		prog.Info("No pending changes — pushing existing commits")
		result = &Result{}
	}

	if err := prog.Step("Pushing to origin", func() error {
		if opts.DryRun {
			prog.Detail("git push -u origin HEAD")
			return nil
		}
		return repo.Push()
	}); err != nil {
		return nil, err
	}

	if provider != nil {
		cfg, _ := config.Load()
		if cfg != nil {
			recordAIUsage("push", cfg, provider.UsageStats())
		}
		printUsage(prog, cfg, provider.UsageStats())
	}
	prog.Success("Ready! 🚀")
	return result, nil
}

func RunPR(ctx context.Context, opts Options) (*Result, error) {
	prog := opts.reporter("pr")
	if opts.Progress == nil {
		opts.session("pr").Header()
	}
	ctx = withAINotices(ctx, prog)

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	base := opts.Base
	if base == "" {
		base = cfg.BaseBranch
	}

	repo, err := gitpkg.New()
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	var resolvedBase string
	if err := prog.Step("Resolving base branch", func() error {
		var err error
		resolvedBase, err = repo.ResolveBase(base)
		return err
	}); err != nil {
		return nil, err
	}

	var provider ai.Provider
	if opts.CachedPRSuggestion == nil {
		var err error
		provider, err = ai.New(cfg)
		if err != nil {
			return nil, err
		}
	}

	if !opts.NoAdd {
		if err := prog.Step("Staging changes", func() error {
			if opts.DryRun {
				prog.Detail("git add .")
				return nil
			}
			return repo.AddAll()
		}); err != nil {
			return nil, err
		}
	}

	result := &Result{}

	hasStaged, err := repo.HasStagedChanges()
	if err != nil {
		return nil, err
	}

	if hasStaged || opts.CachedCommitMessage != "" {
		commitResult, err := commitStaged(ctx, cfg, repo, opts, provider, prog)
		if err != nil {
			return nil, err
		}
		result.Suggestion = commitResult.Suggestion
		result.Message = commitResult.Message
	} else {
		same, err := repo.IsSameAsBase(resolvedBase)
		if err != nil {
			return nil, err
		}
		if same {
			return nil, fmt.Errorf("nenhuma alteração em relação à %s", baseForGH(resolvedBase))
		}
		prog.Info("Using existing commits on branch")
	}

	if err := prog.Step("Pushing to origin", func() error {
		if opts.DryRun {
			prog.Detail("git push -u origin HEAD")
			return nil
		}
		return repo.Push()
	}); err != nil {
		return nil, err
	}

	var branch string
	if err := prog.Step("Reading branch diff", func() error {
		var err error
		branch, err = repo.CurrentBranch()
		return err
	}); err != nil {
		return nil, err
	}

	var prSuggestion *ai.PRSuggestion
	if opts.CachedPRSuggestion != nil {
		prSuggestion = opts.CachedPRSuggestion
	} else {
		diff, err := repo.DiffBranch(resolvedBase)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(diff) == "" {
			return nil, fmt.Errorf("diff vazio em relação à %s", baseForGH(resolvedBase))
		}

		commitLog, err := repo.LogOnBranch(resolvedBase)
		if err != nil {
			return nil, err
		}

		prog.Detail(ai.DescribePreparedInput(cfg, diff, "pr"))
		if err := prog.Step("Thinking", func() error {
			prSuggestion, err = provider.SuggestPR(ctx, diff, branch, baseForGH(resolvedBase), cfg.Language, commitLog)
			return err
		}); err != nil {
			return nil, err
		}
		if line := ai.FormatLatestUsage(provider.UsageStats()); line != "" {
			prog.Detail(line)
		}
	}

	result.PRSuggestion = prSuggestion

	if opts.Verbose {
		printPRVerbose(prSuggestion)
	}

	prClient, err := prpkg.New()
	if err != nil {
		return nil, err
	}

	if opts.DryRun {
		preview := prClient.PreviewCreate(prSuggestion, resolvedBase, opts.Draft, opts.CachedPRBody)
		result.PRPreview = preview
		prog.Detail(preview)
		if provider != nil {
			recordAIUsage("pr", cfg, provider.UsageStats())
			printUsage(prog, cfg, provider.UsageStats())
		}
		prog.Success("Ready! 🚀")
		return result, nil
	}

	var url string
	if err := prog.Step("Creating Pull Request", func() error {
		url, err = prClient.Create(prSuggestion, resolvedBase, opts.Draft, opts.CachedPRBody)
		return err
	}); err != nil {
		return nil, err
	}

	result.PRURL = url
	prog.Detail(url)
	if provider != nil {
		recordAIUsage("pr", cfg, provider.UsageStats())
		printUsage(prog, cfg, provider.UsageStats())
	}
	prog.Success("Ready! 🚀")
	return result, nil
}

func commitFlow(ctx context.Context, opts Options, prog Progress) (*Result, ai.Provider, error) {
	ctx = withAINotices(ctx, prog)
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	repo, err := gitpkg.New()
	if err != nil {
		return nil, nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	if !opts.NoAdd {
		if err := prog.Step("Staging changes", func() error {
			if opts.DryRun {
				prog.Detail("git add .")
				return nil
			}
			return repo.AddAll()
		}); err != nil {
			return nil, nil, err
		}
	}

	var diff string
	if opts.CachedCommitMessage == "" {
		if err := prog.Step("Reading git diff", func() error {
			var err error
			diff, err = repo.DiffForCommit()
			if err != nil {
				return err
			}
			if diff == "" {
				return fmt.Errorf("nenhuma alteração para commitar")
			}
			return nil
		}); err != nil {
			return nil, nil, err
		}
	}

	provider, err := ai.New(cfg)
	if err != nil {
		return nil, nil, err
	}

	var suggestion *ai.CommitSuggestion
	var message string

	if opts.CachedCommitMessage != "" {
		message = opts.CachedCommitMessage
	} else {
		prog.Detail(ai.DescribePreparedInput(cfg, diff, "commit"))
		if err := prog.Step("Thinking", func() error {
			suggestion, err = provider.SuggestCommit(ctx, diff, cfg.Language)
			return err
		}); err != nil {
			return nil, nil, err
		}
		if line := ai.FormatLatestUsage(provider.UsageStats()); line != "" {
			prog.Detail(line)
		}
		message = formatter.FormatCommit(suggestion, cfg.CoAuthor)
	}

	result := &Result{
		Suggestion: suggestion,
		Message:    message,
	}

	if opts.Verbose {
		printCommitVerbose(suggestion, message)
	}

	if err := prog.Step("Writing Conventional Commit", func() error {
		if opts.DryRun {
			prog.Detail("git commit -m " + quoteMessage(message))
			return nil
		}
		return repo.Commit(message)
	}); err != nil {
		return nil, nil, err
	}

	if opts.CachedCommitMessage != "" {
		return result, nil, nil
	}
	return result, provider, nil
}

func commitStaged(ctx context.Context, cfg *config.Config, repo *gitpkg.Repo, opts Options, provider ai.Provider, prog Progress) (*Result, error) {
	var suggestion *ai.CommitSuggestion
	var message string

	if opts.CachedCommitMessage != "" {
		message = opts.CachedCommitMessage
	} else {
		if provider == nil {
			var err error
			provider, err = ai.New(cfg)
			if err != nil {
				return nil, err
			}
		}

		diff, err := repo.DiffStaged()
		if err != nil {
			return nil, err
		}

		prog.Detail(ai.DescribePreparedInput(cfg, diff, "commit"))
		if err := prog.Step("Thinking", func() error {
			suggestion, err = provider.SuggestCommit(ctx, diff, cfg.Language)
			return err
		}); err != nil {
			return nil, err
		}
		if line := ai.FormatLatestUsage(provider.UsageStats()); line != "" {
			prog.Detail(line)
		}
		message = formatter.FormatCommit(suggestion, cfg.CoAuthor)
	}

	result := &Result{
		Suggestion: suggestion,
		Message:    message,
	}

	if opts.Verbose {
		printCommitVerbose(suggestion, message)
	}

	if err := prog.Step("Writing Conventional Commit", func() error {
		if opts.DryRun {
			prog.Detail("git commit -m " + quoteMessage(message))
			return nil
		}
		return repo.Commit(message)
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func printCommitVerbose(s *ai.CommitSuggestion, message string) {
	fmt.Println("--- Sugestão da IA (commit) ---")
	fmt.Printf("type: %s\nscope: %s\ntitle: %s\n", s.Type, s.Scope, s.Title)
	for _, b := range s.Body {
		fmt.Printf("  - %s\n", b)
	}
	fmt.Println("--- Mensagem ---")
	fmt.Println(message)
}

func printPRVerbose(s *ai.PRSuggestion) {
	fmt.Println("--- Sugestão da IA (PR) ---")
	fmt.Printf("title: %s\n", s.Title)
	fmt.Println("summary:")
	for _, line := range s.Summary {
		fmt.Printf("  - %s\n", line)
	}
	fmt.Println("changes:")
	for _, line := range s.Changes {
		fmt.Printf("  - %s\n", line)
	}
	fmt.Println("test_plan:")
	for _, line := range s.TestPlan {
		fmt.Printf("  - %s\n", line)
	}
	if len(s.Notes) > 0 {
		fmt.Println("notes:")
		for _, line := range s.Notes {
			fmt.Printf("  - %s\n", line)
		}
	}
}

func baseForGH(base string) string {
	return strings.TrimPrefix(base, "origin/")
}

func quoteMessage(msg string) string {
	if len(msg) > 80 {
		return fmt.Sprintf("%q...", msg[:80])
	}
	return fmt.Sprintf("%q", msg)
}

func PrintDryRunHint() {
	fmt.Fprintln(os.Stderr, "Use --dry-run para simular sem executar.")
}
