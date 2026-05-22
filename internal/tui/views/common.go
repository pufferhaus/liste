package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/pblca/liste/internal/model"
	"github.com/pblca/liste/internal/output"
)

// ListItem wraps model.Item to implement bubbles list.Item.
type ListItem struct {
	Item *model.Item
}

func (i ListItem) Title() string {
	blocked := i.Item.Blocked != nil
	return fmt.Sprintf("%-10s  %s  %s  %s",
		i.Item.ID,
		output.RenderType(string(i.Item.Type)),
		output.RenderStatus(i.Item.Status, blocked),
		output.RenderPriority(i.Item.Priority),
	)
}

func (i ListItem) Description() string {
	phase := ""
	if i.Item.Phase != nil {
		phase = fmt.Sprintf("  phase %d", *i.Item.Phase)
	}
	tags := ""
	if len(i.Item.Tags) > 0 {
		tags = "  #" + joinTags(i.Item.Tags)
	}
	return lipgloss.NewStyle().Faint(true).Render(i.Item.Title + phase + tags)
}

func (i ListItem) FilterValue() string {
	return i.Item.Title + " " + i.Item.ID
}

func joinTags(tags []string) string {
	var sb strings.Builder
	for i, t := range tags {
		if i > 0 {
			sb.WriteString(" #")
		}
		sb.WriteString(t)
	}
	return sb.String()
}

// ItemsToListItems converts model items to bubbles list items.
func ItemsToListItems(items []*model.Item) []list.Item {
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = ListItem{Item: item}
	}
	return out
}
