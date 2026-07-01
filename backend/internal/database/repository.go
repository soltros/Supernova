package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
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

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback. If tx.Commit() succeeds later, this becomes a safe no-op.
	defer tx.Rollback()

	// 1. Resolve Primary Artist
	artistName := meta.Artist
	if artistName == "" {
		artistName = "Unknown Artist"
	}
	artistID, err := r.upsertArtist(tx, artistName, meta.ArtistMBID, "", "")
	if err != nil {
		return err
	}

	// 2. Resolve Album Artist (Fallback to primary artist if empty)
	albumArtistName := meta.AlbumArtist
	if albumArtistName == "" {
		albumArtistName = artistName
	}
	albumArtistID, err := r.upsertArtist(tx, albumArtistName, "", "", "")
	if err != nil {
		return err
	}

	// 3. Resolve Album
	albumTitle := meta.Album
	if albumTitle == "" {
		albumTitle = "Unknown Album"
	}
	// BUG-3: Pass albumArtistID so that albums with the same title from different artists are not merged
	albumID, err := r.upsertAlbum(tx, albumTitle, meta.AlbumMBID, meta.Year, meta.CoverArtPath, albumArtistID)
	if err != nil {
		return err
	}

	// 4. Link Album to Album Artist
	err = r.linkAlbumArtist(tx, albumID, albumArtistID, "primary")
	if err != nil {
		return err
	}

	// 5. Insert the Track itself (Upsert on FilePath conflict)
	var trackID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO tracks (id, album_id, title, track_number, disc_number, duration_ms, file_path, format, bitrate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			title=excluded.title,
			album_id=excluded.album_id,
			track_number=excluded.track_number,
			disc_number=excluded.disc_number,
			duration_ms=excluded.duration_ms,
			bitrate=excluded.bitrate
		RETURNING id
	`, generateUUID(), albumID, meta.Title, meta.TrackNumber, meta.DiscNumber, meta.DurationMs, meta.FilePath, meta.Format, meta.Bitrate).Scan(&trackID)
	
	if err != nil {
		return fmt.Errorf("failed to insert track: %w", err)
	}

	// 6. Link Track to Primary Artist
	err = r.linkTrackArtist(tx, trackID, artistID, "primary")
	if err != nil {
		return err
	}

	// If everything succeeded, commit the transaction to disk
	return tx.Commit()
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
		SET enriched = 0, image_url = '', bio = ''
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
