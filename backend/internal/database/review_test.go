package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/soltros/Supernova/internal/models"
	"path/filepath"
	"sync"
	"testing"
	"strings"
)

func testRepository(t *testing.T) *Repository {
	t.Helper()
	db, err := Init(filepath.Join(t.TempDir(), "library?#.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRepository(db)
}
func TestRegistrationInviteAndConcurrentBootstrap(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	users := []*models.User{}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u, err := r.RegisterUser(ctx, fmt.Sprintf("user%d", i), "hash", "", "")
			if err != nil && !errors.Is(err, ErrInviteRequired) {
				t.Errorf("register: %v", err)
			}
			if u != nil {
				mu.Lock()
				users = append(users, u)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if len(users) != 1 || !users[0].IsAdmin {
		t.Fatalf("bootstrap: %+v", users)
	}
	if _, err := r.RegisterUser(ctx, "wrong", "hash", "bad", "invite"); !errors.Is(err, ErrInviteRequired) {
		t.Fatalf("wrong invite: %v", err)
	}
	u, err := r.RegisterUser(ctx, "invited", "hash", "invite", "invite")
	if err != nil || u.IsAdmin {
		t.Fatalf("invite: %+v %v", u, err)
	}
	persisted, err := r.GetUserByID(ctx, u.ID)
	if err != nil || persisted.IsAdmin {
		t.Fatalf("persisted admin: %+v %v", persisted, err)
	}
}
func TestRescanRepairsDurationAndPreservesOverrides(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	meta := &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: "/music/song.mp3", Format: "ID3v2.4", FileModifiedAt: 100}
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.Exec(`UPDATE tracks SET title='Manual title'`); err != nil {
		t.Fatal(err)
	}
	meta.DurationMs = 123456
	meta.Bitrate = 192
	meta.Format = "mp3"
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	tracks, err := r.GetTracks(ctx, "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].Title != "Manual title" || tracks[0].DurationMs != 123456 || tracks[0].Format != "mp3" {
		t.Fatalf("rescan: %+v", tracks)
	}
	// Replacing a file with an older timestamp must update its tags.
	meta.FileModifiedAt = 50
	meta.Title = "Replacement"
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	track, err := r.GetTrackByID(ctx, tracks[0].ID)
	if err != nil || track.Title != "Replacement" {
		t.Fatalf("older replacement: %+v %v", track, err)
	}
	// AutoTagger intentionally passes no timestamp and must still apply.
	meta.FileModifiedAt = 0
	meta.Title = "Tagged"
	if err := r.UpsertTrack(ctx, meta); err != nil {
		t.Fatal(err)
	}
	track, err = r.GetTrackByID(ctx, tracks[0].ID)
	if err != nil || track.Title != "Tagged" {
		t.Fatalf("tagger: %+v %v", track, err)
	}
}
func TestPlaylistOwnership(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	a, _ := r.CreateUser(ctx, "alice", "hash")
	b, _ := r.CreateUser(ctx, "bob", "hash")
	p, err := r.CreatePlaylist(ctx, a.ID, "Private")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetPlaylistTracks(ctx, b.ID, p.ID); err == nil {
		t.Fatal("private playlist readable")
	}
	if err := r.AddTrackToPlaylist(ctx, b.ID, p.ID, "missing"); err == nil {
		t.Fatal("private playlist writable")
	}
	if err := r.HeartEntity(ctx, b.ID, "playlist", p.ID); err == nil {
		t.Fatal("private playlist can be favorited")
	}
}

func TestV4UpgradePreservesAccountsAndAssignsOldestAdmin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = old.Exec(schemaSQL + `
 INSERT INTO users(id,username,password_hash,created_at) VALUES ('old','owner','hash','2026-01-01 00:00:00'),('new','member','hash','2026-01-02 00:00:00');
 PRAGMA user_version=4;`); err != nil {
		t.Fatal(err)
	}
	old.Close()
	db, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r := NewRepository(db)
	for _, id := range []string{"old", "new"} {
		u, err := r.GetUserByID(context.Background(), id)
		if err != nil || u.IsAdmin != (id == "old") {
			t.Fatalf("upgrade %s: %+v %v", id, u, err)
		}
	}
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil || result != "ok" {
		t.Fatalf("integrity: %s %v", result, err)
	}
	db.Close()
	again, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
}

func TestPlaylistImportRollsBackOnInsertFailure(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	user, err := r.CreateUser(ctx, "owner", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertTrack(ctx, &models.TrackMetadata{Title: "Song", Album: "Album", Artist: "Artist", FilePath: "/music/song.mp3"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.Exec(`CREATE TRIGGER reject_import BEFORE INSERT ON playlist_tracks BEGIN SELECT RAISE(ABORT,'simulated disk failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := r.ImportPlaylistBackup(ctx, user.ID, models.PlaylistBackup{Name: "Import", Tracks: []string{"/music/song.mp3"}}); err == nil {
		t.Fatal("expected insert failure")
	}
	lists, err := r.GetPlaylists(ctx, user.ID)
	if err != nil || len(lists) != 0 {
		t.Fatalf("partial playlist remains: %+v %v", lists, err)
	}
}


func TestPlaylistAllowsRepeatedTracksAndIndexUpdates(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "playlist-owner", "hash")
	if err != nil { t.Fatal(err) }
	for i, path := range []string{"/music/a.mp3", "/music/b.mp3"} {
		if err := r.UpsertTrack(ctx, &models.TrackMetadata{Title: fmt.Sprintf("Track %d", i), Album:"Album", Artist:"Artist", FilePath:path}); err != nil {
			t.Fatal(err)
		}
	}
	tracks, err := r.GetTracks(ctx,"","",10,0)
	if err != nil || len(tracks) != 2 { t.Fatalf("tracks: %+v %v", tracks, err) }
	a, b := tracks[0].ID, tracks[1].ID
	p, err := r.CreatePlaylistWithTracks(ctx,u.ID,"Repeated",[]string{a,b,a})
	if err != nil { t.Fatal(err) }
	got, err := r.GetPlaylistTracks(ctx,u.ID,p.ID)
	if err != nil { t.Fatal(err) }
	if len(got) != 3 || got[0].ID != a || got[1].ID != b || got[2].ID != a {
		t.Fatalf("duplicate ordering lost: %+v", got)
	}
	name := "Renamed"
	if err := r.UpdatePlaylist(ctx,u.ID,p.ID,&name,[]string{b},[]int{1}); err != nil { t.Fatal(err) }
	got, err = r.GetPlaylistTracks(ctx,u.ID,p.ID)
	if err != nil { t.Fatal(err) }
	if len(got) != 3 || got[0].ID != a || got[1].ID != a || got[2].ID != b {
		t.Fatalf("index update wrong: %+v", got)
	}
	before := append([]models.Track(nil), got...)
	if err := r.UpdatePlaylist(ctx,u.ID,p.ID,nil,nil,[]int{99}); err == nil {
		t.Fatal("invalid removal index unexpectedly succeeded")
	}
	got, _ = r.GetPlaylistTracks(ctx,u.ID,p.ID)
	if len(got) != len(before) {
		t.Fatalf("invalid update mutated playlist: before=%d after=%d",len(before),len(got))
	}
	lists, err := r.GetPlaylists(ctx,u.ID)
	if err != nil || len(lists) != 1 || lists[0].Name != "Renamed" {
		t.Fatalf("rename not persisted: %+v %v",lists,err)
	}
}

func TestV5PlaylistMigrationPreservesOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(),"v5.db")
	old, err := sql.Open("sqlite3","file:"+path)
	if err != nil { t.Fatal(err) }
	oldSchema := strings.Replace(schemaSQL,
		`CREATE TABLE IF NOT EXISTS playlist_tracks (
    entry_id TEXT PRIMARY KEY,
    playlist_id TEXT NOT NULL,
    track_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE,
    UNIQUE(playlist_id, position)
);`,
		`CREATE TABLE IF NOT EXISTS playlist_tracks (
    playlist_id TEXT NOT NULL,
    track_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (playlist_id, track_id),
    FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE
);`,1)
	if _,err=old.Exec(oldSchema+`
		ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;
		INSERT INTO users(id,username,password_hash,is_admin) VALUES('u','owner','hash',1);
		INSERT INTO artists(id,name) VALUES('ar','Artist');
		INSERT INTO albums(id,title) VALUES('al','Album');
		INSERT INTO tracks(id,album_id,title,file_path) VALUES('a','al','A','/a'),('b','al','B','/b');
		INSERT INTO playlists(id,user_id,name) VALUES('p','u','List');
		INSERT INTO playlist_tracks(playlist_id,track_id,position) VALUES('p','b',0),('p','a',1);
		PRAGMA user_version=5;`); err != nil { t.Fatal(err) }
	old.Close()
	db,err:=Init(path)
	if err != nil { t.Fatal(err) }
	defer db.Close()
	var version int
	if err:=db.QueryRow("PRAGMA user_version").Scan(&version); err!=nil || version<6 { t.Fatalf("version=%d err=%v",version,err) }
	rows,err:=db.Query(`SELECT track_id,entry_id FROM playlist_tracks ORDER BY position`)
	if err!=nil { t.Fatal(err) }
	defer rows.Close()
	var ids []string
	for rows.Next(){ var track,entry string; if err:=rows.Scan(&track,&entry);err!=nil{t.Fatal(err)}; if entry==""{t.Fatal("missing entry id")}; ids=append(ids,track) }
	if fmt.Sprint(ids)!="[b a]" { t.Fatalf("order changed: %v",ids) }
}


func TestPlaylistBackupV2UsesFingerprintAndWholeSetRollback(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	user, _ := r.CreateUser(ctx, "backup-owner", "hash")
	if err := r.UpsertTrack(ctx, &models.TrackMetadata{
		Title:"Song", Album:"Album", Artist:"Artist", FilePath:"/music/original.mp3",
		FileFingerprint:"fp-song", FileModifiedNs:1, FileSize:100,
	}); err != nil { t.Fatal(err) }
	tracks, _ := r.GetTracks(ctx,"","",10,0)
	p, err := r.CreatePlaylistWithTracks(ctx,user.ID,"Portable",[]string{tracks[0].ID})
	if err != nil { t.Fatal(err) }
	backups, err := r.ExportPlaylists(ctx,user.ID)
	if err != nil || len(backups)!=1 || len(backups[0].TrackRefs)!=1 || backups[0].TrackRefs[0].Fingerprint!="fp-song" {
		t.Fatalf("v2 export: %+v %v",backups,err)
	}
	originalCreated := backups[0].CreatedAt
	if err:=r.DeletePlaylist(ctx,user.ID,p.ID);err!=nil{t.Fatal(err)}
	if _,err:=r.db.Exec(`UPDATE tracks SET file_path='/music/moved.mp3' WHERE id=?`,tracks[0].ID);err!=nil{t.Fatal(err)}
	if err:=r.ImportPlaylistBackups(ctx,user.ID,backups);err!=nil{t.Fatal(err)}
	lists,err:=r.GetPlaylists(ctx,user.ID)
	if err!=nil || len(lists)!=1 || lists[0].CreatedAt!=originalCreated { t.Fatalf("restored: %+v %v",lists,err) }
	restored,err:=r.GetPlaylistTracks(ctx,user.ID,lists[0].ID)
	if err!=nil || len(restored)!=1 || restored[0].ID!=tracks[0].ID { t.Fatalf("track restore: %+v %v",restored,err) }

	if err:=r.DeletePlaylist(ctx,user.ID,lists[0].ID);err!=nil{t.Fatal(err)}
	bad:=[]models.PlaylistBackup{
		{Name:"First",TrackRefs:[]models.PlaylistTrackBackup{{Fingerprint:"fp-song"}}},
		{Name:"Broken",TrackRefs:[]models.PlaylistTrackBackup{{FilePath:"/missing.mp3"}}},
	}
	if err:=r.ImportPlaylistBackups(ctx,user.ID,bad);err==nil{t.Fatal("expected missing track error")}
	lists,_=r.GetPlaylists(ctx,user.ID)
	if len(lists)!=0{t.Fatalf("whole-set rollback failed: %+v",lists)}
}

func TestFavoriteBackupV2RestoresFingerprintMetadataAndTimestamp(t *testing.T) {
	r:=testRepository(t)
	ctx:=context.Background()
	user,_:=r.CreateUser(ctx,"heart-backup","hash")
	if err:=r.UpsertTrack(ctx,&models.TrackMetadata{Title:"Song",Album:"Album",Artist:"Artist",FilePath:"/music/a.mp3",FileFingerprint:"fp-heart",FileModifiedNs:1,FileSize:10});err!=nil{t.Fatal(err)}
	tracks,_:=r.GetTracks(ctx,"","",10,0)
	if err:=r.HeartEntity(ctx,user.ID,"track",tracks[0].ID);err!=nil{t.Fatal(err)}
	if err:=r.HeartEntityWithMetadata(ctx,user.ID,"radio","station-1",[]byte(`{"stationuuid":"station-1","name":"Station"}`));err!=nil{t.Fatal(err)}
	backups,err:=r.ExportHeartsV2(ctx,user.ID)
	if err!=nil{t.Fatal(err)}
	var trackBackup,radioBackup *models.HeartBackup
	for i:=range backups{
		switch backups[i].EntityType{case "track":trackBackup=&backups[i];case "radio":radioBackup=&backups[i]}
	}
	if trackBackup==nil || trackBackup.ReferenceType!="fingerprint" || trackBackup.Reference!="fp-heart"{t.Fatalf("track backup: %+v",trackBackup)}
	if radioBackup==nil || len(radioBackup.Metadata)==0{t.Fatalf("radio backup: %+v",radioBackup)}
	created:=trackBackup.CreatedAt
	if _,err:=r.db.Exec(`DELETE FROM hearts WHERE user_id=?`,user.ID);err!=nil{t.Fatal(err)}
	if _,err:=r.db.Exec(`UPDATE tracks SET file_path='/music/moved-a.mp3' WHERE id=?`,tracks[0].ID);err!=nil{t.Fatal(err)}
	if err:=r.ImportHeartBackupsV2(ctx,user.ID,backups);err!=nil{t.Fatal(err)}
	hearts,err:=r.GetAllHearts(ctx,user.ID)
	if err!=nil || len(hearts)!=2{t.Fatalf("hearts: %+v %v",hearts,err)}
	var restoredCreated string
	if err:=r.db.QueryRow(`SELECT created_at FROM hearts WHERE user_id=? AND entity_type='track'`,user.ID).Scan(&restoredCreated);err!=nil{t.Fatal(err)}
	if restoredCreated!=created{t.Fatalf("timestamp changed: %q != %q",restoredCreated,created)}
	radio,_,err:=r.GetExternalHeartMetadata(ctx,user.ID)
	if err!=nil || len(radio)!=1 || !strings.Contains(string(radio[0]),"Station"){t.Fatalf("radio metadata: %s %v",radio,err)}
}
