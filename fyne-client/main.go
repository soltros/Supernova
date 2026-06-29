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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var instanceURL = "http://localhost:8080/api"
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
	resp, err := http.Post(instanceURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("login failed: %s - %s", resp.Status, buf.String())
	}
	var authResp AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	jwtToken = authResp.Token
	return nil
}

func register(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(instanceURL+"/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("register failed: %s - %s", resp.Status, buf.String())
	}
	var authResp AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	jwtToken = authResp.Token
	return nil
}

func fetchAPI(endpoint string, target interface{}) error {
	req, _ := http.NewRequest("GET", instanceURL+endpoint, nil)
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

func postAPI(endpoint string, payload interface{}) error {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", instanceURL+endpoint, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
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
	instanceEntry := widget.NewEntry()
	instanceEntry.SetText("http://localhost:8080/api")
	instanceEntry.SetPlaceHolder("Instance URL")
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	loginBtn := widget.NewButton("Login", func() {
		instanceURL = instanceEntry.Text
		go func() {
			err := login(usernameEntry.Text, passwordEntry.Text)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			w.SetContent(buildAppUI())
		}()
	})
	loginBtn.Importance = widget.HighImportance

	registerBtn := widget.NewButton("Register", func() {
		instanceURL = instanceEntry.Text
		go func() {
			err := register(usernameEntry.Text, passwordEntry.Text)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			w.SetContent(buildAppUI())
		}()
	})

	titleText := canvas.NewText("🚀 Supernova", theme.PrimaryColor())
	titleText.TextSize = 32
	titleText.TextStyle.Bold = true
	titleText.Alignment = fyne.TextAlignCenter

	loginForm := container.NewVBox(
		titleText,
		widget.NewLabel(""), // spacer
		instanceEntry,
		usernameEntry,
		passwordEntry,
		widget.NewLabel(""), // spacer
		loginBtn,
		registerBtn,
	)
	loginContainer := container.NewCenter(loginForm)
	w.SetContent(loginContainer)

	// ---- APP UI ----
	buildAppUI = func() *fyne.Container {
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
			// Log scrobble
			go postAPI("/scrobbles", map[string]string{"track_id": track.ID})

			streamURL := fmt.Sprintf("%s/stream/%s", instanceURL, track.ID)
			headerArg := fmt.Sprintf("Authorization: Bearer %s", jwtToken)
			currentFFPlay = exec.Command("ffplay", "-headers", headerArg, "-nodisp", "-autoexit", streamURL)
			currentFFPlay.Start()
			statusLabel.SetText(fmt.Sprintf("▶ %s", track.Title))
		}

		prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {})
		playBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
			statusLabel.SetText("Play/Pause clicked")
		})
		stopBtn := widget.NewButtonWithIcon("", theme.MediaStopIcon(), func() { stopPlayback() })
		nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {})

		progressBar := widget.NewSlider(0, 100)
		progressBar.SetValue(0)
		volumeSlider := widget.NewSlider(0, 100)
		volumeSlider.SetValue(100)
		searchEntry := widget.NewEntry()
		searchEntry.SetPlaceHolder("Search...")

		controlsBox := container.NewHBox(prevBtn, playBtn, stopBtn, nextBtn)
		topPlayerBar := container.NewBorder(nil, nil, controlsBox, container.NewHBox(volumeSlider, searchEntry), container.NewBorder(nil, nil, nil, nil, container.NewVBox(statusLabel, progressBar)))

		// ---------------- VIEWS ----------------

		// 1. Music (Library) View
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

		artistsList.OnSelected = func(id widget.ListItemID) {
			go func() {
				var newAlbums []Album
				if err := fetchAPI("/albums?limit=1000&artist_id="+artists[id].ID, &newAlbums); err == nil {
					albums = newAlbums
					tracks = nil
					albumsList.Refresh()
					tracksList.Refresh()
				}
			}()
		}
		albumsList.OnSelected = func(id widget.ListItemID) {
			go func() {
				var newTracks []Track
				if err := fetchAPI("/tracks?limit=1000&album_id="+albums[id].ID, &newTracks); err == nil {
					tracks = newTracks
					tracksList.Refresh()
				}
			}()
		}
		tracksList.OnSelected = func(id widget.ListItemID) {
			playTrack(tracks[id])
		}

		artistsBox := container.NewBorder(widget.NewLabelWithStyle("Artists", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, artistsList)
		albumsBox := container.NewBorder(widget.NewLabelWithStyle("Albums", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, albumsList)
		tracksBox := container.NewBorder(widget.NewLabelWithStyle("Tracks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, tracksList)

		topSplit := container.NewHSplit(artistsBox, albumsBox)
		musicView := container.NewVSplit(topSplit, tracksBox)

		// 2. Hearts View
		var heartsTracks []Track
		heartsList := widget.NewList(
			func() int { return len(heartsTracks) },
			func() fyne.CanvasObject { return widget.NewLabel("Hearted Track") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(heartsTracks[i].Title)
			},
		)
		heartsList.OnSelected = func(id widget.ListItemID) {
			playTrack(heartsTracks[id])
		}
		heartsView := container.NewBorder(widget.NewLabelWithStyle("Favorite Tracks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, heartsList)

		// 3. Playlists View
		var playlists []Playlist
		var playlistTracks []Track
		playlistsList := widget.NewList(
			func() int { return len(playlists) },
			func() fyne.CanvasObject { return widget.NewLabel("Playlist") },
			func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(playlists[i].Name) },
		)
		playlistTracksList := widget.NewList(
			func() int { return len(playlistTracks) },
			func() fyne.CanvasObject { return widget.NewLabel("Playlist Track") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(playlistTracks[i].Title)
			},
		)
		playlistsList.OnSelected = func(id widget.ListItemID) {
			go func() {
				var newTracks []Track
				if err := fetchAPI("/playlists/"+playlists[id].ID+"/tracks", &newTracks); err == nil {
					playlistTracks = newTracks
					playlistTracksList.Refresh()
				}
			}()
		}
		playlistTracksList.OnSelected = func(id widget.ListItemID) {
			playTrack(playlistTracks[id])
		}
		playlistsView := container.NewHSplit(
			container.NewBorder(widget.NewLabelWithStyle("Playlists", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, playlistsList),
			container.NewBorder(widget.NewLabelWithStyle("Playlist Tracks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, playlistTracksList),
		)

		// 4. Scrobbles View
		var scrobbles []Track
		scrobblesList := widget.NewList(
			func() int { return len(scrobbles) },
			func() fyne.CanvasObject { return widget.NewLabel("Scrobbled Track") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(scrobbles[i].Title)
			},
		)
		scrobblesList.OnSelected = func(id widget.ListItemID) {
			playTrack(scrobbles[id])
		}
		scrobblesView := container.NewBorder(widget.NewLabelWithStyle("Recent Scrobbles", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, scrobblesList)

		// Content Area
		contentArea := container.NewMax(musicView)

		// Navigation Sidebar
		sidebarLabels := []string{"Music", "Hearts", "Playlists", "Scrobbles"}
		sidebar := widget.NewList(
			func() int { return len(sidebarLabels) },
			func() fyne.CanvasObject { return widget.NewLabel("Library") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(sidebarLabels[i])
			},
		)

		sidebar.OnSelected = func(id widget.ListItemID) {
			contentArea.Objects = nil
			switch id {
			case 0:
				contentArea.Add(musicView)
			case 1:
				go func() {
					var details HeartDetails
					if err := fetchAPI("/hearts/details", &details); err == nil {
						heartsTracks = details.Tracks
						heartsList.Refresh()
					}
				}()
				contentArea.Add(heartsView)
			case 2:
				go func() {
					var p []Playlist
					if err := fetchAPI("/playlists", &p); err == nil {
						playlists = p
						playlistsList.Refresh()
					}
				}()
				contentArea.Add(playlistsView)
			case 3:
				go func() {
					var s []Track
					if err := fetchAPI("/scrobbles/recent", &s); err == nil {
						scrobbles = s
						scrobblesList.Refresh()
					}
				}()
				contentArea.Add(scrobblesView)
			}
			contentArea.Refresh()
		}

		sidebarContainer := container.NewBorder(
			widget.NewLabelWithStyle("Supernova", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewButtonWithIcon("Logout", theme.LogoutIcon(), func() {
				stopPlayback()
				jwtToken = ""
				w.SetContent(loginContainer)
			}),
			nil, nil, sidebar,
		)

		// Initial load for Music view
		go func() {
			var initArtists []Artist
			if err := fetchAPI("/artists?limit=1000", &initArtists); err == nil {
				artists = initArtists
				artistsList.Refresh()
			}
		}()
		sidebar.Select(0) // Default to Music view

		mainContent := container.NewHSplit(sidebarContainer, contentArea)
		mainContent.SetOffset(0.2)

		return container.NewBorder(topPlayerBar, nil, nil, nil, mainContent)
	}

	w.ShowAndRun()

	// Cleanup on exit
	if currentFFPlay != nil && currentFFPlay.Process != nil {
		currentFFPlay.Process.Kill()
	}
}
