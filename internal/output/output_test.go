package output_test

import (
	"regexp"
	"strings"
	"testing"

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
