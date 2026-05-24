package tui_test

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/pufferhaus/liste/internal/tui"
)

const (
	testTermWidth  = 100
	testTermHeight = 32
	// teatestTimeout bounds WaitFor checks. Short enough to fail fast on bugs,
	// long enough to absorb scheduler jitter on loaded machines.
	teatestTimeout = 3 * time.Second
)

// seedItem is a compact spec for a fixture item.
type seedItem struct {
	id       string
	itemType model.ItemType
	title    string
	status   string
	priority string
	body     string
	tags     []string
	blocked  string
}

// defaultSeed returns a mixed fixture covering each type, status, and a blocked item.
func defaultSeed() []seedItem {
	return []seedItem{
		{id: "FEAT-001", itemType: model.TypeFeature, title: "OAuth login", status: "active", priority: "high", body: "Implement OAuth2 with PKCE."},
		{id: "FEAT-002", itemType: model.TypeFeature, title: "User profile page", status: "planned", priority: "medium"},
		{id: "BUG-001", itemType: model.TypeBug, title: "Login timeout on Safari", status: "idea", priority: "critical"},
		{id: "BUG-002", itemType: model.TypeBug, title: "Password reset email malformed", status: "idea", priority: "medium", blocked: "Waiting on SMTP creds"},
		{id: "TASK-001", itemType: model.TypeTask, title: "Write integration tests", status: "idea", priority: "medium"},
		{id: "TASK-002", itemType: model.TypeTask, title: "Document API endpoints", status: "done", priority: "low"},
		{id: "EPIC-001", itemType: model.TypeEpic, title: "Auth platform", status: "idea", priority: "medium"},
	}
}

// newTestStore creates a tempdir-backed store seeded with the given items.
func newTestStore(t *testing.T, items []seedItem) (*store.Store, *model.Config) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".liste")
	s := store.New(root)
	if err := s.Init("testproj"); err != nil {
		t.Fatalf("store.Init: %v", err)
	}
	cfg, err := s.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for _, it := range items {
		item := &model.Item{
			ID:       it.id,
			Type:     it.itemType,
			Title:    it.title,
			Status:   it.status,
			Priority: it.priority,
			Created:  now,
			Updated:  now,
			Tags:     it.tags,
			Body:     it.body,
		}
		if it.blocked != "" {
			item.Blocked = &model.Blocked{Reason: it.blocked}
		}
		if err := s.WriteItem(item); err != nil {
			t.Fatalf("WriteItem %s: %v", it.id, err)
		}
	}
	return s, cfg
}

// newTestApp builds an AppModel from a fresh store seeded with defaultSeed().
func newTestApp(t *testing.T) (tui.AppModel, *store.Store, *model.Config) {
	t.Helper()
	s, cfg := newTestStore(t, defaultSeed())
	app, err := tui.NewTestApp(s, cfg, testTermWidth, testTermHeight)
	if err != nil {
		t.Fatalf("NewTestApp: %v", err)
	}
	return app, s, cfg
}

// newTeaTest wraps an AppModel in a teatest harness sized to the test terminal.
func newTeaTest(t *testing.T, app tui.AppModel) *teatest.TestModel {
	t.Helper()
	return teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(testTermWidth, testTermHeight),
	)
}

// waitForContains polls tm.Output() until every substring has been observed.
// teatest.WaitFor calls the condition with whatever buffer it currently has;
// in practice that buffer accumulates per-call but is *reset* between WaitFor
// invocations. So substrings need to all appear within a single WaitFor's
// observation window. To make this robust across many substrings emitted in
// separate frames, we accumulate bytes into a local buffer that survives across
// chunk reads but is checked from the start each time.
func waitForContains(t *testing.T, tm *teatest.TestModel, substrs ...string) {
	t.Helper()
	var seen bytes.Buffer
	remaining := make([][]byte, 0, len(substrs))
	for _, s := range substrs {
		remaining = append(remaining, []byte(s))
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		// teatest may give us only the latest chunk or the full accumulation —
		// merge into our own buffer to make matching deterministic.
		if len(b) > seen.Len() {
			seen.Reset()
			seen.Write(b)
		}
		next := remaining[:0]
		for _, want := range remaining {
			if !bytes.Contains(seen.Bytes(), want) {
				next = append(next, want)
			}
		}
		remaining = next
		return len(remaining) == 0
	}, teatest.WithDuration(teatestTimeout))
}

// keyMsg builds a tea.KeyMsg for a single rune.
func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// finalModel quits the program and returns the final AppModel.
//
// The app's ctrl+c handler only fires at the top-level list view, so we have
// to unwind any open overlays first. Worst case from deepest state:
//   confirmDiscard inside editOverlay inside overlay
//     → y closes discard
//     → esc opens discard again from edit
//     → y closes discard and edit
//     → esc closes detail
//     → q quits
//
// Each key is paused so it doesn't race the queued previous one.
func finalModel(t *testing.T, tm *teatest.TestModel) tui.AppModel {
	t.Helper()
	tm.Send(keyMsg('y'))
	awaitProcessed(t, tm)
	pressKey(tm, tea.KeyEsc)
	awaitProcessed(t, tm)
	tm.Send(keyMsg('y'))
	awaitProcessed(t, tm)
	pressKey(tm, tea.KeyEsc)
	awaitProcessed(t, tm)
	tm.Send(keyMsg('q'))
	m := tm.FinalModel(t, teatest.WithFinalTimeout(teatestTimeout))
	app, ok := m.(tui.AppModel)
	if !ok {
		t.Fatalf("final model is %T, want AppModel", m)
	}
	return app
}

// awaitProcessed sleeps long enough for any pending tea.Cmd dispatched by the
// previous Send to be evaluated and routed back through Update. Used between a
// state-changing key (e.g. 'b' that opens a textinput) and the next key, to
// avoid the second key racing the still-open prior view.
func awaitProcessed(t *testing.T, _ *teatest.TestModel) {
	t.Helper()
	time.Sleep(50 * time.Millisecond)
}

// typeSlow sends each char of s as a KeyRunes message with a short pause
// between each. teatest.Type bursts characters faster than bubbletea drains
// its message channel under load — long bursts get dropped silently. This
// helper preserves every keystroke.
func typeSlow(tm *teatest.TestModel, s string) {
	for _, r := range s {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		time.Sleep(2 * time.Millisecond)
	}
}

// pressKey sends the given KeyType with a short pause so it doesn't race
// other queued sends.
func pressKey(tm *teatest.TestModel, kt tea.KeyType) {
	tm.Send(tea.KeyMsg{Type: kt})
	time.Sleep(2 * time.Millisecond)
}

// pressKeyN sends the same key n times with a short pause between.
func pressKeyN(tm *teatest.TestModel, kt tea.KeyType, n int) {
	for i := 0; i < n; i++ {
		pressKey(tm, kt)
	}
}
