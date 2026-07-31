package ai

import (
	"strings"
	"testing"
)

func TestChangeAreasFromStat_piDoBrasil(t *testing.T) {
	stat := ` app/Console/Commands/ReprocessMetaLeadCommand.php | 138 +++++++++++++
 app/Http/Controllers/LeadController.php           | 229 ++++++++++++++++++----
 app/Http/Controllers/PaymentController.php        | 137 +++++++------
 resources/views/customers/auto-payments.blade.php |  29 +--
 resources/views/customers/payments.blade.php      |  25 +--
 5 files changed, 422 insertions(+), 136 deletions(-)`

	areas := ChangeAreasFromStat(stat)
	if len(areas) != 4 {
		t.Fatalf("areas = %d, want 4: %+v", len(areas), areas)
	}

	keys := make([]string, len(areas))
	for i, a := range areas {
		keys[i] = a.Key
	}
	want := []string{
		"app/Console/Commands/ReprocessMetaLeadCommand",
		"app/Http/Controllers/LeadController",
		"app/Http/Controllers/PaymentController",
		"resources/views/customers",
	}
	for i, w := range want {
		if keys[i] != w {
			t.Fatalf("area[%d] = %q, want %q (all: %v)", i, keys[i], w, keys)
		}
	}
}

func TestShouldSuggestSplit(t *testing.T) {
	stat := ` app/Console/Commands/Foo.php | 1 +
 app/Http/Controllers/Bar.php | 1 +
 1 file changed, 1 insertion(+)`
	areas := ChangeAreasFromStat(stat)
	if !ShouldSuggestSplit(areas) {
		t.Fatal("expected split suggestion for multiple areas")
	}
}

func TestFormatSplitSuggestion(t *testing.T) {
	areas := []ChangeArea{
		{Key: "app/Console/Commands/Foo"},
		{Key: "app/Http/Controllers/Bar"},
	}
	msg := FormatSplitSuggestion(areas)
	if !strings.Contains(msg, "git add -p") {
		t.Fatalf("missing split hint: %s", msg)
	}
	if !strings.Contains(msg, "app/Console/Commands/Foo") {
		t.Fatalf("missing area names: %s", msg)
	}
}

func TestBuildPrompt_includesStatAndRules(t *testing.T) {
	prompt := buildPrompt("diff-body", " foo.go | 1 +", "pt-BR", "standard")
	for _, want := range []string{
		"git diff --stat",
		"foo.go",
		"TODAS as áreas alteradas",
		"não invente funcionalidades",
		"diff-body",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildPrompt_detailLevels(t *testing.T) {
	minimal := buildPrompt("d", "a.go | 1 +", "pt-BR", "minimal")
	if !strings.Contains(minimal, "1-3 bullets") {
		t.Fatalf("minimal prompt missing short body rule:\n%s", minimal)
	}
	thorough := buildPrompt("d", "a.go | 1 +", "pt-BR", "thorough")
	if !strings.Contains(thorough, "4-8 bullets") {
		t.Fatalf("thorough prompt missing dense body rule:\n%s", thorough)
	}
}

func TestBuildPRPrompt_detailLevels(t *testing.T) {
	minimal := buildPRPrompt("d", "feat", "main", "pt-BR", "", "minimal")
	if !strings.Contains(minimal, "1-2 bullets") {
		t.Fatalf("minimal PR prompt missing short summary rule:\n%s", minimal)
	}
	standard := buildPRPrompt("d", "feat", "main", "pt-BR", "abc", "standard")
	if !strings.Contains(standard, "3-8 bullets") {
		t.Fatalf("standard PR prompt missing changes rule:\n%s", standard)
	}
	thorough := buildPRPrompt("d", "feat", "main", "pt-BR", "", "thorough")
	if !strings.Contains(thorough, "5-10 bullets") {
		t.Fatalf("thorough PR prompt missing dense changes rule:\n%s", thorough)
	}
}

func TestTruncateDiff(t *testing.T) {
	in := strings.Repeat("a", 100)
	got := truncateDiff(in, 50)
	if !strings.Contains(got, "[diff truncado]") {
		t.Fatalf("expected truncation marker: %s", got)
	}
	if len(got) <= 50 {
		t.Fatalf("truncated output too short: %d", len(got))
	}
	if truncateDiff(in, 200) != in {
		t.Fatal("expected no truncation under limit")
	}
}
