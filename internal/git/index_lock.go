package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ClearStaleIndexLock removes an orphaned .git/index.lock left by a crashed git
// process. A very recent lock (<2s) is left alone — another command may be active.
func (r *Repo) ClearStaleIndexLock() error {
	if r == nil || r.dir == "" {
		return nil
	}
	lock := filepath.Join(r.dir, ".git", "index.lock")
	info, err := os.Stat(lock)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	age := time.Since(info.ModTime())
	if age < 2*time.Second {
		return fmt.Errorf("outra operação git parece em andamento (.git/index.lock com %dms)", age.Milliseconds())
	}
	if err := os.Remove(lock); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("não foi possível remover .git/index.lock: %w", err)
	}
	return nil
}

// EnsureWritableIndex clears a stale index lock so subsequent writes can proceed.
func (r *Repo) EnsureWritableIndex() error {
	return r.ClearStaleIndexLock()
}
