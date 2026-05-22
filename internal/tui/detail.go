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
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 4
	if innerH < 1 {
		innerH = 1
	}
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
		vw := msg.Width - 4
		if vw < 1 {
			vw = 1
		}
		vh := msg.Height - 4
		if vh < 1 {
			vh = 1
		}
		m.viewport.Width = vw
		m.viewport.Height = vh
		m.viewport.SetContent(renderDetail(m.item, vw))
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
