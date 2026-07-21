package config

import "testing"

func TestEffectiveChatModelFallsBackToGit(t *testing.T) {
	cfg := &Config{Model: "git-model", ChatModel: ""}
	if got := cfg.EffectiveChatModel(); got != "git-model" {
		t.Fatalf("got %q", got)
	}
	cfg.ChatModel = "chat-model"
	if got := cfg.EffectiveChatModel(); got != "chat-model" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyChatModels(t *testing.T) {
	cfg := &Config{
		Model:             "git-primary",
		FallbackModel:     "git-fb",
		ChatModel:         "chat-primary",
		ChatFallbackModel: "chat-fb",
	}
	cfg.ApplyChatModels()
	if cfg.Model != "chat-primary" || cfg.FallbackModel != "chat-fb" {
		t.Fatalf("got model=%q fallback=%q", cfg.Model, cfg.FallbackModel)
	}
}
