package gha

import (
	"strings"
	"testing"
)

func TestParseWorkflowDispatchInputs_choiceRequired(t *testing.T) {
	yaml := `
name: Deploy Cloudflare
on:
  workflow_dispatch:
    inputs:
      target:
        description: What to deploy
        required: true
        type: choice
        options:
          - all
          - frontend
          - sites
        default: all
`
	can, inputs, err := ParseWorkflowDispatchInputs(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if !can {
		t.Fatal("expected canDispatch")
	}
	if len(inputs) != 1 {
		t.Fatalf("inputs=%d want 1: %#v", len(inputs), inputs)
	}
	in := inputs[0]
	if in.ID != "target" || !in.Required || in.Type != "choice" || in.Default != "all" {
		t.Fatalf("input = %#v", in)
	}
	if len(in.Options) != 3 || in.Options[0] != "all" {
		t.Fatalf("options = %#v", in.Options)
	}
}

func TestParseWorkflowDispatchInputs_onAsList(t *testing.T) {
	yaml := `
on:
  - push
  - workflow_dispatch
`
	can, inputs, err := ParseWorkflowDispatchInputs(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if !can {
		t.Fatal("expected canDispatch")
	}
	if len(inputs) != 0 {
		t.Fatalf("expected no inputs, got %#v", inputs)
	}
}

func TestParseWorkflowDispatchInputs_noDispatch(t *testing.T) {
	yaml := `
on:
  push:
    branches: [main]
`
	can, _, err := ParseWorkflowDispatchInputs(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if can {
		t.Fatal("expected no dispatch")
	}
}

func TestMergeAndMissingRequired(t *testing.T) {
	inputs := []DispatchInput{
		{ID: "target", Required: true, Type: "choice", Options: []string{"all", "frontend"}, Default: "all"},
		{ID: "note", Required: false, Type: "string"},
	}
	fields := MergeDispatchDefaults(inputs, nil)
	if fields["target"] != "all" {
		t.Fatalf("default target = %q", fields["target"])
	}
	if miss := MissingRequiredDispatchInputs(inputs, fields); len(miss) != 0 {
		t.Fatalf("unexpected missing: %v", miss)
	}
	delete(fields, "target")
	miss := MissingRequiredDispatchInputs(inputs, fields)
	if len(miss) != 1 || miss[0] != "target" {
		t.Fatalf("missing = %v", miss)
	}
}

func TestParseWorkflowDispatchInputs_yamlOnTrueKey(t *testing.T) {
	// Some YAML 1.1 parsers treat "on" as boolean; our unmarshal uses map keys as strings
	// from gopkg.in/yaml.v3 which keeps "on". Also accept literal key quirks via doc["true"].
	yaml := strings.TrimSpace(`
name: x
on: workflow_dispatch
`)
	can, _, err := ParseWorkflowDispatchInputs(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if !can {
		t.Fatal("expected canDispatch for on: workflow_dispatch")
	}
}
