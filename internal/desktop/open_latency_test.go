package desktop_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/laerciocrestani/openbench/internal/desktop"
)

func TestBenchOpenLatency(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		t0 := time.Now()
		d, err := desktop.LoadDashboard(root)
		shell := time.Since(t0)
		if err != nil {
			t.Fatal(err)
		}
		t1 := time.Now()
		g, err := desktop.LoadGitStatus(root)
		status := time.Since(t1)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("shell[%d]=%v status[%d]=%v branch=%s dirty=%v files=%d",
			i, shell, i, status, d.Branch, g.Dirty, len(g.ChangedFiles))
		fmt.Printf("shell[%d]=%v status[%d]=%v\n", i, shell, i, status)
		if shell > 500*time.Millisecond {
			t.Fatalf("shell open too slow: %v (want <500ms)", shell)
		}
	}
}
