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
	NoAdd   bool
	DryRun  bool
	Draft   bool
	Base    string
	Verbose bool
	UI      *ui.Session
}

type Result struct {
	Suggestion   *ai.CommitSuggestion
	PRSuggestion *ai.PRSuggestion
	Message      string
	PRURL        string
	PRPreview    string
}

func (o Options) session(command string) *ui.Session {
	if o.UI != nil {
		return o.UI
	}
	return ui.New(command, o.DryRun)
}

func RunCommit(ctx context.Context, opts Options) (*Result, error) {
	sess := opts.session("commit")
	sess.Header()

	result, provider, err := commitFlow(ctx, opts, sess)
	if err != nil {
		return nil, err
	}

	if provider != nil {
		cfg, _ := config.Load()
		if cfg != nil {
			recordAIUsage("commit", cfg, provider.UsageStats())
		}
		provider.UsageStats().PrintWith(sess)
	}

	if result != nil && result.Message != "" && !opts.DryRun {
		sess.Detail(formatter.TitleLine(result.Message))
	}
	sess.Success("Ready to ship 🚀")
	return result, nil
}

func RunPush(ctx context.Context, opts Options) (*Result, error) {
	sess := opts.session("push")
	sess.Header()

	repo, err := gitpkg.New()
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	if !opts.NoAdd {
		if err := sess.Step("Staging changes", func() error {
			if opts.DryRun {
				sess.Detail("git add .")
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
		pushOpts.UI = sess
		result, provider, err = commitFlow(ctx, pushOpts, sess)
		if err != nil {
			return nil, err
		}
	} else {
		sess.Info("No pending changes — pushing existing commits")
		result = &Result{}
	}

	if err := sess.Step("Pushing to origin", func() error {
		if opts.DryRun {
			sess.Detail("git push -u origin HEAD")
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
		provider.UsageStats().PrintWith(sess)
	}
	sess.Success("Ready to ship 🚀")
	return result, nil
}

func RunPR(ctx context.Context, opts Options) (*Result, error) {
	sess := opts.session("pr")
	sess.Header()

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
	if err := sess.Step("Resolving base branch", func() error {
		var err error
		resolvedBase, err = repo.ResolveBase(base)
		return err
	}); err != nil {
		return nil, err
	}

	provider, err := ai.New(cfg)
	if err != nil {
		return nil, err
	}

	if !opts.NoAdd {
		if err := sess.Step("Staging changes", func() error {
			if opts.DryRun {
				sess.Detail("git add .")
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

	if hasStaged {
		commitResult, err := commitStaged(ctx, cfg, repo, opts, provider, sess)
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
		sess.Info("Using existing commits on branch")
	}

	if err := sess.Step("Pushing to origin", func() error {
		if opts.DryRun {
			sess.Detail("git push -u origin HEAD")
			return nil
		}
		return repo.Push()
	}); err != nil {
		return nil, err
	}

	var branch string
	if err := sess.Step("Reading branch diff", func() error {
		var err error
		branch, err = repo.CurrentBranch()
		return err
	}); err != nil {
		return nil, err
	}

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

	var prSuggestion *ai.PRSuggestion
	prEstimate := ai.EstimateCost(cfg, diff, "pr")
	sess.Detail("Estimativa: " + prEstimate.Format(cfg.Provider))
	if err := sess.Step("Thinking", func() error {
		prSuggestion, err = provider.SuggestPR(ctx, diff, branch, baseForGH(resolvedBase), cfg.Language, commitLog)
		return err
	}); err != nil {
		return nil, err
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
		preview := prClient.PreviewCreate(prSuggestion, resolvedBase, opts.Draft)
		result.PRPreview = preview
		sess.Detail(preview)
		recordAIUsage("pr", cfg, provider.UsageStats())
		provider.UsageStats().PrintWith(sess)
		sess.Success("Ready to ship 🚀")
		return result, nil
	}

	var url string
	if err := sess.Step("Creating Pull Request", func() error {
		url, err = prClient.Create(prSuggestion, resolvedBase, opts.Draft)
		return err
	}); err != nil {
		return nil, err
	}

	result.PRURL = url
	sess.Detail(url)
	recordAIUsage("pr", cfg, provider.UsageStats())
	provider.UsageStats().PrintWith(sess)
	sess.Success("Ready to ship 🚀")
	return result, nil
}

func commitFlow(ctx context.Context, opts Options, sess *ui.Session) (*Result, ai.Provider, error) {
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
		if err := sess.Step("Staging changes", func() error {
			if opts.DryRun {
				sess.Detail("git add .")
				return nil
			}
			return repo.AddAll()
		}); err != nil {
			return nil, nil, err
		}
	}

	var diff string
	if err := sess.Step("Reading git diff", func() error {
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

	provider, err := ai.New(cfg)
	if err != nil {
		return nil, nil, err
	}

	var suggestion *ai.CommitSuggestion
	commitEstimate := ai.EstimateCost(cfg, diff, "commit")
	sess.Detail("Estimativa: " + commitEstimate.Format(cfg.Provider))
	if err := sess.Step("Thinking", func() error {
		suggestion, err = provider.SuggestCommit(ctx, diff, cfg.Language)
		return err
	}); err != nil {
		return nil, nil, err
	}

	message := formatter.FormatCommit(suggestion, cfg.CoAuthor)
	result := &Result{
		Suggestion: suggestion,
		Message:    message,
	}

	if opts.Verbose {
		printCommitVerbose(suggestion, message)
	}

	if err := sess.Step("Writing Conventional Commit", func() error {
		if opts.DryRun {
			sess.Detail("git commit -m " + quoteMessage(message))
			return nil
		}
		return repo.Commit(message)
	}); err != nil {
		return nil, nil, err
	}

	return result, provider, nil
}

func commitStaged(ctx context.Context, cfg *config.Config, repo *gitpkg.Repo, opts Options, provider ai.Provider, sess *ui.Session) (*Result, error) {
	diff, err := repo.DiffStaged()
	if err != nil {
		return nil, err
	}

	var suggestion *ai.CommitSuggestion
	commitEstimate := ai.EstimateCost(cfg, diff, "commit")
	sess.Detail("Estimativa: " + commitEstimate.Format(cfg.Provider))
	if err := sess.Step("Thinking", func() error {
		suggestion, err = provider.SuggestCommit(ctx, diff, cfg.Language)
		return err
	}); err != nil {
		return nil, err
	}

	message := formatter.FormatCommit(suggestion, cfg.CoAuthor)
	result := &Result{
		Suggestion: suggestion,
		Message:    message,
	}

	if opts.Verbose {
		printCommitVerbose(suggestion, message)
	}

	if err := sess.Step("Writing Conventional Commit", func() error {
		if opts.DryRun {
			sess.Detail("git commit -m " + quoteMessage(message))
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
