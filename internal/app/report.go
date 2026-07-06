package app

import (
	"fmt"
	"sort"
	"time"

	"github.com/laerciocrestani/gitai/internal/pricing"
	"github.com/laerciocrestani/gitai/internal/ui"
	"github.com/laerciocrestani/gitai/internal/usage"
)

type ReportOptions struct {
	Hour  bool
	Hours int
	Days  int
	Month bool
	All   bool
}

func RunReport(opts ReportOptions) error {
	sess := ui.New("report", false)
	sess.Header()

	period, err := usage.ResolvePeriod(usage.PeriodOptions{
		Hour:  opts.Hour,
		Hours: opts.Hours,
		Days:  opts.Days,
		Month: opts.Month,
		All:   opts.All,
	}, time.Now())
	if err != nil {
		return err
	}

	report, err := usage.BuildReport(period)
	if err != nil {
		return err
	}

	store, _ := pricing.Load()
	if store != nil && !store.UpdatedAt.IsZero() {
		sess.Detail(fmt.Sprintf("Preços: atualizados em %s",
			store.UpdatedAt.Format("2006-01-02 15:04 UTC")))
	} else {
		sess.Info("Preços não encontrados. Execute: gitai pricing update")
	}

	ledgerPath, _ := usage.LedgerPath()
	sess.Section("Período")
	sess.KV("Intervalo", period.Label)
	if !opts.All {
		sess.KV("De", period.Since.Local().Format("2006-01-02 15:04"))
		sess.KV("Até", period.Until.Local().Format("2006-01-02 15:04"))
	}
	sess.KV("Arquivo", ledgerPath)

	if report.Summary.TotalEntries == 0 {
		sess.Info("Nenhum uso registrado neste período.")
		return nil
	}

	sess.Section("Resumo")
	sess.KV("Chamadas", fmt.Sprintf("%d", report.Summary.TotalEntries))
	sess.KV("Tokens entrada", usage.FormatTokens(report.Summary.TotalInput))
	sess.KV("Tokens saída", usage.FormatTokens(report.Summary.TotalOutput))
	sess.KV("Tokens total", usage.FormatTokens(report.Summary.TotalInput+report.Summary.TotalOutput))
	if report.Summary.HasCost {
		sess.KV("Custo total", fmt.Sprintf("$%.6f USD", report.Summary.TotalCost))
	}

	if len(report.ByModel) > 0 {
		sess.Section("Por modelo")
		models := sortedModelKeys(report.ByModel)
		for _, model := range models {
			mu := report.ByModel[model]
			line := fmt.Sprintf("%s — %d chamada(s) · %s in · %s out",
				model, mu.Calls,
				usage.FormatTokens(mu.InputTokens),
				usage.FormatTokens(mu.OutputTokens))
			if mu.HasCost {
				line += fmt.Sprintf(" · $%.6f USD", mu.CostUSD)
			}
			if store != nil {
				if in, out, ok := store.PricesForModel(model); ok {
					line += fmt.Sprintf(" · preço $%.4f/$%.4f por 1M", in, out)
				}
			}
			sess.Bullet(line)
		}
	}

	if len(report.ByProject) > 0 {
		sess.Section("Por projeto")
		projects := sortedProjectKeys(report.ByProject)
		for _, project := range projects {
			pu := report.ByProject[project]
			line := fmt.Sprintf("%s — %d chamada(s) · %s in · %s out",
				project, pu.Calls,
				usage.FormatTokens(pu.InputTokens),
				usage.FormatTokens(pu.OutputTokens))
			if pu.HasCost {
				line += fmt.Sprintf(" · $%.6f USD", pu.CostUSD)
			}
			sess.Bullet(line)
		}
	}

	if len(report.Entries) > 0 {
		sess.Section("Detalhes")
		entries := report.Entries
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		})
		for _, e := range entries {
			line := fmt.Sprintf("%s · %s · %s · %s · %s in · %s out",
				e.Timestamp.Local().Format("2006-01-02 15:04"),
				e.Command,
				e.Project,
				e.Model,
				usage.FormatTokens(e.InputTokens),
				usage.FormatTokens(e.OutputTokens),
			)
			if e.CostUSD != nil {
				line += fmt.Sprintf(" · $%.6f USD", *e.CostUSD)
			}
			sess.Bullet(line)
		}
	}

	return nil
}

func sortedModelKeys(m map[string]usage.ModelUsage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedProjectKeys(m map[string]usage.ProjectUsage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
