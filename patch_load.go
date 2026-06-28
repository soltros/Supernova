func loadAppUI() {
	player.InitMPRIS()

	navBar := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetTextAlign(tview.AlignCenter)
	navBar.SetText(`["home"]Home[""]  |  ["hearts"]Hearts[""]  |  ["playlists"]Playlists[""]`)

	queueList = tview.NewList().ShowSecondaryText(false)
	queueList.SetBorder(true).SetTitle(" Playback Queue ").SetTitleColor(tcell.ColorWhite).SetBorderColor(tcell.ColorGray)

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

	go func() {
		var artists []models.Artist
		if err := api.Fetch("/artists?limit=1000", &artists); err == nil {
			app.QueueUpdateDraw(func() {
				for _, artist := range artists {
					artistsList.AddItem(artist.Name, artist.ID, 0, nil)
				}
			})
		}
	}()

	artistsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		albumsList.Clear()
		tracksList.Clear()
		if secondaryText == "" { return }
		go func(artistID string) {
			var albums []models.Album
			if err := api.Fetch("/albums?limit=1000&artist_id="+artistID, &albums); err == nil {
				app.QueueUpdateDraw(func() {
					if artistsList.GetItemCount() == 0 { return }
					_, currentArtistID := artistsList.GetItemText(artistsList.GetCurrentItem())
					if currentArtistID != artistID { return }
					
					albumsList.Clear()
					for _, album := range albums {
						albumsList.AddItem(fmt.Sprintf("%s (%d)", album.Title, album.Year), album.ID, 0, nil)
					}
				})
			}
		}(secondaryText)
	})

	albumsList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tracksList.Clear()
		if secondaryText == "" { return }
		go func(albumID string) {
			var tracks []models.Track
			if err := api.Fetch("/tracks?limit=1000&album_id="+albumID, &tracks); err == nil {
				app.QueueUpdateDraw(func() {
					if albumsList.GetItemCount() == 0 { return }
					_, currentAlbumID := albumsList.GetItemText(albumsList.GetCurrentItem())
					if currentAlbumID != albumID { return }
					
					tracksList.Clear()
					for _, track := range tracks {
						title := fmt.Sprintf("%d. %s", track.TrackNumber, track.Title)
						tracksList.AddItem(title, track.ID, 0, nil)
					}
				})
			}
		}(secondaryText)
	})

	tracksList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		var albumID string
		if albumsList.GetItemCount() > 0 { _, albumID = albumsList.GetItemText(albumsList.GetCurrentItem()) }
		go func(aID string, trackID string) {
			var tracks []models.Track
			if err := api.Fetch("/tracks?limit=1000&album_id="+aID, &tracks); err == nil {
				for _, t := range tracks {
					if t.ID == trackID {
						app.QueueUpdateDraw(func() {
							player.AddToQueue(t)
							queueList.AddItem(t.Title, t.ID, 0, nil)
							if player.CurrentState == player.Stopped { player.PlayTrack(len(player.Queue) - 1) }
						})
						break
					}
				}
			}
		}(albumID, secondaryText)
	})

	tracksList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' || event.Rune() == 'H' {
			if tracksList.GetItemCount() > 0 {
				_, trackID := tracksList.GetItemText(tracksList.GetCurrentItem())
				go func(tid string) {
					api.HeartToggle("track", tid)
					app.QueueUpdateDraw(func() { statusText.SetText(" 💖 Heart toggled! ") })
				}(trackID)
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
		go func(trackID string) {
			var hData models.HeartDetails
			if err := api.Fetch("/hearts/details", &hData); err == nil {
				for _, t := range hData.Tracks {
					if t.ID == trackID {
						app.QueueUpdateDraw(func() {
							player.AddToQueue(t)
							queueList.AddItem(t.Title, t.ID, 0, nil)
							if player.CurrentState == player.Stopped { player.PlayTrack(len(player.Queue) - 1) }
						})
						break
					}
				}
			}
		}(secondaryText)
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
		go func(plID string) {
			var tracks []models.Track
			if err := api.Fetch("/playlists/"+plID+"/tracks", &tracks); err == nil {
				app.QueueUpdateDraw(func() {
					if playlistsList.GetItemCount() == 0 { return }
					_, currentPlID := playlistsList.GetItemText(playlistsList.GetCurrentItem())
					if currentPlID != plID { return }
					
					plTracksList.Clear()
					for _, track := range tracks {
						plTracksList.AddItem(track.Title, track.ID, 0, nil)
					}
				})
			}
		}(secondaryText)
	})
	
	plTracksList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		var playlistID string
		if playlistsList.GetItemCount() > 0 { _, playlistID = playlistsList.GetItemText(playlistsList.GetCurrentItem()) }
		go func(pID string, trackID string) {
			var tracks []models.Track
			if err := api.Fetch("/playlists/"+pID+"/tracks", &tracks); err == nil {
				for _, t := range tracks {
					if t.ID == trackID {
						app.QueueUpdateDraw(func() {
							player.AddToQueue(t)
							queueList.AddItem(t.Title, t.ID, 0, nil)
							if player.CurrentState == player.Stopped { player.PlayTrack(len(player.Queue) - 1) }
						})
						break
					}
				}
			}
		}(playlistID, secondaryText)
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
				innerPages.SwitchToPage("hearts")
				go func() {
					var hData models.HeartDetails
					if err := api.Fetch("/hearts/details", &hData); err == nil {
						app.QueueUpdateDraw(func() {
							heartsList.Clear()
							for _, t := range hData.Tracks { heartsList.AddItem(t.Title, t.ID, 0, nil) }
						})
					}
				}()
			case "playlists":
				playlistsList.Clear()
				innerPages.SwitchToPage("playlists")
				go func() {
					var pData []models.Playlist
					if err := api.Fetch("/playlists", &pData); err == nil {
						app.QueueUpdateDraw(func() {
							playlistsList.Clear()
							for _, p := range pData { playlistsList.AddItem(p.Name, p.ID, 0, nil) }
						})
					}
				}()
			}
		}
	})

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

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(navBar, 1, 1, false).
		AddItem(tview.NewFlex().
			AddItem(innerPages, 0, 3, true).
			AddItem(queueList, 0, 1, false),
		0, 1, true).
		AddItem(bottomBar, 1, 1, false)
	
	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if queueList.HasFocus() { app.SetFocus(innerPages) } else { app.SetFocus(queueList) }
			return nil
		}
		return event
	})

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
