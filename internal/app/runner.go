package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/laerciocrestani/gitia/internal/ai"
	"github.com/laerciocrestani/gitia/internal/config"
	"github.com/laerciocrestani/gitia/internal/formatter"
	gitpkg "github.com/laerciocrestani/gitia/internal/git"
	prpkg "github.com/laerciocrestani/gitia/internal/pr"
)

type Options struct {
	NoAdd   bool
	DryRun  bool
	Draft   bool
	Base    string
	Verbose bool
}

type Result struct {
	Suggestion   *ai.CommitSuggestion
	PRSuggestion *ai.PRSuggestion
	Message      string
	PRURL        string
	PRPreview    string
}

func RunCommit(ctx context.Context, opts Options) (*Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	repo, err := gitpkg.New()
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	if !opts.NoAdd {
		if opts.DryRun {
			fmt.Println("[dry-run] git add .")
		} else {
			if err := repo.AddAll(); err != nil {
				return nil, err
			}
		}
	}

	diff, err := repo.DiffForCommit()
	if err != nil {
		return nil, err
	}
	if diff == "" {
		return nil, fmt.Errorf("nenhuma alteração para commitar")
	}

	provider, err := ai.New(cfg)
	if err != nil {
		return nil, err
	}

	suggestion, err := provider.SuggestCommit(ctx, diff, cfg.Language)
	if err != nil {
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

	if opts.DryRun {
		fmt.Println("[dry-run] git commit -m", quoteMessage(message))
		provider.UsageStats().Print()
		return result, nil
	}

	if err := repo.Commit(message); err != nil {
		return nil, err
	}

	fmt.Println("Commit criado:", formatter.TitleLine(message))
	provider.UsageStats().Print()
	return result, nil
}

func RunPush(ctx context.Context, opts Options) (*Result, error) {
	result, err := RunCommit(ctx, opts)
	if err != nil {
		return nil, err
	}

	if opts.DryRun {
		fmt.Println("[dry-run] git push -u origin HEAD")
		return result, nil
	}

	repo, err := gitpkg.New()
	if err != nil {
		return nil, err
	}
	if err := repo.Push(); err != nil {
		return nil, err
	}

	fmt.Println("Push concluído")
	return result, nil
}

func RunPR(ctx context.Context, opts Options) (*Result, error) {
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

	resolvedBase, err := repo.ResolveBase(base)
	if err != nil {
		return nil, err
	}

	provider, err := ai.New(cfg)
	if err != nil {
		return nil, err
	}

	if !opts.NoAdd {
		if opts.DryRun {
			fmt.Println("[dry-run] git add .")
		} else if err := repo.AddAll(); err != nil {
			return nil, err
		}
	}

	result := &Result{}

	hasStaged, err := repo.HasStagedChanges()
	if err != nil {
		return nil, err
	}

	if hasStaged {
		commitResult, err := commitStaged(ctx, cfg, repo, opts, provider)
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
		fmt.Println("Nenhuma alteração pendente; usando commits já existentes na branch")
	}

	if opts.DryRun {
		fmt.Println("[dry-run] git push -u origin HEAD")
	} else if err := repo.Push(); err != nil {
		return nil, err
	} else {
		fmt.Println("Push concluído")
	}

	branch, err := repo.CurrentBranch()
	if err != nil {
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

	prSuggestion, err := provider.SuggestPR(ctx, diff, branch, baseForGH(resolvedBase), cfg.Language, commitLog)
	if err != nil {
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
		fmt.Println("[dry-run]", preview)
		provider.UsageStats().Print()
		return result, nil
	}

	url, err := prClient.Create(prSuggestion, resolvedBase, opts.Draft)
	if err != nil {
		return nil, err
	}

	result.PRURL = url
	fmt.Println("PR criado:", url)
	provider.UsageStats().Print()
	return result, nil
}

func commitStaged(ctx context.Context, cfg *config.Config, repo *gitpkg.Repo, opts Options, provider ai.Provider) (*Result, error) {
	diff, err := repo.DiffStaged()
	if err != nil {
		return nil, err
	}

	suggestion, err := provider.SuggestCommit(ctx, diff, cfg.Language)
	if err != nil {
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

	if opts.DryRun {
		fmt.Println("[dry-run] git commit -m", quoteMessage(message))
		return result, nil
	}

	if err := repo.Commit(message); err != nil {
		return nil, err
	}

	fmt.Println("Commit criado:", formatter.TitleLine(message))
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
