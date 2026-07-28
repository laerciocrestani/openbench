package desktop

import (
	"testing"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

func TestSetOriginRemote_add(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitGitAndOpen(dir, false); err != nil {
		t.Fatal(err)
	}
	url := "https://github.com/example/api.git"
	dash, err := SetOriginRemote(dir, url)
	if err != nil {
		t.Fatal(err)
	}
	if dash.RemoteURL != url {
		t.Fatalf("remoteURL: got %q want %q", dash.RemoteURL, url)
	}
	repo, err := gitpkg.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.RemoteOriginURL()
	if err != nil {
		t.Fatal(err)
	}
	if got != url {
		t.Fatalf("origin: got %q want %q", got, url)
	}
}

func TestSetOriginRemote_setURL(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitGitAndOpen(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := SetOriginRemote(dir, "https://github.com/example/old.git"); err != nil {
		t.Fatal(err)
	}
	next := "https://github.com/example/new.git"
	dash, err := SetOriginRemote(dir, next)
	if err != nil {
		t.Fatal(err)
	}
	if dash.RemoteURL != next {
		t.Fatalf("remoteURL: got %q want %q", dash.RemoteURL, next)
	}
}

func TestCreateGitHubRepo_requiresNoOrigin(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitGitAndOpen(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := SetOriginRemote(dir, "https://github.com/example/api.git"); err != nil {
		t.Fatal(err)
	}
	_, err := CreateGitHubRepo(dir, "api", "private", "")
	if err == nil {
		t.Fatal("expected error when origin already exists")
	}
}

func TestValidateGitHubRepoName(t *testing.T) {
	if err := validateGitHubRepoName(""); err == nil {
		t.Fatal("empty")
	}
	if err := validateGitHubRepoName("api"); err != nil {
		t.Fatal(err)
	}
	if err := validateGitHubRepoName("PI-do-Brasil/api"); err != nil {
		t.Fatal(err)
	}
	if err := validateGitHubRepoName("a/b/c"); err == nil {
		t.Fatal("too many slashes")
	}
}
