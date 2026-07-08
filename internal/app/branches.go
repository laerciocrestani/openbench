package app

import (
	"fmt"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

// LoadBranchDetail returns contextual information for a branch.
func LoadBranchDetail(name string, snap *WorkspaceSnapshot) (*gitpkg.BranchDetail, error) {
	if name == "" {
		return nil, fmt.Errorf("nome da branch vazio")
	}
	base := "main"
	if snap != nil && snap.Overview != nil && snap.Overview.BaseBranch != "" {
		base = snap.Overview.BaseBranch
	}
	repo, err := gitpkg.New()
	if err != nil {
		return nil, err
	}
	return repo.BranchDetail(name, base)
}

// CheckoutBranch switches to the given local branch.
func CheckoutBranch(name string) error {
	if name == "" {
		return fmt.Errorf("nome da branch vazio")
	}
	repo, err := gitpkg.New()
	if err != nil {
		return err
	}
	return repo.Checkout(name)
}
