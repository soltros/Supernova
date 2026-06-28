package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const baseURL = "http://localhost:8080/api"

var jwtToken string
var app *tview.Application
var pages *tview.Pages
var currentFFPlay *exec.Cmd

// Supernova Colors
var (
	ColorPrimary   = tcell.NewHexColor(0x9d4edd) // Deep Purple
	ColorSecondary = tcell.NewHexColor(0xff006e) // Vibrant Pink
	ColorAccent    = tcell.NewHexColor(0x00f5d4) // Cyan/Mint
	ColorText      = tcell.ColorWhite
	ColorBg        = tcell.ColorDefault
)

// UI Elements
var (
	visualizerText *tview.TextView
	controlsText   *tview.TextView
	statusText     *tview.TextView
)

// State
var (
	isPlaying      bool
	currentTrack   string
	visualizerQuit chan struct{}
)

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

func init() {
	// Rounded borders for elegant feel
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'
}

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

func startVisualizer() {
	if visualizerQuit != nil {
		close(visualizerQuit)
	}
	visualizerQuit = make(chan struct{})
	
	bars := []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-visualizerQuit:
				app.QueueUpdateDraw(func() {
					visualizerText.SetText("")
				})
				return
			case <-ticker.C:
				if !isPlaying {
					continue
				}
				// Generate random bars
				var out strings.Builder
				for i := 0; i < 40; i++ {
					out.WriteRune(bars[rand.Intn(len(bars))])
				}
				app.QueueUpdateDraw(func() {
					visualizerText.SetText(fmt.Sprintf("[#00f5d4]%s[-]", out.String()))
				})
			}
		}
	}()
}

func stopVisualizer() {
	if visualizerQuit != nil {
		close(visualizerQuit)
		visualizerQuit = nil
	}
}

func playTrack(trackID, trackTitle string) {
	stopPlayback()
	
	streamURL := fmt.Sprintf("%s/stream/%s", baseURL, trackID)
	headerArg := fmt.Sprintf("Authorization: Bearer %s", jwtToken)
	
	currentFFPlay = exec.Command("ffplay", "-headers", headerArg, "-nodisp", "-autoexit", streamURL)
	currentFFPlay.Start()

	currentTrack = trackTitle
	isPlaying = true
	
	statusText.SetText(fmt.Sprintf(" [#9d4edd]▶ Playing:[-] %s ", trackTitle))
	updateControls()
	startVisualizer()
}

func stopPlayback() {
	if currentFFPlay != nil && currentFFPlay.Process != nil {
		currentFFPlay.Process.Kill()
	}
	isPlaying = false
	currentTrack = ""
	statusText.SetText(" [#ff006e]⏸ Stopped[-] ")
	updateControls()
	stopVisualizer()
}

func togglePause() {
	// ffplay doesn't easily support pausing via stdin in this mode, 
	// so for this TUI we'll just stop it. A true robust client would use mpv IPC or beep.
	if isPlaying {
		stopPlayback()
	}
}

func updateControls() {
	var playBtn, stopBtn string
	if isPlaying {
		playBtn = `["play"]⏸ Pause[""]`
	} else {
		playBtn = `["play"]▶ Play[""]`
	}
	stopBtn = `["stop"]⏹ Stop[""]`
	controlsText.SetText(fmt.Sprintf("%s  |  %s", playBtn, stopBtn))
}

func buildLoginForm() *tview.Flex {
	form := tview.NewForm()
	
	usernameInput := tview.NewInputField().SetLabel("Username: ").SetFieldWidth(30)
	passwordInput := tview.NewInputField().SetLabel("Password: ").SetFieldWidth(30).SetMaskCharacter('*')

	form.AddFormItem(usernameInput).
		AddFormItem(passwordInput).
		AddButton("Login", func() {
			user := usernameInput.GetText()
			pass := passwordInput.GetText()
			if err := login(user, pass); err != nil {
				usernameInput.SetText("")
				passwordInput.SetText("")
				usernameInput.SetTitle(" Login Failed ")
				return
			}
			loadAppUI()
			pages.SwitchToPage("App")
		}).
		AddButton("Quit", func() {
			app.Stop()
		})

	form.SetBorder(true).
		SetTitle(" 🚀 Supernova TUI ").
		SetTitleColor(ColorSecondary).
		SetBorderColor(ColorPrimary)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 11, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)

	return flex
}

func loadAppUI() {
	artistsList := tview.NewList().ShowSecondaryText(false)
	artistsList.SetBorder(true).SetTitle(" Artists ").SetTitleColor(ColorPrimary).SetBorderColor(ColorPrimary)

	albumsList := tview.NewList().ShowSecondaryText(false)
	albumsList.SetBorder(true).SetTitle(" Albums ").SetTitleColor(ColorSecondary).SetBorderColor(ColorSecondary)

	tracksList := tview.NewList().ShowSecondaryText(false)
	tracksList.SetBorder(true).SetTitle(" Tracks (Enter to Play) ").SetTitleColor(ColorAccent).SetBorderColor(ColorAccent)

	// Bottom Bar
	bottomBar := tview.NewFlex().SetDirection(tview.FlexColumn)
	
	visualizerText = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	statusText = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	controlsText = tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetTextAlign(tview.AlignRight)
	
	statusText.SetText(" 🎵 Ready. (Tab: Switch Panels, Mouse: Supported) ")
	updateControls()

	controlsText.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			if added[0] == "play" {
				togglePause()
			} else if added[0] == "stop" {
				stopPlayback()
			}
		}
	})

	bottomBar.AddItem(visualizerText, 0, 1, false)
	bottomBar.AddItem(statusText, 0, 2, false)
	bottomBar.AddItem(controlsText, 0, 1, false)

	// Layout
	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(artistsList, 0, 1, true).
			AddItem(albumsList, 0, 1, false).
			AddItem(tracksList, 0, 2, false),
		0, 1, true).
		AddItem(bottomBar, 1, 1, false)

	// Focus switching logic
	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
		if event.Rune() == 's' || event.Rune() == 'S' {
			stopPlayback()
			return nil
		}
		return event
	})

	// Fetch Data
	var artists []Artist
	if err := fetchAPI("/artists?limit=1000", &artists); err == nil {
		for _, artist := range artists {
			artistsList.AddItem(artist.Name, artist.ID, 0, nil)
		}
	}

	// Handlers
	artistsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		albumsList.Clear()
		tracksList.Clear()
		if secondaryText == "" {
			return
		}
		var albums []Album
		if err := fetchAPI("/albums?limit=1000&artist_id="+secondaryText, &albums); err == nil {
			for _, album := range albums {
				title := fmt.Sprintf("%s (%d)", album.Title, album.Year)
				if album.Year == 0 {
					title = album.Title
				}
				albumsList.AddItem(title, album.ID, 0, nil)
			}
		}
	})

	albumsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tracksList.Clear()
		if secondaryText == "" {
			return
		}
		var tracks []Track
		if err := fetchAPI("/tracks?limit=1000&album_id="+secondaryText, &tracks); err == nil {
			for _, track := range tracks {
				title := fmt.Sprintf("%d. %s [%s]", track.TrackNumber, track.Title, formatTime(track.Duration))
				tracksList.AddItem(title, track.ID, 0, nil)
			}
		}
	})

	tracksList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		playTrack(secondaryText, mainText)
	})

	pages.AddPage("App", mainLayout, true, false)
}

func main() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	loginForm := buildLoginForm()
	pages.AddPage("Login", loginForm, true, true)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

	stopPlayback()
}
