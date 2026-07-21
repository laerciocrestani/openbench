package desktop

import (
	"testing"

	"github.com/laerciocrestani/openbench/internal/usage"
)

func TestSplitUsageByCommand(t *testing.T) {
	chatCost := 0.002
	otherCost := 0.01
	entries := []usage.Entry{
		{Command: "chat", InputTokens: 100, OutputTokens: 20, CostUSD: &chatCost},
		{Command: "commit", InputTokens: 200, OutputTokens: 40, CostUSD: &otherCost},
		{Command: "pr", InputTokens: 50, OutputTokens: 10},
		{Command: "CHAT", InputTokens: 10, OutputTokens: 5, CostUSD: &chatCost},
	}

	chat, other := splitUsageByCommand(entries)
	if chat.Calls != 2 || chat.Input != 110 || chat.Output != 25 {
		t.Fatalf("chat bucket: %+v", chat)
	}
	if !chat.HasCost || chat.Cost < 0.0039 || chat.Cost > 0.0041 {
		t.Fatalf("chat cost: %+v", chat)
	}
	if other.Calls != 2 || other.Input != 250 || other.Output != 50 {
		t.Fatalf("other bucket: %+v", other)
	}
	if !other.HasCost || other.Cost != otherCost {
		t.Fatalf("other cost: %+v", other)
	}
}

func TestSplitUsageByCommandEmpty(t *testing.T) {
	chat, other := splitUsageByCommand(nil)
	if chat.Calls != 0 || other.Calls != 0 {
		t.Fatalf("expected empty buckets")
	}
}
