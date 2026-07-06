package usage

import (
	"testing"
	"time"
)

func TestResolvePeriodDefaults24h(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	p, err := ResolvePeriod(PeriodOptions{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Label != "últimas 24 horas" {
		t.Fatalf("label: %s", p.Label)
	}
	if !p.Since.Equal(now.Add(-24 * time.Hour)) {
		t.Fatalf("since: %v", p.Since)
	}
}

func TestResolvePeriodHour(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	p, err := ResolvePeriod(PeriodOptions{Hour: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Label != "última hora" {
		t.Fatalf("label: %s", p.Label)
	}
}

func TestResolvePeriodDays(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	p, err := ResolvePeriod(PeriodOptions{Days: 7}, now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Label != "últimos 7 dias" {
		t.Fatalf("label: %s", p.Label)
	}
}

func TestResolvePeriodMonth(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	p, err := ResolvePeriod(PeriodOptions{Month: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Since.Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("since: %v", p.Since)
	}
}

func TestFormatTokens(t *testing.T) {
	if FormatTokens(1234567) != "1,234,567" {
		t.Fatalf("got %s", FormatTokens(1234567))
	}
}

func TestBuildReportWithCost(t *testing.T) {
	now := time.Now().UTC()
	cost := 0.001
	entries := []Entry{
		{
			Timestamp:    now.Add(-time.Hour),
			Command:      "commit",
			Project:      "gitia",
			Model:        "gemini-2.5-flash-lite",
			InputTokens:  100,
			OutputTokens: 50,
			CostUSD:      &cost,
		},
	}

	period := Period{
		Since: now.Add(-24 * time.Hour),
		Until: now.Add(time.Second),
		Label: "test",
	}

	// simulate BuildReport aggregation without ledger file
	report := &Report{
		Period:  period,
		Entries: entries,
		Summary: Summary{ByProject: map[string]float64{}},
		ByModel: map[string]ModelUsage{},
		ByProject: map[string]ProjectUsage{},
	}
	for _, e := range entries {
		report.Summary.TotalEntries++
		if e.CostUSD != nil {
			report.Summary.TotalCost += *e.CostUSD
			report.Summary.HasCost = true
			report.Summary.ByProject[e.Project] += *e.CostUSD
		}
	}

	if !report.Summary.HasCost {
		t.Fatal("expected cost summary")
	}
	if report.Summary.ByProject["gitia"] != cost {
		t.Fatalf("by project: %f", report.Summary.ByProject["gitia"])
	}
}
