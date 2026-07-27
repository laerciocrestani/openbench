package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laerciocrestani/openbench/internal/ai"
)

func TestDockerFixRunnerWriteFile(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(compose, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &dockerFixRunner{
		projectPath: dir,
		composeFile: compose,
		steps: []ai.DockerFixStep{
			{ID: "1", Kind: "write_file", Title: "edit env", Target: ".env", Risk: "warn"},
		},
		files: map[string]ai.DockerFixFile{
			".env": {Path: ".env", Content: "FOO=1\n", Reason: "porta"},
		},
	}
	sr, done, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if sr.Status != "ok" {
		t.Fatalf("status=%s detail=%s", sr.Status, sr.Detail)
	}
	if !done {
		t.Fatal("expected done")
	}
	got, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "FOO=1\n" {
		t.Fatalf("content=%q", got)
	}
}

func TestDockerFixRunnerRejectsUnknownKind(t *testing.T) {
	r := &dockerFixRunner{
		projectPath: t.TempDir(),
		composeFile: "x",
		steps: []ai.DockerFixStep{
			{ID: "1", Kind: "kill_host", Title: "bad", Target: "1", Risk: "destructive"},
		},
	}
	sr, _, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if sr.Status != "error" {
		t.Fatalf("want error, got %s", sr.Status)
	}
}

func TestMapDockerFixSuggestionCanFix(t *testing.T) {
	view := mapDockerFixSuggestion(&ai.DockerFixSuggestion{
		Problem:    "p",
		Resolution: "r",
		Steps: []ai.DockerFixStep{
			{ID: "1", Kind: "docker_stop", Title: "stop", Target: "web", Risk: "ok"},
		},
	}, "start")
	if !view.CanFix || view.Action != "start" {
		t.Fatalf("%+v", view)
	}
	empty := mapDockerFixSuggestion(&ai.DockerFixSuggestion{
		Problem: "p", Resolution: "r",
	}, "up")
	if empty.CanFix {
		t.Fatal("expected CanFix false")
	}
}
