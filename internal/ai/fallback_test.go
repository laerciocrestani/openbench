package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/laerciocrestani/gitai/internal/config"
)

func TestWithModelFallbackUsesSecondary(t *testing.T) {
	cfg := &config.Config{
		Model:         "primary",
		FallbackModel: "fallback",
	}
	calls := []string{}

	_, err := withModelFallback(context.Background(), cfg, cfg.Model, func(model string) (string, error) {
		calls = append(calls, model)
		if model == "primary" {
			return "", &APIError{Provider: "Gemini", StatusCode: 503}
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[1] != "fallback" {
		t.Fatalf("calls = %v, want primary then fallback", calls)
	}
}

func TestWithModelFallbackSkipsWhenSameModel(t *testing.T) {
	cfg := &config.Config{
		Model:         "same",
		FallbackModel: "same",
	}
	_, err := withModelFallback(context.Background(), cfg, cfg.Model, func(model string) (string, error) {
		return "", &APIError{Provider: "Gemini", StatusCode: 503}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
}

func TestWithModelFallbackMigratesDeprecatedFallback(t *testing.T) {
	cfg := &config.Config{
		Model:         "gemini-2.5-flash-lite",
		FallbackModel: "gemini-2.0-flash-lite",
	}
	calls := []string{}

	_, err := withModelFallback(context.Background(), cfg, cfg.Model, func(model string) (string, error) {
		calls = append(calls, model)
		if model == "gemini-2.5-flash-lite" {
			return "", &APIError{Provider: "Gemini", StatusCode: 503}
		}
		if model == "gemini-3.1-flash-lite" {
			return "ok", nil
		}
		t.Fatalf("unexpected model %q", model)
		return "", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[1] != "gemini-3.1-flash-lite" {
		t.Fatalf("calls = %v, want fallback migrated to gemini-3.1-flash-lite", calls)
	}
}

func TestWithModelFallbackOnNotFound(t *testing.T) {
	cfg := &config.Config{
		Model:         "gemini-2.5-flash-lite",
		FallbackModel: "gemini-3.1-flash-lite",
	}
	calls := []string{}

	_, err := withModelFallback(context.Background(), cfg, cfg.Model, func(model string) (string, error) {
		calls = append(calls, model)
		if model == "gemini-2.5-flash-lite" {
			return "", &APIError{Provider: "Gemini", StatusCode: 404}
		}
		if model == "gemini-3.1-flash-lite" {
			return "ok", nil
		}
		t.Fatalf("unexpected model %q", model)
		return "", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[1] != "gemini-3.1-flash-lite" {
		t.Fatalf("calls = %v, want fallback after 404", calls)
	}
}

func TestFallbackErrorDeprecatedMessage(t *testing.T) {
	err := &FallbackError{
		PrimaryModel:  "gemini-2.5-flash-lite",
		PrimaryErr:    &APIError{Provider: "Gemini", StatusCode: 503},
		FallbackModel: "gemini-2.0-flash-lite",
		FallbackErr:   &APIError{Provider: "Gemini", StatusCode: 404},
	}
	msg := err.Error()
	if !strings.Contains(msg, "descontinuado") {
		t.Fatalf("expected deprecated hint, got: %q", msg)
	}
	if !strings.Contains(msg, "gemini-3.1-flash-lite") {
		t.Fatalf("expected migration suggestion, got: %q", msg)
	}
}
