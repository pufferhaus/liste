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
