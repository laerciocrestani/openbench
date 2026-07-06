package main

import (
	"fmt"
	"os"

	"github.com/laerciocrestani/gitia/internal/app"
	"github.com/laerciocrestani/gitia/internal/config"
	gitpkg "github.com/laerciocrestani/gitia/internal/git"
	"github.com/laerciocrestani/gitia/internal/setup"
	"github.com/laerciocrestani/gitia/internal/ui"
	"github.com/spf13/cobra"
)

var (
	noAdd               bool
	dryRun              bool
	draft               bool
	base                string
	verbose             bool
	pruneBranches       bool
	pruneRemoteBranches bool
	reportHour          bool
	reportHours         int
	reportDays          int
	reportMonth         bool
	reportAll           bool
)

func main() {
	root := &cobra.Command{
		Use:   "gitia",
		Short: "CLI para commit/PR com IA barata",
		Long:  "Gera conventional commits a partir de git diff usando IA configurável e integra com gh pr create.",
		RunE:  runOverview,
		Args:  cobra.NoArgs,
	}

	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simula sem executar git/gh")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "exibe detalhes da sugestão da IA")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if config.ClearScreenEnabled() {
			ui.ClearScreen()
		}
	}

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
			sess := ui.New("status", false)
			sess.Header()

			repo, err := gitpkg.New()
			if err != nil {
				return err
			}
			if err := repo.IsRepo(); err != nil {
				return fmt.Errorf("diretório atual não é um repositório git")
			}

			sess.Info("Reading repository status...")
			fmt.Println()
			if err := repo.Status(args...); err != nil {
				return err
			}
			return nil
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

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sincroniza com origin (pull da branch base)",
		Long:  "Atualiza a branch base com git fetch + pull. Use --prune para remover branches mergeadas (local e GitHub) ou --prune-remote só no GitHub.",
		RunE:  runSync,
	}
	syncCmd.Flags().BoolVar(&pruneBranches, "prune", false, "remove branches mergeadas no local e no GitHub")
	syncCmd.Flags().BoolVar(&pruneRemoteBranches, "prune-remote", false, "remove branches mergeadas só no GitHub")
	syncCmd.Flags().StringVar(&base, "base", "", "branch base (default: config base_branch)")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Exibe versão instalada",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess := ui.New("version", false)
			sess.Header()
			info := ui.VersionInfo()
			fmt.Println()
			sess.MetaRow("Version", info.Version)
			if info.Commit != "" {
				sess.MetaRow("Commit", info.Commit)
			}
			if info.Commits > 0 {
				sess.MetaRow("Commits", fmt.Sprintf("%d", info.Commits))
			}
			if info.Dirty {
				sess.MetaRow("Tree", "dirty")
			}
			return nil
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
			sess := ui.New("config", false)
			sess.Header()

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Print(cfg.Display())
			return nil
		},
	}

	configCmd.AddCommand(configInit, configShow)

	pricingCmd := &cobra.Command{
		Use:   "pricing",
		Short: "Atualiza e consulta preços da API Gemini",
		Long:  "Busca preços oficiais do Gemini na web e mantém estimativas de custo atualizadas.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPricingUpdate()
		},
	}
	pricingUpdate := &cobra.Command{
		Use:   "update",
		Short: "Busca preços oficiais e salva localmente",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPricingUpdate()
		},
	}
	pricingShow := &cobra.Command{
		Use:   "show",
		Short: "Exibe preços salvos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPricingShow()
		},
	}
	pricingReport := &cobra.Command{
		Use:   "report",
		Short: "Relatório de gastos registrados",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPricingReport()
		},
	}
	pricingCmd.AddCommand(pricingUpdate, pricingShow, pricingReport)

	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Relatório de uso e custos de IA",
		Long:  "Lê o ledger de uso e exibe tokens e custos. Padrão: últimas 24 horas.",
		RunE:  runReport,
	}
	reportCmd.Flags().BoolVar(&reportHour, "hour", false, "última hora")
	reportCmd.Flags().IntVar(&reportHours, "hours", 0, "últimas N horas (padrão: 24 se nenhum período for informado)")
	reportCmd.Flags().IntVar(&reportDays, "days", 0, "últimos N dias")
	reportCmd.Flags().BoolVar(&reportMonth, "month", false, "mês atual (calendário)")
	reportCmd.Flags().BoolVar(&reportAll, "all", false, "todo o histórico")

	root.AddCommand(installCmd, updateCmd, syncCmd, versionCmd, statusCmd, commitCmd, pushCmd, prCmd, configCmd, pricingCmd, reportCmd)

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

func runOverview(cmd *cobra.Command, args []string) error {
	return app.RunOverview()
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

func runSync(cmd *cobra.Command, args []string) error {
	return app.RunSync(app.SyncOptions{
		Prune:       pruneBranches,
		PruneRemote: pruneRemoteBranches,
		Base:        base,
		DryRun:      dryRun,
	})
}

func runReport(cmd *cobra.Command, args []string) error {
	return app.RunReport(app.ReportOptions{
		Hour:  reportHour,
		Hours: reportHours,
		Days:  reportDays,
		Month: reportMonth,
		All:   reportAll,
	})
}
