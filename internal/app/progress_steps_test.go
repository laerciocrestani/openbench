package app_test

import (
	"testing"

	"github.com/laerciocrestani/gitai/internal/app"
)

func TestStepWeightForKnownLabels(t *testing.T) {
	if got := app.StepWeightFor("Thinking"); got != 55 {
		t.Fatalf("Thinking weight = %d, want 55", got)
	}
	if got := app.StepWeightFor("Pulling main"); got != 25 {
		t.Fatalf("Pulling prefix weight = %d, want 25", got)
	}
}

func TestStepWeightForUnknownDefaults(t *testing.T) {
	if got := app.StepWeightFor("Custom step"); got != 10 {
		t.Fatalf("default weight = %d, want 10", got)
	}
}
