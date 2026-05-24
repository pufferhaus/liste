package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestDeleteFlowKeyboardConfirm opens detail → 'x' → 'y' to confirm delete.
// Verifies the item file is removed from the store.
func TestDeleteFlowKeyboardConfirm(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[x] Delete")

	tm.Send(keyMsg('x'))
	waitForContains(t, tm, "Delete BUG-001?", "Confirm Delete", "Cancel")

	tm.Send(keyMsg('y'))
	waitForContains(t, tm, "BUG-001 deleted")
	finalModel(t, tm)

	if _, err := s.ReadItem("BUG-001"); err == nil {
		t.Fatal("BUG-001 still exists after delete confirm")
	}
}

// TestDeleteFlowKeyboardCancelEsc opens delete modal then Esc — item must remain.
func TestDeleteFlowKeyboardCancelEsc(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[x] Delete")

	tm.Send(keyMsg('x'))
	waitForContains(t, tm, "Delete BUG-001?")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	awaitProcessed(t, tm)
	finalModel(t, tm)

	// Item must still exist.
	if _, err := s.ReadItem("BUG-001"); err != nil {
		t.Fatalf("BUG-001 should still exist after esc-cancel, got %v", err)
	}
}

// TestDeleteFlowKeyboardCancelN opens delete modal then 'n' — item must remain.
func TestDeleteFlowKeyboardCancelN(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[x] Delete")

	tm.Send(keyMsg('x'))
	waitForContains(t, tm, "Delete BUG-001?")

	tm.Send(keyMsg('n'))
	awaitProcessed(t, tm)
	finalModel(t, tm)

	if _, err := s.ReadItem("BUG-001"); err != nil {
		t.Fatalf("BUG-001 should still exist after n-cancel, got %v", err)
	}
}

// TestDeleteFlowMouseConfirm clicks the Confirm Delete button.
// Modal sits centered; with width=100, height=32 the modal is roughly 28
// columns wide and 7 tall. The exact button X is derived from confirmModalLayout
// at runtime, so we use the rendered model state — capture the layout by
// asserting via a click at the modal-content row.
func TestDeleteFlowMouseConfirm(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[x] Delete")

	tm.Send(keyMsg('x'))
	waitForContains(t, tm, "Confirm Delete")

	// Modal centered at width=100 height=32. Button row = (32-7)/2 + 4 = 16.
	// startX ≈ (100 - modalW)/2; modalW ≈ "Confirm Delete" + "[ Cancel ]" widths
	// plus border+padding (~10). Confirm button starts at startX+3 and is
	// ~lipgloss.Width("Confirm Delete")+2 = 16 wide.
	// Click safely inside the confirm button.
	tm.Send(tea.MouseMsg{
		X:      40,
		Y:      16,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})

	waitForContains(t, tm, "BUG-001 deleted")
	finalModel(t, tm)

	if _, err := s.ReadItem("BUG-001"); err == nil {
		t.Fatal("BUG-001 still exists after mouse-confirm delete")
	}
}

// TestDeleteFlowMouseCancel clicks the Cancel button.
func TestDeleteFlowMouseCancel(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[x] Delete")

	tm.Send(keyMsg('x'))
	waitForContains(t, tm, "Confirm Delete")

	// Cancel sits to the right of Confirm + gap. Click well past x=40.
	tm.Send(tea.MouseMsg{
		X:      60,
		Y:      16,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	awaitProcessed(t, tm)
	finalModel(t, tm)

	if _, err := s.ReadItem("BUG-001"); err != nil {
		t.Fatalf("BUG-001 should still exist after mouse-cancel, got %v", err)
	}
}
