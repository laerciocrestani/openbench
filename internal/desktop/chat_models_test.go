package desktop

import "testing"

func TestLoadChatModelsIncludesConfigured(t *testing.T) {
	view, err := LoadChatModels()
	if err != nil {
		t.Fatal(err)
	}
	if view == nil {
		t.Fatal("nil view")
	}
	if len(view.Models) == 0 {
		t.Fatal("expected at least one model")
	}
	if view.DefaultModel == "" {
		t.Fatal("expected default model")
	}
	found := false
	for _, m := range view.Models {
		if m == view.DefaultModel {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("default %q not in models %v", view.DefaultModel, view.Models)
	}
}
