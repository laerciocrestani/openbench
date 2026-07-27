package git

import (
	"testing"
)

func TestParseBranchHeader(t *testing.T) {
	cases := []struct {
		in                string
		branch, upstream  string
		ahead, behind     int
	}{
		{"main", "main", "", 0, 0},
		{"main...origin/main", "main", "origin/main", 0, 0},
		{"main...origin/main [ahead 2]", "main", "origin/main", 2, 0},
		{"main...origin/main [behind 3]", "main", "origin/main", 0, 3},
		{"feat...origin/feat [ahead 1, behind 2]", "feat", "origin/feat", 1, 2},
		{"HEAD (no branch)", "HEAD", "", 0, 0},
	}
	for _, tc := range cases {
		b, u, a, be := parseBranchHeader(tc.in)
		if b != tc.branch || u != tc.upstream || a != tc.ahead || be != tc.behind {
			t.Fatalf("%q: got branch=%q upstream=%q ahead=%d behind=%d", tc.in, b, u, a, be)
		}
	}
}

func TestShell_openbench(t *testing.T) {
	repo, err := Open("../..")
	if err != nil {
		t.Fatal(err)
	}
	info, err := repo.Shell()
	if err != nil {
		t.Fatal(err)
	}
	if info.Root == "" || info.Branch == "" || info.HeadHash == "" {
		t.Fatalf("incomplete shell: %+v", info)
	}
}
