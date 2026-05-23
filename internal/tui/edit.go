package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pufferhaus/liste/internal/model"
)

var (
	editBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#74b9ff")).
			Padding(0, 1)
	editHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4"))
	editLabelStyle  = lipgloss.NewStyle().Faint(true).Width(10)
	editHintStyle   = lipgloss.NewStyle().Faint(true)
	editErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	editBodyLabel   = lipgloss.NewStyle().Faint(true)
)

const (
	efTitle    = 0
	efStatus   = 1
	efPriority = 2
	efPhase    = 3
	efTags     = 4
	efBody     = 5
	efCount    = 6
)

// ItemSavedMsg signals that the edit overlay saved an item.
type ItemSavedMsg struct{ Item *model.Item }

// EditModel is an overlay for editing all fields of an item.
type EditModel struct {
	item    *model.Item
	inputs  [5]textinput.Model
	body    textarea.Model
	focused int
	width   int
	height  int
	errMsg  string
	config  *model.Config
}

var editFieldLabels = [5]string{"Title:", "Status:", "Priority:", "Phase:", "Tags:"}

// NewEditModel creates an EditModel pre-populated with the item's current values.
func NewEditModel(item *model.Item, cfg *model.Config, width, height int) EditModel {
	var inputs [5]textinput.Model
	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 200
		inputs[i] = ti
	}
	inputs[efTitle].SetValue(item.Title)
	inputs[efStatus].SetValue(item.Status)
	inputs[efPriority].SetValue(item.Priority)
	if item.Phase != nil {
		inputs[efPhase].SetValue(strconv.Itoa(*item.Phase))
	}
	inputs[efTags].SetValue(strings.Join(item.Tags, ", "))
	inputs[efTitle].Placeholder = "item title"
	inputs[efStatus].Placeholder = strings.Join(cfg.Statuses, " | ")
	inputs[efPriority].Placeholder = strings.Join(cfg.Priorities, " | ")
	inputs[efPhase].Placeholder = "number (blank = none)"
	inputs[efTags].Placeholder = "comma-separated"

	ta := textarea.New()
	ta.SetValue(item.Body)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	m := EditModel{
		item:   item,
		inputs: inputs,
		body:   ta,
		config: cfg,
		width:  width,
		height: height,
	}
	m.applySize(width, height)
	m.focusField(efTitle)
	return m
}

func (m *EditModel) applySize(width, height int) {
	innerW := width - 6 // border (2) + padding (2) + label (10) + gap (1) - overcount
	if innerW < 10 {
		innerW = 10
	}
	inputW := innerW - 11 // label=10 + space=1
	if inputW < 10 {
		inputW = 10
	}
	for i := range m.inputs {
		m.inputs[i].Width = inputW
	}

	bodyH := height - 18 // border(2)+padding(2)+header(1)+blank(1)+inputs(5)+blank(1)+label(1)+blank(1)+hint(1)
	if bodyH < 3 {
		bodyH = 3
	}
	bodyW := width - 8
	if bodyW < 10 {
		bodyW = 10
	}
	m.body.SetWidth(bodyW)
	m.body.SetHeight(bodyH)
}

func (m *EditModel) focusField(idx int) {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.body.Blur()
	m.focused = idx
	if idx < efBody {
		m.inputs[idx].Focus()
	} else {
		m.body.Focus()
	}
}

func (m EditModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m EditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseDetailMsg{} }
		case "ctrl+s":
			return m.save()
		case "tab":
			next := (m.focused + 1) % efCount
			m.focusField(next)
			return m, textinput.Blink
		case "shift+tab":
			prev := (m.focused - 1 + efCount) % efCount
			m.focusField(prev)
			return m, textinput.Blink
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applySize(msg.Width, msg.Height)
		return m, nil
	}

	// Route to focused field.
	if m.focused < efBody {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(msg)
	return m, cmd
}

func (m EditModel) save() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.inputs[efTitle].Value())
	if title == "" {
		m.errMsg = "title cannot be empty"
		return m, nil
	}

	status := strings.TrimSpace(m.inputs[efStatus].Value())
	if !m.config.IsValidStatus(status) {
		m.errMsg = fmt.Sprintf("invalid status %q (valid: %s)", status, strings.Join(m.config.Statuses, ", "))
		return m, nil
	}

	priority := strings.TrimSpace(m.inputs[efPriority].Value())
	if !m.config.IsValidPriority(priority) {
		m.errMsg = fmt.Sprintf("invalid priority %q (valid: %s)", priority, strings.Join(m.config.Priorities, ", "))
		return m, nil
	}

	phaseStr := strings.TrimSpace(m.inputs[efPhase].Value())
	var phase *int
	if phaseStr != "" {
		n, err := strconv.Atoi(phaseStr)
		if err != nil {
			m.errMsg = "phase must be a whole number"
			return m, nil
		}
		phase = &n
	}

	var tags []string
	for _, t := range strings.Split(m.inputs[efTags].Value(), ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	updated := *m.item
	updated.Title = title
	updated.Status = status
	updated.Priority = priority
	updated.Phase = phase
	updated.Tags = tags
	updated.Body = m.body.Value()
	updated.Updated = time.Now()

	return m, func() tea.Msg { return ItemSavedMsg{Item: &updated} }
}

func (m EditModel) View() string {
	var sb strings.Builder

	sb.WriteString(editHeaderStyle.Render("Edit  "+m.item.ID) + "\n\n")

	for i, label := range editFieldLabels {
		labelStr := editLabelStyle.Render(label)
		var inputStr string
		if i == m.focused {
			inputStr = m.inputs[i].View()
		} else {
			inputStr = m.inputs[i].View()
		}
		sb.WriteString(labelStr + " " + inputStr + "\n")
	}

	sb.WriteString("\n" + editBodyLabel.Render("Body:") + "\n")
	sb.WriteString(m.body.View() + "\n")

	if m.errMsg != "" {
		sb.WriteString(editErrorStyle.Render("✗ "+m.errMsg) + "\n")
	}

	sb.WriteString(editHintStyle.Render("tab: next field · ctrl+s: save · esc: cancel"))

	return editBorderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(sb.String())
}
