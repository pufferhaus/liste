# Charmbracelet Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add lipgloss styled output, glamour markdown rendering, huh interactive forms, and a bubbletea TUI (`liste -i`) to the liste CLI while leaving all JSON/quiet/agent paths untouched.

**Architecture:** Incremental in-place edits — lipgloss and glamour go into `internal/output/output.go`, huh into `cmd/add.go` and `cmd/init.go` (TTY-gated), and bubbletea into a new `internal/tui/` package wired via a `-i` persistent flag on the root cobra command.

**Tech Stack:** Go 1.26, charmbracelet/lipgloss, charmbracelet/glamour, charmbracelet/huh, charmbracelet/bubbletea, charmbracelet/bubbles, mattn/go-isatty

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `go.mod` / `go.sum` | Modify | Add 6 new deps |
| `internal/model/project.go` | Modify | Add `TUIConfig` struct + `Resolved()` |
| `internal/output/output.go` | Modify | Lipgloss styles, exported render helpers, glamour body render |
| `internal/output/output_test.go` | Create | Style helper tests (ANSI-stripped) |
| `cmd/root.go` | Modify | Add `-i` flag, `PersistentPreRunE` that launches TUI |
| `cmd/add.go` | Modify | TTY-gated huh form when 0 args |
| `cmd/init.go` | Modify | TTY-gated huh form when 0 args |
| `cmd/roadmap.go` | Modify | Use exported output render helpers for styled phase headers/rows |
| `internal/tui/app.go` | Create | Bubbletea root model, tab bar, window resize, item mutations |
| `internal/tui/detail.go` | Create | Viewport overlay for item detail + glamour body |
| `internal/tui/views/list.go` | Create | Bubbles list — all items |
| `internal/tui/views/blocked.go` | Create | Bubbles list — blocked items |
| `internal/tui/views/next.go` | Create | Bubbles list — next queue |
| `internal/tui/views/search.go` | Create | Textinput + filtered list |
| `internal/tui/views/roadmap.go` | Create | Phase-grouped viewport |
| `internal/tui/views/common.go` | Create | Shared item rendering helpers for TUI views |

---

## Task 1: Add charmbracelet dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add all deps**

Run from the repo root:
```bash
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/glamour@latest
go get github.com/charmbracelet/huh@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/mattn/go-isatty@latest
go mod tidy
```

- [ ] **Step 2: Verify build still passes**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add charmbracelet deps (lipgloss, glamour, huh, bubbletea, bubbles)"
```

---

## Task 2: Add TUIConfig to model

**Files:**
- Modify: `internal/model/project.go`

- [ ] **Step 1: Write the failing test**

Create `internal/model/project_test.go`:

```go
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
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/model/...
```
Expected: FAIL — `TUIConfig` undefined.

- [ ] **Step 3: Add TUIConfig to project.go**

In `internal/model/project.go`, add after the `Defaults` struct:

```go
// TUIConfig holds configuration for the interactive TUI (liste -i).
type TUIConfig struct {
	DefaultView string   `yaml:"default_view"` // list | roadmap | blocked | next | search
	Views       []string `yaml:"views"`         // ordered list of enabled views
}

// Resolved returns TUIConfig with defaults applied when fields are empty.
func (c *TUIConfig) Resolved() TUIConfig {
	r := *c
	if r.DefaultView == "" {
		r.DefaultView = "list"
	}
	if len(r.Views) == 0 {
		r.Views = []string{"list", "roadmap", "blocked", "next", "search"}
	}
	return r
}
```

Add `TUI TUIConfig` to the `Config` struct (after the existing `Defaults` field):

```go
type Config struct {
	Project    string   `yaml:"project"`
	Statuses   []string `yaml:"statuses"`
	Blocked    bool     `yaml:"blocked"`
	Types      []string `yaml:"types"`
	Priorities []string `yaml:"priorities"`
	Defaults   Defaults `yaml:"defaults"`
	TUI        TUIConfig `yaml:"tui,omitempty"`
}
```

- [ ] **Step 4: Run tests to confirm pass**

```bash
go test ./internal/model/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/project.go internal/model/project_test.go
git commit -m "feat: add TUIConfig to model with configurable views and default view"
```

---

## Task 3: Add lipgloss style helpers to output package

**Files:**
- Modify: `internal/output/output.go`
- Create: `internal/output/output_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/output/output_test.go`:

```go
package output_test

import (
	"regexp"
	"testing"

	"github.com/pufferhaus/liste/internal/output"
)

// stripANSI removes ANSI escape codes so we can test text content.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func TestRenderStatus(t *testing.T) {
	tests := []struct {
		status  string
		blocked bool
		want    string
	}{
		{"active", false, "● active"},
		{"planned", false, "○ planned"},
		{"done", false, "✓ done"},
		{"cancelled", false, "✗ cancelled"},
		{"active", true, "⊘ blocked"},
	}
	for _, tt := range tests {
		got := stripANSI(output.RenderStatus(tt.status, tt.blocked))
		if got != tt.want {
			t.Errorf("RenderStatus(%q, %v) = %q, want %q", tt.status, tt.blocked, got, tt.want)
		}
	}
}

func TestRenderType(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{"feature", "■ feature"},
		{"bug", "■ bug"},
		{"task", "■ task"},
		{"idea", "■ idea"},
		{"epic", "■ epic"},
	}
	for _, tt := range tests {
		got := stripANSI(output.RenderType(tt.typ))
		if got != tt.want {
			t.Errorf("RenderType(%q) = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

func TestRenderPriority(t *testing.T) {
	tests := []struct {
		priority string
		want     string
	}{
		{"critical", "▲ critical"},
		{"high", "▲ high"},
		{"medium", "▸ medium"},
		{"low", "▽ low"},
	}
	for _, tt := range tests {
		got := stripANSI(output.RenderPriority(tt.priority))
		if got != tt.want {
			t.Errorf("RenderPriority(%q) = %q, want %q", tt.priority, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/output/...
```
Expected: FAIL — `output.RenderStatus` undefined.

- [ ] **Step 3: Add lipgloss imports and style vars to output.go**

Add `"github.com/charmbracelet/lipgloss"` to the imports block in `internal/output/output.go`.

Add the following package-level vars and exported functions **before** the `Format` type declaration:

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pufferhaus/liste/internal/model"
)

var (
	styleStatusActive    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00b894"))
	styleStatusBlocked   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e17055"))
	styleStatusPlanned   = lipgloss.NewStyle().Foreground(lipgloss.Color("#74b9ff"))
	styleStatusDone      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7a89")).Faint(true)
	styleStatusCancelled = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7a89")).Faint(true)

	styleTypeFeature = lipgloss.NewStyle().Foreground(lipgloss.Color("#a29bfe"))
	styleTypeBug     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7675"))
	styleTypeTask    = lipgloss.NewStyle().Foreground(lipgloss.Color("#81ecec"))
	styleTypeIdea    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffeaa7"))
	styleTypeEpic    = lipgloss.NewStyle().Bold(true)

	stylePriCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("#d63031")).Bold(true)
	stylePriHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("#fdcb6e"))
	stylePriMedium   = lipgloss.NewStyle()
	stylePriLow      = lipgloss.NewStyle().Faint(true)

	styleHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a8b2d8"))
	styleFaint   = lipgloss.NewStyle().Faint(true)
	stylePhase   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4"))
	styleDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("#313244"))
)

// RenderStatus returns a styled status string (e.g. "● active", "⊘ blocked").
// Safe to call even without a terminal — lipgloss respects NO_COLOR.
func RenderStatus(status string, blocked bool) string {
	if blocked {
		return styleStatusBlocked.Render("⊘ blocked")
	}
	switch status {
	case "active":
		return styleStatusActive.Render("● active")
	case "planned":
		return styleStatusPlanned.Render("○ planned")
	case "done":
		return styleStatusDone.Render("✓ done")
	case "cancelled":
		return styleStatusCancelled.Render("✗ cancelled")
	default:
		return status
	}
}

// RenderType returns a styled type string (e.g. "■ feature").
func RenderType(t string) string {
	switch t {
	case "feature":
		return styleTypeFeature.Render("■ feature")
	case "bug":
		return styleTypeBug.Render("■ bug")
	case "task":
		return styleTypeTask.Render("■ task")
	case "idea":
		return styleTypeIdea.Render("■ idea")
	case "epic":
		return styleTypeEpic.Render("■ epic")
	default:
		return "■ " + t
	}
}

// RenderPriority returns a styled priority string (e.g. "▲ high").
func RenderPriority(p string) string {
	switch p {
	case "critical":
		return stylePriCritical.Render("▲ critical")
	case "high":
		return stylePriHigh.Render("▲ high")
	case "medium":
		return stylePriMedium.Render("▸ medium")
	case "low":
		return stylePriLow.Render("▽ low")
	default:
		return p
	}
}

// RenderPhaseHeader returns a styled phase header line for roadmap output.
func RenderPhaseHeader(phase int, status string, done, total int) string {
	label := fmt.Sprintf("PHASE %d  %s  %d/%d", phase, status, done, total)
	divider := styleDivider.Render(strings.Repeat("─", 60))
	return stylePhase.Render(label) + "\n" + divider
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/output/...
```
Expected: PASS.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: add lipgloss style helpers to output package"
```

---

## Task 4: Apply lipgloss styling to ItemList

**Files:**
- Modify: `internal/output/output.go`
- Modify: `internal/output/output_test.go`

- [ ] **Step 1: Add failing test**

Add to `internal/output/output_test.go`:

```go
import (
	"bytes"
	"strings"
	"time"

	"github.com/pufferhaus/liste/internal/model"
)

func makeTestItem(id, typ, status, priority string) *model.Item {
	return &model.Item{
		ID:       id,
		Type:     model.ItemType(typ),
		Title:    "Test title for " + id,
		Status:   status,
		Priority: priority,
		Created:  time.Now(),
		Updated:  time.Now(),
	}
}

func TestItemListTableContainsExpectedFields(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)

	items := []*model.Item{
		makeTestItem("FEAT-001", "feature", "active", "high"),
		makeTestItem("BUG-002", "bug", "planned", "critical"),
	}
	f.ItemList(items)

	got := stripANSI(buf.String())

	checks := []string{"FEAT-001", "■ feature", "● active", "▲ high", "Test title for FEAT-001",
		"BUG-002", "■ bug", "○ planned", "▲ critical"}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("ItemList output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestItemListBlockedShowsBlockedStatus(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)

	item := makeTestItem("BUG-001", "bug", "active", "high")
	item.Blocked = &model.Blocked{Reason: "waiting"}
	f.ItemList([]*model.Item{item})

	got := stripANSI(buf.String())
	if !strings.Contains(got, "⊘ blocked") {
		t.Errorf("expected blocked item to show ⊘ blocked, got:\n%s", got)
	}
}

func TestItemListDoneRowsDimmed(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)
	item := makeTestItem("TASK-003", "task", "done", "low")
	f.ItemList([]*model.Item{item})
	// Just verify it renders without error and contains the ID
	got := buf.String()
	if !strings.Contains(stripANSI(got), "TASK-003") {
		t.Errorf("done item not rendered, got:\n%s", got)
	}
}
```

- [ ] **Step 2: Run to confirm tests compile and fail**

```bash
go test ./internal/output/... -run TestItemList
```
Expected: FAIL — output missing styled symbols (old plain fmt.Fprintf format).

- [ ] **Step 3: Replace ItemList table case in output.go**

Replace the `default:` case inside `ItemList` (the table rendering block) with:

```go
default:
	if len(items) == 0 {
		fmt.Fprintln(f.Writer, "No items found.")
		return
	}
	// Column widths sized for visible content
	idW, typeW, statusW, priW := 10, 12, 14, 12

	// Header
	fmt.Fprintf(f.Writer, "%s %s %s %s %s\n",
		styleHeader.Width(idW).Render("ID"),
		styleHeader.Width(typeW).Render("TYPE"),
		styleHeader.Width(statusW).Render("STATUS"),
		styleHeader.Width(priW).Render("PRIORITY"),
		styleHeader.Render("TITLE"),
	)
	fmt.Fprintln(f.Writer, styleDivider.Render(strings.Repeat("─", 70)))

	for _, item := range items {
		isDone := item.Status == "done" || item.Status == "cancelled"
		idCell     := lipgloss.NewStyle().Width(idW).Render(item.ID)
		typeCell   := lipgloss.NewStyle().Width(typeW).Render(RenderType(string(item.Type)))
		statusCell := lipgloss.NewStyle().Width(statusW).Render(RenderStatus(item.Status, item.Blocked != nil))
		priCell    := lipgloss.NewStyle().Width(priW).Render(RenderPriority(item.Priority))
		row := fmt.Sprintf("%s %s %s %s %s", idCell, typeCell, statusCell, priCell, item.Title)
		if isDone {
			row = styleFaint.Render(row)
		}
		fmt.Fprintln(f.Writer, row)
	}
	fmt.Fprintf(f.Writer, "\n%d item(s)\n", len(items))
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/output/... -run TestItemList
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: apply lipgloss styling to ItemList table output"
```

---

## Task 5: Apply lipgloss to ItemDetail + glamour markdown body

**Files:**
- Modify: `internal/output/output.go`
- Modify: `internal/output/output_test.go`

- [ ] **Step 1: Add failing test**

Add to `internal/output/output_test.go`:

```go
func TestItemDetailTableContainsExpectedFields(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)

	item := makeTestItem("FEAT-001", "feature", "active", "high")
	item.Tags = []string{"backend", "auth"}
	item.Body = "## Description\n\nDoes the thing."

	f.ItemDetail(item, nil)

	got := stripANSI(buf.String())
	for _, want := range []string{"FEAT-001", "■ feature", "● active", "▲ high", "backend", "auth", "Description", "Does the thing"} {
		if !strings.Contains(got, want) {
			t.Errorf("ItemDetail missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestItemDetailBlockedShown(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)
	item := makeTestItem("BUG-001", "bug", "active", "critical")
	item.Blocked = &model.Blocked{Reason: "OAuth approval pending"}
	f.ItemDetail(item, nil)

	got := stripANSI(buf.String())
	if !strings.Contains(got, "OAuth approval pending") {
		t.Errorf("block reason not shown, got:\n%s", got)
	}
}
```

- [ ] **Step 2: Run to confirm tests currently pass or fail appropriately**

```bash
go test ./internal/output/... -run TestItemDetail
```
The tests may pass with the current plain text output (body content would be present). That's OK — the next step adds glamour, which changes the body rendering. Re-run after glamour to confirm the body text still appears.

- [ ] **Step 3: Add glamour import and update ItemDetail**

Add `"github.com/charmbracelet/glamour"` to the import block.

Replace the `default:` case inside `ItemDetail` with:

```go
default:
	// Header line
	fmt.Fprintf(f.Writer, "%s  %s\n",
		styleHeader.Render(item.ID),
		item.Title,
	)
	// Metadata row
	fmt.Fprintf(f.Writer, "%s  %s  %s\n",
		RenderType(string(item.Type)),
		RenderStatus(item.Status, item.Blocked != nil),
		RenderPriority(item.Priority),
	)
	fmt.Fprintf(f.Writer, "%s  Created: %s  Updated: %s\n",
		styleFaint.Render("·"),
		styleFaint.Render(item.Created.Format("2006-01-02")),
		styleFaint.Render(item.Updated.Format("2006-01-02")),
	)

	if len(item.Tags) > 0 {
		fmt.Fprintf(f.Writer, "Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if item.Blocked != nil {
		reason := item.Blocked.Reason
		if reason == "" {
			reason = "(no reason)"
		}
		fmt.Fprintf(f.Writer, "%s %s\n", styleStatusBlocked.Render("⊘ BLOCKED:"), reason)
	}
	if len(item.Links) > 0 {
		fmt.Fprintln(f.Writer, "\n"+styleHeader.Render("Links:"))
		for _, l := range item.Links {
			proj := ""
			if l.Project != "" {
				proj = "  [" + l.Project + "]"
			}
			fmt.Fprintf(f.Writer, "  %s %s%s\n", styleFaint.Render(string(l.Type)), l.Target, proj)
		}
	}
	if len(inverseLinks) > 0 {
		fmt.Fprintln(f.Writer, "\n"+styleHeader.Render("Referenced by:"))
		for _, l := range inverseLinks {
			fmt.Fprintf(f.Writer, "  %s %s\n", styleFaint.Render(l.Type), l.SourceID)
		}
	}
	if item.Body != "" {
		fmt.Fprintln(f.Writer)
		rendered, err := glamour.Render(item.Body, "auto")
		if err != nil {
			// Fall back to raw body on any glamour error
			fmt.Fprintf(f.Writer, "%s\n", item.Body)
		} else {
			fmt.Fprint(f.Writer, rendered)
		}
	}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/output/... -run TestItemDetail
```
Expected: PASS (glamour renders the markdown but the text content is still present in the output).

- [ ] **Step 5: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: apply lipgloss to ItemDetail and add glamour markdown rendering"
```

---

## Task 6: Apply lipgloss to StatusSummary and ItemCreated

**Files:**
- Modify: `internal/output/output.go`
- Modify: `internal/output/output_test.go`

- [ ] **Step 1: Add failing tests**

Add to `internal/output/output_test.go`:

```go
func TestStatusSummaryContainsStyledStatus(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)
	items := []*model.Item{
		makeTestItem("FEAT-001", "feature", "active", "high"),
		makeTestItem("BUG-002", "bug", "done", "low"),
	}
	f.StatusSummary(items, "my-project")
	got := stripANSI(buf.String())
	for _, want := range []string{"● active", "✓ done", "FEAT-001", "BUG-002"} {
		if !strings.Contains(got, want) {
			t.Errorf("StatusSummary missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestItemCreatedContainsID(t *testing.T) {
	var buf bytes.Buffer
	f := output.New(&buf, output.FormatTable)
	item := makeTestItem("FEAT-001", "feature", "idea", "medium")
	f.ItemCreated(item)
	got := stripANSI(buf.String())
	if !strings.Contains(got, "FEAT-001") {
		t.Errorf("ItemCreated missing ID, got:\n%s", got)
	}
}
```

- [ ] **Step 2: Run to confirm they compile**

```bash
go test ./internal/output/... -run "TestStatusSummary|TestItemCreated"
```
These will likely pass already (plain text contains the IDs). The goal is to add lipgloss styling to these methods and verify they still pass.

- [ ] **Step 3: Update StatusSummary table case**

Replace the `default:` case inside `StatusSummary` with:

```go
default:
	fmt.Fprintf(f.Writer, "%s  (%d items)\n\n",
		styleHeader.Render("Project: "+projectName), len(items))

	statusOrder := []string{"active", "planned", "blocked", "idea", "done", "cancelled"}
	for _, status := range statusOrder {
		group, ok := groups[status]
		if !ok || len(group) == 0 {
			continue
		}
		label := RenderStatus(status, status == "blocked")
		fmt.Fprintf(f.Writer, "%s (%d)\n", label, len(group))
		for _, item := range group {
			fmt.Fprintf(f.Writer, "  %-10s  %s  %s\n",
				item.ID,
				RenderPriority(item.Priority),
				item.Title,
			)
		}
		fmt.Fprintln(f.Writer)
	}
```

- [ ] **Step 4: Update ItemCreated table case**

Replace the `default:` case inside `ItemCreated` with:

```go
default:
	fmt.Fprintf(f.Writer, "Created %s: %s\n",
		styleHeader.Render(item.ID),
		item.Title,
	)
	fmt.Fprintf(f.Writer, "  %s  %s  %s\n",
		RenderType(string(item.Type)),
		RenderStatus(item.Status, false),
		RenderPriority(item.Priority),
	)
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/output/...
```
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: apply lipgloss styling to StatusSummary and ItemCreated"
```

---

## Task 7: Style roadmap.go output using render helpers

**Files:**
- Modify: `cmd/roadmap.go`

- [ ] **Step 1: Update renderRoadmapTable in cmd/roadmap.go**

Add `"github.com/pufferhaus/liste/internal/output"` to the import block in `cmd/roadmap.go`.

Replace `renderRoadmapTable` with:

```go
func renderRoadmapTable(phases []phaseGroup, unphased []projectItems) {
	for _, pg := range phases {
		status := detectPhaseStatus(pg)
		done, total := phaseGroupProgress(pg)

		if status == phaseComplete {
			fmt.Fprintln(os.Stdout, output.RenderPhaseHeader(pg.Phase, "complete", done, total))
			fmt.Fprintln(os.Stdout)
			continue
		}

		fmt.Fprintln(os.Stdout, output.RenderPhaseHeader(pg.Phase, string(status), done, total))
		for _, proj := range pg.Projects {
			if hasMultipleProjects(phases, unphased) {
				fmt.Fprintf(os.Stdout, "  %s\n", proj.Name)
			}
			for _, item := range proj.Items {
				indent := "  "
				if hasMultipleProjects(phases, unphased) {
					indent = "    "
				}
				fmt.Fprintf(os.Stdout, "%s%s  %-10s  %s\n",
					indent,
					output.RenderStatus(item.Status, item.Blocked != nil),
					item.ID,
					item.Title,
				)
			}
		}
		fmt.Fprintln(os.Stdout)
	}

	if len(unphased) > 0 {
		totalUnphased := 0
		for _, proj := range unphased {
			totalUnphased += len(proj.Items)
		}
		fmt.Fprintf(os.Stdout, "%s\n", output.RenderPhaseHeader(0, "unphased", 0, totalUnphased))
		for _, proj := range unphased {
			if hasMultipleProjects(phases, unphased) {
				fmt.Fprintf(os.Stdout, "  %s\n", proj.Name)
			}
			for _, item := range proj.Items {
				indent := "  "
				if hasMultipleProjects(phases, unphased) {
					indent = "    "
				}
				fmt.Fprintf(os.Stdout, "%s%s  %-10s  %s\n",
					indent,
					output.RenderStatus(item.Status, item.Blocked != nil),
					item.ID,
					item.Title,
				)
			}
		}
		fmt.Fprintln(os.Stdout)
	}
}
```

- [ ] **Step 2: Update renderPhaseDetail to use render helpers**

Replace the table output section of `renderPhaseDetail` (the non-JSON block at the bottom):

```go
	// Table output
	fmt.Fprintln(os.Stdout, output.RenderPhaseHeader(phaseNum, string(status), done, total))
	fmt.Fprintln(os.Stdout)
	for _, proj := range target.Projects {
		fmt.Fprintf(os.Stdout, "  %s\n", proj.Name)
		for _, item := range proj.Items {
			fmt.Fprintf(os.Stdout, "    %s  %-10s  %s  %s\n",
				output.RenderStatus(item.Status, item.Blocked != nil),
				item.ID,
				output.RenderPriority(item.Priority),
				item.Title,
			)
		}
		fmt.Fprintln(os.Stdout)
	}
```

- [ ] **Step 3: Build to verify no import errors**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/roadmap.go
git commit -m "feat: apply lipgloss styling to roadmap command output"
```

---

## Task 8: huh interactive form for liste add

**Files:**
- Modify: `cmd/add.go`

- [ ] **Step 1: Update add.go**

Replace the entire contents of `cmd/add.go` with:

```go
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/spf13/cobra"
)

var (
	addPriority string
	addTags     []string
	addStatus   string
	addPhase    int
)

var addCmd = &cobra.Command{
	Use:   "add <type> <title>",
	Short: "Create a new item",
	Long:  "Create a new item of the given type (feature, bug, idea, task, epic). Run without arguments in a terminal to use an interactive form.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil // TTY-gated form handled in RunE
		}
		if len(args) < 2 {
			return fmt.Errorf("requires at least 2 arg(s) (<type> <title>), received %d", len(args))
		}
		return nil
	},
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addPriority, "priority", "", "Priority (critical, high, medium, low)")
	addCmd.Flags().StringSliceVar(&addTags, "tag", nil, "Tags (can be specified multiple times)")
	addCmd.Flags().StringVar(&addStatus, "status", "", "Initial status (overrides default)")
	addCmd.Flags().IntVar(&addPhase, "phase", 0, "Phase number (0 = unphased)")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	s, err := getStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("'liste add' requires <type> and <title> arguments when not running in a terminal")
		}
		return runAddInteractive(s)
	}

	typeStr := args[0]
	title := strings.Join(args[1:], " ")

	itemType, ok := model.ParseItemType(typeStr)
	if !ok {
		return fmt.Errorf("invalid type %q (valid: feature, bug, idea, task, epic)", typeStr)
	}

	cfg, err := s.ReadConfig()
	if err != nil {
		return err
	}

	item, err := s.CreateItem(itemType, title, cfg)
	if err != nil {
		return err
	}

	changed := false
	if addPriority != "" {
		if !cfg.IsValidPriority(addPriority) {
			return fmt.Errorf("invalid priority %q (valid: %s)", addPriority, strings.Join(cfg.Priorities, ", "))
		}
		item.Priority = addPriority
		changed = true
	}
	if addStatus != "" {
		if !cfg.IsValidStatus(addStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", addStatus, strings.Join(cfg.Statuses, ", "))
		}
		item.Status = addStatus
		changed = true
	}
	if len(addTags) > 0 {
		item.Tags = addTags
		changed = true
	}
	if addPhase > 0 {
		p := addPhase
		item.Phase = &p
		changed = true
	}
	if changed {
		item.Updated = time.Now()
		if err := s.WriteItem(item); err != nil {
			return err
		}
	}

	f := getFormatter()
	f.ItemCreated(item)
	return nil
}

func runAddInteractive(s *store.Store) error {
	cfg, err := s.ReadConfig()
	if err != nil {
		return err
	}

	var (
		itemType string = "feature"
		title    string
		priority string = cfg.Defaults.Priority
		phaseStr string
		tagsStr  string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Type").
				Options(
					huh.NewOption("Feature", "feature"),
					huh.NewOption("Bug", "bug"),
					huh.NewOption("Task", "task"),
					huh.NewOption("Idea", "idea"),
					huh.NewOption("Epic", "epic"),
				).
				Value(&itemType),
			huh.NewInput().
				Title("Title").
				Value(&title).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("title is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Priority").
				Options(
					huh.NewOption("Critical", "critical"),
					huh.NewOption("High", "high"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("Low", "low"),
				).
				Value(&priority),
			huh.NewInput().
				Title("Phase (optional, positive integer)").
				Value(&phaseStr),
			huh.NewInput().
				Title("Tags (optional, comma-separated)").
				Value(&tagsStr),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	t, _ := model.ParseItemType(itemType)
	item, err := s.CreateItem(t, strings.TrimSpace(title), cfg)
	if err != nil {
		return err
	}

	changed := false
	if priority != cfg.Defaults.Priority {
		item.Priority = priority
		changed = true
	}
	if phaseStr != "" {
		p, err := strconv.Atoi(strings.TrimSpace(phaseStr))
		if err != nil || p < 1 {
			return fmt.Errorf("phase must be a positive integer, got %q", phaseStr)
		}
		item.Phase = &p
		changed = true
	}
	if tagsStr != "" {
		var tags []string
		for _, tag := range strings.Split(tagsStr, ",") {
			if t := strings.TrimSpace(tag); t != "" {
				tags = append(tags, t)
			}
		}
		if len(tags) > 0 {
			item.Tags = tags
			changed = true
		}
	}
	if changed {
		item.Updated = time.Now()
		if err := s.WriteItem(item); err != nil {
			return err
		}
	}

	f := getFormatter()
	f.ItemCreated(item)
	return nil
}
```

- [ ] **Step 2: Build to confirm no compile errors**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Smoke-test non-interactive path still works**

```bash
# In a directory with a .liste/ (or create a test one)
liste add feature "My feature title"
```
Expected: item created, output shows ID + title.

- [ ] **Step 4: Commit**

```bash
git add cmd/add.go
git commit -m "feat: add TTY-gated huh interactive form to liste add"
```

---

## Task 9: huh interactive form for liste init

**Files:**
- Modify: `cmd/init.go`

- [ ] **Step 1: Update init.go**

Replace the entire contents of `cmd/init.go` with:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new .liste/ in the current directory",
	Long:  "Creates a .liste/ directory with default config and state files. Run without arguments in a terminal for an interactive prompt.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var name string
	switch {
	case len(args) > 0:
		name = args[0]
	case isatty.IsTerminal(os.Stdin.Fd()):
		name, err = promptProjectName(filepath.Base(cwd))
		if err != nil {
			return err
		}
	default:
		name = filepath.Base(cwd)
	}

	roadmapPath := filepath.Join(cwd, ".liste")
	s := store.New(roadmapPath)

	if s.Exists() {
		return fmt.Errorf(".liste/ already exists in %s", cwd)
	}

	if err := s.Init(name); err != nil {
		return err
	}

	f := getFormatter()
	f.Message(fmt.Sprintf("Initialized .liste/ for project %q", name))
	return nil
}

func promptProjectName(defaultName string) (string, error) {
	var name string = defaultName
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Value(&name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("project name is required")
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}
```

- [ ] **Step 2: Build to confirm no compile errors**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Verify non-interactive path still works**

```bash
cd /tmp && mkdir test-liste && cd test-liste
liste init my-test-project
# Expected: "Initialized .liste/ for project "my-test-project""
cd - && rm -rf /tmp/test-liste
```

- [ ] **Step 4: Commit**

```bash
git add cmd/init.go
git commit -m "feat: add TTY-gated huh interactive form to liste init"
```

---

## Task 10: TUI shared item helpers and list/blocked/next views

**Files:**
- Create: `internal/tui/views/common.go`
- Create: `internal/tui/views/list.go`
- Create: `internal/tui/views/blocked.go`
- Create: `internal/tui/views/next.go`

- [ ] **Step 1: Create common.go — shared item type and rendering helpers**

Create `internal/tui/views/common.go`:

```go
package views

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/output"
)

// ListItem wraps model.Item to implement bubbles list.Item.
type ListItem struct {
	Item *model.Item
}

// Title returns the styled first line shown in the bubbles list.
func (i ListItem) Title() string {
	blocked := i.Item.Blocked != nil
	return fmt.Sprintf("%-10s  %s  %s  %s",
		i.Item.ID,
		output.RenderType(string(i.Item.Type)),
		output.RenderStatus(i.Item.Status, blocked),
		output.RenderPriority(i.Item.Priority),
	)
}

// Description returns the second line shown in the bubbles list.
func (i ListItem) Description() string {
	phase := ""
	if i.Item.Phase != nil {
		phase = fmt.Sprintf("  phase %d", *i.Item.Phase)
	}
	tags := ""
	if len(i.Item.Tags) > 0 {
		tags = "  #" + joinTags(i.Item.Tags)
	}
	return lipgloss.NewStyle().Faint(true).Render(i.Item.Title + phase + tags)
}

// FilterValue is used by bubbles list for fuzzy filtering.
func (i ListItem) FilterValue() string {
	return i.Item.Title + " " + i.Item.ID
}

func joinTags(tags []string) string {
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += " #"
		}
		result += t
	}
	return result
}

// ItemsToListItems converts model items to bubbles list items.
func ItemsToListItems(items []*model.Item) []list.Item {
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = ListItem{Item: item}
	}
	return out
}
```

Note: `list.Item` from `github.com/charmbracelet/bubbles/list` must be added to the import:

```go
import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/output"
)
```

- [ ] **Step 2: Create views/list.go**

Create `internal/tui/views/list.go`:

```go
package views

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

// ItemSelectedMsg is sent when the user presses enter on an item.
type ItemSelectedMsg struct{ Item *model.Item }

// ItemDoneMsg is sent when the user presses 'd' on an item.
type ItemDoneMsg struct{ ID string }

// ItemBlockMsg is sent when the user presses 'b' on an item.
type ItemBlockMsg struct{ ID string }

// ListView shows all items in a scrollable bubbles list.
type ListView struct {
	list  list.Model
	store *store.Store
}

// NewListView creates a list view pre-loaded with all items from the given store.
func NewListView(s *store.Store, width, height int) (ListView, error) {
	items, err := s.ListItems()
	if err != nil {
		return ListView{}, err
	}
	l := newBubblesList("All Items", ItemsToListItems(items), width, height)
	return ListView{list: l, store: s}, nil
}

func newBubblesList(title string, items []list.Item, width, height int) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	return l
}

func (m ListView) Init() tea.Cmd { return nil }

func (m ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemSelectedMsg{Item: li.Item} }
			}
		case "d":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemDoneMsg{ID: li.Item.ID} }
			}
		case "b":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemBlockMsg{ID: li.Item.ID} }
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ListView) View() string { return m.list.View() }

// Reload refreshes the item list from the store.
func (m *ListView) Reload() error {
	items, err := m.store.ListItems()
	if err != nil {
		return err
	}
	m.list.SetItems(ItemsToListItems(items))
	return nil
}
```

- [ ] **Step 3: Create views/blocked.go**

Create `internal/tui/views/blocked.go`:

```go
package views

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

// BlockedView shows only blocked items.
type BlockedView struct {
	list  list.Model
	store *store.Store
}

// NewBlockedView creates a view showing all currently blocked items.
func NewBlockedView(s *store.Store, width, height int) (BlockedView, error) {
	items, err := s.ListItems()
	if err != nil {
		return BlockedView{}, err
	}
	var blocked []*model.Item
	for _, item := range items {
		if item.Blocked != nil {
			blocked = append(blocked, item)
		}
	}
	l := newBubblesList("Blocked Items", ItemsToListItems(blocked), width, height)
	return BlockedView{list: l, store: s}, nil
}

func (m BlockedView) Init() tea.Cmd { return nil }

func (m BlockedView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemSelectedMsg{Item: li.Item} }
			}
		case "d":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemDoneMsg{ID: li.Item.ID} }
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m BlockedView) View() string { return m.list.View() }

// Reload refreshes from store.
func (m *BlockedView) Reload() error {
	items, err := m.store.ListItems()
	if err != nil {
		return err
	}
	var blocked []*model.Item
	for _, item := range items {
		if item.Blocked != nil {
			blocked = append(blocked, item)
		}
	}
	m.list.SetItems(ItemsToListItems(blocked))
	return nil
}
```

- [ ] **Step 4: Create views/next.go**

Create `internal/tui/views/next.go`:

```go
package views

import (
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

// NextView shows the priority-sorted queue of items ready to work on.
type NextView struct {
	list  list.Model
	store *store.Store
}

// NewNextView creates the next-queue view.
func NewNextView(s *store.Store, width, height int) (NextView, error) {
	items, err := nextItems(s)
	if err != nil {
		return NextView{}, err
	}
	l := newBubblesList("Next Up", ItemsToListItems(items), width, height)
	return NextView{list: l, store: s}, nil
}

func nextItems(s *store.Store) ([]*model.Item, error) {
	all, err := s.ListItems()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*model.Item, len(all))
	for _, item := range all {
		byID[item.ID] = item
	}

	var candidates []*model.Item
	for _, item := range all {
		if item.Status == "done" || item.Status == "cancelled" || item.Status == "active" {
			continue
		}
		if item.Blocked != nil {
			continue
		}
		if !depsResolved(item, byID) {
			continue
		}
		candidates = append(candidates, item)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		ap, bp := phaseOrder(a), phaseOrder(b)
		if ap != bp {
			return ap < bp
		}
		if priorityWeight(a.Priority) != priorityWeight(b.Priority) {
			return priorityWeight(a.Priority) < priorityWeight(b.Priority)
		}
		return a.Created.Before(b.Created)
	})
	return candidates, nil
}

// depsResolved returns true when all depends-on links point to done/cancelled items.
func depsResolved(item *model.Item, byID map[string]*model.Item) bool {
	for _, link := range item.Links {
		if link.Type != model.LinkDependsOn {
			continue
		}
		target, ok := byID[link.Target]
		if !ok || (target.Status != "done" && target.Status != "cancelled") {
			return false
		}
	}
	return true
}

func phaseOrder(item *model.Item) int {
	if item.Phase == nil {
		return 9999
	}
	return *item.Phase
}

func priorityWeight(p string) int {
	switch p {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

func (m NextView) Init() tea.Cmd { return nil }

func (m NextView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemSelectedMsg{Item: li.Item} }
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m NextView) View() string { return m.list.View() }

// Reload refreshes from store.
func (m *NextView) Reload() error {
	items, err := nextItems(m.store)
	if err != nil {
		return err
	}
	m.list.SetItems(ItemsToListItems(items))
	return nil
}
```

- [ ] **Step 5: Build to verify no errors**

```bash
go build ./internal/tui/...
```
Expected: no errors (may need to create placeholder app.go first — see note below).

If the build fails because `internal/tui` package doesn't exist yet, create a minimal placeholder:
```bash
mkdir -p internal/tui
cat > internal/tui/app.go << 'EOF'
package tui
EOF
```
Then retry `go build ./internal/tui/views/...`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI list, blocked, and next views (bubbles list)"
```

---

## Task 11: TUI search view

**Files:**
- Create: `internal/tui/views/search.go`

- [ ] **Step 1: Create views/search.go**

Create `internal/tui/views/search.go`:

```go
package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

var searchInputStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#74b9ff")).
	Padding(0, 1)

// SearchView provides a text input that filters items in real time.
type SearchView struct {
	input    textinput.Model
	list     list.Model
	allItems []*model.Item
	store    *store.Store
	width    int
	height   int
}

// NewSearchView creates the search view.
func NewSearchView(s *store.Store, width, height int) (SearchView, error) {
	items, err := s.ListItems()
	if err != nil {
		return SearchView{}, err
	}

	ti := textinput.New()
	ti.Placeholder = "Search items..."
	ti.Focus()
	ti.CharLimit = 100

	l := newBubblesList("Results", ItemsToListItems(items), width, height-5)
	l.SetShowTitle(false)

	return SearchView{
		input:    ti,
		list:     l,
		allItems: items,
		store:    s,
		width:    width,
		height:   height,
	}, nil
}

func (m SearchView) Init() tea.Cmd { return textinput.Blink }

func (m SearchView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.input.SetValue("")
			m.list.SetItems(ItemsToListItems(m.allItems))
			return m, nil
		case "enter":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemSelectedMsg{Item: li.Item} }
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-5)
	}

	var cmds []tea.Cmd

	// Update text input
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)

	// Filter items based on current query
	query := strings.ToLower(m.input.Value())
	if query != "" {
		var filtered []*model.Item
		for _, item := range m.allItems {
			if strings.Contains(strings.ToLower(item.Title), query) ||
				strings.Contains(strings.ToLower(item.ID), query) ||
				containsTag(item.Tags, query) {
				filtered = append(filtered, item)
			}
		}
		m.list.SetItems(ItemsToListItems(filtered))
	} else {
		m.list.SetItems(ItemsToListItems(m.allItems))
	}

	// Update list
	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	cmds = append(cmds, listCmd)

	return m, tea.Batch(cmds...)
}

func (m SearchView) View() string {
	return searchInputStyle.Width(m.width - 4).Render(m.input.View()) +
		"\n" +
		m.list.View()
}

// Reload refreshes from store.
func (m *SearchView) Reload() error {
	items, err := m.store.ListItems()
	if err != nil {
		return err
	}
	m.allItems = items
	m.list.SetItems(ItemsToListItems(items))
	return nil
}

func containsTag(tags []string, query string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), query) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./internal/tui/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/views/search.go
git commit -m "feat: add TUI search view with real-time filtering"
```

---

## Task 12: TUI roadmap view

**Files:**
- Create: `internal/tui/views/roadmap.go`

- [ ] **Step 1: Create views/roadmap.go**

Create `internal/tui/views/roadmap.go`:

```go
package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/output"
	"github.com/pufferhaus/liste/internal/store"
)

var (
	phaseHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4"))
	phaseDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#313244"))
	phaseItemStyle    = lipgloss.NewStyle().PaddingLeft(2)
)

// RoadmapView renders a phase-grouped roadmap in a scrollable viewport.
type RoadmapView struct {
	viewport viewport.Model
	store    *store.Store
	width    int
	height   int
}

// NewRoadmapView creates the roadmap view.
func NewRoadmapView(s *store.Store, width, height int) (RoadmapView, error) {
	vp := viewport.New(width, height-3)
	content, err := buildRoadmapContent(s, width)
	if err != nil {
		return RoadmapView{}, err
	}
	vp.SetContent(content)
	return RoadmapView{viewport: vp, store: s, width: width, height: height}, nil
}

func buildRoadmapContent(s *store.Store, width int) (string, error) {
	items, err := s.ListItems()
	if err != nil {
		return "", err
	}

	// Group by phase
	type phaseItems struct {
		phase int
		items []*model.Item
	}
	phaseMap := make(map[int][]*model.Item)
	var unphased []*model.Item
	for _, item := range items {
		if item.Phase == nil {
			unphased = append(unphased, item)
		} else {
			phaseMap[*item.Phase] = append(phaseMap[*item.Phase], item)
		}
	}
	var phases []int
	for p := range phaseMap {
		phases = append(phases, p)
	}
	sort.Ints(phases)

	var sb strings.Builder
	divider := phaseDividerStyle.Render(strings.Repeat("─", min(width-4, 60)))

	for _, p := range phases {
		pItems := phaseMap[p]
		done, total := countDone(pItems)
		status := phaseStatus(pItems)
		header := phaseHeaderStyle.Render(fmt.Sprintf("PHASE %d  %s  %d/%d", p, status, done, total))
		sb.WriteString(header + "\n" + divider + "\n")
		for _, item := range pItems {
			row := phaseItemStyle.Render(fmt.Sprintf("%s  %-10s  %s  %s",
				output.RenderStatus(item.Status, item.Blocked != nil),
				item.ID,
				output.RenderPriority(item.Priority),
				item.Title,
			))
			sb.WriteString(row + "\n")
		}
		sb.WriteString("\n")
	}

	if len(unphased) > 0 {
		header := phaseHeaderStyle.Render(fmt.Sprintf("UNPHASED  (%d)", len(unphased)))
		sb.WriteString(header + "\n" + divider + "\n")
		for _, item := range unphased {
			row := phaseItemStyle.Render(fmt.Sprintf("%s  %-10s  %s  %s",
				output.RenderStatus(item.Status, item.Blocked != nil),
				item.ID,
				output.RenderPriority(item.Priority),
				item.Title,
			))
			sb.WriteString(row + "\n")
		}
	}

	return sb.String(), nil
}

func countDone(items []*model.Item) (int, int) {
	done := 0
	for _, item := range items {
		if item.Status == "done" || item.Status == "cancelled" {
			done++
		}
	}
	return done, len(items)
}

func phaseStatus(items []*model.Item) string {
	done, total := countDone(items)
	if total > 0 && done == total {
		return "complete"
	}
	for _, item := range items {
		if item.Status == "active" {
			return "active"
		}
	}
	return "upcoming"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m RoadmapView) Init() tea.Cmd { return nil }

func (m RoadmapView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
		// Rebuild content at new width
		if content, err := buildRoadmapContent(m.store, msg.Width); err == nil {
			m.viewport.SetContent(content)
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m RoadmapView) View() string { return m.viewport.View() }

// Reload refreshes the viewport content from the store.
func (m *RoadmapView) Reload() error {
	content, err := buildRoadmapContent(m.store, m.width)
	if err != nil {
		return err
	}
	m.viewport.SetContent(content)
	return nil
}

// SelectedItem returns nil — roadmap view does not support item selection.
func (m RoadmapView) SelectedItem() *model.Item { return nil }
```

- [ ] **Step 2: Build to verify**

```bash
go build ./internal/tui/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/views/roadmap.go
git commit -m "feat: add TUI roadmap view (phase-grouped scrollable viewport)"
```

---

## Task 13: TUI detail overlay

**Files:**
- Create: `internal/tui/detail.go`

- [ ] **Step 1: Create detail.go**

Create `internal/tui/detail.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/output"
)

var (
	detailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#74b9ff")).
				Padding(0, 1)
	detailHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4"))
	detailFaintStyle  = lipgloss.NewStyle().Faint(true)
)

// DetailModel is a scrollable overlay showing full item detail.
type DetailModel struct {
	item     *model.Item
	viewport viewport.Model
	width    int
	height   int
}

// NewDetailModel creates a detail overlay for the given item.
func NewDetailModel(item *model.Item, width, height int) DetailModel {
	innerW := width - 4
	innerH := height - 4
	vp := viewport.New(innerW, innerH)
	vp.SetContent(renderDetail(item, innerW))
	return DetailModel{item: item, viewport: vp, width: width, height: height}
}

func renderDetail(item *model.Item, width int) string {
	var sb strings.Builder

	sb.WriteString(detailHeaderStyle.Render(item.ID+"  "+item.Title) + "\n")
	sb.WriteString(fmt.Sprintf("%s  %s  %s\n",
		output.RenderType(string(item.Type)),
		output.RenderStatus(item.Status, item.Blocked != nil),
		output.RenderPriority(item.Priority),
	))
	sb.WriteString(detailFaintStyle.Render(fmt.Sprintf(
		"Created: %s  Updated: %s",
		item.Created.Format("2006-01-02"),
		item.Updated.Format("2006-01-02"),
	)) + "\n")

	if len(item.Tags) > 0 {
		sb.WriteString("Tags: " + strings.Join(item.Tags, ", ") + "\n")
	}
	if item.Blocked != nil {
		reason := item.Blocked.Reason
		if reason == "" {
			reason = "(no reason)"
		}
		sb.WriteString(output.RenderStatus("blocked", true) + " " + reason + "\n")
	}
	if len(item.Links) > 0 {
		sb.WriteString("\n" + detailHeaderStyle.Render("Links:") + "\n")
		for _, l := range item.Links {
			sb.WriteString(fmt.Sprintf("  %s %s\n", detailFaintStyle.Render(string(l.Type)), l.Target))
		}
	}

	if item.Body != "" {
		sb.WriteString("\n")
		rendered, err := glamour.Render(item.Body, "auto")
		if err != nil {
			sb.WriteString(item.Body + "\n")
		} else {
			sb.WriteString(rendered)
		}
	}

	return sb.String()
}

// CloseDetailMsg signals that the detail overlay should close.
type CloseDetailMsg struct{}

func (m DetailModel) Init() tea.Cmd { return nil }

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseDetailMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 4
		m.viewport.SetContent(renderDetail(m.item, msg.Width-4))
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DetailModel) View() string {
	return detailBorderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(m.viewport.View())
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./internal/tui/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/detail.go
git commit -m "feat: add TUI detail overlay with glamour markdown and viewport"
```

---

## Task 14: TUI app root model

**Files:**
- Modify: `internal/tui/app.go` (replace placeholder)

- [ ] **Step 1: Write app_test.go**

Create `internal/tui/app_test.go`:

```go
package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/tui"
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

	// Press tab — should move from list (0) to roadmap (1)
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
	// Two tab presses wraps back to 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	app := updated.(tui.AppModel)
	if app.ActiveTab() != 0 {
		t.Errorf("ActiveTab after wrap: got %d, want 0", app.ActiveTab())
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/tui/... -run TestTab
```
Expected: FAIL — `tui.NewAppForTest` undefined.

- [ ] **Step 3: Replace app.go with full implementation**

Replace `internal/tui/app.go` with:

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/discovery"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/pufferhaus/liste/internal/tui/views"
)

var (
	tabBarStyle     = lipgloss.NewStyle().Padding(0, 1)
	tabActiveStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4")).Underline(true)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7a89"))
	statusBarStyle  = lipgloss.NewStyle().Faint(true).Padding(0, 1)
)

// AppModel is the root bubbletea model for liste -i.
// Exported so tests can inspect it.
type AppModel struct {
	tabs      []string
	activeTab int
	viewMap   map[string]tea.Model
	overlay   *DetailModel
	blockInput *blockInputModel
	store     *store.Store
	tuiCfg    model.TUIConfig
	width     int
	height    int
	statusMsg string
}

// blockInputModel handles the 'b' key — prompts for block reason inline.
type blockInputModel struct {
	input  textinput.Model
	itemID string
}

// ActiveTab returns the current active tab index (for tests).
func (m AppModel) ActiveTab() int { return m.activeTab }

// NewAppForTest creates an AppModel with no store, for unit testing tab logic.
func NewAppForTest(cfg *model.Config) AppModel {
	tuiCfg := cfg.TUI.Resolved()
	startIdx := 0
	for i, v := range tuiCfg.Views {
		if v == tuiCfg.DefaultView {
			startIdx = i
			break
		}
	}
	return AppModel{
		tabs:      tuiCfg.Views,
		activeTab: startIdx,
		viewMap:   make(map[string]tea.Model),
		tuiCfg:    tuiCfg,
	}
}

// newApp creates a fully initialized AppModel backed by a real store.
func newApp(result *discovery.Result, rootCfg *model.Config) (AppModel, error) {
	m := NewAppForTest(rootCfg)
	m.store = store.New(result.Root)

	// Initialize the default view immediately
	view, err := m.initView(rootCfg.TUI.Resolved().DefaultView, 80, 24)
	if err != nil {
		return AppModel{}, fmt.Errorf("initializing default view: %w", err)
	}
	m.viewMap[rootCfg.TUI.Resolved().DefaultView] = view
	return m, nil
}

// initView creates a view model for the given view name.
func (m AppModel) initView(name string, width, height int) (tea.Model, error) {
	contentH := height - 3 // subtract tab bar
	switch name {
	case "list":
		return views.NewListView(m.store, width, contentH)
	case "blocked":
		return views.NewBlockedView(m.store, width, contentH)
	case "next":
		return views.NewNextView(m.store, width, contentH)
	case "search":
		return views.NewSearchView(m.store, width, contentH)
	case "roadmap":
		return views.NewRoadmapView(m.store, width, contentH)
	default:
		return views.NewListView(m.store, width, contentH)
	}
}

// currentViewName returns the name of the currently active view.
func (m AppModel) currentViewName() string {
	if m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab]
	}
	return "list"
}

// Run starts the bubbletea program.
func Run(result *discovery.Result, rootCfg *model.Config) error {
	m, err := newApp(result, rootCfg)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func (m AppModel) Init() tea.Cmd {
	if v, ok := m.viewMap[m.currentViewName()]; ok {
		return v.Init()
	}
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route to block input if active
	if m.blockInput != nil {
		return m.updateBlockInput(msg)
	}
	// Route to detail overlay if open
	if m.overlay != nil {
		return m.updateOverlay(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to all initialized views
		for name, v := range m.viewMap {
			updated, _ := v.Update(msg)
			m.viewMap[name] = updated
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return m, m.ensureViewLoaded()
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m, m.ensureViewLoaded()
		}

	case views.ItemSelectedMsg:
		overlay := NewDetailModel(msg.Item, m.width, m.height)
		m.overlay = &overlay
		return m, nil

	case views.ItemDoneMsg:
		if err := m.markDone(msg.ID); err != nil {
			m.statusMsg = "Error: " + err.Error()
		} else {
			m.statusMsg = msg.ID + " marked done"
			m.reloadCurrentView()
		}
		return m, nil

	case views.ItemBlockMsg:
		ti := textinput.New()
		ti.Placeholder = "Block reason (optional, press enter to confirm)"
		ti.Focus()
		m.blockInput = &blockInputModel{input: ti, itemID: msg.ID}
		return m, textinput.Blink

	case CloseDetailMsg:
		m.overlay = nil
		return m, nil
	}

	// Delegate to current view
	return m.updateCurrentView(msg)
}

func (m AppModel) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.overlay.Update(msg)
	if detail, ok := updated.(DetailModel); ok {
		m.overlay = &detail
	}
	return m, cmd
}

func (m AppModel) updateBlockInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			reason := m.blockInput.input.Value()
			id := m.blockInput.itemID
			m.blockInput = nil
			if err := m.markBlocked(id, reason); err != nil {
				m.statusMsg = "Error: " + err.Error()
			} else {
				m.statusMsg = id + " blocked"
				m.reloadCurrentView()
			}
			return m, nil
		case "esc":
			m.blockInput = nil
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.blockInput.input, cmd = m.blockInput.input.Update(msg)
	return m, cmd
}

func (m AppModel) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	name := m.currentViewName()
	v, ok := m.viewMap[name]
	if !ok {
		return m, nil
	}
	updated, cmd := v.Update(msg)
	m.viewMap[name] = updated
	return m, cmd
}

func (m AppModel) ensureViewLoaded() tea.Cmd {
	name := m.currentViewName()
	if _, ok := m.viewMap[name]; !ok {
		return func() tea.Msg {
			view, err := m.initView(name, m.width, m.height)
			if err != nil {
				return nil
			}
			m.viewMap[name] = view
			return nil
		}
	}
	return nil
}

func (m *AppModel) reloadCurrentView() {
	name := m.currentViewName()
	v, ok := m.viewMap[name]
	if !ok {
		return
	}
	type reloader interface{ Reload() error }
	if r, ok := v.(reloader); ok {
		_ = r.Reload()
	}
}

func (m AppModel) markDone(id string) error {
	item, err := m.store.ReadItem(id)
	if err != nil {
		return err
	}
	item.Status = "done"
	item.Updated = time.Now()
	return m.store.WriteItem(item)
}

func (m AppModel) markBlocked(id, reason string) error {
	item, err := m.store.ReadItem(id)
	if err != nil {
		return err
	}
	item.Blocked = &model.Blocked{Reason: reason}
	item.Updated = time.Now()
	return m.store.WriteItem(item)
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Overlay takes full screen
	if m.overlay != nil {
		return m.overlay.View()
	}

	// Block input prompt
	if m.blockInput != nil {
		return m.blockInput.input.View()
	}

	// Tab bar
	var tabParts []string
	for i, name := range m.tabs {
		if i == m.activeTab {
			tabParts = append(tabParts, tabActiveStyle.Render(strings.ToUpper(name)))
		} else {
			tabParts = append(tabParts, tabInactiveStyle.Render(name))
		}
	}
	tabBar := tabBarStyle.Render(strings.Join(tabParts, "  "))

	// Current view
	viewStr := ""
	if v, ok := m.viewMap[m.currentViewName()]; ok {
		viewStr = v.View()
	}

	// Status bar
	statusBar := statusBarStyle.Render(m.statusMsg + "  tab/shift+tab: switch  d: done  b: block  enter: detail  q: quit")

	return tabBar + "\n" + viewStr + "\n" + statusBar
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/tui/... -run TestTab
```
Expected: PASS.

- [ ] **Step 5: Build to verify**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat: add TUI root model with tab navigation, detail overlay, and item mutations"
```

---

## Task 15: Wire -i flag to root command

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Update root.go**

Replace the entire contents of `cmd/root.go` with:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/pufferhaus/liste/internal/discovery"
	"github.com/pufferhaus/liste/internal/output"
	"github.com/pufferhaus/liste/internal/store"
	"github.com/pufferhaus/liste/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagJSON        bool
	flagQuiet       bool
	flagProject     string
	flagInteractive bool
)

var rootCmd = &cobra.Command{
	Use:   "liste",
	Short: "Portable roadmap and project tracker",
	Long:  "A CLI tool for managing project roadmaps as markdown files. Designed for both humans and AI agents.",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "Minimal output (IDs only)")
	rootCmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "Target a specific sub-project")
	rootCmd.PersistentFlags().BoolVarP(&flagInteractive, "interactive", "i", false, "Launch interactive TUI")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if !flagInteractive {
			return nil
		}
		if flagJSON || flagQuiet {
			return fmt.Errorf("--interactive cannot be used with --json or --quiet")
		}
		result, err := getDiscovery()
		if err != nil {
			return err
		}
		rootStore := store.New(result.Root)
		cfg, err := rootStore.ReadConfig()
		if err != nil {
			return err
		}
		if err := tui.Run(result, cfg); err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// getFormatter returns a formatter based on current flags.
func getFormatter() *output.Formatter {
	format := output.FormatTable
	if flagJSON {
		format = output.FormatJSON
	} else if flagQuiet {
		format = output.FormatQuiet
	}
	return output.New(os.Stdout, format)
}

// getStore resolves the store for the current context (CWD + project flag).
func getStore() (*store.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	result, err := discovery.Discover(cwd)
	if err != nil {
		return nil, fmt.Errorf("discovering projects: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("no .liste/ found (run 'liste init' to create one)")
	}

	if flagProject != "" {
		s := discovery.StoreForProject(result, flagProject)
		if s == nil {
			return nil, fmt.Errorf("project %q not found", flagProject)
		}
		return s, nil
	}

	return store.New(result.Root), nil
}

// getDiscovery returns the full discovery result for multi-project commands.
func getDiscovery() (*discovery.Result, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	result, err := discovery.Discover(cwd)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("no .liste/ found (run 'liste init' to create one)")
	}

	return result, nil
}
```

- [ ] **Step 2: Build and run full test suite**

```bash
go build ./...
go test ./...
```
Expected: build succeeds, all tests pass.

- [ ] **Step 3: Smoke-test the full flow**

In a directory that has a `.liste/` initialized:
```bash
# Verify existing non-interactive commands still work
liste list
liste roadmap
liste status

# Verify JSON path untouched
liste list --json

# Verify error on conflicting flags
liste -i --json
# Expected: "error: --interactive cannot be used with --json or --quiet"

# Launch TUI (manual verification)
liste -i
```

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat: wire -i flag to launch bubbletea TUI (liste -i)"
```

---

## Self-Review Notes

- **Spec coverage:** lipgloss ✅, glamour ✅, huh (add + init) ✅, TUI views (list/blocked/next/search/roadmap) ✅, detail overlay ✅, `-i` flag ✅, TUIConfig + default view ✅, JSON/quiet unchanged ✅, NO_COLOR ✅ (lipgloss built-in)
- **Type consistency:** `views.ItemSelectedMsg`, `views.ItemDoneMsg`, `views.ItemBlockMsg` defined in Task 10 and used in Task 14. `CloseDetailMsg` defined in Task 13 and used in Task 14. `AppModel` and `NewAppForTest` defined and tested in Task 14. `TUIConfig.Resolved()` defined in Task 2, called in Task 14.
- **Ordering note:** Tasks 1 and 2 must complete before any other task. Tasks 3–7 (output styling) are independent of Tasks 8–9 (huh) and Tasks 10–15 (TUI). Within the TUI group, Tasks 10–13 (views + detail) must precede Task 14 (app.go), which must precede Task 15 (root wiring).
