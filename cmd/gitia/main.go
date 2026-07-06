package main

import (
	"fmt"
	"os"

	"github.com/laerciocrestani/gitia/internal/app"
	"github.com/laerciocrestani/gitia/internal/config"
	gitpkg "github.com/laerciocrestani/gitia/internal/git"
	"github.com/laerciocrestani/gitia/internal/setup"
	"github.com/spf13/cobra"
)

var (
	noAdd   bool
	dryRun  bool
	draft   bool
	base    string
	verbose bool
)

func main() {
	root := &cobra.Command{
		Use:   "gitia",
		Short: "CLI para commit/PR com IA barata",
		Long:  "Gera conventional commits a partir de git diff usando IA configurável e integra com gh pr create.",
	}

	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simula sem executar git/gh")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "exibe detalhes da sugestão da IA")

	commitCmd := &cobra.Command{
		Use:   "commit",
		Short: "Gera commit com IA",
		RunE:  runCommit,
	}
	commitCmd.Flags().BoolVar(&noAdd, "no-add", false, "não executa git add .")

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Commit + push para origin",
		RunE:  runPush,
	}
	pushCmd.Flags().BoolVar(&noAdd, "no-add", false, "não executa git add .")

	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Commit + push + cria PR via gh",
		RunE:  runPR,
	}
	prCmd.Flags().BoolVar(&noAdd, "no-add", false, "não executa git add .")
	prCmd.Flags().BoolVar(&draft, "draft", false, "cria PR como draft")
	prCmd.Flags().StringVar(&base, "base", "", "branch base (default: config base_branch)")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Alias para git status",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := gitpkg.New()
			if err != nil {
				return err
			}
			if err := repo.IsRepo(); err != nil {
				return fmt.Errorf("diretório atual não é um repositório git")
			}
			return repo.Status(args...)
		},
	}

	installCmd := &cobra.Command{
		Use:    "install",
		Hidden: true,
		Short:  "Instala o binário e configura PATH (bootstrap via go run)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setup.Install()
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Atualiza o repositório e reinstala o binário",
		Long:  "Executa git pull e go install dentro do clone do repositório gitia.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setup.Update()
		},
	}

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configura provider, API key e preferências",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.InitInteractive()
		},
	}

	configInit := &cobra.Command{
		Use:   "init",
		Short: "Wizard interativo para criar config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.InitInteractive()
		},
	}

	configShow := &cobra.Command{
		Use:   "show",
		Short: "Exibe configuração atual (API key mascarada)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Print(cfg.Display())
			return nil
		},
	}

	configCmd.AddCommand(configInit, configShow)
	root.AddCommand(installCmd, updateCmd, statusCmd, commitCmd, pushCmd, prCmd, configCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func opts() app.Options {
	return app.Options{
		NoAdd:   noAdd,
		DryRun:  dryRun,
		Draft:   draft,
		Base:    base,
		Verbose: verbose,
	}
}

func runCommit(cmd *cobra.Command, args []string) error {
	_, err := app.RunCommit(cmd.Context(), opts())
	return err
}

func runPush(cmd *cobra.Command, args []string) error {
	_, err := app.RunPush(cmd.Context(), opts())
	return err
}

func runPR(cmd *cobra.Command, args []string) error {
	_, err := app.RunPR(cmd.Context(), opts())
	return err
}
