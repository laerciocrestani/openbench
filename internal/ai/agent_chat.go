package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
)

const maxAgentRounds = 8

// ToolExecutor runs an approved (or auto-allowed) tool call and returns a text
// result for the model.
type ToolExecutor func(ctx context.Context, call ToolCall) (string, error)

// ToolApprover asks the user whether a privileged tool may run.
// Returns false if denied; error if cancelled/failed.
type ToolApprover func(ctx context.Context, call ToolCall) (bool, error)

// RunAgentChat runs a multi-turn chat with optional tool calls.
// Privileged tools invoke approve before execute.
func RunAgentChat(
	ctx context.Context,
	cfg *config.Config,
	messages []ChatMessage,
	execute ToolExecutor,
	approve ToolApprover,
	onChunk func(delta string),
) (*ChatStreamResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config ausente")
	}
	if execute == nil {
		return nil, fmt.Errorf("executor de tools ausente")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("nenhuma mensagem")
	}

	msgs := append([]ChatMessage(nil), messages...)
	var (
		finalContent strings.Builder
		usage        UsageSummary
		lastModel    string
	)

	for round := 0; round < maxAgentRounds; round++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		turn, err := completeChatTurn(ctx, cfg, msgs, true)
		if err != nil {
			return nil, err
		}
		usage.Add(turn.Usage)
		if turn.Usage.Model != "" {
			lastModel = turn.Usage.Model
		}

		if len(turn.ToolCalls) == 0 {
			text := turn.Content
			if text != "" {
				finalContent.WriteString(text)
				if onChunk != nil {
					onChunk(text)
				}
			}
			if finalContent.Len() == 0 {
				return nil, fmt.Errorf("API retornou resposta vazia")
			}
			p, c, t, cost, hasCost := usage.Totals()
			var costPtr *float64
			if hasCost {
				costCopy := cost
				costPtr = &costCopy
			}
			model := lastModel
			if model == "" {
				model = turn.Usage.Model
			}
			return &ChatStreamResult{
				Content: finalContent.String(),
				Usage: UsageRecord{
					Label:            "chat",
					Model:            model,
					PromptTokens:     p,
					CompletionTokens: c,
					TotalTokens:      t,
					CostUSD:          costPtr,
				},
			}, nil
		}

		// Emit any preamble text from the model before tools.
		if turn.Content != "" {
			finalContent.WriteString(turn.Content)
			if onChunk != nil {
				onChunk(turn.Content)
			}
		}

		assistant := ChatMessage{
			Role:      "assistant",
			Content:   turn.Content,
			ToolCalls: turn.ToolCalls,
		}
		msgs = append(msgs, assistant)

		for _, call := range turn.ToolCalls {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			resultText, err := runOneTool(ctx, call, execute, approve, onChunk)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, ChatMessage{
				Role:       "tool",
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    resultText,
			})
		}
	}

	return nil, fmt.Errorf("limite de rounds do agent atingido (%d)", maxAgentRounds)
}

func runOneTool(
	ctx context.Context,
	call ToolCall,
	execute ToolExecutor,
	approve ToolApprover,
	onChunk func(delta string),
) (string, error) {
	name := strings.TrimSpace(call.Name)
	if name == "" {
		return "erro: tool sem nome", nil
	}

	if ToolNeedsApproval(name) {
		if approve == nil {
			return fmt.Sprintf("erro: tool %q requer aprovação, mas nenhum approver está configurado", name), nil
		}
		ok, err := approve(ctx, call)
		if err != nil {
			return "", err
		}
		if !ok {
			msg := fmt.Sprintf("usuário negou a execução de %s", name)
			if onChunk != nil {
				onChunk("\n\n⚠️ " + msg + "\n")
			}
			return msg, nil
		}
	}

	out, err := execute(ctx, call)
	if err != nil {
		return fmt.Sprintf("erro ao executar %s: %v", name, err), nil
	}
	return out, nil
}

type chatTurnResult struct {
	Content   string
	ToolCalls []ToolCall
	Usage     UsageRecord
}

func completeChatTurn(
	ctx context.Context,
	cfg *config.Config,
	messages []ChatMessage,
	withTools bool,
) (*chatTurnResult, error) {
	switch cfg.Provider {
	case config.ProviderOpenAI:
		return completeOpenAIChatTurn(ctx, cfg, "https://api.openai.com/v1/chat/completions", messages, withTools)
	case config.ProviderOpenRouter:
		return completeOpenAIChatTurn(ctx, cfg, "https://openrouter.ai/api/v1/chat/completions", messages, withTools)
	case config.ProviderGemini:
		return completeGeminiChatTurn(ctx, cfg, messages, withTools)
	default:
		return nil, fmt.Errorf("provider desconhecido: %s", cfg.Provider)
	}
}
