package ai

import (
	"encoding/json"
	"strings"
)

// Tool names available to the project chat agent.
const (
	ToolReadFile   = "read_file"
	ToolListDir    = "list_dir"
	ToolWriteFile  = "write_file"
	ToolRunCommand = "run_command"
)

// ToolCall is a model-requested function invocation.
type ToolCall struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// ToolNeedsApproval reports whether the tool must be confirmed by the user.
func ToolNeedsApproval(name string) bool {
	switch strings.TrimSpace(name) {
	case ToolWriteFile, ToolRunCommand:
		return true
	default:
		return false
	}
}

// ChatToolDefinitions returns OpenAI-style tool schemas (also mapped for Gemini).
func ChatToolDefinitions() []map[string]any {
	return []map[string]any{
		openaiTool(ToolReadFile, "Lê o conteúdo de um arquivo relativo à raiz do projeto.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Caminho relativo à raiz do projeto (ex.: .gitignore, src/main.go).",
				},
			},
			"required": []string{"path"},
		}),
		openaiTool(ToolListDir, "Lista arquivos e pastas em um diretório relativo à raiz do projeto.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Diretório relativo (vazio ou \".\" = raiz do projeto).",
				},
			},
			"required": []string{},
		}),
		openaiTool(ToolWriteFile, "Cria ou sobrescreve um arquivo no projeto. Requer aprovação do usuário.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Caminho relativo à raiz do projeto.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Conteúdo completo do arquivo.",
				},
			},
			"required": []string{"path", "content"},
		}),
		openaiTool(ToolRunCommand, "Executa um comando de shell na raiz do projeto. Requer aprovação do usuário.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Comando shell (ex.: git status, ls -la).",
				},
			},
			"required": []string{"command"},
		}),
	}
}

func openaiTool(name, description string, parameters map[string]any) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": description,
			"parameters":  parameters,
		},
	}
}

// GeminiFunctionDeclarations converts ChatToolDefinitions to Gemini REST format.
func GeminiFunctionDeclarations() []map[string]any {
	out := make([]map[string]any, 0, 4)
	for _, t := range ChatToolDefinitions() {
		fn, _ := t["function"].(map[string]any)
		if fn == nil {
			continue
		}
		params, _ := fn["parameters"].(map[string]any)
		out = append(out, map[string]any{
			"name":        fn["name"],
			"description": fn["description"],
			"parameters":  geminiSchema(params),
		})
	}
	return out
}

// geminiSchema uppercases JSON Schema type keywords expected by the Gemini REST API.
func geminiSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	out := make(map[string]any, len(schema))
	for k, v := range schema {
		switch k {
		case "type":
			if s, ok := v.(string); ok {
				out[k] = strings.ToUpper(s)
			} else {
				out[k] = v
			}
		case "properties":
			props, ok := v.(map[string]any)
			if !ok {
				out[k] = v
				continue
			}
			mapped := make(map[string]any, len(props))
			for pk, pv := range props {
				if pm, ok := pv.(map[string]any); ok {
					mapped[pk] = geminiSchema(pm)
				} else {
					mapped[pk] = pv
				}
			}
			out[k] = mapped
		default:
			out[k] = v
		}
	}
	return out
}

// ArgString reads a string tool argument.
func ArgString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
