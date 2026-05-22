package store

import (
	"testing"
	"time"

	"github.com/pufferhaus/liste/internal/model"
)

func TestParseItem(t *testing.T) {
	input := `---
id: FEAT-001
type: feature
title: Test feature
status: active
priority: high
phase: 2
created: "2026-01-15"
updated: "2026-05-01"
tags:
    - backend
    - api
links:
    - type: depends-on
      target: TASK-001
---

## Description

This is a test feature.
`
	item, err := ParseItem([]byte(input))
	if err != nil {
		t.Fatalf("ParseItem failed: %v", err)
	}

	if item.ID != "FEAT-001" {
		t.Errorf("ID = %q, want %q", item.ID, "FEAT-001")
	}
	if item.Type != model.TypeFeature {
		t.Errorf("Type = %q, want %q", item.Type, model.TypeFeature)
	}
	if item.Title != "Test feature" {
		t.Errorf("Title = %q, want %q", item.Title, "Test feature")
	}
	if item.Status != "active" {
		t.Errorf("Status = %q, want %q", item.Status, "active")
	}
	if item.Priority != "high" {
		t.Errorf("Priority = %q, want %q", item.Priority, "high")
	}
	if item.Phase == nil || *item.Phase != 2 {
		t.Errorf("Phase = %v, want 2", item.Phase)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "backend" || item.Tags[1] != "api" {
		t.Errorf("Tags = %v, want [backend, api]", item.Tags)
	}
	if len(item.Links) != 1 {
		t.Fatalf("Links length = %d, want 1", len(item.Links))
	}
	if item.Links[0].Type != model.LinkDependsOn || item.Links[0].Target != "TASK-001" {
		t.Errorf("Link = %+v, want depends-on TASK-001", item.Links[0])
	}
	if item.Body == "" {
		t.Error("Body is empty, expected content")
	}
}

func TestParseItemMinimal(t *testing.T) {
	input := `---
id: BUG-001
type: bug
title: A bug
status: idea
priority: medium
created: "2026-03-01"
updated: "2026-03-01"
---
`
	item, err := ParseItem([]byte(input))
	if err != nil {
		t.Fatalf("ParseItem failed: %v", err)
	}

	if item.ID != "BUG-001" {
		t.Errorf("ID = %q, want %q", item.ID, "BUG-001")
	}
	if item.Phase != nil {
		t.Errorf("Phase = %v, want nil", item.Phase)
	}
	if len(item.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", item.Tags)
	}
	if len(item.Links) != 0 {
		t.Errorf("Links = %v, want empty", item.Links)
	}
	if item.Body != "" {
		t.Errorf("Body = %q, want empty", item.Body)
	}
}

func TestParseItemBlocked(t *testing.T) {
	input := `---
id: FEAT-002
type: feature
title: Blocked feature
status: planned
priority: high
created: "2026-04-01"
updated: "2026-04-10"
blocked:
    reason: Waiting on external API
---
`
	item, err := ParseItem([]byte(input))
	if err != nil {
		t.Fatalf("ParseItem failed: %v", err)
	}

	if item.Blocked == nil {
		t.Fatal("Blocked is nil, want non-nil")
	}
	if item.Blocked.Reason != "Waiting on external API" {
		t.Errorf("Blocked.Reason = %q, want %q", item.Blocked.Reason, "Waiting on external API")
	}
}

func TestMarshalItem(t *testing.T) {
	phase := 1
	item := &model.Item{
		ID:       "TASK-001",
		Type:     model.TypeTask,
		Title:    "Write tests",
		Status:   "planned",
		Priority: "medium",
		Phase:    &phase,
		Created:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated:  time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
		Tags:     []string{"testing"},
		Links: []model.Link{
			{Type: model.LinkChildOf, Target: "EPIC-001"},
		},
		Body: "## Notes\n\n- Write unit tests first\n",
	}

	data, err := MarshalItem(item)
	if err != nil {
		t.Fatalf("MarshalItem failed: %v", err)
	}

	// Re-parse and verify roundtrip
	parsed, err := ParseItem(data)
	if err != nil {
		t.Fatalf("ParseItem after Marshal failed: %v", err)
	}

	if parsed.ID != item.ID {
		t.Errorf("Roundtrip ID = %q, want %q", parsed.ID, item.ID)
	}
	if parsed.Title != item.Title {
		t.Errorf("Roundtrip Title = %q, want %q", parsed.Title, item.Title)
	}
	if parsed.Phase == nil || *parsed.Phase != 1 {
		t.Errorf("Roundtrip Phase = %v, want 1", parsed.Phase)
	}
	if len(parsed.Links) != 1 || parsed.Links[0].Target != "EPIC-001" {
		t.Errorf("Roundtrip Links = %v, want child-of EPIC-001", parsed.Links)
	}
	if parsed.Body == "" {
		t.Error("Roundtrip Body is empty")
	}
}

func TestParseItemInvalidNoFrontmatter(t *testing.T) {
	input := `# Just a markdown file
No frontmatter here.`

	_, err := ParseItem([]byte(input))
	if err == nil {
		t.Error("Expected error for missing frontmatter, got nil")
	}
}

func TestParseItemInvalidNoClosingDelimiter(t *testing.T) {
	input := `---
id: FEAT-001
type: feature
title: broken
`
	_, err := ParseItem([]byte(input))
	if err == nil {
		t.Error("Expected error for missing closing delimiter, got nil")
	}
}

func TestParseItemEmpty(t *testing.T) {
	_, err := ParseItem([]byte(""))
	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

func TestParseItemMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing id", "---\ntype: feature\ntitle: test\nstatus: idea\n---\n"},
		{"missing type", "---\nid: FEAT-001\ntitle: test\nstatus: idea\n---\n"},
		{"missing title", "---\nid: FEAT-001\ntype: feature\nstatus: idea\n---\n"},
		{"missing status", "---\nid: FEAT-001\ntype: feature\ntitle: test\n---\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseItem([]byte(tc.input))
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestParseItemCorruptedDates(t *testing.T) {
	input := `---
id: FEAT-001
type: feature
title: Bad dates
status: idea
priority: medium
created: "not-a-date"
updated: "also-bad"
---
`
	item, err := ParseItem([]byte(input))
	if err != nil {
		t.Fatalf("Expected graceful fallback for bad dates, got error: %v", err)
	}
	if item.Created.IsZero() {
		t.Error("Created should fallback to now, not be zero")
	}
}

func TestParseItemEmptyFrontmatter(t *testing.T) {
	input := `---
---
`
	_, err := ParseItem([]byte(input))
	if err == nil {
		t.Error("Expected error for empty frontmatter, got nil")
	}
}
