package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/soltros/Supernova/tui-client/models"
	"github.com/soltros/Supernova/tui-client/player"
)

var (
	primaryColor   = lipgloss.Color("#9d4edd")
	secondaryColor = lipgloss.Color("#ff006e")
	accentColor    = lipgloss.Color("#00f5d4")

	baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor)

	titleStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Margin(1, 0, 1, 0)
)

type rootModel struct {
	isLoggedIn bool
	login      loginModel
	app        appModel
}

func initialModel() rootModel {
	return rootModel{
		login: initialLoginModel(),
		app:   initialAppModel(),
	}
}

func (m rootModel) Init() tea.Cmd {
	return tea.Batch(m.login.Init(), m.app.Init())
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.isLoggedIn {
		lm, cmd := m.login.Update(msg)
		if lm.isLoggedIn {
			m.isLoggedIn = true
			player.InitMPRIS()
			player.OnStateChange = func(state player.State, track *models.Track) {
				// We won't try to send async msgs to bubbletea right now for state changes
				// just rely on standard updates or tick
			}
			return m, m.app.Init() // start app data fetch
		}
		m.login = lm
		return m, cmd
	}

	am, cmd := m.app.Update(msg)
	m.app = am
	return m, cmd
}

func (m rootModel) View() string {
	if !m.isLoggedIn {
		return m.login.View()
	}
	return m.app.View()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
