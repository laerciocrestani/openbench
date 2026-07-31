package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/laerciocrestani/openbench/internal/app"
	"github.com/laerciocrestani/openbench/internal/config"
	"github.com/laerciocrestani/openbench/internal/desktop"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	"github.com/laerciocrestani/openbench/internal/setup"
	"github.com/laerciocrestani/openbench/internal/tui"
	"github.com/laerciocrestani/openbench/internal/ui"
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
	hygieneFull         bool
	hygieneLocal        bool
	reportHour          bool
	reportHours         int
	reportDays          int
	reportMonth         bool
	reportAll           bool
	doctorExplain       bool
	dockerBuild         bool
	dockerProfile       string
	dockerForceRecreate bool
	dockerNoDeps        bool
	dockerAll           bool
	dockerTail          int
	dockerFollow        bool
	dockerComposeFile   string
	dockerPresetService string
	detail              string
)

func main() {
	root := &cobra.Command{
		Use:   "ob",
		Short: "Dev environment orchestrator with AI-powered git workflow",
		Long:  "openbench (ob) — Docker, conventional commits com IA e integração com GitHub CLI.",
		RunE:  runOverview,
		Args:  cobra.NoArgs,
	}

	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simula sem executar git/gh/docker")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "exibe detalhes da sugestão da IA")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if skipClearScreen(cmd) {
			return
		}
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
	commitCmd.Flags().StringVar(&detail, "detail", "", "profundidade do contexto/saída: minimal|standard|thorough (override do .openbench.yaml)")

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
	prCmd.Flags().StringVar(&detail, "detail", "", "profundidade do contexto/saída: minimal|standard|thorough (override do .openbench.yaml)")

	prViewCmd := &cobra.Command{
		Use:   "view",
		Short: "Abre o PR da branch atual no browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPRView()
		},
	}
	prCmd.AddCommand(prViewCmd)

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
		Long:  "Executa git pull e go install dentro do clone do repositório openbench.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setup.Update()
		},
	}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sincroniza a branch base com origin",
		Long:  "Atualiza refs remotas (fetch --prune) e faz fast-forward da branch base. Para limpar branches mergeadas, use `ob hygiene`.",
		RunE:  runSync,
	}
	syncCmd.Flags().StringVar(&base, "base", "", "branch base (default: config base_branch)")
	// Deprecated flags — kept for clear migration errors.
	syncCmd.Flags().BoolVar(&pruneBranches, "prune", false, "deprecated: use `ob hygiene --full`")
	syncCmd.Flags().BoolVar(&pruneRemoteBranches, "prune-remote", false, "deprecated: use `ob hygiene --full` ou `--local`")
	_ = syncCmd.Flags().MarkDeprecated("prune", "use `ob hygiene --full`")
	_ = syncCmd.Flags().MarkDeprecated("prune-remote", "use `ob hygiene --full` ou `--local`")

	hygieneCmd := &cobra.Command{
		Use:   "hygiene",
		Short: "Limpa branches mergeadas/absorvidas (local e/ou remoto)",
		Long:  "Fetch das refs e prune de branches já mergeadas/absorvidas. Use --full (local+remoto) ou --local (só local).",
		RunE:  runHygiene,
	}
	hygieneCmd.Flags().BoolVar(&hygieneFull, "full", false, "remove branches mergeadas no local e no GitHub")
	hygieneCmd.Flags().BoolVar(&hygieneLocal, "local", false, "remove branches mergeadas só no local (mantém remoto)")
	hygieneCmd.Flags().StringVar(&base, "base", "", "branch base (default: config base_branch)")
	hygieneCmd.MarkFlagsMutuallyExclusive("full", "local")

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

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Interface interativa no terminal (TUI)",
		Long:  "Abre o dashboard fullscreen com ambiente Docker, git e próximos passos.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Panorama de saúde do repositório e ambiente",
		Long:  "Analisa divergências, working tree, Docker e recomenda próximos passos. Use --explain para enriquecer com IA.",
		RunE:  runDoctor,
	}
	doctorCmd.Flags().BoolVar(&doctorExplain, "explain", false, "consulta IA para explicação detalhada")
	doctorCmd.Flags().StringVar(&base, "base", "", "branch base (default: config base_branch)")

	dockerCmd := &cobra.Command{
		Use:   "docker",
		Short: "Controle Docker Compose do projeto",
	}

	dockerStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Exibe status do Docker e containers do compose",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerStatus()
		},
	}

	dockerPSCmd := &cobra.Command{
		Use:   "ps",
		Short: "Lista containers do compose",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerPS(dockerOpts())
		},
	}
	dockerPSCmd.Flags().BoolVar(&dockerAll, "all", false, "inclui containers parados")

	dockerUpCmd := &cobra.Command{
		Use:   "up [service...]",
		Short: "Sobe serviços com docker compose up -d",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			opts.Services = args
			return app.RunDockerUp(opts)
		},
	}
	dockerUpCmd.Flags().BoolVar(&dockerBuild, "build", false, "reconstrói imagens antes de subir")
	dockerUpCmd.Flags().StringVar(&dockerProfile, "profile", "", "profile do compose")
	dockerUpCmd.Flags().BoolVar(&dockerForceRecreate, "force-recreate", false, "recria containers antes de subir")
	dockerUpCmd.Flags().BoolVar(&dockerNoDeps, "no-deps", false, "não inicia serviços dependentes")

	dockerDownCmd := &cobra.Command{
		Use:   "down",
		Short: "Para serviços com docker compose down",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerDown(dockerOpts())
		},
	}

	dockerStopCmd := &cobra.Command{
		Use:   "stop [service...]",
		Short: "Para serviços específicos com docker compose stop",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			opts.Services = args
			return app.RunDockerStop(opts)
		},
	}

	dockerStartCmd := &cobra.Command{
		Use:   "start [service...]",
		Short: "Inicia serviços parados com docker compose start",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			opts.Services = args
			return app.RunDockerStart(opts)
		},
	}

	dockerRecreateCmd := &cobra.Command{
		Use:   "recreate [service]",
		Short: "Recria um serviço com docker compose up -d --force-recreate --no-deps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			opts.Service = args[0]
			return app.RunDockerRecreate(opts)
		},
	}

	dockerExecCmd := &cobra.Command{
		Use:   "exec [service] -- [command...]",
		Short: "Executa comando em um serviço (docker compose exec)",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			if len(args) == 0 {
				return fmt.Errorf("informe o serviço: ob docker exec <service> -- <command>")
			}
			opts.Service = args[0]
			if len(args) > 1 {
				opts.Command = args[1:]
			}
			opts.Interactive = false
			return app.RunDockerExec(opts)
		},
	}

	dockerLogsCmd := &cobra.Command{
		Use:   "logs [service]",
		Short: "Exibe logs de um serviço",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			if len(args) > 0 {
				opts.Service = args[0]
			}
			return app.RunDockerLogs(opts)
		},
	}
	dockerLogsCmd.Flags().IntVar(&dockerTail, "tail", 100, "número de linhas")
	dockerLogsCmd.Flags().BoolVarP(&dockerFollow, "follow", "f", false, "segue logs em tempo real")

	dockerShCmd := &cobra.Command{
		Use:   "sh [service]",
		Short: "Abre shell interativo no serviço",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dockerOpts()
			if len(args) > 0 {
				opts.Service = args[0]
			}
			return app.RunDockerShell(opts)
		},
	}

	dockerPresetCmd := &cobra.Command{
		Use:   "preset",
		Short: "Presets de comandos docker exec do projeto",
	}
	dockerPresetListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lista presets em .openbench/docker-presets.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerPresetList("")
		},
	}
	dockerPresetRunCmd := &cobra.Command{
		Use:   "run <preset-id>",
		Short: "Executa um preset no serviço (one-shot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerPresetRun("", dockerPresetService, args[0], dryRun)
		},
	}
	dockerPresetRunCmd.Flags().StringVar(&dockerPresetService, "service", "", "serviço compose (default: primeiro running)")
	dockerPresetCmd.AddCommand(dockerPresetListCmd, dockerPresetRunCmd)

	dockerKitCmd := &cobra.Command{
		Use:   "kit",
		Short: "Kits de presets empacotados (ex: laravel)",
	}
	dockerKitListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lista kits disponíveis para importar",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerKitList()
		},
	}
	dockerKitImportCmd := &cobra.Command{
		Use:   "import <kit-id>",
		Short: "Importa um kit para .openbench/docker-presets.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDockerKitImport("", args[0])
		},
	}
	dockerKitCmd.AddCommand(dockerKitListCmd, dockerKitImportCmd)

	dockerCmd.PersistentFlags().StringVarP(&dockerComposeFile, "file", "f", "", "caminho do compose file")
	dockerCmd.AddCommand(dockerStatusCmd, dockerPSCmd, dockerUpCmd, dockerDownCmd, dockerStopCmd, dockerStartCmd, dockerRecreateCmd, dockerExecCmd, dockerLogsCmd, dockerShCmd, dockerPresetCmd, dockerKitCmd)

	var (
		ciFailedOnly  bool
		ciLimit       int
		ciBranch      string
		ciAllBranches bool
		ciJobID       int64
		ciLogFailed   bool
		ciYes         bool
		ciRef         string
		ciFields      []string
	)
	ciCmd := &cobra.Command{
		Use:   "ci",
		Short: "GitHub Actions — observar runs do repositório atual",
	}
	ciStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Lista workflow runs (branch atual por padrão)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunCIStatus(app.CIStatusOptions{
				FailedOnly:  ciFailedOnly,
				Limit:       ciLimit,
				Branch:      ciBranch,
				AllBranches: ciAllBranches,
			})
		},
	}
	ciStatusCmd.Flags().BoolVar(&ciFailedOnly, "failed", false, "somente runs falhos")
	ciStatusCmd.Flags().IntVar(&ciLimit, "limit", 20, "máximo de runs (até 50)")
	ciStatusCmd.Flags().StringVar(&ciBranch, "branch", "", "filtrar branch (default: branch atual)")
	ciStatusCmd.Flags().BoolVar(&ciAllBranches, "all", false, "todas as branches")
	ciViewCmd := &cobra.Command{
		Use:   "view <run-id>",
		Short: "Detalha um workflow run (jobs/steps)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("run-id inválido: %s", args[0])
			}
			return app.RunCIView(id)
		},
	}
	ciUsageCmd := &cobra.Command{
		Use:   "usage",
		Short: "Mostra evidência de minutos/uso de Actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunCIUsage()
		},
	}
	ciLogsCmd := &cobra.Command{
		Use:   "logs <run-id>",
		Short: "Baixa log do run sob demanda (sempre redigido)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("run-id inválido: %s", args[0])
			}
			return app.RunCILogs(app.CILogsOptions{
				RunID:      id,
				JobID:      ciJobID,
				FailedOnly: ciLogFailed,
			})
		},
	}
	ciLogsCmd.Flags().Int64Var(&ciJobID, "job", 0, "job id específico")
	ciLogsCmd.Flags().BoolVar(&ciLogFailed, "failed", false, "somente steps falhos (--log-failed)")
	ciRerunCmd := &cobra.Command{
		Use:   "rerun <run-id>",
		Short: "Re-executa um run/job (pede confirmação; avisa minutos)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("run-id inválido: %s", args[0])
			}
			return app.RunCIRerun(app.CIRerunOptions{
				RunID:      id,
				JobID:      ciJobID,
				FailedOnly: ciLogFailed,
				Yes:        ciYes,
			})
		},
	}
	ciRerunCmd.Flags().Int64Var(&ciJobID, "job", 0, "job id específico (databaseId)")
	ciRerunCmd.Flags().BoolVar(&ciLogFailed, "failed", false, "somente jobs falhos")
	ciRerunCmd.Flags().BoolVar(&ciYes, "yes", false, "confirma sem prompt")
	ciWorkflowsCmd := &cobra.Command{
		Use:   "workflows",
		Short: "Lista workflows do repositório",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunCIWorkflows()
		},
	}
	ciDispatchCmd := &cobra.Command{
		Use:   "dispatch <workflow>",
		Short: "Dispara workflow_dispatch (pede confirmação; avisa minutos)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields := map[string]string{}
			for _, f := range ciFields {
				k, v, ok := strings.Cut(f, "=")
				if !ok {
					continue
				}
				fields[strings.TrimSpace(k)] = v
			}
			return app.RunCIDispatch(app.CIDispatchOptions{
				Workflow: args[0],
				Ref:      ciRef,
				Fields:   fields,
				Yes:      ciYes,
			})
		},
	}
	ciDispatchCmd.Flags().StringVar(&ciRef, "ref", "", "branch/tag do workflow")
	ciDispatchCmd.Flags().StringArrayVarP(&ciFields, "field", "f", nil, "input key=value (repetível)")
	ciDispatchCmd.Flags().BoolVar(&ciYes, "yes", false, "confirma sem prompt")
	var ciFixPush bool
	ciFixCmd := &cobra.Command{
		Use:   "fix <run-id>",
		Short: "Corrige falha de CI com IA (preview → confirmação → commit/push)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("run-id inválido: %s", args[0])
			}
			return desktop.RunCIFixCLI(cmd.Context(), desktop.CIFixCLIOptions{
				RunID: id,
				JobID: ciJobID,
				Yes:   ciYes,
				Push:  ciFixPush,
			})
		},
	}
	ciFixCmd.Flags().Int64Var(&ciJobID, "job", 0, "job id específico")
	ciFixCmd.Flags().BoolVar(&ciYes, "yes", false, "confirma sem prompt")
	ciFixCmd.Flags().BoolVar(&ciFixPush, "push", true, "após aplicar, commit+push e acompanhar CI")
	ciCmd.AddCommand(ciStatusCmd, ciViewCmd, ciUsageCmd, ciLogsCmd, ciRerunCmd, ciWorkflowsCmd, ciDispatchCmd, ciFixCmd)

	root.AddCommand(installCmd, updateCmd, syncCmd, hygieneCmd, versionCmd, statusCmd, commitCmd, pushCmd, prCmd, configCmd, pricingCmd, reportCmd, uiCmd, doctorCmd, dockerCmd, ciCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func dockerOpts() app.DockerOptions {
	return app.DockerOptions{
		ComposeFile:   dockerComposeFile,
		Build:         dockerBuild,
		Profile:       dockerProfile,
		ForceRecreate: dockerForceRecreate,
		NoDeps:        dockerNoDeps,
		All:           dockerAll,
		Tail:          dockerTail,
		Follow:        dockerFollow,
		DryRun:        dryRun,
	}
}

func opts() app.Options {
	return app.Options{
		NoAdd:   noAdd,
		DryRun:  dryRun,
		Draft:   draft,
		Base:    base,
		Verbose: verbose,
		Detail:  detail,
	}
}

func runOverview(cmd *cobra.Command, args []string) error {
	if tui.ShouldLaunch() {
		return tui.Run()
	}
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
	if pruneBranches || pruneRemoteBranches {
		return fmt.Errorf("sync não faz mais prune — use `ob hygiene --full` ou `ob hygiene --local`")
	}
	return app.RunSync(app.SyncOptions{
		Base:   base,
		DryRun: dryRun,
	})
}

func runHygiene(cmd *cobra.Command, args []string) error {
	mode := app.HygieneModeFull
	switch {
	case hygieneLocal:
		mode = app.HygieneModeLocal
	case hygieneFull:
		mode = app.HygieneModeFull
	default:
		return fmt.Errorf("informe --full (local+remoto) ou --local (só local)")
	}
	return app.RunHygiene(app.HygieneOptions{
		Mode:   mode,
		Base:   base,
		DryRun: dryRun,
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

func runDoctor(cmd *cobra.Command, args []string) error {
	report, err := app.RunDoctor(cmd.Context(), app.DoctorOptions{
		Explain: doctorExplain,
		Base:    base,
	})
	if err != nil {
		return err
	}
	app.PrintDoctor(report, nil)
	return nil
}

func isConfigCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "config" {
			return true
		}
	}
	return false
}

func skipClearScreen(cmd *cobra.Command) bool {
	if isConfigCommand(cmd) {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "ui" {
			return true
		}
	}
	if cmd.Parent() == nil && tui.ShouldLaunch() {
		return true
	}
	return false
}
