package desktop

import (
	"strings"
	"testing"

	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
)

func TestParseComposeServiceLine_starting(t *testing.T) {
	name, status, _ := parseComposeServiceLine("Container myapp-db-1  Starting")
	if name != "db" || status != "running" {
		t.Fatalf("got name=%q status=%q", name, status)
	}
}

func TestParseComposeServiceLine_started(t *testing.T) {
	name, status, _ := parseComposeServiceLine("Container myapp-api-1  Started")
	if name != "api" || status != "ok" {
		t.Fatalf("got name=%q status=%q", name, status)
	}
}

func TestParseComposeServiceLine_portConflict(t *testing.T) {
	line := "Error response from daemon: Bind for 0.0.0.0:5432 failed: port is already allocated"
	_, status, detail := parseComposeServiceLine(line)
	if status != "error" || detail == "" {
		t.Fatalf("got status=%q detail=%q", status, detail)
	}
}

func TestSummarizeDockerError_port(t *testing.T) {
	out := "Container x Creating\nError response from daemon: Bind for 0.0.0.0:5432 failed: port is already allocated\n"
	got := summarizeDockerError(out, nil)
	lower := strings.ToLower(got)
	if !strings.Contains(lower, "port is already") && !strings.Contains(lower, "bind for") {
		t.Fatalf("got %q", got)
	}
}

func TestShouldVerifyComposeHealth(t *testing.T) {
	if !shouldVerifyComposeHealth("start") || !shouldVerifyComposeHealth("up") || !shouldVerifyComposeHealth("up --build") {
		t.Fatal("expected verify for start/up")
	}
	if shouldVerifyComposeHealth("stop") || shouldVerifyComposeHealth("down") {
		t.Fatal("should not verify stop/down")
	}
}

func TestContainerFailedAfterStart(t *testing.T) {
	if !containerFailedAfterStart(dockerpkg.ContainerSummary{State: "exited"}) {
		t.Fatal("exited should fail")
	}
	if !containerFailedAfterStart(dockerpkg.ContainerSummary{State: "running", Health: "unhealthy"}) {
		t.Fatal("unhealthy should fail")
	}
	if containerFailedAfterStart(dockerpkg.ContainerSummary{State: "running"}) {
		t.Fatal("running healthy should pass")
	}
}

func TestJoinDockerOutput(t *testing.T) {
	got := joinDockerOutput("a", "b")
	if got != "a\n\nb" {
		t.Fatalf("got %q", got)
	}
}
