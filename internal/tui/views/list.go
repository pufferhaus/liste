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

// ItemEditMsg is sent when the user presses 'e' on an item.
type ItemEditMsg struct{ Item *model.Item }

// listItemHeight is the visual height of each list item (default delegate height=2 + spacing=1).
const listItemHeight = 3

// listHeaderLines is the number of lines consumed by the list header (title + status bar).
const listHeaderLines = 2

// handleListMouse processes mouse events for a list view backed by a list.Model.
// tabBarY is the terminal Y of the tab bar (0 for the top of the screen).
// Returns the updated list, an optional cmd, and whether the event was consumed.
func handleListMouse(l list.Model, msg tea.MouseMsg, tabBarY int) (list.Model, tea.Cmd, bool) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		l.CursorUp()
		return l, nil, true
	case tea.MouseButtonWheelDown:
		l.CursorDown()
		return l, nil, true
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionRelease {
			return l, nil, true
		}
		itemsStartY := tabBarY + 1 + listHeaderLines
		if msg.Y < itemsStartY {
			return l, nil, true
		}
		itemOnPage := (msg.Y - itemsStartY) / listItemHeight
		absIdx := l.Paginator.Page*l.Paginator.PerPage + itemOnPage
		items := l.Items()
		if absIdx >= 0 && absIdx < len(items) {
			l.Select(absIdx)
		}
		if li, ok := l.SelectedItem().(ListItem); ok {
			return l, func() tea.Msg { return ItemSelectedMsg(li) }, true
		}
		return l, nil, true
	}
	return l, nil, false
}

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
				return m, func() tea.Msg { return ItemSelectedMsg(li) }
			}
		case "d":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemDoneMsg{ID: li.Item.ID} }
			}
		case "b":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemBlockMsg{ID: li.Item.ID} }
			}
		case "e":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemEditMsg{Item: li.Item} }
			}
		}
	case tea.MouseMsg:
		updated, cmd, handled := handleListMouse(m.list, msg, 0)
		m.list = updated
		if handled {
			return m, cmd
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
