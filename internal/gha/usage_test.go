package gha

import (
	"errors"
	"strings"
	"testing"
)

func TestUsageForRuns_hidesOptionalBillingAuthNoise(t *testing.T) {
	c := &Client{}
	// Force orgBilling path with a stub by using empty owner first.
	usage := c.UsageForRuns(nil, "")
	if usage.State != UsageStateRepoWindow {
		t.Fatalf("state=%q", usage.State)
	}
	if !strings.Contains(usage.Message, "nenhum workflow run") {
		t.Fatalf("message=%q", usage.Message)
	}
}

func TestIsOptionalBillingErr(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New(`gh: This API operation needs the "admin:org" scope. To request it, run: gh auth refresh -h github.com -s admin:org`), true},
		{errors.New("HTTP 403: Forbidden"), true},
		{errors.New("You must be an org admin or have the actions policies fine-grained permission"), true},
		{errors.New("network timeout talking to api.github.com"), false},
	}
	for _, tc := range cases {
		if got := isOptionalBillingErr(tc.err); got != tc.want {
			t.Fatalf("err=%q got=%v want=%v", tc.err, got, tc.want)
		}
	}
}

func TestClassifyGhErr_billingScopeNotAuthLogin(t *testing.T) {
	err := classifyGhErr(
		`gh: This API operation needs the "admin:org" scope. To request it, run:  gh auth refresh -h github.com -s admin:org`,
		errors.New("exit status 1"),
	)
	var ge *Error
	if !errors.As(err, &ge) {
		t.Fatalf("want *Error, got %T", err)
	}
	if !errors.Is(ge, ErrForbidden) {
		t.Fatalf("kind=%v want Forbidden", ge.Kind)
	}
	if strings.Contains(strings.ToLower(ge.Error()), "auth login") {
		t.Fatalf("should not ask auth login for billing scope: %q", ge.Error())
	}
}
