package player

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/soltros/Supernova/tui-client/models"
)

const (
	mprisName = "org.mpris.MediaPlayer2.SupernovaTUI"
	mprisPath = "/org/mpris/MediaPlayer2"
)

type mprisPlayer struct{}

func (p mprisPlayer) Next() *dbus.Error {
	Next()
	return nil
}

func (p mprisPlayer) Previous() *dbus.Error {
	Prev()
	return nil
}

func (p mprisPlayer) Pause() *dbus.Error {
	Stop()
	return nil
}

func (p mprisPlayer) PlayPause() *dbus.Error {
	TogglePause()
	return nil
}

func (p mprisPlayer) Stop() *dbus.Error {
	Stop()
	return nil
}

func (p mprisPlayer) Play() *dbus.Error {
	if CurrentState != Playing && len(Queue) > 0 {
		PlayTrack(CurrentIndex)
	}
	return nil
}

var props *prop.Properties

func InitMPRIS() {
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Println("Failed to connect to session bus:", err)
		return
	}
	reply, err := conn.RequestName(mprisName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return
	}

	player := mprisPlayer{}
	conn.Export(player, mprisPath, "org.mpris.MediaPlayer2.Player")

	props, err = prop.Export(conn, mprisPath, prop.Map{
		"org.mpris.MediaPlayer2": {
			"CanQuit":      {false, true, prop.EmitTrue, nil},
			"CanRaise":     {false, true, prop.EmitTrue, nil},
			"HasTrackList": {false, true, prop.EmitTrue, nil},
			"Identity":     {"Supernova TUI", true, prop.EmitTrue, nil},
			"SupportedUriSchemes": {[]string{"http"}, true, prop.EmitTrue, nil},
			"SupportedMimeTypes":  {[]string{"audio/mpeg"}, true, prop.EmitTrue, nil},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {"Stopped", true, prop.EmitTrue, nil},
			"CanGoNext":      {true, true, prop.EmitTrue, nil},
			"CanGoPrevious":  {true, true, prop.EmitTrue, nil},
			"CanPlay":        {true, true, prop.EmitTrue, nil},
			"CanPause":       {true, true, prop.EmitTrue, nil},
			"CanSeek":        {false, true, prop.EmitTrue, nil},
			"CanControl":     {true, true, prop.EmitTrue, nil},
			"Metadata":       {map[string]dbus.Variant{}, true, prop.EmitTrue, nil},
		},
	})
	
	node := &introspect.Node{
		Name: mprisPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: "org.mpris.MediaPlayer2",
				Methods: []introspect.Method{
					{Name: "Raise"},
					{Name: "Quit"},
				},
				Properties: []introspect.Property{
					{Name: "CanQuit", Type: "b", Access: "read"},
					{Name: "CanRaise", Type: "b", Access: "read"},
					{Name: "HasTrackList", Type: "b", Access: "read"},
					{Name: "Identity", Type: "s", Access: "read"},
				},
			},
			{
				Name: "org.mpris.MediaPlayer2.Player",
				Methods: []introspect.Method{
					{Name: "Next"},
					{Name: "Previous"},
					{Name: "Pause"},
					{Name: "PlayPause"},
					{Name: "Stop"},
					{Name: "Play"},
				},
				Properties: []introspect.Property{
					{Name: "PlaybackStatus", Type: "s", Access: "read"},
					{Name: "Metadata", Type: "a{sv}", Access: "read"},
				},
			},
		},
	}
	conn.Export(introspect.NewIntrospectable(node), mprisPath, "org.freedesktop.DBus.Introspectable")
}

func UpdateMPRIS(track models.Track, state State) {
	if props == nil {
		return
	}
	
	status := "Stopped"
	if state == Playing {
		status = "Playing"
	}
	
	props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", status)
	
	metadata := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/supernova/track/" + track.ID)),
		"xesam:title":   dbus.MakeVariant(track.Title),
		"mpris:length":  dbus.MakeVariant(int64(track.Duration * 1000)), // microseconds
	}
	props.SetMust("org.mpris.MediaPlayer2.Player", "Metadata", metadata)
}
