package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const baseURL = "http://localhost:8080/api"

var jwtToken string
var currentFFPlay *exec.Cmd

// Models
type AuthResponse struct {
	Token string `json:"token"`
}
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

// API functions
func login(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", resp.Status)
	}
	var authResp AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	jwtToken = authResp.Token
	return nil
}

func fetchAPI(endpoint string, target interface{}) error {
	req, _ := http.NewRequest("GET", baseURL+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func formatTime(ms int) string {
	s := ms / 1000
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func main() {
	myApp := app.NewWithID("com.soltros.supernova")
	w := myApp.NewWindow("Supernova Desktop")
	w.Resize(fyne.NewSize(1000, 600))

	var buildAppUI func() *fyne.Container

	// ---- LOGIN UI ----
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	loginBtn := widget.NewButton("Login", func() {
		err := login(usernameEntry.Text, passwordEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		w.SetContent(buildAppUI())
	})
	loginBtn.Importance = widget.HighImportance

	loginForm := container.NewVBox(
		widget.NewLabelWithStyle("🚀 Supernova", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		usernameEntry,
		passwordEntry,
		loginBtn,
	)
	loginContainer := container.NewCenter(loginForm)
	w.SetContent(loginContainer)

	// ---- APP UI ----
	var artists []Artist
	var albums []Album
	var tracks []Track

	artistsList := widget.NewList(
		func() int { return len(artists) },
		func() fyne.CanvasObject { return widget.NewLabel("Artist Name") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(artists[i].Name) },
	)

	albumsList := widget.NewList(
		func() int { return len(albums) },
		func() fyne.CanvasObject { return widget.NewLabel("Album Title") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("%s (%d)", albums[i].Title, albums[i].Year))
		},
	)

	tracksList := widget.NewList(
		func() int { return len(tracks) },
		func() fyne.CanvasObject { return widget.NewLabel("Track Title") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("%d. %s [%s]", tracks[i].TrackNumber, tracks[i].Title, formatTime(tracks[i].Duration)))
		},
	)

	// Playback Controls
	statusLabel := widget.NewLabel("Ready.")
	
	stopPlayback := func() {
		if currentFFPlay != nil && currentFFPlay.Process != nil {
			currentFFPlay.Process.Kill()
		}
		statusLabel.SetText("⏹ Stopped")
	}

	playTrack := func(track Track) {
		stopPlayback()
		streamURL := fmt.Sprintf("%s/stream/%s", baseURL, track.ID)
		headerArg := fmt.Sprintf("Authorization: Bearer %s", jwtToken)
		currentFFPlay = exec.Command("ffplay", "-headers", headerArg, "-nodisp", "-autoexit", streamURL)
		currentFFPlay.Start()
		statusLabel.SetText(fmt.Sprintf("▶ Playing: %s", track.Title))
	}

	playBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		// Toggle pause not directly supported in headless ffplay without IPC, so we just log
		statusLabel.SetText("Play/Pause clicked")
	})
	stopBtn := widget.NewButtonWithIcon("", theme.MediaStopIcon(), func() { stopPlayback() })
	
	controls := container.NewHBox(layout.NewSpacer(), playBtn, stopBtn, statusLabel, layout.NewSpacer())

	// Wiring List Selections
	artistsList.OnSelected = func(id widget.ListItemID) {
		artistID := artists[id].ID
		fetchAPI("/albums?limit=1000&artist_id="+artistID, &albums)
		albumsList.Refresh()
		tracks = nil
		tracksList.Refresh()
	}

	albumsList.OnSelected = func(id widget.ListItemID) {
		albumID := albums[id].ID
		fetchAPI("/tracks?limit=1000&album_id="+albumID, &tracks)
		tracksList.Refresh()
	}

	tracksList.OnSelected = func(id widget.ListItemID) {
		playTrack(tracks[id])
	}

	buildAppUI = func() *fyne.Container {
		// Load initial artists
		fetchAPI("/artists?limit=1000", &artists)
		artistsList.Refresh()

		splitAlbumsTracks := container.NewHSplit(albumsList, tracksList)
		splitMain := container.NewHSplit(artistsList, splitAlbumsTracks)
		splitMain.SetOffset(0.25)
		splitAlbumsTracks.SetOffset(0.33)

		return container.NewBorder(nil, controls, nil, nil, splitMain)
	}

	w.ShowAndRun()
	
	// Cleanup on exit
	stopPlayback()
}
