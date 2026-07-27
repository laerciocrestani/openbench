package ai

import "testing"

func TestParseDockerFixSuggestion(t *testing.T) {
	raw := `{
	  "problem": "porta 5173 já alocada pelo serviço onboarding",
	  "resolution": "parar o container conflitante e subir de novo",
	  "steps": [
	    {"id": "1", "kind": "docker_stop", "title": "Stop onboarding", "target": "onboarding", "risk": "ok"},
	    {"id": "2", "kind": "docker_up", "title": "Up onboarding", "target": "onboarding", "risk": "ok"},
	    {"id": "x", "kind": "kill_host", "title": "kill", "target": "1234", "risk": "destructive"},
	    {"id": "3", "kind": "write_file", "title": "Ajustar porta", "target": "compose.yaml", "risk": "warn"}
	  ],
	  "files": [
	    {"path": "compose.yaml", "content": "services:\n", "reason": "mudar mapping da porta 5173"},
	    {"path": "../evil", "content": "x", "reason": "nope"},
	    {"path": "no-reason.yaml", "content": "x", "reason": ""}
	  ],
	  "notes": ["se a porta for processo no host, mate manualmente"]
	}`
	sug, err := parseDockerFixSuggestion(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sug.Problem == "" || sug.Resolution == "" {
		t.Fatalf("incomplete: %+v", sug)
	}
	if len(sug.Steps) != 3 {
		t.Fatalf("steps=%+v want 3 (kill_host filtered)", sug.Steps)
	}
	if len(sug.Files) != 1 || sug.Files[0].Path != "compose.yaml" {
		t.Fatalf("files=%+v", sug.Files)
	}
}

func TestParseDockerFixSuggestionRequiresProblem(t *testing.T) {
	raw := `{"problem":"","resolution":"x","steps":[],"files":[]}`
	if _, err := parseDockerFixSuggestion(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDockerFixSuggestionDropsOrphanWriteFile(t *testing.T) {
	raw := `{
	  "problem": "p",
	  "resolution": "r",
	  "steps": [
	    {"id": "1", "kind": "write_file", "title": "edit", "target": "missing.yml", "risk": "ok"}
	  ],
	  "files": []
	}`
	sug, err := parseDockerFixSuggestion(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(sug.Steps) != 0 {
		t.Fatalf("expected orphan write_file dropped, got %+v", sug.Steps)
	}
}
