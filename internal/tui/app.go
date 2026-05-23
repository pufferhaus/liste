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

// tabBarHeight is the number of terminal rows the tab bar occupies.
const tabBarHeight = 3

var (
	tabHighlightColor = lipgloss.Color("#74b9ff")
	tabInactiveColor  = lipgloss.Color("#6c7a89")

	activeTabStyle = lipgloss.NewStyle().
			Border(tabBorderWithBottom("┘", " ", "└"), true).
			BorderForeground(tabHighlightColor).
			Foreground(lipgloss.Color("#cdd6f4")).
			Bold(true).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Border(tabBorderWithBottom("┴", "─", "┴"), true).
				BorderForeground(tabInactiveColor).
				Foreground(tabInactiveColor).
				Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().Faint(true).Padding(0, 1)
)

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	b := lipgloss.RoundedBorder()
	b.BottomLeft = left
	b.Bottom = middle
	b.BottomRight = right
	return b
}

// AppModel is the root bubbletea model for liste -i.
// Exported so tests can inspect it.
type AppModel struct {
	tabs        []string
	activeTab   int
	viewMap     map[string]tea.Model
	overlay     *DetailModel
	editOverlay *EditModel
	blockInput  *blockInputModel
	store       *store.Store
	config      *model.Config
	tuiCfg      model.TUIConfig
	width       int
	height      int
	statusMsg   string
}

// blockInputModel handles the 'b' key — prompts for block reason inline.
type blockInputModel struct {
	input  textinput.Model
	itemID string
}

// viewLoadedMsg carries a lazily-initialized view back into the Update loop.
type viewLoadedMsg struct {
	name string
	view tea.Model
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
		config:    cfg,
		tuiCfg:    tuiCfg,
	}
}

// newApp creates a fully initialized AppModel backed by a real store.
func newApp(result *discovery.Result, rootCfg *model.Config) (AppModel, error) {
	m := NewAppForTest(rootCfg)
	m.store = store.New(result.Root)

	tuiCfg := rootCfg.TUI.Resolved()
	view, err := m.initView(tuiCfg.DefaultView, 80, 24)
	if err != nil {
		return AppModel{}, fmt.Errorf("initializing default view: %w", err)
	}
	m.viewMap[tuiCfg.DefaultView] = view
	return m, nil
}

// initView creates a view model for the given view name.
func (m AppModel) initView(name string, width, height int) (tea.Model, error) {
	contentH := height - tabBarHeight - 1 // tabs + status bar
	if contentH < 1 {
		contentH = 1
	}
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

func (m AppModel) currentViewName() string {
	if m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab]
	}
	return "list"
}

// Run starts the bubbletea program with mouse support enabled.
func Run(result *discovery.Result, rootCfg *model.Config) error {
	m, err := newApp(result, rootCfg)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
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
	if m.blockInput != nil {
		return m.updateBlockInput(msg)
	}

	// CloseDetailMsg closes whichever overlay is open.
	if _, ok := msg.(CloseDetailMsg); ok {
		m.overlay = nil
		m.editOverlay = nil
		return m, nil
	}

	// Mouse: tab bar clicks always take priority, even during editing.
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if mouse.Button == tea.MouseButtonLeft &&
			mouse.Action == tea.MouseActionRelease &&
			mouse.Y < tabBarHeight {
			x := 0
			for i := range m.tabs {
				rendered := m.renderTab(i)
				w := lipgloss.Width(rendered)
				if mouse.X >= x && mouse.X < x+w {
					if i != m.activeTab {
						m.activeTab = i
						m.editOverlay = nil // discard unsaved edit on tab switch
						return m, m.ensureViewLoaded()
					}
					return m, nil
				}
				x += w
			}
		}
	}

	if m.overlay != nil {
		return m.updateOverlay(msg)
	}

	if m.editOverlay != nil {
		return m.updateEditOverlay(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		for name, v := range m.viewMap {
			updated, _ := v.Update(msg)
			m.viewMap[name] = updated
		}
		return m, nil

	case tea.MouseMsg:
		return m.updateCurrentView(msg)

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

	case viewLoadedMsg:
		m.viewMap[msg.name] = msg.view
		return m, nil

	case views.ItemSelectedMsg:
		overlay := NewDetailModel(msg.Item, m.width, m.height)
		m.overlay = &overlay
		return m, nil

	case views.ItemEditMsg:
		m.overlay = nil
		if m.config != nil {
			edit := NewEditModel(msg.Item, m.config, m.width, m.height)
			m.editOverlay = &edit
			return m, edit.Init()
		}
		return m, nil

	case ItemSavedMsg:
		if err := m.store.WriteItem(msg.Item); err != nil {
			m.statusMsg = "Error: " + err.Error()
		} else {
			m.statusMsg = msg.Item.ID + " saved"
			m.editOverlay = nil
			m.reloadCurrentView()
		}
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
	}

	return m.updateCurrentView(msg)
}

func (m AppModel) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.overlay.Update(msg)
	if detail, ok := updated.(DetailModel); ok {
		m.overlay = &detail
	}
	return m, cmd
}

func (m AppModel) updateEditOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.editOverlay.Update(msg)
	if edit, ok := updated.(EditModel); ok {
		m.editOverlay = &edit
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
	if _, ok := m.viewMap[name]; ok {
		return nil
	}
	width, height := m.width, m.height
	appCopy := m
	return func() tea.Msg {
		view, err := appCopy.initView(name, width, height)
		if err != nil {
			return nil
		}
		return viewLoadedMsg{name: name, view: view}
	}
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

// renderTab returns the styled string for tab at index idx.
func (m AppModel) renderTab(idx int) string {
	isActive := idx == m.activeTab
	isFirst := idx == 0
	isLast := idx == len(m.tabs)-1
	name := strings.ToUpper(m.tabs[idx])

	var border lipgloss.Border
	if isActive {
		border = tabBorderWithBottom("┘", " ", "└")
	} else {
		border = tabBorderWithBottom("┴", "─", "┴")
	}
	if isFirst {
		if isActive {
			border.BottomLeft = "│"
		} else {
			border.BottomLeft = "├"
		}
	}
	if isLast {
		if isActive {
			border.BottomRight = "│"
		} else {
			border.BottomRight = "┤"
		}
	}

	if isActive {
		return activeTabStyle.Border(border).Render(name)
	}
	return inactiveTabStyle.Border(border).Render(name)
}

func (m AppModel) renderTabBar() string {
	var parts []string
	for i := range m.tabs {
		parts = append(parts, m.renderTab(i))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	gapWidth := m.width - lipgloss.Width(row)
	if gapWidth < 0 {
		gapWidth = 0
	}
	gap := lipgloss.NewStyle().Foreground(tabInactiveColor).Render(strings.Repeat("─", gapWidth))
	return lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.overlay != nil {
		return m.overlay.View()
	}

	if m.blockInput != nil {
		return m.blockInput.input.View()
	}

	tabBar := m.renderTabBar()

	if m.editOverlay != nil {
		statusBar := statusBarStyle.Render(m.statusMsg + "  tab: next field  ctrl+s: save  esc: cancel")
		return tabBar + "\n" + m.editOverlay.View() + "\n" + statusBar
	}

	viewStr := ""
	if v, ok := m.viewMap[m.currentViewName()]; ok {
		viewStr = v.View()
	}

	hint := "  tab/shift+tab: switch  d: done  b: block  e: edit  enter: detail  q: quit"
	statusBar := statusBarStyle.Render(m.statusMsg + hint)

	return tabBar + "\n" + viewStr + "\n" + statusBar
}
