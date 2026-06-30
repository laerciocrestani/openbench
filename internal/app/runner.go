package app

import (
	"context"
	"fmt"
	"os"

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
	Suggestion *ai.CommitSuggestion
	Message    string
	PRURL      string
	PRPreview  string
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
		fmt.Println("--- Sugestão da IA ---")
		fmt.Printf("type: %s\nscope: %s\ntitle: %s\n", suggestion.Type, suggestion.Scope, suggestion.Title)
		for _, b := range suggestion.Body {
			fmt.Printf("  - %s\n", b)
		}
		fmt.Println("--- Mensagem ---")
		fmt.Println(message)
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

	result, err := RunPush(ctx, opts)
	if err != nil {
		return nil, err
	}

	prClient, err := prpkg.New()
	if err != nil {
		return nil, err
	}

	title := formatter.TitleLine(result.Message)

	if opts.DryRun {
		preview := prClient.PreviewCreate(title, result.Message, base, opts.Draft)
		result.PRPreview = preview
		fmt.Println("[dry-run]", preview)
		return result, nil
	}

	url, err := prClient.Create(title, result.Message, base, opts.Draft)
	if err != nil {
		return nil, err
	}

	result.PRURL = url
	fmt.Println("PR criado:", url)
	return result, nil
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
