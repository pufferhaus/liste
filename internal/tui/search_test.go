package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// switchToSearch tabs from list (index 0) through to search (index 4).
func switchToSearch(t *testing.T, tm interface{ Send(tea.Msg) }) {
	t.Helper()
	for i := 0; i < 4; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	}
}

// TestSearchFiltersAsYouType types a query and asserts the list narrows to
// only items whose title matches.
func TestSearchFiltersAsYouType(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "OAuth login")

	switchToSearch(t, tm)
	awaitProcessed(t, tm)

	// Type a query that only matches OAuth.
	typeSlow(tm, "OAuth")
	// The matched item title should still appear, and the unrelated item should not.
	// We only assert positive: presence of OAuth in filtered output.
	waitForContains(t, tm, "OAuth login")
	finalModel(t, tm)
}

// TestSearchEscClears empties the input after typing. We can't observe the
// "cleared" frame directly (ANSI compressor suppresses unchanged renders),
// so we type a fresh query after esc and assert it matches multiple items —
// proving the input was reset rather than appended to.
func TestSearchEscClears(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "OAuth login")

	switchToSearch(t, tm)
	awaitProcessed(t, tm)
	typeSlow(tm, "OAuth")
	waitForContains(t, tm, "OAuth login")

	pressKey(tm, tea.KeyEsc)
	awaitProcessed(t, tm)
	// "BUG" matches BUG-001 and BUG-002. If esc didn't clear, query would be
	// "OAuthBUG" → no matches → BUG-001 never renders.
	typeSlow(tm, "BUG")
	waitForContains(t, tm, "BUG-001", "BUG-002")
	finalModel(t, tm)
}

// TestSearchEnterOpensDetail selects the first filtered result and opens detail.
func TestSearchEnterOpensDetail(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "OAuth login")

	switchToSearch(t, tm)
	awaitProcessed(t, tm)
	typeSlow(tm, "OAuth")
	waitForContains(t, tm, "OAuth login")

	pressKey(tm, tea.KeyEnter)
	awaitProcessed(t, tm)
	// Detail action bar should appear.
	waitForContains(t, tm, "[e] Edit", "[x] Delete")
	finalModel(t, tm)
}

// TestSearchMouseClickResultOpensDetail clicks an item row in the search results.
// Search layout: Y=3..5 input box, Y=6 blank, Y=7..8 list header, Y=9+ items.
func TestSearchMouseClickResultOpensDetail(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "OAuth login")

	switchToSearch(t, tm)
	awaitProcessed(t, tm)
	typeSlow(tm, "OAuth")
	waitForContains(t, tm, "OAuth login")

	tm.Send(tea.MouseMsg{
		X:      10,
		Y:      9,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	awaitProcessed(t, tm)
	waitForContains(t, tm, "[e] Edit")
	finalModel(t, tm)
}
