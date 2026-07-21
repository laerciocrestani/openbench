package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/laerciocrestani/openbench/internal/ai"
	"github.com/laerciocrestani/openbench/internal/app"
	"github.com/laerciocrestani/openbench/internal/config"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// ChatMessageView is a conversational turn for the desktop chat UI.
type ChatMessageView struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatDonePayload is emitted on chat:done after a successful stream.
type ChatDonePayload struct {
	Content          string   `json:"content"`
	Model            string   `json:"model"`
	PromptTokens     int      `json:"promptTokens"`
	CompletionTokens int      `json:"completionTokens"`
	TotalTokens      int      `json:"totalTokens"`
	CostUSD          *float64 `json:"costUSD,omitempty"`
	UsageLine        string   `json:"usageLine,omitempty"`
}

// ChatAgentHooks wires UI approval into the agent loop.
type ChatAgentHooks struct {
	OnChunk          func(delta string)
	RequestTool      func(ctx context.Context, req ChatToolRequest) (approved bool, err error)
	OnProjectMutated func() // after write_file / run_command succeeds
}

// BuildProjectChatSystemPrompt builds a short system prompt from the open project.
func BuildProjectChatSystemPrompt(projectPath string) string {
	projectPath = strings.TrimSpace(projectPath)
	name := filepath.Base(projectPath)
	branch := "(desconhecida)"
	stat := "(indisponível)"

	if repo, err := gitpkg.Open(projectPath); err == nil {
		if b, err := repo.CurrentBranch(); err == nil && b != "" {
			branch = b
		}
		if s, err := repo.DiffStatForCommit(); err == nil {
			s = strings.TrimSpace(s)
			if s == "" {
				stat = "(working tree limpa)"
			} else if len(s) > 2500 {
				stat = s[:2500] + "\n… [stat truncado]"
			} else {
				stat = s
			}
		}
		name = repo.ProjectName()
	}

	var b strings.Builder
	b.WriteString("Você é o assistente do openbench, um app de desenvolvimento (git, Docker Compose, commits/PR com IA).\n")
	b.WriteString("Responda em português brasileiro, de forma objetiva e prática.\n")
	b.WriteString("Use o contexto do projeto abaixo. Se faltar informação, diga o que precisa.\n\n")
	b.WriteString("Você tem tools para agir no projeto:\n")
	b.WriteString("- read_file / list_dir: leitura (automática)\n")
	b.WriteString("- write_file / run_command: exigem aprovação explícita do usuário no app\n")
	b.WriteString("Quando o usuário pedir para criar/editar arquivos ou rodar comandos, USE as tools — ")
	b.WriteString("não diga que não tem acesso ao filesystem. Nunca afirme que escreveu um arquivo ")
	b.WriteString("ou executou um comando sem ter chamado a tool correspondente.\n")
	b.WriteString("Paths são relativos à raiz do projeto.\n\n")
	b.WriteString("Projeto: ")
	b.WriteString(name)
	b.WriteString("\nCaminho: ")
	b.WriteString(projectPath)
	b.WriteString("\nBranch: ")
	b.WriteString(branch)
	b.WriteString("\n\nResumo do working tree (git diff --stat):\n")
	b.WriteString(stat)
	return b.String()
}

// RunProjectChatStream streams a chat reply for the open project and records usage.
// When hooks.RequestTool is set, privileged tools pause for user approval.
// modelOverride, when non-empty, replaces cfg.Model for this request only.
func RunProjectChatStream(
	ctx context.Context,
	projectPath string,
	history []ChatMessageView,
	userMessage string,
	modelOverride string,
	hooks ChatAgentHooks,
) (*ChatDonePayload, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return nil, fmt.Errorf("abra um projeto para usar o chat")
	}
	userMessage = strings.TrimSpace(userMessage)
	if userMessage == "" {
		return nil, fmt.Errorf("mensagem vazia")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("api_key não configurada — use Configurações")
	}
	cfg.ApplyChatModels()
	if m := strings.TrimSpace(modelOverride); m != "" {
		cfg.Model = m
	}

	messages := make([]ai.ChatMessage, 0, len(history)+2)
	messages = append(messages, ai.ChatMessage{
		Role:    "system",
		Content: BuildProjectChatSystemPrompt(projectPath),
	})
	for _, h := range history {
		role := strings.TrimSpace(h.Role)
		content := strings.TrimSpace(h.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		messages = append(messages, ai.ChatMessage{Role: role, Content: content})
	}
	messages = append(messages, ai.ChatMessage{Role: "user", Content: userMessage})

	execute := func(ctx context.Context, call ai.ToolCall) (string, error) {
		out, err := ExecuteChatTool(ctx, projectPath, call)
		if err == nil && chatToolMutatesProject(call.Name) && hooks.OnProjectMutated != nil {
			hooks.OnProjectMutated()
		}
		return out, err
	}

	approve := func(ctx context.Context, call ai.ToolCall) (bool, error) {
		if hooks.RequestTool == nil {
			return false, fmt.Errorf("aprovação de tool não disponível")
		}
		req := BuildChatToolRequest(call)
		if hooks.OnChunk != nil {
			hooks.OnChunk(fmt.Sprintf("\n\n⏳ Aguardando permissão: %s\n", req.Summary))
		}
		ok, err := hooks.RequestTool(ctx, req)
		if err != nil {
			return false, err
		}
		if ok && hooks.OnChunk != nil {
			hooks.OnChunk("✅ Permissão concedida — executando…\n")
		}
		return ok, nil
	}

	result, err := ai.RunAgentChat(ctx, cfg, messages, execute, approve, hooks.OnChunk)
	if err != nil {
		return nil, err
	}

	summary := ai.UsageSummary{}
	summary.Add(result.Usage)
	app.RecordAIUsageForProject("chat", projectPath, cfg, summary)

	done := &ChatDonePayload{
		Content:          result.Content,
		Model:            result.Usage.Model,
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
		CostUSD:          result.Usage.CostUSD,
		UsageLine:        ai.FormatLatestUsage(summary),
	}
	return done, nil
}
