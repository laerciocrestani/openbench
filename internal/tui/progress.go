package tui

import (
	"sync"

	"github.com/laerciocrestani/gitai/internal/app"
)

const maxProgressBeforeDone = 99

// ActionProgress implementa app.Progress para ações na TUI.
type ActionProgress struct {
	mu         sync.Mutex
	Status     string
	Logs       []string
	percent    int
	cumulative int
	done       bool
}

func NewActionProgress() *ActionProgress {
	return &ActionProgress{}
}

// Reset clears progress state for a new operation.
func (p *ActionProgress) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Status = ""
	p.Logs = nil
	p.percent = 0
	p.cumulative = 0
	p.done = false
}

func (p *ActionProgress) Step(label string, fn func() error) error {
	weight := app.StepWeightFor(label)
	p.setStatus(label + "…")
	p.advanceTo(p.cumulative + weight/2)

	err := fn()

	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.Logs = append(p.Logs, "✗ "+label)
	} else {
		p.Logs = append(p.Logs, "✓ "+label)
		p.cumulative += weight
		if p.cumulative > maxProgressBeforeDone {
			p.cumulative = maxProgressBeforeDone
		}
		p.percent = p.cumulative
	}
	return err
}

func (p *ActionProgress) StepQuiet(fn func() error) error {
	return fn()
}

func (p *ActionProgress) Detail(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Logs = append(p.Logs, "  "+msg)
}

func (p *ActionProgress) Info(msg string) {
	p.Detail(msg)
}

func (p *ActionProgress) Success(msg string) {
	p.setStatus(msg)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done = true
	p.percent = 100
	p.cumulative = 100
	p.Logs = append(p.Logs, "✓ "+msg)
}

func (p *ActionProgress) setStatus(s string) {
	p.mu.Lock()
	p.Status = s
	p.mu.Unlock()
}

func (p *ActionProgress) advanceTo(target int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return
	}
	if target > maxProgressBeforeDone {
		target = maxProgressBeforeDone
	}
	if target > p.percent {
		p.percent = target
	}
}

// Percent returns the current progress (0–100).
func (p *ActionProgress) Percent() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.percent
}

func (p *ActionProgress) Snapshot() (status string, logs []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	status = p.Status
	logs = append([]string(nil), p.Logs...)
	return status, logs
}

var _ app.Progress = (*ActionProgress)(nil)
