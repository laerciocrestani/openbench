package desktop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laerciocrestani/openbench/internal/ai"
)

func TestExecuteChatToolWriteRead(t *testing.T) {
	dir := t.TempDir()
	_, err := ExecuteChatTool(context.Background(), dir, ai.ToolCall{
		Name: ai.ToolWriteFile,
		Args: map[string]any{
			"path":    ".gitignore",
			"content": "node_modules/\n",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "node_modules/\n" {
		t.Fatalf("unexpected content: %q", data)
	}

	out, err := ExecuteChatTool(context.Background(), dir, ai.ToolCall{
		Name: ai.ToolReadFile,
		Args: map[string]any{"path": ".gitignore"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "node_modules/\n" {
		t.Fatalf("read mismatch: %q", out)
	}
}

func TestExecuteChatToolRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	_, err := ExecuteChatTool(context.Background(), dir, ai.ToolCall{
		Name: ai.ToolWriteFile,
		Args: map[string]any{
			"path":    "../evil.txt",
			"content": "x",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "fora do projeto") {
		t.Fatalf("expected path escape error, got %v", err)
	}
}
