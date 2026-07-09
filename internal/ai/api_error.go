package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (e *APIError) Error() string {
	return e.UserMessage()
}

// UserMessage retorna uma mensagem legível em pt-BR, sem JSON bruto.
func (e *APIError) UserMessage() string {
	if hint := statusHint(e.StatusCode); hint != "" {
		return fmt.Sprintf("%s: %s", e.Provider, hint)
	}
	if msg := extractAPIMessage(e.Body); msg != "" {
		return fmt.Sprintf("%s: %s", e.Provider, msg)
	}
	return fmt.Sprintf("%s retornou erro %d", e.Provider, e.StatusCode)
}

func (e *APIError) retryHint() string {
	switch e.StatusCode {
	case http.StatusServiceUnavailable:
		return "Modelo sobrecarregado"
	case http.StatusTooManyRequests:
		return "Limite de requisições atingido"
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return e.Provider + " temporariamente indisponível"
	default:
		return e.Provider + " indisponível"
	}
}

func statusHint(code int) string {
	switch code {
	case http.StatusServiceUnavailable:
		return "modelo com alta demanda no momento — tente novamente em alguns minutos ou escolha outro modelo (`gitai config`)"
	case http.StatusTooManyRequests:
		return "muitas requisições — aguarde um momento e tente novamente"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "chave API inválida ou sem permissão — verifique com `gitai config`"
	case http.StatusNotFound:
		return "modelo não encontrado — confira o nome em `gitai config`"
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return "serviço temporariamente indisponível — tente novamente em instantes"
	case http.StatusInternalServerError:
		return "erro interno do provedor — tente novamente em instantes"
	default:
		return ""
	}
}

func extractAPIMessage(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	var wrapped struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &wrapped); err == nil {
		if msg := strings.TrimSpace(wrapped.Error.Message); msg != "" {
			return msg
		}
	}

	var flat struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &flat); err == nil {
		return strings.TrimSpace(flat.Message)
	}

	if len(body) > 160 {
		return body[:157] + "..."
	}
	return body
}
