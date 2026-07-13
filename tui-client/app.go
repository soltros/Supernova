package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/soltros/Supernova/tui-client/api"
	"github.com/soltros/Supernova/tui-client/models"
	"github.com/soltros/Supernova/tui-client/player"
)

type focusState int

const (
	focusArtists focusState = iota
	focusAlbums
	focusTracks
	focusQueue
)

type appModel struct {
	artistsList list.Model
	albumsList  list.Model
	tracksList  list.Model
	queueList   list.Model
	focus       focusState
	width       int
	height      int
	status      string
}

type trackItem struct{ models.Track }
func (i trackItem) Title() string       { return i.Track.Title }
func (i trackItem) Description() string { return fmt.Sprintf("Duration: %d", i.Track.Duration) }
func (i trackItem) FilterValue() string { return i.Track.Title }

type artistItem struct{ models.Artist }
func (i artistItem) Title() string       { return i.Artist.Name }
func (i artistItem) Description() string { return "" }
func (i artistItem) FilterValue() string { return i.Artist.Name }

type albumItem struct{ models.Album }
func (i albumItem) Title() string       { return i.Album.Title }
func (i albumItem) Description() string { return fmt.Sprintf("%d", i.Album.Year) }
func (i albumItem) FilterValue() string { return i.Album.Title }

func initialAppModel() appModel {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(primaryColor).BorderLeftForeground(primaryColor)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(primaryColor).BorderLeftForeground(primaryColor)
	d.ShowDescription = true

	al := list.New([]list.Item{}, d, 0, 0)
	al.Title = "Artists"
	al.SetShowStatusBar(false)

	bl := list.New([]list.Item{}, d, 0, 0)
	bl.Title = "Albums"
	bl.SetShowStatusBar(false)

	tl := list.New([]list.Item{}, d, 0, 0)
	tl.Title = "Tracks"
	tl.SetShowStatusBar(false)

	ql := list.New([]list.Item{}, d, 0, 0)
	ql.Title = "Queue"
	ql.SetShowStatusBar(false)

	return appModel{
		artistsList: al,
		albumsList:  bl,
		tracksList:  tl,
		queueList:   ql,
		focus:       focusArtists,
		status:      "Ready.",
	}
}

// Commands
type artistsLoadedMsg []list.Item
type albumsLoadedMsg []list.Item
type tracksLoadedMsg []list.Item
type errMsg struct{ err error }
type StateChangeMsg struct {
	State player.State
	Track *models.Track
}

func fetchArtists() tea.Cmd {
	return func() tea.Msg {
		var artists []models.Artist
		if err := api.Fetch("/artists?limit=1000", &artists); err == nil {
			items := make([]list.Item, len(artists))
			for i, a := range artists { items[i] = artistItem{a} }
			return artistsLoadedMsg(items)
		} else {
			return errMsg{err}
		}
	}
}

func fetchAlbums(artistID string) tea.Cmd {
	return func() tea.Msg {
		var albums []models.Album
		if err := api.Fetch("/albums?limit=1000&artist_id="+artistID, &albums); err == nil {
			items := make([]list.Item, len(albums))
			for i, a := range albums { items[i] = albumItem{a} }
			return albumsLoadedMsg(items)
		} else {
			return errMsg{err}
		}
	}
}

func fetchTracks(albumID string) tea.Cmd {
	return func() tea.Msg {
		var tracks []models.Track
		if err := api.Fetch("/tracks?limit=1000&album_id="+albumID, &tracks); err == nil {
			items := make([]list.Item, len(tracks))
			for i, t := range tracks { items[i] = trackItem{t} }
			return tracksLoadedMsg(items)
		} else {
			return errMsg{err}
		}
	}
}

func (m appModel) Init() tea.Cmd {
	return fetchArtists()
}

func (m appModel) Update(msg tea.Msg) (appModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		paneW := (m.width - 8) / 4
		paneH := m.height - 4
		m.artistsList.SetSize(paneW, paneH)
		m.albumsList.SetSize(paneW, paneH)
		m.tracksList.SetSize(paneW, paneH)
		m.queueList.SetSize(paneW, paneH)

	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg.err)
		
	case StateChangeMsg:
		// Trigger a re-render to update the status bar
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % 4
		case "shift+tab":
			m.focus--
			if m.focus < 0 { m.focus = 3 }
		case "enter":
			if m.focus == focusTracks {
				if i, ok := m.tracksList.SelectedItem().(trackItem); ok {
					player.AddToQueue(i.Track)
					m.queueList.InsertItem(len(m.queueList.Items()), i)
					if player.GetCurrentState() == player.Stopped {
						player.PlayTrack(player.GetQueueLength() - 1)
					}
					m.status = fmt.Sprintf("Enqueued: %s", i.Track.Title)
				}
			}
		}

	case artistsLoadedMsg:
		cmds = append(cmds, m.artistsList.SetItems(msg))
		if len(msg) > 0 {
			if item, ok := msg[0].(artistItem); ok {
				cmds = append(cmds, fetchAlbums(item.ID))
			}
		}
	case albumsLoadedMsg:
		cmds = append(cmds, m.albumsList.SetItems(msg))
		if len(msg) > 0 {
			if item, ok := msg[0].(albumItem); ok {
				cmds = append(cmds, fetchTracks(item.ID))
			}
		}
	case tracksLoadedMsg:
		cmds = append(cmds, m.tracksList.SetItems(msg))
	}

	var cmd tea.Cmd
	switch m.focus {
	case focusArtists:
		m.artistsList, cmd = m.artistsList.Update(msg)
		if _, ok := msg.(tea.KeyMsg); ok && m.artistsList.SelectedItem() != nil {
			if item, ok := m.artistsList.SelectedItem().(artistItem); ok {
				cmds = append(cmds, fetchAlbums(item.ID))
			}
		}
	case focusAlbums:
		m.albumsList, cmd = m.albumsList.Update(msg)
		if _, ok := msg.(tea.KeyMsg); ok && m.albumsList.SelectedItem() != nil {
			if item, ok := m.albumsList.SelectedItem().(albumItem); ok {
				cmds = append(cmds, fetchTracks(item.ID))
			}
		}
	case focusTracks:
		m.tracksList, cmd = m.tracksList.Update(msg)
	case focusQueue:
		m.queueList, cmd = m.queueList.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	paneW := (m.width - 8) / 4
	paneH := m.height - 4

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(paneW).
		Height(paneH)
	
	styleList := func(l list.Model, active bool) string {
		s := paneStyle.Copy()
		if active {
			s = s.BorderForeground(primaryColor)
		} else {
			s = s.BorderForeground(lipgloss.Color("240"))
		}
		return s.Render(l.View())
	}

	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styleList(m.artistsList, m.focus == focusArtists),
		styleList(m.albumsList, m.focus == focusAlbums),
		styleList(m.tracksList, m.focus == focusTracks),
		styleList(m.queueList, m.focus == focusQueue),
	)
	
	statusStr := ""
	if player.GetCurrentState() == player.Playing {
		statusStr = lipgloss.NewStyle().Foreground(secondaryColor).Render("▶ Playing")
	} else {
		statusStr = "⏹ Stopped"
	}

	statusBar := lipgloss.NewStyle().
		Margin(1, 0, 0, 2).
		Render(fmt.Sprintf("%s | %s", statusStr, m.status))

	return panes + "\n" + statusBar
}
