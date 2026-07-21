package docker

import "testing"

func TestFirstContainerName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"web", "web"},
		{"/web", "web"},
		{"web,web-alias", "web"},
		{"/web,/alias", "web"},
	}
	for _, tt := range tests {
		if got := firstContainerName(tt.in); got != tt.want {
			t.Fatalf("firstContainerName(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveComposeFromLabels(t *testing.T) {
	tests := []struct {
		dir   string
		files string
		want  string
	}{
		{"", "", ""},
		{"/proj", "", ""},
		{"/proj", "compose.yaml", "/proj/compose.yaml"},
		{"/proj", "/abs/compose.yml", "/abs/compose.yml"},
		{"/proj", "compose.yaml,/proj/override.yml", "/proj/compose.yaml"},
		{"", "compose.yaml", "compose.yaml"},
	}
	for _, tt := range tests {
		got := resolveComposeFromLabels(tt.dir, tt.files)
		if got != tt.want {
			t.Fatalf("resolveComposeFromLabels(%q,%q)=%q want %q", tt.dir, tt.files, got, tt.want)
		}
	}
}
