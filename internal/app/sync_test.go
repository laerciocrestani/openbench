package app

import (
	"strings"
	"testing"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

type recordedProgress struct {
	steps   []string
	details []string
}

func (r *recordedProgress) Step(label string, fn func() error) error {
	r.steps = append(r.steps, label)
	return fn()
}

func (r *recordedProgress) StepQuiet(fn func() error) error { return fn() }
func (r *recordedProgress) Detail(msg string)               { r.details = append(r.details, msg) }
func (r *recordedProgress) Info(msg string)                 { r.Detail(msg) }
func (r *recordedProgress) Warn(msg string)                 { r.Detail(msg) }
func (r *recordedProgress) Success(msg string)              { r.steps = append(r.steps, "success:"+msg) }

func TestPrunePhaseOrder_remoteBeforeLocal(t *testing.T) {
	rec := &recordedProgress{}
	repo, err := gitpkg.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.IsRepo(); err != nil {
		t.Skip("not a git repo")
	}
	clean, err := repo.IsClean()
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Skip("dirty working tree")
	}

	// Dry-run exercises the full prune path without mutating branches.
	err = RunSync(SyncOptions{
		Prune:    true,
		Base:     "main",
		DryRun:   true,
		Progress: rec,
	})
	if err != nil {
		t.Fatalf("RunSync: %v", err)
	}

	joined := strings.Join(rec.steps, "|")
	remoteIdx := strings.Index(joined, "Removing remote ")
	refreshIdx := strings.Index(joined, "Refreshing origin")
	localIdx := strings.Index(joined, "Removing local ")

	if remoteIdx >= 0 && localIdx >= 0 && localIdx < remoteIdx {
		t.Fatalf("local prune ran before remote prune: %v", rec.steps)
	}
	if remoteIdx >= 0 && refreshIdx >= 0 && refreshIdx < remoteIdx {
		t.Fatalf("expected refresh after remote deletes: %v", rec.steps)
	}
	if refreshIdx >= 0 && localIdx >= 0 && localIdx < refreshIdx {
		t.Fatalf("expected local prune after refresh: %v", rec.steps)
	}
}
