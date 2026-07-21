package desktop

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/usage"
)

// UsageReportView is the JSON-friendly usage report for the desktop chart UI.
type UsageReportView struct {
	PeriodKey   string           `json:"periodKey"`
	PeriodLabel string           `json:"periodLabel"`
	Calls       int              `json:"calls"`
	TotalInput  int              `json:"totalInput"`
	TotalOutput int              `json:"totalOutput"`
	TotalCost   float64          `json:"totalCost"`
	HasCost     bool             `json:"hasCost"`
	Granularity string           `json:"granularity"` // "hour" | "day"
	Series      []UsagePointView `json:"series"`
	ByModel     []UsageGroupView `json:"byModel"`
	ByProject   []UsageGroupView `json:"byProject"`
	// Chat aggregates ledger rows with command=chat.
	Chat UsageBucketView `json:"chat"`
	// Other aggregates commit/push/pr/doctor and any non-chat command.
	Other UsageBucketView `json:"other"`
}

// UsageBucketView is a cost/token slice (chat vs commits/PRs/etc).
type UsageBucketView struct {
	Calls   int     `json:"calls"`
	Input   int     `json:"input"`
	Output  int     `json:"output"`
	Cost    float64 `json:"cost"`
	HasCost bool    `json:"hasCost"`
}

// UsagePointView is one bar on the token usage chart.
type UsagePointView struct {
	Date   string `json:"date"`
	Input  int    `json:"input"`
	Output int    `json:"output"`
}

// UsageGroupView aggregates tokens by model or project.
type UsageGroupView struct {
	Name   string  `json:"name"`
	Calls  int     `json:"calls"`
	Input  int     `json:"input"`
	Output int     `json:"output"`
	Cost   float64 `json:"cost"`
	HasCost bool   `json:"hasCost"`
}

// LoadUsageReport builds a usage report for the given period key.
// Supported keys: "24h" (default), "7d", "30d", "month", "all".
func LoadUsageReport(periodKey string) (*UsageReportView, error) {
	periodKey = normalizeUsagePeriod(periodKey)
	opts, err := usagePeriodOptions(periodKey)
	if err != nil {
		return nil, err
	}

	period, err := usage.ResolvePeriod(opts, time.Now())
	if err != nil {
		return nil, err
	}

	report, err := usage.BuildReport(period)
	if err != nil {
		return nil, err
	}

	hourly := periodKey == "24h" || opts.Hour || (opts.Hours > 0 && opts.Hours <= 48)
	series := buildUsageSeries(report.Entries, period, hourly)

	view := &UsageReportView{
		PeriodKey:   periodKey,
		PeriodLabel: period.Label,
		Calls:       report.Summary.TotalEntries,
		TotalInput:  report.Summary.TotalInput,
		TotalOutput: report.Summary.TotalOutput,
		TotalCost:   report.Summary.TotalCost,
		HasCost:     report.Summary.HasCost,
		Granularity: "day",
		Series:      series,
		ByModel:     make([]UsageGroupView, 0, len(report.ByModel)),
		ByProject:   make([]UsageGroupView, 0, len(report.ByProject)),
	}
	view.Chat, view.Other = splitUsageByCommand(report.Entries)
	if hourly {
		view.Granularity = "hour"
	}

	modelKeys := make([]string, 0, len(report.ByModel))
	for k := range report.ByModel {
		modelKeys = append(modelKeys, k)
	}
	sort.Strings(modelKeys)
	for _, k := range modelKeys {
		mu := report.ByModel[k]
		view.ByModel = append(view.ByModel, UsageGroupView{
			Name:    mu.Model,
			Calls:   mu.Calls,
			Input:   mu.InputTokens,
			Output:  mu.OutputTokens,
			Cost:    mu.CostUSD,
			HasCost: mu.HasCost,
		})
	}

	projectKeys := make([]string, 0, len(report.ByProject))
	for k := range report.ByProject {
		projectKeys = append(projectKeys, k)
	}
	sort.Strings(projectKeys)
	for _, k := range projectKeys {
		pu := report.ByProject[k]
		view.ByProject = append(view.ByProject, UsageGroupView{
			Name:    pu.Project,
			Calls:   pu.Calls,
			Input:   pu.InputTokens,
			Output:  pu.OutputTokens,
			Cost:    pu.CostUSD,
			HasCost: pu.HasCost,
		})
	}

	return view, nil
}

func splitUsageByCommand(entries []usage.Entry) (chat, other UsageBucketView) {
	for _, e := range entries {
		b := &other
		if strings.EqualFold(strings.TrimSpace(e.Command), "chat") {
			b = &chat
		}
		b.Calls++
		b.Input += e.InputTokens
		b.Output += e.OutputTokens
		if e.CostUSD != nil {
			b.Cost += *e.CostUSD
			b.HasCost = true
		}
	}
	return chat, other
}

func normalizeUsagePeriod(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	switch key {
	case "", "day", "1d":
		return "24h"
	case "7", "week":
		return "7d"
	case "30":
		return "30d"
	case "24h", "7d", "30d", "90d", "month", "all":
		return key
	default:
		return "24h"
	}
}

func usagePeriodOptions(key string) (usage.PeriodOptions, error) {
	switch key {
	case "24h":
		return usage.PeriodOptions{Hours: 24}, nil
	case "7d":
		return usage.PeriodOptions{Days: 7}, nil
	case "30d":
		return usage.PeriodOptions{Days: 30}, nil
	case "90d":
		return usage.PeriodOptions{Days: 90}, nil
	case "month":
		return usage.PeriodOptions{Month: true}, nil
	case "all":
		return usage.PeriodOptions{All: true}, nil
	default:
		return usage.PeriodOptions{}, fmt.Errorf("período inválido: %s", key)
	}
}

func buildUsageSeries(entries []usage.Entry, period usage.Period, hourly bool) []UsagePointView {
	loc := time.Local
	buckets := map[string]UsagePointView{}

	for _, e := range entries {
		ts := e.Timestamp.In(loc)
		key := seriesKey(ts, hourly)
		pt := buckets[key]
		pt.Date = key
		pt.Input += e.InputTokens
		pt.Output += e.OutputTokens
		buckets[key] = pt
	}

	keys := fillSeriesKeys(period, hourly, loc)
	out := make([]UsagePointView, 0, len(keys))
	for _, key := range keys {
		if pt, ok := buckets[key]; ok {
			out = append(out, pt)
			continue
		}
		out = append(out, UsagePointView{Date: key})
	}
	return out
}

func seriesKey(ts time.Time, hourly bool) string {
	if hourly {
		return ts.Format("2006-01-02T15")
	}
	return ts.Format("2006-01-02")
}

func fillSeriesKeys(period usage.Period, hourly bool, loc *time.Location) []string {
	since := period.Since.In(loc)
	until := period.Until.In(loc)
	if until.Before(since) {
		return nil
	}

	var keys []string
	if hourly {
		cur := time.Date(since.Year(), since.Month(), since.Day(), since.Hour(), 0, 0, 0, loc)
		end := time.Date(until.Year(), until.Month(), until.Day(), until.Hour(), 0, 0, 0, loc)
		for !cur.After(end) {
			keys = append(keys, seriesKey(cur, true))
			cur = cur.Add(time.Hour)
		}
		return keys
	}

	cur := time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, loc)
	end := time.Date(until.Year(), until.Month(), until.Day(), 0, 0, 0, 0, loc)
	// Cap dense daily series for "all" to keep the chart usable.
	const maxDays = 120
	daySpan := int(end.Sub(cur).Hours()/24) + 1
	if daySpan > maxDays {
		cur = end.AddDate(0, 0, -(maxDays - 1))
	}
	for !cur.After(end) {
		keys = append(keys, seriesKey(cur, false))
		cur = cur.AddDate(0, 0, 1)
	}
	return keys
}
