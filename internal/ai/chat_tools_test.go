package ai

import "testing"

func TestToolNeedsApproval(t *testing.T) {
	if !ToolNeedsApproval(ToolWriteFile) {
		t.Fatal("write_file should need approval")
	}
	if !ToolNeedsApproval(ToolRunCommand) {
		t.Fatal("run_command should need approval")
	}
	if ToolNeedsApproval(ToolReadFile) {
		t.Fatal("read_file should be auto")
	}
	if ToolNeedsApproval(ToolListDir) {
		t.Fatal("list_dir should be auto")
	}
}

func TestGeminiFunctionDeclarations(t *testing.T) {
	decls := GeminiFunctionDeclarations()
	if len(decls) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(decls))
	}
	params, ok := decls[0]["parameters"].(map[string]any)
	if !ok {
		t.Fatal("missing parameters")
	}
	typ, _ := params["type"].(string)
	if typ != "OBJECT" {
		t.Fatalf("gemini type should be OBJECT, got %q", typ)
	}
}
