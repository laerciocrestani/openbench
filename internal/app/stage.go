package app

import (
	"fmt"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

// CanAdd reports whether there are unstaged files that can be added to the index.
func CanAdd(snap *WorkspaceSnapshot) bool {
	return len(AddableFiles(snap)) > 0
}

// AddableFiles returns unstaged/untracked files from the workspace snapshot.
func AddableFiles(snap *WorkspaceSnapshot) []gitpkg.FileChange {
	if snap == nil || snap.Overview == nil {
		return nil
	}
	return gitpkg.FilterAddable(snap.Overview.FileChanges)
}

// StageAll runs git add .
func StageAll() error {
	repo, err := gitpkg.New()
	if err != nil {
		return err
	}
	if err := repo.IsRepo(); err != nil {
		return fmt.Errorf("diretório atual não é um repositório git")
	}
	return repo.AddAll()
}

// StageFiles runs git add on the given paths.
func StageFiles(paths []string) error {
	if len(paths) == 0 {
		return StageAll()
	}
	repo, err := gitpkg.New()
	if err != nil {
		return err
	}
	if err := repo.IsRepo(); err != nil {
		return fmt.Errorf("diretório atual não é um repositório git")
	}
	return repo.Add(paths...)
}
