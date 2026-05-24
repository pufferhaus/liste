package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pufferhaus/liste/internal/tui"
)

// TestTabBarRenders confirms the tab bar shows every configured tab name
// uppercased after the initial window-size frame.
func TestTabBarRenders(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)

	waitForContains(t, tm, "LIST", "ROADMAP", "BLOCKED", "NEXT", "SEARCH")
	finalModel(t, tm)
}

// TestTabCyclesAllViewsForward walks through every tab via Tab and asserts
// the visible view content changes per tab.
func TestTabCyclesAllViewsForward(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)

	// List is the default view — wait for any item title to render.
	waitForContains(t, tm, "OAuth login")

	// Tab to roadmap — bubbles list title is "PHASE 0" (no phases set in seed).
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitForContains(t, tm, "PHASE")

	// Tab to blocked — bubbles list title + the blocked item title.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitForContains(t, tm, "Blocked Items", "Password reset email malformed")

	// Tab to next — bubbles list title is "Next Up".
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitForContains(t, tm, "Next Up")

	// Tab to search — assert via final model state (textinput placeholder is
	// rendered in a faint style that doesn't appear as literal text in the
	// captured byte stream).
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	final := finalModel(t, tm)
	if final.ActiveTab() != 4 {
		t.Fatalf("after 4 tabs, ActiveTab=%d, want 4 (search)", final.ActiveTab())
	}
}

// TestTabCyclesBackward verifies Shift+Tab moves backward and wraps.
func TestTabCyclesBackward(t *testing.T) {
	app, _, _ := newTestApp(t)

	// shift+tab from list (index 0) should wrap to the last tab (search, index 4).
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a := updated.(tui.AppModel)
	if a.ActiveTab() != 4 {
		t.Fatalf("after shift+tab from 0, ActiveTab=%d, want 4", a.ActiveTab())
	}

	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = updated.(tui.AppModel)
	if a.ActiveTab() != 3 {
		t.Fatalf("after second shift+tab, ActiveTab=%d, want 3", a.ActiveTab())
	}
}

// TestTabClickSwitchesView verifies a left-mouse click on the third tab
// (BLOCKED at index 2) switches to that view.
func TestTabClickSwitchesView(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "OAuth login")

	// Each tab is ~11 columns wide ("│ LIST │"). Click well inside BLOCKED tab.
	// "LIST" + "ROADMAP" widths put BLOCKED around x=22-30. Y=1 is inside the tab bar.
	tm.Send(tea.MouseMsg{
		X:      24,
		Y:      1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})

	waitForContains(t, tm, "Blocked Items", "Password reset email malformed")
	final := finalModel(t, tm)
	if final.ActiveTab() != 2 {
		t.Fatalf("after click at x=24, ActiveTab=%d, want 2 (blocked)", final.ActiveTab())
	}
}
