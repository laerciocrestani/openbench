package usage

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Period struct {
	Since time.Time
	Until time.Time
	Label string
}

type Report struct {
	Period   Period
	Entries  []Entry
	Summary  Summary
	ByModel  map[string]ModelUsage
	ByProject map[string]ProjectUsage
}

type ModelUsage struct {
	Model        string
	Calls        int
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	HasCost      bool
}

type ProjectUsage struct {
	Project      string
	Calls        int
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	HasCost      bool
}

func LoadEntries() ([]Entry, error) {
	path, err := LedgerPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(records))
	for i, row := range records {
		if i == 0 || len(row) < 9 {
			continue
		}
		entry, err := parseEntry(row)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseEntry(row []string) (Entry, error) {
	ts, err := time.Parse(time.RFC3339, row[0])
	if err != nil {
		return Entry{}, err
	}

	in, _ := strconv.Atoi(row[6])
	out, _ := strconv.Atoi(row[7])

	entry := Entry{
		Timestamp:    ts,
		Command:      row[1],
		Project:      row[2],
		Provider:     row[3],
		Model:        row[4],
		Label:        row[5],
		InputTokens:  in,
		OutputTokens: out,
	}

	if row[8] != "" {
		cost, err := strconv.ParseFloat(row[8], 64)
		if err == nil {
			entry.CostUSD = &cost
		}
	}
	return entry, nil
}

func BuildReport(period Period) (*Report, error) {
	all, err := LoadEntries()
	if err != nil {
		return nil, err
	}

	filtered := make([]Entry, 0, len(all))
	for _, e := range all {
		if e.Timestamp.Before(period.Since) {
			continue
		}
		if !e.Timestamp.Before(period.Until) {
			continue
		}
		filtered = append(filtered, e)
	}

	report := &Report{
		Period:  period,
		Entries: filtered,
		Summary: Summary{ByProject: map[string]float64{}},
		ByModel: map[string]ModelUsage{},
		ByProject: map[string]ProjectUsage{},
	}

	for _, e := range filtered {
		report.Summary.TotalEntries++
		report.Summary.TotalInput += e.InputTokens
		report.Summary.TotalOutput += e.OutputTokens

		if e.CostUSD != nil {
			report.Summary.TotalCost += *e.CostUSD
			report.Summary.HasCost = true
			report.Summary.ByProject[e.Project] += *e.CostUSD
		}

		mu := report.ByModel[e.Model]
		mu.Model = e.Model
		mu.Calls++
		mu.InputTokens += e.InputTokens
		mu.OutputTokens += e.OutputTokens
		if e.CostUSD != nil {
			mu.CostUSD += *e.CostUSD
			mu.HasCost = true
		}
		report.ByModel[e.Model] = mu

		pu := report.ByProject[e.Project]
		pu.Project = e.Project
		pu.Calls++
		pu.InputTokens += e.InputTokens
		pu.OutputTokens += e.OutputTokens
		if e.CostUSD != nil {
			pu.CostUSD += *e.CostUSD
			pu.HasCost = true
		}
		report.ByProject[e.Project] = pu
	}

	return report, nil
}

type PeriodOptions struct {
	Hour  bool
	Hours int
	Days  int
	Month bool
	All   bool
}

func ResolvePeriod(opts PeriodOptions, now time.Time) (Period, error) {
	until := now.UTC()

	switch {
	case opts.All:
		return Period{
			Since: time.Time{}.UTC(),
			Until: until.Add(time.Second),
			Label: "todo o histórico",
		}, nil
	case opts.Month:
		since := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, time.UTC)
		return Period{
			Since: since,
			Until: until,
			Label: fmt.Sprintf("mês atual (%s)", since.Format("2006-01")),
		}, nil
	case opts.Days > 0:
		since := until.Add(-time.Duration(opts.Days) * 24 * time.Hour)
		label := fmt.Sprintf("últimos %d dias", opts.Days)
		if opts.Days == 1 {
			label = "últimas 24 horas"
		}
		return Period{Since: since, Until: until, Label: label}, nil
	case opts.Hour:
		since := until.Add(-time.Hour)
		return Period{Since: since, Until: until, Label: "última hora"}, nil
	case opts.Hours > 0:
		since := until.Add(-time.Duration(opts.Hours) * time.Hour)
		label := fmt.Sprintf("últimas %d horas", opts.Hours)
		if opts.Hours == 1 {
			label = "última hora"
		}
		return Period{Since: since, Until: until, Label: label}, nil
	default:
		since := until.Add(-24 * time.Hour)
		return Period{Since: since, Until: until, Label: "últimas 24 horas"}, nil
	}
}

func FormatTokens(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}

	var b strings.Builder
	start := len(s) % 3
	if start > 0 {
		b.WriteString(s[:start])
	}
	for i := start; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
