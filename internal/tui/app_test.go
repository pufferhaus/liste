package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pblca/liste/internal/model"
	"github.com/pblca/liste/internal/tui"
)

func TestTabCyclesForward(t *testing.T) {
	cfg := &model.Config{
		Project: "test",
		TUI: model.TUIConfig{
			DefaultView: "list",
			Views:       []string{"list", "roadmap", "blocked"},
		},
	}
	m := tui.NewAppForTest(cfg)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	app, ok := updated.(tui.AppModel)
	if !ok {
		t.Fatal("Update did not return AppModel")
	}
	if app.ActiveTab() != 1 {
		t.Errorf("ActiveTab after tab: got %d, want 1", app.ActiveTab())
	}
}

func TestTabCyclesWraps(t *testing.T) {
	cfg := &model.Config{
		Project: "test",
		TUI: model.TUIConfig{
			DefaultView: "list",
			Views:       []string{"list", "roadmap"},
		},
	}
	m := tui.NewAppForTest(cfg)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	app := updated.(tui.AppModel)
	if app.ActiveTab() != 0 {
		t.Errorf("ActiveTab after wrap: got %d, want 0", app.ActiveTab())
	}
}
