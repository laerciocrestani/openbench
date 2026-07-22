package main

import "testing"

func TestTrayDiffLabel(t *testing.T) {
	tests := []struct {
		files, ins, del int
		want            string
	}{
		{0, 0, 0, ""},
		{4, 0, 0, "4"},
		{4, 168, 14, "4 +168 -14"},
		{0, 10, 2, "0 +10 -2"},
		{1, 5, 0, "1 +5"},
		{2, 0, 3, "2 -3"},
	}
	for _, tc := range tests {
		got := trayDiffLabel(tc.files, tc.ins, tc.del)
		if got != tc.want {
			t.Fatalf("trayDiffLabel(%d,%d,%d)=%q want %q", tc.files, tc.ins, tc.del, got, tc.want)
		}
	}
}
