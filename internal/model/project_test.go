package model_test

import (
	"testing"

	"github.com/pufferhaus/liste/internal/model"
)

func TestTUIConfigResolvedDefaults(t *testing.T) {
	cfg := model.TUIConfig{}
	got := cfg.Resolved()

	if got.DefaultView != "list" {
		t.Errorf("DefaultView: got %q, want %q", got.DefaultView, "list")
	}
	if len(got.Views) != 5 {
		t.Errorf("Views len: got %d, want 5", len(got.Views))
	}
	expected := []string{"list", "roadmap", "blocked", "next", "search"}
	for i, v := range expected {
		if got.Views[i] != v {
			t.Errorf("Views[%d]: got %q, want %q", i, got.Views[i], v)
		}
	}
}

func TestTUIConfigResolvedPreservesExisting(t *testing.T) {
	cfg := model.TUIConfig{
		DefaultView: "roadmap",
		Views:       []string{"roadmap", "list"},
	}
	got := cfg.Resolved()

	if got.DefaultView != "roadmap" {
		t.Errorf("DefaultView: got %q, want %q", got.DefaultView, "roadmap")
	}
	if len(got.Views) != 2 {
		t.Errorf("Views len: got %d, want 2", len(got.Views))
	}
}
