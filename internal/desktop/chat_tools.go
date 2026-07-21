package desktop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/laerciocrestani/openbench/internal/ai"
)

const (
	maxReadBytes    = 200_000
	maxWriteBytes   = 512_000
	maxCommandBytes = 100_000
	commandTimeout  = 60 * time.Second
)

// ChatToolRequest is emitted on chat:tool_request for user approval.
type ChatToolRequest struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Summary        string `json:"summary"`
	Path           string `json:"path,omitempty"`
	Command        string `json:"command,omitempty"`
	ContentPreview string `json:"contentPreview,omitempty"`
}

// ExecuteChatTool runs a chat tool confined to projectPath.
func ExecuteChatTool(ctx context.Context, projectPath string, call ai.ToolCall) (string, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return "", fmt.Errorf("projeto não aberto")
	}
	switch strings.TrimSpace(call.Name) {
	case ai.ToolReadFile:
		return toolReadFile(projectPath, ai.ArgString(call.Args, "path"))
	case ai.ToolListDir:
		return toolListDir(projectPath, ai.ArgString(call.Args, "path"))
	case ai.ToolWriteFile:
		return toolWriteFile(projectPath, ai.ArgString(call.Args, "path"), ai.ArgString(call.Args, "content"))
	case ai.ToolRunCommand:
		return toolRunCommand(ctx, projectPath, ai.ArgString(call.Args, "command"))
	default:
		return "", fmt.Errorf("tool desconhecida: %s", call.Name)
	}
}

// BuildChatToolRequest builds a UI-friendly approval payload.
func BuildChatToolRequest(call ai.ToolCall) ChatToolRequest {
	req := ChatToolRequest{
		ID:   call.ID,
		Name: call.Name,
	}
	if req.ID == "" {
		req.ID = call.Name
	}
	switch call.Name {
	case ai.ToolWriteFile:
		path := ai.ArgString(call.Args, "path")
		content := ai.ArgString(call.Args, "content")
		req.Path = path
		req.ContentPreview = truncateRunes(content, 4000)
		req.Summary = fmt.Sprintf("Escrever arquivo %s (%d caracteres)", path, utf8.RuneCountInString(content))
	case ai.ToolRunCommand:
		cmd := ai.ArgString(call.Args, "command")
		req.Command = cmd
		req.Summary = fmt.Sprintf("Executar comando: %s", cmd)
	default:
		req.Summary = fmt.Sprintf("Executar %s", call.Name)
	}
	return req
}

func toolReadFile(projectPath, rel string) (string, error) {
	abs, err := resolveUnderProject(projectPath, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s é um diretório — use list_dir", rel)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	if len(data) > maxReadBytes {
		return string(data[:maxReadBytes]) + "\n… [arquivo truncado]", nil
	}
	return string(data), nil
}

func toolListDir(projectPath, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		rel = "."
	}
	abs, err := resolveUnderProject(projectPath, rel)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			b.WriteString(name)
			b.WriteString("/\n")
		} else {
			b.WriteString(name)
			b.WriteByte('\n')
		}
	}
	if b.Len() == 0 {
		return "(diretório vazio)", nil
	}
	return b.String(), nil
}

func toolWriteFile(projectPath, rel, content string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("path obrigatório")
	}
	if len(content) > maxWriteBytes {
		return "", fmt.Errorf("conteúdo muito grande (máx %d bytes)", maxWriteBytes)
	}
	abs, err := resolveUnderProject(projectPath, rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("arquivo escrito: %s (%d bytes)", rel, len(content)), nil
}

func toolRunCommand(ctx context.Context, projectPath, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("comando vazio")
	}
	cctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "sh", "-c", command)
	cmd.Dir = projectPath
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	text := string(out)
	if len(text) > maxCommandBytes {
		text = text[:maxCommandBytes] + "\n… [saída truncada]"
	}
	if err != nil {
		if text == "" {
			return "", fmt.Errorf("%v", err)
		}
		return fmt.Sprintf("exit error: %v\n%s", err, text), nil
	}
	if text == "" {
		return "(sem saída)", nil
	}
	return text, nil
}

// resolveUnderProject resolves rel against projectRoot and ensures the result
// stays inside the project tree.
func resolveUnderProject(projectRoot, rel string) (string, error) {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", err
	}
	root = filepath.Clean(root)

	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("path vazio")
	}
	// Disallow absolute paths that escape; allow abs only if under root.
	var candidate string
	if filepath.IsAbs(rel) {
		candidate = filepath.Clean(rel)
	} else {
		candidate = filepath.Clean(filepath.Join(root, rel))
	}

	sep := string(filepath.Separator)
	if candidate != root && !strings.HasPrefix(candidate, root+sep) {
		return "", fmt.Errorf("path fora do projeto: %s", rel)
	}
	return candidate, nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "\n… [preview truncado]"
}

func chatToolMutatesProject(name string) bool {
	switch strings.TrimSpace(name) {
	case ai.ToolWriteFile, ai.ToolRunCommand:
		return true
	default:
		return false
	}
}
