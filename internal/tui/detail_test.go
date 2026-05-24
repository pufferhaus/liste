package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestListEnterOpensDetail simulates pressing enter on the default-selected
// item and expects the detail overlay (with its action bar) to render.
func TestListEnterOpensDetail(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)

	waitForContains(t, tm, "BUG-001") // first item alphabetically
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Detail overlay action bar is unique to the detail view.
	waitForContains(t, tm, "[e] Edit", "[d] Done", "[b] Block", "[x] Delete")
	finalModel(t, tm)
}

// TestDetailEscClosesOverlay opens detail and asserts Esc returns to list.
func TestDetailEscClosesOverlay(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)

	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[e] Edit")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	// Status bar hint only appears when no overlay is active.
	waitForContains(t, tm, "tab/shift+tab: switch")
	finalModel(t, tm)
}

// TestListMouseClickOpensDetail clicks the third row in the list.
// listItemHeight=3 + listHeaderLines=2 + tabBarHeight=3 → first item at Y=8.
func TestListMouseClickOpensDetail(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")

	tm.Send(tea.MouseMsg{
		X:      10,
		Y:      8,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})

	waitForContains(t, tm, "[e] Edit")
	finalModel(t, tm)
}

// TestListDoneKeyMarksItemDone presses 'd' on the first item and expects the
// store to reflect the new status.
func TestListDoneKeyMarksItemDone(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(keyMsg('d'))
	waitForContains(t, tm, "BUG-001 marked done")
	finalModel(t, tm)

	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Status != "done" {
		t.Fatalf("BUG-001.Status = %q, want %q", item.Status, "done")
	}
}

// TestDetailDoneAction opens detail then presses 'd' to mark done.
// The detail view closes (markDone clears overlay) and status bar updates.
func TestDetailDoneAction(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[e] Edit")

	tm.Send(keyMsg('d'))
	waitForContains(t, tm, "BUG-001 marked done")
	finalModel(t, tm)

	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Status != "done" {
		t.Fatalf("BUG-001.Status = %q, want done", item.Status)
	}
}

// TestDetailBlockSetsReason presses 'b' from the detail view of the first
// list item (BUG-001 by alpha sort), types a reason, hits enter, and asserts
// the item is persisted with the new block reason. (textinput placeholder
// doesn't render as plain text in the byte stream, so we drive the flow end-
// to-end and verify the side effect instead.)
func TestDetailBlockSetsReason(t *testing.T) {
	app, s, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[b] Block")

	tm.Send(keyMsg('b'))
	// Yield so the ItemBlockMsg cmd dispatched from 'b' is processed before
	// we start typing — otherwise the next keystroke races the still-open
	// detail overlay and gets dropped into the viewport.
	awaitProcessed(t, tm)
	tm.Type("waiting on review")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "BUG-001 blocked")
	finalModel(t, tm)

	item, err := s.ReadItem("BUG-001")
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Blocked == nil || item.Blocked.Reason != "waiting on review" {
		t.Fatalf("BUG-001 blocked = %+v, want reason %q", item.Blocked, "waiting on review")
	}
}

// TestDetailEditOpensEditOverlay presses 'e' from detail and expects the edit form.
func TestDetailEditOpensEditOverlay(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "FEAT-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[e] Edit")

	tm.Send(keyMsg('e'))
	// Edit overlay header is "Edit  <ID>".
	waitForContains(t, tm, "Edit ", "Title:", "Status:", "Priority:")
	finalModel(t, tm)
}

// TestDetailMouseClickOnEditButton clicks the [e] Edit button in the action bar.
// detail action bar lives at Y = height - 3 = 32 - 3 = 29.
// "[e] Edit" occupies X=2..9 inside the detail border.
func TestDetailMouseClickOnEditButton(t *testing.T) {
	app, _, _ := newTestApp(t)
	tm := newTeaTest(t, app)
	waitForContains(t, tm, "BUG-001")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForContains(t, tm, "[e] Edit")

	tm.Send(tea.MouseMsg{
		X:      5,
		Y:      29,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})

	waitForContains(t, tm, "Edit ", "Title:")
	finalModel(t, tm)
}
