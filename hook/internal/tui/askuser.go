package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/internal/server"
)

type askUserModel struct {
	req        *pb.PermissionRequest
	input      server.AskUserQuestionInput
	currentQ   int
	answers    map[string]string
	cursor     int
	selected   map[int]bool // for multi-select
	customMode bool
	completed  bool
	textInput  textinput.Model
	width      int
	height     int
}

func newAskUserModel(req *pb.PermissionRequest, input server.AskUserQuestionInput, width, height int) askUserModel {
	ti := textinput.New()
	ti.Placeholder = "Type your answer..."
	ti.CharLimit = 512
	ti.Width = 50

	return askUserModel{
		req:      req,
		input:    input,
		answers:  make(map[string]string),
		selected: make(map[int]bool),
		textInput: ti,
		width:    width,
		height:   height,
	}
}

func (m askUserModel) currentQuestion() server.AskQuestion {
	return m.input.Questions[m.currentQ]
}

// optionCount returns number of options plus the "Other" entry.
func (m askUserModel) optionCount() int {
	return len(m.currentQuestion().Options) + 1
}

func (m askUserModel) Update(msg tea.Msg) (askUserModel, tea.Cmd) {
	if m.completed {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.customMode {
			return m.updateCustomInput(msg)
		}
		return m.updateOptionSelection(msg)
	}
	return m, nil
}

func (m askUserModel) updateOptionSelection(msg tea.KeyMsg) (askUserModel, tea.Cmd) {
	q := m.currentQuestion()

	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < m.optionCount()-1 {
			m.cursor++
		}
	case tea.KeyRunes:
		switch msg.String() {
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j":
			if m.cursor < m.optionCount()-1 {
				m.cursor++
			}
		case " ":
			if q.MultiSelect {
				// Toggle selection
				if m.selected[m.cursor] {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = true
				}
			}
		}
	case tea.KeyEnter:
		return m.confirmSelection()
	}

	return m, nil
}

func (m askUserModel) confirmSelection() (askUserModel, tea.Cmd) {
	q := m.currentQuestion()
	otherIdx := len(q.Options)

	if q.MultiSelect {
		// If "Other" is selected, enter custom mode
		if m.selected[otherIdx] {
			delete(m.selected, otherIdx)
			m.customMode = true
			m.textInput.Focus()
			return m, textinput.Blink
		}

		// Collect selected labels
		var labels []string
		for i, opt := range q.Options {
			if m.selected[i] {
				labels = append(labels, opt.Label)
			}
		}
		if len(labels) == 0 {
			// Nothing selected, treat as "Other"
			m.customMode = true
			m.textInput.Focus()
			return m, textinput.Blink
		}
		m.answers[q.Question] = strings.Join(labels, ", ")
	} else {
		// Single select
		if m.cursor == otherIdx {
			m.customMode = true
			m.textInput.Focus()
			return m, textinput.Blink
		}
		m.answers[q.Question] = q.Options[m.cursor].Label
	}

	return m.advanceQuestion()
}

func (m askUserModel) updateCustomInput(msg tea.KeyMsg) (askUserModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		q := m.currentQuestion()
		m.answers[q.Question] = m.textInput.Value()
		m.customMode = false
		m.textInput.Blur()
		m.textInput.Reset()
		return m.advanceQuestion()
	case tea.KeyEsc:
		m.customMode = false
		m.textInput.Blur()
		m.textInput.Reset()
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m askUserModel) advanceQuestion() (askUserModel, tea.Cmd) {
	m.currentQ++
	m.cursor = 0
	m.selected = make(map[int]bool)

	if m.currentQ >= len(m.input.Questions) {
		m.completed = true
		// All questions answered, build response
		resp, err := server.BuildAskUserResponse(m.req, m.input, m.answers)
		if err != nil {
			return m, func() tea.Msg {
				return promptCompleteMsg{err: err}
			}
		}
		return m, func() tea.Msg {
			return promptCompleteMsg{response: resp}
		}
	}

	return m, nil
}

func (m askUserModel) View() string {
	if m.completed {
		return headerStyle.Render("AskUserQuestion") + "\n\n" +
			progressStyle.Render("  Completing...") + "\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("AskUserQuestion"))
	b.WriteString("\n\n")

	// Progress
	b.WriteString(progressStyle.Render(fmt.Sprintf("  Question %d of %d", m.currentQ+1, len(m.input.Questions))))
	b.WriteString("\n\n")

	q := m.currentQuestion()

	// Question header + text
	b.WriteString(fmt.Sprintf("  %s %s\n\n", questionHeaderStyle.Render(fmt.Sprintf("[%s]", q.Header)), q.Question))

	if m.customMode {
		b.WriteString("  Type your answer:\n\n")
		b.WriteString("  " + m.textInput.View())
		b.WriteString("\n")
		return b.String()
	}

	// Options
	for i, opt := range q.Options {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		label := opt.Label
		if opt.Description != "" {
			label = fmt.Sprintf("%s - %s", opt.Label, opt.Description)
		}

		// Show checkmark for multi-select
		check := ""
		if q.MultiSelect && m.selected[i] {
			check = checkStyle.Render(" [x]")
		} else if q.MultiSelect {
			check = " [ ]"
		}

		if m.cursor == i {
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, selectedStyle.Render(label), check))
		} else {
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, unselectedStyle.Render(label), check))
		}
	}

	// "Other" option
	otherIdx := len(q.Options)
	cursor := "  "
	if m.cursor == otherIdx {
		cursor = cursorStyle.Render("> ")
	}
	otherLabel := "Other (type custom answer)"
	check := ""
	if q.MultiSelect && m.selected[otherIdx] {
		check = checkStyle.Render(" [x]")
	} else if q.MultiSelect {
		check = " [ ]"
	}
	if m.cursor == otherIdx {
		b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, selectedStyle.Render(otherLabel), check))
	} else {
		b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, unselectedStyle.Render(otherLabel), check))
	}

	// Help text
	b.WriteString("\n")
	if q.MultiSelect {
		b.WriteString(unselectedStyle.Render("  space: toggle  enter: confirm  j/k: navigate"))
	} else {
		b.WriteString(unselectedStyle.Render("  enter: select  j/k: navigate"))
	}
	b.WriteString("\n")

	return b.String()
}
