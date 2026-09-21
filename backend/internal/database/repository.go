package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/soltros/Supernova/internal/models"
)

// Repository provides all database access methods for Supernova
type Repository struct {
	db      *DB
	writeMu sync.Mutex
}

func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *DB {
	return r.db
}

// UpsertTrack safely inserts or updates a track and its relational metadata.
// It uses a mutex to serialize writes to SQLite, enabling extreme concurrency for scanning
// without triggering "database is locked" timeouts.
func (r *Repository) UpsertTrack(ctx context.Context, meta *models.TrackMetadata) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	var ignored string
	err := r.db.QueryRowContext(ctx, "SELECT file_path FROM ignored_files WHERE file_path = ?", meta.FilePath).Scan(&ignored)
	if err == nil { return nil }
	if err != nil && !errors.Is(err, sql.ErrNoRows) { return err }

	var existingID string
	var existingMod, existingNs, existingSize int64
	err = r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(file_modified_at,0), COALESCE(file_modified_ns,0), COALESCE(file_size,0)
		FROM tracks WHERE file_path=?
	`, meta.FilePath).Scan(&existingID,&existingMod,&existingNs,&existingSize)
	if err != nil && !errors.Is(err,sql.ErrNoRows) { return err }

	// If this path is new, a unique fingerprint whose old path disappeared is a move,
	// not a new logical track. Move the existing row first so all user references survive.
	if errors.Is(err,sql.ErrNoRows) && meta.FileFingerprint != "" {
		rows,qerr:=r.db.QueryContext(ctx,`
			SELECT id,file_path FROM tracks WHERE file_fingerprint=? ORDER BY id LIMIT 2
		`,meta.FileFingerprint)
		if qerr!=nil{return qerr}
		type candidate struct{id,path string}
		var candidates []candidate
		for rows.Next(){var c candidate;if qerr=rows.Scan(&c.id,&c.path);qerr!=nil{rows.Close();return qerr};candidates=append(candidates,c)}
		if qerr=rows.Close();qerr!=nil{return qerr}
		if len(candidates)==1 {
			if _,statErr:=os.Stat(candidates[0].path);errors.Is(statErr,os.ErrNotExist) {
				if _,qerr=r.db.ExecContext(ctx,`
					UPDATE tracks SET file_path=?,file_modified_at=?,file_modified_ns=?,file_size=?,file_fingerprint=?
					WHERE id=?
				`,meta.FilePath,meta.FileModifiedAt,meta.FileModifiedNs,meta.FileSize,meta.FileFingerprint,candidates[0].id);qerr!=nil{return qerr}
				existingID=candidates[0].id
				existingMod=meta.FileModifiedAt
				existingNs=0 // force metadata refresh after relocation
				existingSize=0
				err=nil
			}
		}
	}

	unchanged := err == nil && meta.FileModifiedNs > 0 && existingNs == meta.FileModifiedNs && existingSize == meta.FileSize
	if unchanged {
		_, err := r.db.ExecContext(ctx, `
			UPDATE tracks SET
				duration_ms=CASE WHEN ?>0 THEN ? ELSE duration_ms END,
				bitrate=CASE WHEN ?>0 THEN ? ELSE bitrate END,
				format=CASE WHEN ?!='' THEN ? ELSE format END,
				file_fingerprint=CASE WHEN ?!='' THEN ? ELSE file_fingerprint END
			WHERE id=?
		`,meta.DurationMs,meta.DurationMs,meta.Bitrate,meta.Bitrate,meta.Format,meta.Format,meta.FileFingerprint,meta.FileFingerprint,existingID)
		return err
	}
	// Older databases/plugins may supply only second-resolution timestamps.
	if err==nil && meta.FileModifiedNs==0 && meta.FileModifiedAt>0 && existingMod==meta.FileModifiedAt {
		_,err:=r.db.ExecContext(ctx,`UPDATE tracks SET duration_ms=CASE WHEN ?>0 THEN ? ELSE duration_ms END,bitrate=CASE WHEN ?>0 THEN ? ELSE bitrate END,format=CASE WHEN ?!='' THEN ? ELSE format END WHERE id=?`,
			meta.DurationMs,meta.DurationMs,meta.Bitrate,meta.Bitrate,meta.Format,meta.Format,existingID)
		return err
	}

	tx,err:=r.db.BeginTx(ctx,nil)
	if err!=nil{return fmt.Errorf("failed to begin transaction: %w",err)}
	defer tx.Rollback()

	artistName:=meta.Artist
	if artistName==""{artistName="Unknown Artist"}
	artistID,err:=r.upsertArtist(tx,artistName,meta.ArtistMBID,"","")
	if err!=nil{return err}

	albumArtistName:=meta.AlbumArtist
	if albumArtistName==""{albumArtistName=artistName}
	albumArtistID,err:=r.upsertArtist(tx,albumArtistName,"","","")
	if err!=nil{return err}

	albumTitle:=meta.Album
	if albumTitle==""{
		folder:=filepath.Base(filepath.Dir(meta.FilePath))
		if folder=="" || folder=="." || folder==string(filepath.Separator){folder="Unknown Album"}
		albumTitle=folder
	}
	albumID,err:=r.upsertAlbum(tx,albumTitle,meta.AlbumMBID,meta.Year,meta.CoverArtPath,albumArtistID)
	if err!=nil{return err}
	if err=r.linkAlbumArtist(tx,albumID,albumArtistID,"primary");err!=nil{return err}

	var trackID string
	err=tx.QueryRowContext(ctx,`
		INSERT INTO tracks(id,album_id,title,track_number,disc_number,duration_ms,file_path,format,bitrate,file_modified_at,file_modified_ns,file_size,file_fingerprint)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(file_path) DO UPDATE SET
			title=excluded.title,album_id=excluded.album_id,track_number=excluded.track_number,
			disc_number=excluded.disc_number,duration_ms=excluded.duration_ms,bitrate=excluded.bitrate,
			format=excluded.format,
			file_modified_at=CASE WHEN excluded.file_modified_at>0 THEN excluded.file_modified_at ELSE tracks.file_modified_at END,
			file_modified_ns=CASE WHEN excluded.file_modified_ns>0 THEN excluded.file_modified_ns ELSE tracks.file_modified_ns END,
			file_size=CASE WHEN excluded.file_size>0 THEN excluded.file_size ELSE tracks.file_size END,
			file_fingerprint=CASE WHEN excluded.file_fingerprint!='' THEN excluded.file_fingerprint ELSE tracks.file_fingerprint END
		RETURNING id
	`,generateUUID(),albumID,meta.Title,meta.TrackNumber,meta.DiscNumber,meta.DurationMs,meta.FilePath,meta.Format,meta.Bitrate,meta.FileModifiedAt,meta.FileModifiedNs,meta.FileSize,meta.FileFingerprint).Scan(&trackID)
	if err!=nil{return fmt.Errorf("failed to insert track: %w",err)}
	if _,err=tx.ExecContext(ctx,"DELETE FROM track_artists WHERE track_id=?",trackID);err!=nil{return err}
	if err=r.linkTrackArtist(tx,trackID,artistID,"primary");err!=nil{return err}
	return tx.Commit()
}

func (r *Repository) RemoveTrackByPath(ctx context.Context, path string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_,err:=r.db.ExecContext(ctx,`DELETE FROM tracks WHERE file_path=?`,path)
	return err
}

// ReconcileLibraryPaths removes stale rows for files that disappeared while the watcher was offline.
// Moved files retain their IDs because UpsertTrack relocates unique fingerprints before this runs.
func (r *Repository) ReconcileLibraryPaths(ctx context.Context, mediaRoot string, seen map[string]struct{}) error {
	rows,err:=r.db.QueryContext(ctx,`SELECT file_path FROM tracks`)
	if err!=nil{return err}
	var stale []string
	for rows.Next(){
		var path string
		if err:=rows.Scan(&path);err!=nil{rows.Close();return err}
		rel,relErr:=filepath.Rel(mediaRoot,path)
		if relErr!=nil || rel==".." || filepath.IsAbs(rel) || (len(rel)>3 && rel[:3]==".."+string(filepath.Separator)){continue}
		if _,ok:=seen[path];!ok{
			if _,statErr:=os.Stat(path);errors.Is(statErr,os.ErrNotExist){stale=append(stale,path)}
		}
	}
	if err:=rows.Close();err!=nil{return err}
	for _,path:=range stale{if err:=r.RemoveTrackByPath(ctx,path);err!=nil{return err}}
	return nil
}

// upsertArtist looks up an artist by name. If they don't exist, it creates them.
func (r *Repository) upsertArtist(tx *sql.Tx, name, mbid, imageURL, bio string) (string, error) {
	var id string
	// Using name matching here. If we have MBID we could prefer it, but name is a safe generic fallback.
	err := tx.QueryRow(`SELECT id FROM artists WHERE name = ? LIMIT 1`, name).Scan(&id)

	if err == sql.ErrNoRows {
		id = generateUUID()
		_, err = tx.Exec(`
			INSERT INTO artists (id, name, musicbrainz_id, image_url, bio)
			VALUES (?, ?, ?, ?, ?)
		`, id, name, mbid, imageURL, bio)
		if err != nil {
			return "", fmt.Errorf("failed to insert artist: %w", err)
		}
	} else if err != nil {
		return "", err
	}
	return id, nil
}

// upsertAlbum looks up an album by (title, album_artist_id). If it doesn't exist, it creates it.
// BUG-3 fix: matching only on title caused different artists' albums with the same title (e.g. "Greatest Hits") to merge.
func (r *Repository) upsertAlbum(tx *sql.Tx, title, mbid string, year int, coverArtPath string, albumArtistID string) (string, error) {
	var id string
	err := tx.QueryRow(`
		SELECT a.id FROM albums a
		JOIN album_artists aa ON a.id = aa.album_id
		WHERE a.title = ? AND aa.artist_id = ? LIMIT 1
	`, title, albumArtistID).Scan(&id)

	if err == sql.ErrNoRows {
		id = generateUUID()
		_, err = tx.Exec(`
			INSERT INTO albums (id, title, release_year, musicbrainz_id, cover_art_path)
			VALUES (?, ?, ?, ?, ?)
		`, id, title, year, mbid, coverArtPath)
		if err != nil {
			return "", fmt.Errorf("failed to insert album: %w", err)
		}
	} else if err != nil {
		return "", err
	} else {
		// If the album exists but is missing cover art, try to update it
		if coverArtPath != "" {
			_, _ = tx.Exec(`UPDATE albums SET cover_art_path = ? WHERE id = ? AND (cover_art_path IS NULL OR cover_art_path = '')`, coverArtPath, id)
		}
	}
	return id, nil
}

func (r *Repository) linkTrackArtist(tx *sql.Tx, trackID, artistID, role string) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO track_artists (track_id, artist_id)
		VALUES (?, ?)
	`, trackID, artistID)
	return err
}

func (r *Repository) linkAlbumArtist(tx *sql.Tx, albumID, artistID, role string) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO album_artists (album_id, artist_id, role)
		VALUES (?, ?, ?)
	`, albumID, artistID, role)
	return err
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// CONC-3: crypto/rand failure would produce all-zero UUIDs causing PK collisions — unrecoverable
		panic("crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ResetArtistEnrichment resets any cached/enriched artist fields so enrichment can be rerun.
// LEAK-4 fix: use ExecContext so the operation respects request cancellation.
func (r *Repository) ResetArtistEnrichment(ctx context.Context) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	_, err := r.db.ExecContext(ctx, `
		UPDATE artists 
		SET image_url = '', bio = ''
	`)
	return err
}

// GetAlbumsByArtistID returns all albums for a given artist ID
func (r *Repository) GetAlbumsByArtistID(ctx context.Context, artistID string) ([]models.Album, error) {
	query := `
		SELECT DISTINCT a.id, a.title, a.release_year, a.musicbrainz_id, a.cover_art_path
		FROM albums a
		JOIN album_artists aa ON a.id = aa.album_id
		WHERE aa.artist_id = ?
		ORDER BY a.release_year DESC, a.title ASC
	`
	rows, err := r.db.QueryContext(ctx, query, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var album models.Album
		if err := rows.Scan(&album.ID, &album.Title, &album.ReleaseYear, &album.MusicBrainzID, &album.CoverArtPath); err != nil {
			return nil, err
		}
		albums = append(albums, album)
	}
	return albums, rows.Err()
}

// GetTracksByAlbumID returns all tracks for a given album ID
func (r *Repository) GetTracksByAlbumID(ctx context.Context, albumID string) ([]models.Track, error) {
	query := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.file_path, t.format, t.bitrate
		FROM tracks t
		WHERE t.album_id = ?
		ORDER BY t.disc_number ASC, t.track_number ASC
	`
	rows, err := r.db.QueryContext(ctx, query, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var track models.Track
		if err := rows.Scan(&track.ID, &track.AlbumID, &track.Title, &track.TrackNumber, &track.DiscNumber, &track.DurationMs, &track.FilePath, &track.Format, &track.Bitrate); err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, rows.Err()
}
