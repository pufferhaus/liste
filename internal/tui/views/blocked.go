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
