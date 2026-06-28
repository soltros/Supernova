package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/soltros/Supernova/tui-client/api"
)

type loginModel struct {
	inputs     []textinput.Model
	focusIndex int
	err        error
	isLoggedIn bool
}

func initialLoginModel() loginModel {
	m := loginModel{
		inputs: make([]textinput.Model, 3),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(accentColor)
		t.CharLimit = 64

		switch i {
		case 0:
			t.Placeholder = "http://localhost:8080/api"
			t.Prompt = "Instance URL: "
			t.SetValue("http://localhost:8080/api")
			t.Focus()
			t.PromptStyle = lipgloss.NewStyle().Foreground(primaryColor)
			t.TextStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
		case 1:
			t.Placeholder = "Username"
			t.Prompt = "Username: "
		case 2:
			t.Placeholder = "Password"
			t.Prompt = "Password: "
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}

		m.inputs[i] = t
	}

	return m
}

func (m loginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()
			
			if s == "enter" && m.focusIndex == len(m.inputs) {
				instance := m.inputs[0].Value()
				if instance != "" {
					api.BaseURL = instance
				}
				username := m.inputs[1].Value()
				password := m.inputs[2].Value()
				if err := api.Login(username, password); err != nil {
					m.err = err
					return m, nil
				}
				m.isLoggedIn = true
				return m, nil
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(primaryColor)
					m.inputs[i].TextStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
				m.inputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *loginModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m loginModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🚀 Supernova") + "\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := "[ Login ]"
	if m.focusIndex == len(m.inputs) {
		button = lipgloss.NewStyle().Foreground(primaryColor).Render("[ Login ]")
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", button)

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(secondaryColor).Render(m.err.Error()) + "\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 4).
		Render(b.String())
	
	return lipgloss.Place(80, 20, lipgloss.Center, lipgloss.Center, box)
}
