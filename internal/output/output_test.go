package output_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pblca/liste/internal/model"
	"github.com/pblca/liste/internal/output"
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

func TestRenderPhaseHeader(t *testing.T) {
	got := stripANSI(output.RenderPhaseHeader(1, "active", 2, 5))
	if !strings.Contains(got, "PHASE 1") {
		t.Errorf("RenderPhaseHeader missing phase number, got: %q", got)
	}
	if !strings.Contains(got, "active") {
		t.Errorf("RenderPhaseHeader missing status, got: %q", got)
	}
	if !strings.Contains(got, "2/5") {
		t.Errorf("RenderPhaseHeader missing progress, got: %q", got)
	}
	if !strings.Contains(got, "─") {
		t.Errorf("RenderPhaseHeader missing divider, got: %q", got)
	}
}
