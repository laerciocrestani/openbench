package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultRetryAttempts = 3
	defaultRetryDelay    = 3 * time.Second
)

var gatewayBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

type APIError struct {
	Provider   string
	StatusCode int
	Body       string
}

func (e *APIError) Retryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func callWithRetry(ctx context.Context, provider string, fn func() (string, error)) (string, error) {
	for attempt := 1; ; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		maxAttempts := maxAttemptsFor(err)
		if !isRetryableError(err) || attempt >= maxAttempts {
			return "", err
		}

		delay := retryDelayFor(err, attempt)
		hint := retryMessage(err, provider)
		emitNotice(ctx, fmt.Sprintf("%s — tentando novamente em %s (%d/%d)...",
			hint, formatDelay(delay), attempt, maxAttempts))

		if err := sleep(ctx, delay); err != nil {
			return "", err
		}
	}
}

func maxAttemptsFor(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) && isGatewayError(apiErr.StatusCode) {
		return len(gatewayBackoff) + 1
	}
	return defaultRetryAttempts
}

func retryDelayFor(err error, attempt int) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) && isGatewayError(apiErr.StatusCode) {
		idx := attempt - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(gatewayBackoff) {
			return gatewayBackoff[len(gatewayBackoff)-1]
		}
		return gatewayBackoff[idx]
	}
	return defaultRetryDelay
}

func isGatewayError(code int) bool {
	return code == http.StatusBadGateway || code == http.StatusGatewayTimeout
}

func formatDelay(d time.Duration) string {
	secs := d.Round(time.Second) / time.Second
	return fmt.Sprintf("%ds", secs)
}

func isRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Retryable()
	}
	return false
}

func retryMessage(err error, provider string) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.retryHint()
	}
	return provider + " indisponível"
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
