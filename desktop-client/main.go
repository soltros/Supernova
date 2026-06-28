package main

import (
	"fmt"
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/soltros/Supernova/desktop-client/api"
)

func main() {
	app := gtk.NewApplication("com.soltros.supernova", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() {
		loadCSS()
		activate(app)
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func loadCSS() {
	cssProvider := gtk.NewCSSProvider()
	cssProvider.LoadFromData(`
		window {
			background-color: rgba(30, 30, 35, 0.65); /* Semi-transparent dark glass */
		}
		headerbar {
			background-color: rgba(255, 255, 255, 0.05);
			border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		}
		list {
			background-color: transparent;
		}
		list > row {
			background-color: rgba(255, 255, 255, 0.05);
			border-radius: 12px;
			margin: 8px 16px;
			padding: 8px;
			border: 1px solid rgba(255, 255, 255, 0.08);
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
			transition: all 250ms ease-in-out;
		}
		list > row:hover {
			background-color: rgba(255, 255, 255, 0.12);
			box-shadow: 0 6px 16px rgba(0, 0, 0, 0.3);
		}
		label {
			color: #eeeeee;
			font-weight: 500;
		}
		.year-label {
			color: #aaaaaa;
		}
	`)
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(),
		cssProvider,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
	)
}

func activate(app *gtk.Application) {
	window := gtk.NewApplicationWindow(app)
	window.SetTitle("Supernova")
	window.SetDefaultSize(800, 600)

	// In GTK4, to get a custom headerbar we need to set the titlebar
	header := gtk.NewHeaderBar()
	window.SetTitlebar(header)

	vbox := gtk.NewBox(gtk.OrientationVertical, 0)
	window.SetChild(vbox)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	vbox.Append(scrolled)

	listBox := gtk.NewListBox()
	listBox.SetSelectionMode(gtk.SelectionSingle)
	// Remove listbox styling
	listBox.AddCSSClass("navigation-sidebar")
	scrolled.SetChild(listBox)

	statusLabel := gtk.NewLabel("Connecting to API...")
	statusLabel.SetMarginBottom(8)
	statusLabel.SetMarginTop(8)
	vbox.Append(statusLabel)

	baseURL := os.Getenv("SUPERNOVA_API_URL")
	if baseURL == "" {
		baseURL = api.DefaultBaseURL
	}
	client := api.NewClient(baseURL)

	go func() {
		albums, err := client.GetAlbums(50, 0)
		
		glib.IdleAdd(func() {
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Failed to load albums: %v", err))
				return
			}

			statusLabel.SetText(fmt.Sprintf("Successfully loaded %d albums", len(albums)))

			for _, album := range albums {
				row := gtk.NewListBoxRow()
				hbox := gtk.NewBox(gtk.OrientationHorizontal, 10)
				hbox.SetMarginStart(10)
				hbox.SetMarginEnd(10)
				hbox.SetMarginTop(8)
				hbox.SetMarginBottom(8)

				titleLabel := gtk.NewLabel(album.Title)
				titleLabel.SetHAlign(gtk.AlignStart)
				titleLabel.SetHExpand(true)

				yearLabel := gtk.NewLabel(fmt.Sprintf("%d", album.Year))
				yearLabel.SetHAlign(gtk.AlignEnd)
				yearLabel.AddCSSClass("year-label")

				hbox.Append(titleLabel)
				hbox.Append(yearLabel)

				row.SetChild(hbox)
				listBox.Append(row)
			}
		})
	}()

	window.Show()
}
