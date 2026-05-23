package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
	"github.com/pufferhaus/liste/internal/store"
)

var searchInputStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#74b9ff")).
	Padding(0, 1)

// SearchView provides a text input that filters items in real time.
type SearchView struct {
	input     textinput.Model
	list      list.Model
	allItems  []*model.Item
	store     *store.Store
	width     int
	height    int
	lastQuery string
}

// NewSearchView creates the search view.
func NewSearchView(s *store.Store, width, height int) (SearchView, error) {
	items, err := s.ListItems()
	if err != nil {
		return SearchView{}, err
	}

	ti := textinput.New()
	ti.Placeholder = "Search items..."
	ti.Focus()
	ti.CharLimit = 100

	l := newBubblesList("Results", ItemsToListItems(items), width, height-5)
	l.SetShowTitle(false)

	return SearchView{
		input:    ti,
		list:     l,
		allItems: items,
		store:    s,
		width:    width,
		height:   height,
	}, nil
}

func (m SearchView) Init() tea.Cmd { return textinput.Blink }

func (m SearchView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.input.SetValue("")
			m.list.SetItems(ItemsToListItems(m.allItems))
			return m, nil
		case "enter":
			if li, ok := m.list.SelectedItem().(ListItem); ok {
				return m, func() tea.Msg { return ItemSelectedMsg(li) }
			}
		}
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.list.CursorUp()
			return m, nil
		case tea.MouseButtonWheelDown:
			m.list.CursorDown()
			return m, nil
		case tea.MouseButtonLeft:
			if msg.Action != tea.MouseActionRelease {
				return m, nil
			}
			// Layout (terminal Y, after 3-row tab bar):
			//   Y=3: search box border top
			//   Y=4: search input content
			//   Y=5: search box border bottom
			//   Y=6: blank ("\n" separator)
			//   Y=7,8: list header (filter bar + status bar)
			//   Y=9+: list items (listItemHeight=3 each)
			const (
				searchInputY    = 4
				searchListStart = 9
			)
			switch {
			case msg.Y == searchInputY:
				m.input.Focus()
				// border(1)+padding(1)=2 chars before input content
				col := msg.X - 2
				if col < 0 {
					col = 0
				}
				m.input.SetCursor(col)
				return m, textinput.Blink
			case msg.Y >= searchListStart:
				itemOnPage := (msg.Y - searchListStart) / listItemHeight
				absIdx := m.list.Paginator.Page*m.list.Paginator.PerPage + itemOnPage
				items := m.list.Items()
				if absIdx >= 0 && absIdx < len(items) {
					m.list.Select(absIdx)
					if li, ok := m.list.SelectedItem().(ListItem); ok {
						return m, func() tea.Msg { return ItemSelectedMsg(li) }
					}
				}
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-7)
	}

	var cmds []tea.Cmd

	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)

	query := strings.ToLower(m.input.Value())
	if query != m.lastQuery {
		m.lastQuery = query
		if query != "" {
			var filtered []*model.Item
			for _, item := range m.allItems {
				if strings.Contains(strings.ToLower(item.Title), query) ||
					strings.Contains(strings.ToLower(item.ID), query) ||
					containsTag(item.Tags, query) {
					filtered = append(filtered, item)
				}
			}
			m.list.SetItems(ItemsToListItems(filtered))
		} else {
			m.list.SetItems(ItemsToListItems(m.allItems))
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	cmds = append(cmds, listCmd)

	return m, tea.Batch(cmds...)
}

func (m SearchView) View() string {
	return searchInputStyle.Width(m.width - 4).Render(m.input.View()) +
		"\n" +
		m.list.View()
}

// Reload refreshes from store.
func (m *SearchView) Reload() error {
	items, err := m.store.ListItems()
	if err != nil {
		return err
	}
	m.allItems = items
	m.list.SetItems(ItemsToListItems(items))
	return nil
}

func containsTag(tags []string, query string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), query) {
			return true
		}
	}
	return false
}
