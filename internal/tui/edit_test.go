package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestEditSavePersistsTitle opens the edit overlay, replaces the title, ctrl+s
// to save, and asserts the new title hit disk.
func TestEditSavePersistsTitle(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	awaitProcessed(t, tm)
	tm.Send(keyMsg('e'))
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Title:")

	pressKeyN(tm, tea.KeyBackspace, 30)
	typeSlow(tm, "Login timeout - investigated")
	pressKey(tm, tea.KeyCtrlS)
	// After save, edit overlay closes and detail refreshes with the new title.
	waitForContains(t, tm, "Login timeout - investigated")
	finalModel(t, tm)

	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Title != "Login timeout - investigated" {
		t.Fatalf("Title = %q, want %q", item.Title, "Login timeout - investigated")
	}
}

// TestEditTabCyclesFields confirms tab moves focus to the next field.
// Drives: tab from title → status, edits status, saves, asserts on store.
func TestEditTabCyclesFields(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	awaitProcessed(t, tm)
	tm.Send(keyMsg('e'))
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Title:")

	pressKey(tm, tea.KeyTab) // → status field
	pressKeyN(tm, tea.KeyBackspace, 15)
	typeSlow(tm, "active")
	pressKey(tm, tea.KeyCtrlS)
	// After save the detail overlay re-renders with the new status (● active).
	waitForContains(t, tm, "active")
	finalModel(t, tm)

	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Status != "active" {
		t.Fatalf("Status = %q, want active", item.Status)
	}
	if item.Title != "Login timeout on Safari" {
		t.Fatalf("Title accidentally changed to %q", item.Title)
	}
}

// TestEditEscOpensDiscardModal triggers the discard-changes confirmation modal.
func TestEditEscOpensDiscardModal(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	awaitProcessed(t, tm)
	tm.Send(keyMsg('e'))
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Title:")

	pressKey(tm, tea.KeyEsc)
	waitForContains(t, tm, "Discard changes?")
	finalModel(t, tm)
}

// TestEditDiscardConfirmReturnsToDetail confirms the discard modal returns to
// the detail overlay (still in memory) rather than dropping back to the list.
func TestEditDiscardConfirmReturnsToDetail(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	awaitProcessed(t, tm)
	tm.Send(keyMsg('e'))
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Title:")

	typeSlow(tm, "dirty")
	pressKey(tm, tea.KeyEsc)
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Discard changes?")

	tm.Send(keyMsg('y'))
	// Back to detail view — action bar reappears.
	waitForContains(t, tm, "[e] Edit")
	finalModel(t, tm)

	// The "dirty" text must NOT have been persisted.
	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Title != "Login timeout on Safari" {
		t.Fatalf("Title = %q, want original (discard failed)", item.Title)
	}
}

// TestEditDiscardCancelStaysInEdit asserts that hitting 'n' on the discard
// modal returns to the edit overlay with the in-progress text intact.
// (We type a sentinel after 'n' to force a fresh render — the ANSI compressor
// suppresses re-emission of identical frames.)
func TestEditDiscardCancelStaysInEdit(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	awaitProcessed(t, tm)
	tm.Send(keyMsg('e'))
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Title:")

	pressKey(tm, tea.KeyEsc)
	awaitProcessed(t, tm)
	waitForContains(t, tm, "Discard changes?")

	tm.Send(keyMsg('n'))
	awaitProcessed(t, tm)
	// Type a sentinel — only the title textinput would receive it. If we were
	// pushed back to the list/detail this sentinel goes nowhere.
	typeSlow(tm, "ZZZ")
	waitForContains(t, tm, "ZZZ")
	// Save and verify the sentinel made it into the persisted title — proves
	// we were still in the edit form's title field.
	pressKey(tm, tea.KeyCtrlS)
	waitForContains(t, tm, "ZZZ")
	finalModel(t, tm)

	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Title == "Login timeout on Safari" {
		t.Fatalf("Title unchanged — 'n' on discard did not return to edit")
	}
}
