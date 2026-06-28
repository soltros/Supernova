package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
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
type HeartDetails struct {
	Tracks []Track `json:"tracks"`
}
type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// API functions
func login(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("login failed: %s", resp.Status) }
	var authResp AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	jwtToken = authResp.Token
	return nil
}

func fetchAPI(endpoint string, target interface{}) error {
	req, _ := http.NewRequest("GET", baseURL+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("HTTP %d", resp.StatusCode) }
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
	myApp.Settings().SetTheme(supernovaTheme{})
	
	w := myApp.NewWindow("Supernova Desktop")
	w.Resize(fyne.NewSize(1100, 700))

	var buildAppUI func() *fyne.Container

	// ---- LOGIN UI ----
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	loginBtn := widget.NewButton("Login", func() {
		// Async login
		username := usernameEntry.Text
		password := passwordEntry.Text
		go func() {
			err := login(username, password)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			appUI := buildAppUI()
			w.SetContent(appUI)
		}()
	})
	loginBtn.Importance = widget.HighImportance

	titleText := canvas.NewText("🚀 Supernova", theme.PrimaryColor())
	titleText.TextSize = 32
	titleText.TextStyle.Bold = true
	titleText.Alignment = fyne.TextAlignCenter

	loginForm := container.NewVBox(
		titleText,
		widget.NewLabel(""), // spacer
		usernameEntry,
		passwordEntry,
		widget.NewLabel(""), // spacer
		loginBtn,
	)
	loginContainer := container.NewCenter(loginForm)
	w.SetContent(loginContainer)

	// ---- APP UI ----
	buildAppUI = func() *fyne.Container {
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
		statusLabel.TextStyle.Bold = true
		
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
			statusLabel.SetText("Play/Pause clicked")
		})
		stopBtn := widget.NewButtonWithIcon("", theme.MediaStopIcon(), func() { stopPlayback() })
		
		progressBar := widget.NewProgressBar()
		progressBar.SetValue(0)
		
		controlsBox := container.NewHBox(playBtn, stopBtn)
		nowPlayingBox := container.NewVBox(statusLabel, progressBar)
		playerContainer := container.NewBorder(nil, nil, controlsBox, nil, nowPlayingBox)

		// Navigation Sidebar
		navHome := widget.NewButtonWithIcon("Home", theme.HomeIcon(), func() {})
		navHearts := widget.NewButtonWithIcon("Hearts", theme.DocumentIcon(), func() {})
		navPlaylists := widget.NewButtonWithIcon("Playlists", theme.FolderIcon(), func() {})
		navHome.Importance = widget.HighImportance
		
		sidebar := container.NewVBox(
			widget.NewLabelWithStyle("Supernova", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			navHome, navHearts, navPlaylists,
			layout.NewSpacer(),
			widget.NewButtonWithIcon("Logout", theme.LogoutIcon(), func() {
				stopPlayback()
				jwtToken = ""
				w.SetContent(loginContainer)
			}),
		)

		// Async Loaders
		loadAlbums := func(artistID string) {
			go func() {
				var newAlbums []Album
				if err := fetchAPI("/albums?limit=1000&artist_id="+artistID, &newAlbums); err == nil {
					albums = newAlbums
					tracks = nil
					albumsList.Refresh()
					tracksList.Refresh()
				}
			}()
		}
		
		loadTracks := func(albumID string) {
			go func() {
				var newTracks []Track
				if err := fetchAPI("/tracks?limit=1000&album_id="+albumID, &newTracks); err == nil {
					tracks = newTracks
					tracksList.Refresh()
				}
			}()
		}

		// Wiring List Selections
		artistsList.OnSelected = func(id widget.ListItemID) {
			loadAlbums(artists[id].ID)
		}

		albumsList.OnSelected = func(id widget.ListItemID) {
			loadTracks(albums[id].ID)
		}

		tracksList.OnSelected = func(id widget.ListItemID) {
			playTrack(tracks[id])
		}

		// Initial Data Load
		go func() {
			var initArtists []Artist
			if err := fetchAPI("/artists?limit=1000", &initArtists); err == nil {
				artists = initArtists
				artistsList.Refresh()
			}
		}()

		// Main Layout Assembly
		artistsBox := container.NewBorder(widget.NewLabelWithStyle("Artists", fyne.TextAlignCenter, fyne.TextStyle{Bold:true}), nil, nil, nil, artistsList)
		albumsBox := container.NewBorder(widget.NewLabelWithStyle("Albums", fyne.TextAlignCenter, fyne.TextStyle{Bold:true}), nil, nil, nil, albumsList)
		tracksBox := container.NewBorder(widget.NewLabelWithStyle("Tracks", fyne.TextAlignCenter, fyne.TextStyle{Bold:true}), nil, nil, nil, tracksList)

		splitAlbumsTracks := container.NewHSplit(albumsBox, tracksBox)
		splitMain := container.NewHSplit(artistsBox, splitAlbumsTracks)
		splitMain.SetOffset(0.25)
		splitAlbumsTracks.SetOffset(0.33)

		mainContent := container.NewBorder(nil, nil, sidebar, nil, splitMain)
		return container.NewBorder(nil, playerContainer, nil, nil, mainContent)
	}

	w.ShowAndRun()
	
	// Cleanup on exit
	if currentFFPlay != nil && currentFFPlay.Process != nil {
		currentFFPlay.Process.Kill()
	}
}
