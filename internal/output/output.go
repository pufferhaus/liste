package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
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

// Format represents the output format.
type Format int

const (
	FormatTable Format = iota
	FormatJSON
	FormatQuiet
)

// Formatter handles output rendering.
type Formatter struct {
	Writer io.Writer
	Format Format
}

// New creates a formatter for the given writer and format.
func New(w io.Writer, format Format) *Formatter {
	return &Formatter{Writer: w, Format: format}
}

// ItemCreated outputs the result of creating an item.
func (f *Formatter) ItemCreated(item *model.Item) {
	switch f.Format {
	case FormatJSON:
		f.json(map[string]any{
			"id":       item.ID,
			"type":     string(item.Type),
			"title":    item.Title,
			"status":   item.Status,
			"priority": item.Priority,
			"created":  item.Created.Format("2006-01-02"),
		})
	case FormatQuiet:
		fmt.Fprintln(f.Writer, item.ID)
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
	}
}

// ItemDetail outputs the full detail of an item.
func (f *Formatter) ItemDetail(item *model.Item, inverseLinks []InverseLinkDisplay) {
	switch f.Format {
	case FormatJSON:
		data := map[string]any{
			"id":       item.ID,
			"type":     string(item.Type),
			"title":    item.Title,
			"status":   item.Status,
			"priority": item.Priority,
			"created":  item.Created.Format("2006-01-02"),
			"updated":  item.Updated.Format("2006-01-02"),
			"tags":     item.Tags,
			"links":    item.Links,
			"body":     item.Body,
		}
		if item.Blocked != nil {
			data["blocked"] = item.Blocked
		}
		if len(inverseLinks) > 0 {
			data["referenced_by"] = inverseLinks
		}
		f.json(data)
	case FormatQuiet:
		fmt.Fprintln(f.Writer, item.ID)
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
	}
}

// InverseLinkDisplay is a simplified inverse link for display.
type InverseLinkDisplay struct {
	Type     string `json:"type"`
	SourceID string `json:"source_id"`
}

// ItemList outputs a list of items.
func (f *Formatter) ItemList(items []*model.Item) {
	switch f.Format {
	case FormatJSON:
		list := make([]map[string]any, 0, len(items))
		for _, item := range items {
			entry := map[string]any{
				"id":       item.ID,
				"type":     string(item.Type),
				"title":    item.Title,
				"status":   item.Status,
				"priority": item.Priority,
			}
			if item.Blocked != nil {
				entry["blocked"] = true
			}
			list = append(list, entry)
		}
		f.json(list)
	case FormatQuiet:
		for _, item := range items {
			fmt.Fprintln(f.Writer, item.ID)
		}
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
			idCell := lipgloss.NewStyle().Width(idW).Render(item.ID)
			typeCell := lipgloss.NewStyle().Width(typeW).Render(RenderType(string(item.Type)))
			statusCell := lipgloss.NewStyle().Width(statusW).Render(RenderStatus(item.Status, item.Blocked != nil))
			priCell := lipgloss.NewStyle().Width(priW).Render(RenderPriority(item.Priority))
			row := fmt.Sprintf("%s %s %s %s %s", idCell, typeCell, statusCell, priCell, item.Title)
			if isDone {
				row = styleFaint.Render(row)
			}
			fmt.Fprintln(f.Writer, row)
		}
		fmt.Fprintf(f.Writer, "\n%d item(s)\n", len(items))
	}
}

// StatusSummary outputs a dashboard-style summary.
func (f *Formatter) StatusSummary(items []*model.Item, projectName string) {
	// Group by status
	groups := make(map[string][]*model.Item)
	for _, item := range items {
		status := item.Status
		if item.Blocked != nil {
			status = "blocked"
		}
		groups[status] = append(groups[status], item)
	}

	switch f.Format {
	case FormatJSON:
		summary := map[string]any{
			"project": projectName,
			"total":   len(items),
			"by_status": func() map[string]int {
				counts := make(map[string]int)
				for k, v := range groups {
					counts[k] = len(v)
				}
				return counts
			}(),
			"items": func() []map[string]any {
				list := make([]map[string]any, 0, len(items))
				for _, item := range items {
					list = append(list, map[string]any{
						"id":       item.ID,
						"type":     string(item.Type),
						"title":    item.Title,
						"status":   item.Status,
						"priority": item.Priority,
						"blocked":  item.Blocked != nil,
					})
				}
				return list
			}(),
		}
		f.json(summary)
	case FormatQuiet:
		fmt.Fprintf(f.Writer, "%d items\n", len(items))
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
	}
}

// ProjectList outputs the list of discovered projects.
func (f *Formatter) ProjectList(root string, subProjects []ProjectDisplay) {
	switch f.Format {
	case FormatJSON:
		f.json(map[string]any{
			"root":         root,
			"sub_projects": subProjects,
		})
	default:
		fmt.Fprintf(f.Writer, "Root: %s\n", root)
		if len(subProjects) > 0 {
			fmt.Fprintln(f.Writer, "\nSub-projects:")
			for _, p := range subProjects {
				fmt.Fprintf(f.Writer, "  %s (%d items)\n", p.Name, p.ItemCount)
			}
		}
	}
}

// ProjectDisplay is a simplified project for display.
type ProjectDisplay struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	ItemCount int    `json:"item_count"`
}

// Message outputs a simple message.
func (f *Formatter) Message(msg string) {
	switch f.Format {
	case FormatJSON:
		f.json(map[string]string{"message": msg})
	default:
		fmt.Fprintln(f.Writer, msg)
	}
}

// Error outputs an error message.
func (f *Formatter) Error(err error) {
	switch f.Format {
	case FormatJSON:
		f.json(map[string]string{"error": err.Error()})
	default:
		fmt.Fprintf(f.Writer, "Error: %s\n", err)
	}
}

func (f *Formatter) json(v any) {
	enc := json.NewEncoder(f.Writer)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
