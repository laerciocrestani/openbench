package git

// NeedsAdd reports whether the file has unstaged changes that git add can stage.
func (f FileChange) NeedsAdd() bool {
	switch f.Status {
	case "untracked", "modified", "staged+modified", "deleted", "changed", "renamed":
		return true
	default:
		return false
	}
}

// FilterAddable returns files that can be staged with git add.
func FilterAddable(changes []FileChange) []FileChange {
	var out []FileChange
	for _, f := range changes {
		if f.NeedsAdd() {
			out = append(out, f)
		}
	}
	return out
}
