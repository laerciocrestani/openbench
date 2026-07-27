package desktop

import (
	"strings"
	"testing"
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
