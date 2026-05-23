package views

import (
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

// NextView shows the priority-sorted queue of items ready to work on.
type NextView struct {
	list  list.Model
	store *store.Store
}

// NewNextView creates the next-queue view.
func NewNextView(s *store.Store, width, height int) (NextView, error) {
	items, err := nextItems(s)
	if err != nil {
		return NextView{}, err
	}
	l := newBubblesList("Next Up", ItemsToListItems(items), width, height)
	return NextView{list: l, store: s}, nil
}

func nextItems(s *store.Store) ([]*model.Item, error) {
	all, err := s.ListItems()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*model.Item, len(all))
	for _, item := range all {
		byID[item.ID] = item
	}

	var candidates []*model.Item
	for _, item := range all {
		if item.Status == "done" || item.Status == "cancelled" || item.Status == "active" {
			continue
		}
		if item.Blocked != nil {
			continue
		}
		if !depsResolved(item, byID) {
			continue
		}
		candidates = append(candidates, item)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		ap, bp := phaseOrder(a), phaseOrder(b)
		if ap != bp {
			return ap < bp
		}
		if priorityWeight(a.Priority) != priorityWeight(b.Priority) {
			return priorityWeight(a.Priority) < priorityWeight(b.Priority)
		}
		return a.Created.Before(b.Created)
	})
	return candidates, nil
}

func depsResolved(item *model.Item, byID map[string]*model.Item) bool {
	for _, link := range item.Links {
		if link.Type != model.LinkDependsOn {
			continue
		}
		target, ok := byID[link.Target]
		if !ok || (target.Status != "done" && target.Status != "cancelled") {
			return false
		}
	}
	return true
}

func phaseOrder(item *model.Item) int {
	if item.Phase == nil {
		return 9999
	}
	return *item.Phase
}

func priorityWeight(p string) int {
	switch p {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

func (m NextView) Init() tea.Cmd { return nil }

func (m NextView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemSelectedMsg(li) }
			}
		case "e":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemEditMsg{Item: li.Item} }
			}
		}
	case tea.MouseMsg:
		updated, cmd, handled := handleListMouse(m.list, msg, 3)
		m.list = updated
		if handled {
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m NextView) View() string { return m.list.View() }

// Reload refreshes from store.
func (m *NextView) Reload() error {
	items, err := nextItems(m.store)
	if err != nil {
		return err
	}
	m.list.SetItems(ItemsToListItems(items))
	return nil
}
