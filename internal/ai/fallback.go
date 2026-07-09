package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/laerciocrestani/gitai/internal/config"
)

type FallbackError struct {
	PrimaryModel  string
	PrimaryErr    error
	FallbackModel string
	FallbackErr   error
}

func (e *FallbackError) Error() string {
	primary := userFacingAPIError(e.PrimaryErr)
	fallback := userFacingAPIError(e.FallbackErr)

	if isDeprecatedGeminiModel(e.FallbackModel) {
		return fmt.Sprintf(
			"Gemini: %s indisponível e fallback %s foi descontinuado — atualize fallback_model em `gitai config` (sugestão: gemini-3.1-flash-lite)",
			e.PrimaryModel, e.FallbackModel,
		)
	}

	return fmt.Sprintf(
		"Gemini: %s indisponível (%s); fallback %s também falhou (%s)",
		e.PrimaryModel, primary, e.FallbackModel, fallback,
	)
}

func userFacingAPIError(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.UserMessage()
	}
	return err.Error()
}

func withModelFallback(
	ctx context.Context,
	cfg *config.Config,
	primaryModel string,
	fn func(model string) (string, error),
) (string, error) {
	primaryModel = resolveGeminiModel(primaryModel)

	result, err := fn(primaryModel)
	if err == nil {
		return result, nil
	}
	primaryErr := err

	fallback := resolveGeminiModel(strings.TrimSpace(cfg.FallbackModel))
	if fallback == "" || fallback == primaryModel || !shouldTryModelFallback(err) {
		return "", err
	}

	emitNotice(ctx, fmt.Sprintf("%s indisponível — usando fallback %s...", primaryModel, fallback))
	result, err = fn(fallback)
	if err != nil {
		return "", &FallbackError{
			PrimaryModel:  primaryModel,
			PrimaryErr:    primaryErr,
			FallbackModel: strings.TrimSpace(cfg.FallbackModel),
			FallbackErr:   err,
		}
	}
	return result, nil
}

func shouldTryModelFallback(err error) bool {
	if isRetryableError(err) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
