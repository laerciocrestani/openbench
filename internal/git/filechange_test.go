package git

import "testing"

func TestFileChangeNeedsAdd(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"untracked", true},
		{"modified", true},
		{"staged+modified", true},
		{"deleted", true},
		{"staged", false},
		{"new", false},
	}

	for _, tc := range tests {
		f := FileChange{Path: "x.go", Status: tc.status}
		if got := f.NeedsAdd(); got != tc.want {
			t.Errorf("NeedsAdd(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestFilterAddable(t *testing.T) {
	changes := []FileChange{
		{Path: "a.go", Status: "untracked"},
		{Path: "b.go", Status: "staged"},
		{Path: "c.go", Status: "modified"},
	}
	got := FilterAddable(changes)
	if len(got) != 2 {
		t.Fatalf("FilterAddable len = %d, want 2", len(got))
	}
}
