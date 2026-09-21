package subsonic

import (
	"context"
	"encoding/json"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormAuthenticationExtensionsAndScrobble(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repo.CreateUser(ctx, "listener", string(hash))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertTrack(ctx, &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: "/music/song.flac", Format: "flac", DurationMs: 120000}); err != nil {
		t.Fatal(err)
	}
	tracks, err := repo.GetTracks(ctx, "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := &SubsonicPlugin{repo: repo}
	mux := http.NewServeMux()
	p.SetupRoutes(mux)
	request := func(path string, values url.Values) map[string]any {
		values.Set("f", "json")
		r := httptest.NewRequest("POST", path, strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var data map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("%s: %s %v", path, w.Body, err)
		}
		return data["subsonic-response"].(map[string]any)
	}
	auth := func() url.Values { return url.Values{"u": {"listener"}, "p": {"password"}} }
	if got := request("/rest/ping.view", auth())["status"]; got != "ok" {
		t.Fatalf("POST auth: %v", got)
	}
	extensions := request("/rest/getOpenSubsonicExtensions", url.Values{})
	if extensions["status"] != "ok" {
		t.Fatal("extensions need auth")
	}
	if _, ok := extensions["openSubsonicExtensions"].([]any); !ok {
		t.Fatalf("invalid extensions shape: %+v", extensions)
	}
	values := auth()
	values.Set("id", tracks[0].ID)
	values.Set("time", "1700000000000")
	if result := request("/rest/scrobble", values); result["status"] != "ok" {
		t.Fatalf("scrobble: %+v", result)
	}
	var count int
	var timestamp string
	if err := db.QueryRow("SELECT COUNT(*), listened_at FROM scrobbles WHERE user_id=?", user.ID).Scan(&count, &timestamp); err != nil {
		t.Fatal(err)
	}
	if count != 1 || !strings.HasPrefix(timestamp, "2023-11-14") {
		t.Fatalf("scrobble not recorded: %d %s", count, timestamp)
	}
	values.Set("submission", "false")
	request("/rest/scrobble", values)
	if err := db.QueryRow("SELECT COUNT(*) FROM scrobbles").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("now playing inserted a play")
	}
	random := request("/rest/getRandomSongs", auth())["randomSongs"].(map[string]any)["song"].([]any)
	if len(random) != 1 {
		t.Fatal("random songs still empty")
	}
}


func TestPlaylistWritesAreTransactionalAndIndexAware(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(),"playlist.db"))
	if err != nil { t.Fatal(err) }
	defer db.Close()
	repo := database.NewRepository(db)
	ctx := context.Background()
	hash,_ := bcrypt.GenerateFromPassword([]byte("password"),bcrypt.MinCost)
	user,err := repo.CreateUser(ctx,"listener",string(hash))
	if err != nil { t.Fatal(err) }
	for _, meta := range []*models.TrackMetadata{
		{Title:"A",Album:"Album",Artist:"Artist",FilePath:"/music/a.mp3"},
		{Title:"B",Album:"Album",Artist:"Artist",FilePath:"/music/b.mp3"},
	} {
		if err:=repo.UpsertTrack(ctx,meta);err!=nil{t.Fatal(err)}
	}
	tracks,_:=repo.GetTracks(ctx,"","",10,0)
	if len(tracks)!=2{t.Fatalf("tracks=%d",len(tracks))}
	p:=&SubsonicPlugin{repo:repo}
	mux:=http.NewServeMux(); p.SetupRoutes(mux)
	call:=func(path string,v url.Values) map[string]any{
		v.Set("u","listener");v.Set("p","password");v.Set("f","json")
		req:=httptest.NewRequest("POST",path,strings.NewReader(v.Encode()))
		req.Header.Set("Content-Type","application/x-www-form-urlencoded")
		w:=httptest.NewRecorder();mux.ServeHTTP(w,req)
		var out map[string]any
		if err:=json.Unmarshal(w.Body.Bytes(),&out);err!=nil{t.Fatalf("%s: %s",path,w.Body.String())}
		return out["subsonic-response"].(map[string]any)
	}
	create:=url.Values{"name":{"List"},"songId":{tracks[0].ID,tracks[1].ID,tracks[0].ID}}
	resp:=call("/rest/createPlaylist",create)
	if resp["status"]!="ok"{t.Fatalf("create: %+v",resp)}
	pl:=resp["playlist"].(map[string]any)
	if int(pl["songCount"].(float64))!=3{t.Fatalf("count: %+v",pl)}
	pid:=pl["id"].(string)

	update:=url.Values{"playlistId":{pid},"name":{"Renamed"},"songIndexToRemove":{"1"},"songIdToAdd":{tracks[1].ID}}
	if got:=call("/rest/updatePlaylist",update);got["status"]!="ok"{t.Fatalf("update: %+v",got)}
	got,err:=repo.GetPlaylistTracks(ctx,user.ID,pid)
	if err!=nil{t.Fatal(err)}
	if len(got)!=3 || got[0].ID!=tracks[0].ID || got[1].ID!=tracks[0].ID || got[2].ID!=tracks[1].ID {
		t.Fatalf("playlist state: %+v",got)
	}

	bad:=url.Values{"playlistId":{pid},"songIndexToRemove":{"99"},"songIdToAdd":{tracks[0].ID}}
	if got:=call("/rest/updatePlaylist",bad);got["status"]!="failed"{t.Fatalf("bad update succeeded: %+v",got)}
	after,_:=repo.GetPlaylistTracks(ctx,user.ID,pid)
	if len(after)!=3{t.Fatalf("bad update partially applied: %+v",after)}

	replace:=url.Values{"playlistId":{pid},"songId":{tracks[1].ID,tracks[1].ID}}
	if got:=call("/rest/createPlaylist",replace);got["status"]!="ok"{t.Fatalf("replace: %+v",got)}
	after,_=repo.GetPlaylistTracks(ctx,user.ID,pid)
	if len(after)!=2 || after[0].ID!=tracks[1].ID || after[1].ID!=tracks[1].ID {
		t.Fatalf("replace did not preserve duplicate: %+v",after)
	}
}
