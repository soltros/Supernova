package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const baseURL = "http://localhost:8080/api"

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Album struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ArtistID string `json:"artist_id"`
	Year     int    `json:"release_year"`
}

type Track struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	TrackNumber int    `json:"track_number"`
	Duration    int    `json:"duration_ms"`
}

func fetchArtists() ([]Artist, error) {
	resp, err := http.Get(baseURL + "/artists?limit=100")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var artists []Artist
	json.Unmarshal(body, &artists)
	return artists, nil
}

func fetchAlbums(artistID string) ([]Album, error) {
	resp, err := http.Get(baseURL + "/albums?limit=100&artist_id=" + artistID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var albums []Album
	json.Unmarshal(body, &albums)
	return albums, nil
}

func fetchTracks(albumID string) ([]Track, error) {
	resp, err := http.Get(baseURL + "/tracks?limit=100&album_id=" + albumID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tracks []Track
	json.Unmarshal(body, &tracks)
	return tracks, nil
}

func formatTime(ms int) string {
	s := ms / 1000
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func main() {
	app := tview.NewApplication()

	artistsList := tview.NewList().ShowSecondaryText(false)
	artistsList.SetBorder(true).SetTitle("Artists (Press Tab to switch)").SetTitleColor(tcell.ColorYellow)

	albumsList := tview.NewList().ShowSecondaryText(false)
	albumsList.SetBorder(true).SetTitle("Albums").SetTitleColor(tcell.ColorGreen)

	tracksList := tview.NewList().ShowSecondaryText(false)
	tracksList.SetBorder(true).SetTitle("Tracks").SetTitleColor(tcell.ColorBlue)

	// Layout
	flex := tview.NewFlex().
		AddItem(artistsList, 0, 1, true).
		AddItem(albumsList, 0, 1, false).
		AddItem(tracksList, 0, 2, false)

	// Focus switching logic
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if artistsList.HasFocus() {
				app.SetFocus(albumsList)
			} else if albumsList.HasFocus() {
				app.SetFocus(tracksList)
			} else {
				app.SetFocus(artistsList)
			}
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			app.Stop()
		}
		return event
	})

	// Load Initial Data
	artists, err := fetchArtists()
	if err != nil {
		fmt.Println("Error connecting to Supernova backend:", err)
		os.Exit(1)
	}

	for _, artist := range artists {
		artistsList.AddItem(artist.Name, artist.ID, 0, nil)
	}

	// Handlers
	artistsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		albumsList.Clear()
		tracksList.Clear()
		if secondaryText == "" {
			return
		}
		albums, _ := fetchAlbums(secondaryText)
		for _, album := range albums {
			title := fmt.Sprintf("%s (%d)", album.Title, album.Year)
			if album.Year == 0 {
				title = album.Title
			}
			albumsList.AddItem(title, album.ID, 0, nil)
		}
	})

	albumsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tracksList.Clear()
		if secondaryText == "" {
			return
		}
		tracks, _ := fetchTracks(secondaryText)
		for _, track := range tracks {
			title := fmt.Sprintf("%d. %s [%s]", track.TrackNumber, track.Title, formatTime(track.Duration))
			tracksList.AddItem(title, track.ID, 0, nil)
		}
	})

	tracksList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		// Just a modal for demonstration
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Playing:\n\n%s\n\nStream URL: %s/stream/%s", mainText, baseURL, secondaryText)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.SetRoot(flex, true).SetFocus(tracksList)
			})
		app.SetRoot(modal, false).SetFocus(modal)
	})

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
