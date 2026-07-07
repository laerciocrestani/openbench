package ui

import (
	"strings"
	"testing"
)

func TestIndexOf(t *testing.T) {
	opts := []string{"a", "b", "c"}
	if got := indexOf(opts, "b"); got != 1 {
		t.Fatalf("indexOf(b) = %d, want 1", got)
	}
	if got := indexOf(opts, "z"); got != -1 {
		t.Fatalf("indexOf(z) = %d, want -1", got)
	}
}

func TestWizardBuildFrameKeepsEntries(t *testing.T) {
	sess := New("config", false)
	sess.enabled = false
	w := NewWizard(sess, "Configuração", "intro")
	w.entries = []wizardEntry{
		{label: "Provedor", value: "gemini"},
		{label: "Modelo", value: "gemini-2.5-flash-lite"},
	}
	frame := w.buildFrame(nil, "", "")
	if !strings.Contains(frame, "Provedor: gemini") || !strings.Contains(frame, "Modelo: gemini-2.5-flash-lite") {
		t.Fatalf("frame missing entries: %q", frame)
	}
}

func TestWizardBuildFrameHighlightsSelection(t *testing.T) {
	sess := New("config", false)
	sess.enabled = true
	w := NewWizard(sess, "Configuração", "intro")
	frame := w.buildFrame(&selectState{
		label:   "Tamanho da fonte na interface",
		options: []string{"Pequeno", "Normal", "Grande"},
		cursor:  1,
	}, "", "")
	if !strings.Contains(frame, "▸") || !strings.Contains(frame, "Normal") {
		t.Fatalf("frame missing highlighted selection: %q", frame)
	}
}
