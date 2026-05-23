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
	editHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4"))
	editLabelStyle  = lipgloss.NewStyle().Faint(true).Width(10)
	editHintStyle   = lipgloss.NewStyle().Faint(true)
	editErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
)

const (
	efTitle    = 0
	efStatus   = 1
	efPriority = 2
	efPhase    = 3
	efTags     = 4
	efBody     = 5
	efCount    = 6

	// editLabelWidth is the fixed width of field labels (matches editLabelStyle.Width).
	editLabelWidth = 10
	// editFieldStartX is the terminal X where textinput content begins (label + space).
	editFieldStartX = editLabelWidth + 1
	// editContentStartY is the terminal Y where edit content begins (after 3-line tab bar).
	editContentStartY = tabBarHeight
	// editFieldsStartY is the terminal Y of the first input field.
	// Layout: header(1) + blank(1) = 2 lines before fields.
	editFieldsStartY = editContentStartY + 2
	// editBodyY is the terminal Y where the textarea begins.
	// Layout: fields(5) + blank(1) + label(1) = 7 lines after fieldsStart.
	editBodyStartY = editFieldsStartY + efBody + 2
)

var editFieldLabels = [5]string{"Title:", "Status:", "Priority:", "Phase:", "Tags:"}

// ItemSavedMsg signals that the edit form saved an item.
type ItemSavedMsg struct{ Item *model.Item }

// EditCancelRequestMsg signals the user wants to leave the edit form (esc key).
// App handles it by showing a discard-confirmation modal.
type EditCancelRequestMsg struct{}

// EditModel renders in the content area (view space) for editing an item's fields.
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

// fieldAtY returns the field index for a click at terminal Y, or -1 if not a field.
func fieldAtY(termY int) int {
	switch {
	case termY >= editFieldsStartY && termY < editFieldsStartY+efBody:
		return termY - editFieldsStartY // 0-4
	case termY >= editBodyStartY:
		return efBody
	default:
		return -1
	}
}

func (m *EditModel) applySize(width, height int) {
	// Content area = height - tabBarHeight - 1 (status bar)
	inputW := width - editFieldStartX - 1
	if inputW < 10 {
		inputW = 10
	}
	for i := range m.inputs {
		m.inputs[i].Width = inputW
	}

	// body height = content area - header(1) - blank(1) - inputs(5) - blank(1) - label(1) - hint(1) - err(1)
	bodyH := height - tabBarHeight - 12
	if bodyH < 3 {
		bodyH = 3
	}
	bodyW := width - 1
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
			return m, func() tea.Msg { return EditCancelRequestMsg{} }
		case "ctrl+s":
			return m.save()
		case "tab":
			m.focusField((m.focused + 1) % efCount)
			return m, textinput.Blink
		case "shift+tab":
			m.focusField((m.focused - 1 + efCount) % efCount)
			return m, textinput.Blink
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease {
			if field := fieldAtY(msg.Y); field >= 0 {
				m.focusField(field)
				if field < efBody {
					clickPos := msg.X - editFieldStartX
					if clickPos < 0 {
						clickPos = 0
					}
					m.inputs[field].SetCursor(clickPos)
				} else {
					// Navigate textarea to clicked row then column.
					// textarea prompt = "┃ " = 2 chars wide; content starts at X=2.
					const promptW = 2
					targetRow := msg.Y - editBodyStartY
					if targetRow < 0 {
						targetRow = 0
					}
					delta := targetRow - m.body.Line()
					for i := 0; i < delta; i++ {
						m.body.CursorDown()
					}
					for i := 0; i > delta; i-- {
						m.body.CursorUp()
					}
					col := msg.X - promptW
					if col < 0 {
						col = 0
					}
					m.body.SetCursor(col)
				}
				return m, textinput.Blink
			}
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
		sb.WriteString(editLabelStyle.Render(label) + " " + m.inputs[i].View() + "\n")
	}

	sb.WriteString("\n" + editLabelStyle.Render("Body:") + "\n")
	sb.WriteString(m.body.View() + "\n")

	sb.WriteString(editHintStyle.Render("tab: next field · ctrl+s: save · esc: cancel"))

	if m.errMsg != "" {
		sb.WriteString("\n" + editErrorStyle.Render("✗ "+m.errMsg))
	}

	return sb.String()
}
