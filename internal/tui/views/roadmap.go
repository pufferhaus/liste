package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pblca/liste/internal/model"
	"github.com/pblca/liste/internal/output"
	"github.com/pblca/liste/internal/store"
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
	divW := width - 4
	if divW > 60 {
		divW = 60
	}
	if divW < 1 {
		divW = 1
	}
	divider := phaseDividerStyle.Render(strings.Repeat("─", divW))

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

func (m RoadmapView) Init() tea.Cmd { return nil }

func (m RoadmapView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
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
