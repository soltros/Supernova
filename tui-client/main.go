package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	
	"github.com/soltros/Supernova/tui-client/api"
	"github.com/soltros/Supernova/tui-client/models"
	"github.com/soltros/Supernova/tui-client/player"
)

var app *tview.Application
var pages *tview.Pages

// Colors
var (
	ColorPrimary   = tcell.NewHexColor(0x9d4edd)
	ColorSecondary = tcell.NewHexColor(0xff006e)
	ColorAccent    = tcell.NewHexColor(0x00f5d4)
)

var (
	visualizerText *tview.TextView
	controlsText   *tview.TextView
	statusText     *tview.TextView
	queueList      *tview.List
	visualizerQuit chan struct{}
)

func init() {
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'
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
				app.QueueUpdateDraw(func() { visualizerText.SetText("") })
				return
			case <-ticker.C:
				if player.CurrentState != player.Playing { continue }
				var out strings.Builder
				for i := 0; i < 20; i++ {
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

func updateControls() {
	var playBtn string
	if player.CurrentState == player.Playing {
		playBtn = `["play"]⏸ Pause[""]`
	} else {
		playBtn = `["play"]▶ Play[""]`
	}
	controlsText.SetText(fmt.Sprintf(`["prev"]⏮ Prev[""]  %s  ["next"]⏭ Next[""]  |  ["stop"]⏹ Stop[""]`, playBtn))
}

func buildLoginForm() *tview.Flex {
	form := tview.NewForm()
	
	userIn := tview.NewInputField().SetLabel("Username: ").SetFieldWidth(30)
	passIn := tview.NewInputField().SetLabel("Password: ").SetFieldWidth(30).SetMaskCharacter('*')

	loginFunc := func() {
		if err := api.Login(userIn.GetText(), passIn.GetText()); err != nil {
			userIn.SetTitle(" Login Failed ")
			return
		}
		loadAppUI()
		pages.SwitchToPage("App")
	}

	regFunc := func() {
		if err := api.Register(userIn.GetText(), passIn.GetText()); err == nil {
			loginFunc()
		} else {
			userIn.SetTitle(" Reg Failed ")
		}
	}

	form.AddFormItem(userIn).
		AddFormItem(passIn).
		AddButton("Login", loginFunc).
		AddButton("Register", regFunc).
		AddButton("Quit", func() { app.Stop() })

	form.SetBorder(true).SetTitle(" 🚀 Supernova ").SetTitleColor(ColorSecondary).SetBorderColor(ColorPrimary)

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 15, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)
}

func loadAppUI() {
	player.InitMPRIS()

	// Nav
	navBar := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetTextAlign(tview.AlignCenter)
	navBar.SetText(`["home"]Home[""]  |  ["hearts"]Hearts[""]  |  ["playlists"]Playlists[""]`)

	// Queue pane (Always visible)
	queueList = tview.NewList().ShowSecondaryText(false)
	queueList.SetBorder(true).SetTitle(" Playback Queue ").SetTitleColor(tcell.ColorWhite).SetBorderColor(tcell.ColorGray)

	// Inner Pages for swapping views
	innerPages := tview.NewPages()

	// ----------------- HOME VIEW -----------------
	artistsList := tview.NewList().ShowSecondaryText(false)
	artistsList.SetBorder(true).SetTitle(" Artists ").SetTitleColor(ColorPrimary).SetBorderColor(ColorPrimary)
	albumsList := tview.NewList().ShowSecondaryText(false)
	albumsList.SetBorder(true).SetTitle(" Albums ").SetTitleColor(ColorSecondary).SetBorderColor(ColorSecondary)
	tracksList := tview.NewList().ShowSecondaryText(false)
	tracksList.SetBorder(true).SetTitle(" Tracks (Enter: Enqueue, H: Heart) ").SetTitleColor(ColorAccent).SetBorderColor(ColorAccent)

	homeView := tview.NewFlex().
		AddItem(artistsList, 0, 1, true).
		AddItem(albumsList, 0, 1, false).
		AddItem(tracksList, 0, 2, false)

	// Fetch Data Home
	var artists []models.Artist
	if err := api.Fetch("/artists?limit=1000", &artists); err == nil {
		for _, artist := range artists {
			artistsList.AddItem(artist.Name, artist.ID, 0, nil)
		}
	}
	artistsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		albumsList.Clear()
		tracksList.Clear()
		if secondaryText == "" { return }
		var albums []models.Album
		if err := api.Fetch("/albums?limit=1000&artist_id="+secondaryText, &albums); err == nil {
			for _, album := range albums {
				albumsList.AddItem(fmt.Sprintf("%s (%d)", album.Title, album.Year), album.ID, 0, nil)
			}
		}
	})
	albumsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tracksList.Clear()
		if secondaryText == "" { return }
		var tracks []models.Track
		if err := api.Fetch("/tracks?limit=1000&album_id="+secondaryText, &tracks); err == nil {
			for _, track := range tracks {
				title := fmt.Sprintf("%d. %s", track.TrackNumber, track.Title)
				tracksList.AddItem(title, track.ID, 0, nil)
			}
		}
	})
	tracksList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		var albumID string
		if albumsList.GetItemCount() > 0 { _, albumID = albumsList.GetItemText(albumsList.GetCurrentItem()) }
		var tracks []models.Track
		api.Fetch("/tracks?limit=1000&album_id="+albumID, &tracks)
		for _, t := range tracks {
			if t.ID == secondaryText {
				player.AddToQueue(t)
				queueList.AddItem(t.Title, t.ID, 0, nil)
				if player.CurrentState == player.Stopped { player.PlayTrack(len(player.Queue) - 1) }
				break
			}
		}
	})
	tracksList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' || event.Rune() == 'H' {
			if tracksList.GetItemCount() > 0 {
				_, trackID := tracksList.GetItemText(tracksList.GetCurrentItem())
				api.HeartToggle("track", trackID)
				statusText.SetText(" 💖 Heart toggled! ")
			}
			return nil
		}
		return event
	})

	// ----------------- HEARTS VIEW -----------------
	heartsList := tview.NewList().ShowSecondaryText(false)
	heartsList.SetBorder(true).SetTitle(" Hearted Tracks ").SetTitleColor(ColorSecondary).SetBorderColor(ColorSecondary)
	heartsView := tview.NewFlex().AddItem(heartsList, 0, 1, true)

	heartsList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		var hData models.HeartDetails
		api.Fetch("/hearts/details", &hData)
		for _, t := range hData.Tracks {
			if t.ID == secondaryText {
				player.AddToQueue(t)
				queueList.AddItem(t.Title, t.ID, 0, nil)
				if player.CurrentState == player.Stopped { player.PlayTrack(len(player.Queue) - 1) }
				break
			}
		}
	})

	// ----------------- PLAYLISTS VIEW -----------------
	playlistsList := tview.NewList().ShowSecondaryText(false)
	playlistsList.SetBorder(true).SetTitle(" Playlists ").SetTitleColor(ColorPrimary).SetBorderColor(ColorPrimary)
	plTracksList := tview.NewList().ShowSecondaryText(false)
	plTracksList.SetBorder(true).SetTitle(" Playlist Tracks ").SetTitleColor(ColorAccent).SetBorderColor(ColorAccent)

	playlistsView := tview.NewFlex().
		AddItem(playlistsList, 0, 1, true).
		AddItem(plTracksList, 0, 2, false)

	playlistsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		plTracksList.Clear()
		if secondaryText == "" { return }
		var tracks []models.Track
		if err := api.Fetch("/playlists/"+secondaryText+"/tracks", &tracks); err == nil {
			for _, track := range tracks {
				plTracksList.AddItem(track.Title, track.ID, 0, nil)
			}
		}
	})
	
	plTracksList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		var playlistID string
		if playlistsList.GetItemCount() > 0 { _, playlistID = playlistsList.GetItemText(playlistsList.GetCurrentItem()) }
		var tracks []models.Track
		api.Fetch("/playlists/"+playlistID+"/tracks", &tracks)
		for _, t := range tracks {
			if t.ID == secondaryText {
				player.AddToQueue(t)
				queueList.AddItem(t.Title, t.ID, 0, nil)
				if player.CurrentState == player.Stopped { player.PlayTrack(len(player.Queue) - 1) }
				break
			}
		}
	})

	// Setup Pages
	innerPages.AddPage("home", homeView, true, true)
	innerPages.AddPage("hearts", heartsView, true, false)
	innerPages.AddPage("playlists", playlistsView, true, false)

	navBar.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			switch added[0] {
			case "home":
				innerPages.SwitchToPage("home")
			case "hearts":
				heartsList.Clear()
				var hData models.HeartDetails
				if err := api.Fetch("/hearts/details", &hData); err == nil {
					for _, t := range hData.Tracks { heartsList.AddItem(t.Title, t.ID, 0, nil) }
				}
				innerPages.SwitchToPage("hearts")
			case "playlists":
				playlistsList.Clear()
				var pData []models.Playlist
				if err := api.Fetch("/playlists", &pData); err == nil {
					for _, p := range pData { playlistsList.AddItem(p.Name, p.ID, 0, nil) }
				}
				innerPages.SwitchToPage("playlists")
			}
		}
	})

	// Bottom bar
	visualizerText = tview.NewTextView().SetDynamicColors(true)
	statusText = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	controlsText = tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetTextAlign(tview.AlignRight)
	
	updateControls()
	controlsText.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			switch added[0] {
			case "play": player.TogglePause()
			case "stop": player.Stop()
			case "next": player.Next()
			case "prev": player.Prev()
			}
		}
	})

	bottomBar := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(visualizerText, 22, 1, false).
		AddItem(statusText, 0, 1, false).
		AddItem(controlsText, 40, 1, false)

	// Main Layout
	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(navBar, 1, 1, false).
		AddItem(tview.NewFlex().
			AddItem(innerPages, 0, 3, true).
			AddItem(queueList, 0, 1, false),
		0, 1, true).
		AddItem(bottomBar, 1, 1, false)
	
	// App input capture for fast navigation
	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Basic tab cycling. To do it perfectly requires recursive focus checking, 
			// but we'll do a simple swap between library and queue
			if queueList.HasFocus() {
				app.SetFocus(innerPages)
			} else {
				app.SetFocus(queueList)
			}
			return nil
		}
		return event
	})

	// Player State Callback
	player.OnStateChange = func(state player.State, track *models.Track) {
		app.QueueUpdateDraw(func() {
			updateControls()
			if state == player.Playing && track != nil {
				statusText.SetText(fmt.Sprintf(" [#9d4edd]▶ Playing:[-] %s ", track.Title))
				startVisualizer()
			} else {
				statusText.SetText(" [#ff006e]⏸ Stopped[-] ")
				stopVisualizer()
			}
		})
	}

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
	player.Stop()
}
