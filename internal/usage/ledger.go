package usage

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const csvHeader = "timestamp,command,project,provider,model,label,input_tokens,output_tokens,cost_usd"

type Entry struct {
	Timestamp    time.Time
	Command      string
	Project      string
	Provider     string
	Model        string
	Label        string
	InputTokens  int
	OutputTokens int
	CostUSD      *float64
}

var writeMu sync.Mutex

func LedgerPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gitai", "usage", "ledger.csv"), nil
}

func Log(entry Entry) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	path, err := LedgerPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	needsHeader := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		needsHeader = true
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if needsHeader {
		if err := w.Write(stringsSplitCSV(csvHeader)); err != nil {
			return err
		}
	}

	cost := ""
	if entry.CostUSD != nil {
		cost = fmt.Sprintf("%.8f", *entry.CostUSD)
	}

	ts := entry.Timestamp.UTC().Format(time.RFC3339)
	if entry.Timestamp.IsZero() {
		ts = time.Now().UTC().Format(time.RFC3339)
	}

	row := []string{
		ts,
		entry.Command,
		entry.Project,
		entry.Provider,
		entry.Model,
		entry.Label,
		strconv.Itoa(entry.InputTokens),
		strconv.Itoa(entry.OutputTokens),
		cost,
	}
	if err := w.Write(row); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func stringsSplitCSV(s string) []string {
	return []string{
		"timestamp", "command", "project", "provider", "model",
		"label", "input_tokens", "output_tokens", "cost_usd",
	}
}

type Summary struct {
	TotalEntries int
	TotalInput   int
	TotalOutput  int
	TotalCost    float64
	HasCost      bool
	ByProject    map[string]float64
}

func LoadSummary() (*Summary, error) {
	path, err := LedgerPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Summary{ByProject: map[string]float64{}}, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	summary := &Summary{ByProject: map[string]float64{}}
	for i, row := range records {
		if i == 0 || len(row) < 9 {
			continue
		}
		summary.TotalEntries++
		in, _ := strconv.Atoi(row[6])
		out, _ := strconv.Atoi(row[7])
		summary.TotalInput += in
		summary.TotalOutput += out

		if row[8] != "" {
			cost, err := strconv.ParseFloat(row[8], 64)
			if err == nil {
				summary.TotalCost += cost
				summary.HasCost = true
				summary.ByProject[row[2]] += cost
			}
		}
	}
	return summary, nil
}
