package database

import (
	"context"

	"github.com/soltros/Supernova/internal/models"
)

// GetArtists returns all artists in the library sorted alphabetically, with pagination.
func (r *Repository) GetArtists(ctx context.Context, limit, offset int) ([]models.Artist, error) {
	query := `SELECT id, name, musicbrainz_id, image_url, bio FROM artists ORDER BY name ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var a models.Artist
		if err := rows.Scan(&a.ID, &a.Name, &a.MusicBrainzID, &a.ImageURL, &a.Bio); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	
	if artists == nil {
		return []models.Artist{}, nil
	}
	return artists, nil
}

// GetAlbums returns all albums in the library sorted alphabetically, with pagination.
func (r *Repository) GetAlbums(ctx context.Context, limit, offset int) ([]models.Album, error) {
	query := `SELECT id, title, release_year, musicbrainz_id, cover_art_path FROM albums ORDER BY title ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var a models.Album
		if err := rows.Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	
	if albums == nil {
		return []models.Album{}, nil
	}
	return albums, nil
}

// GetTracks returns tracks from the library. It can optionally be filtered by a specific album.
func (r *Repository) GetTracks(ctx context.Context, albumID string, limit, offset int) ([]models.Track, error) {
	query := `SELECT id, album_id, title, track_number, disc_number, duration_ms, format, bitrate FROM tracks`
	args := []any{}
	
	if albumID != "" {
		query += ` WHERE album_id = ?`
		args = append(args, albumID)
	}
	query += ` ORDER BY disc_number ASC, track_number ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	
	if tracks == nil {
		return []models.Track{}, nil
	}
	return tracks, nil
}

// UnenrichedAlbum is a DTO for the background worker
type UnenrichedAlbum struct {
	AlbumID    string
	AlbumTitle string
	ArtistID   string
	ArtistName string
	TrackTitle string // A sample track to help MusicBrainz identify the release
}

// GetUnenrichedAlbums safely fetches albums that haven't been checked against MusicBrainz yet.
func (r *Repository) GetUnenrichedAlbums(ctx context.Context, limit int) ([]UnenrichedAlbum, error) {
	query := `
		SELECT 
			a.id, a.title, 
			art.id, art.name,
			t.title
		FROM albums a
		JOIN album_artists aa ON a.id = aa.album_id
		JOIN artists art ON aa.artist_id = art.id
		JOIN tracks t ON a.id = t.album_id
		WHERE a.musicbrainz_id IS NULL OR a.musicbrainz_id = ''
		GROUP BY a.id
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []UnenrichedAlbum
	for rows.Next() {
		var a UnenrichedAlbum
		if err := rows.Scan(&a.AlbumID, &a.AlbumTitle, &a.ArtistID, &a.ArtistName, &a.TrackTitle); err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, nil
}

// UpdateMBIDs securely saves the official MusicBrainz IDs found by the background worker.
func (r *Repository) UpdateMBIDs(ctx context.Context, albumID, albumMBID, artistID, artistMBID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if albumMBID != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE albums SET musicbrainz_id = ? WHERE id = ?`, albumMBID, albumID); err != nil {
			return err
		}
	}
	if artistMBID != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE artists SET musicbrainz_id = ? WHERE id = ?`, artistMBID, artistID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetTrackByID fetches a single track by its ID to locate its file path for streaming.
func (r *Repository) GetTrackByID(ctx context.Context, id string) (*models.Track, error) {
	query := `SELECT id, album_id, title, track_number, disc_number, duration_ms, file_path, format, bitrate FROM tracks WHERE id = ?`
	var t models.Track
	err := r.db.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.FilePath, &t.Format, &t.Bitrate)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetAlbumByID fetches a single album by its ID.
func (r *Repository) GetAlbumByID(ctx context.Context, id string) (*models.Album, error) {
	query := `SELECT id, title, release_year, musicbrainz_id, cover_art_path FROM albums WHERE id = ?`
	var a models.Album
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// HeartEntity adds a heart for an entity, safely generating its own UUID
func (r *Repository) HeartEntity(ctx context.Context, entityType, entityID string) error {
	id := generateUUID()
	query := `INSERT INTO hearts (id, entity_type, entity_id) VALUES (?, ?, ?) ON CONFLICT(entity_type, entity_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, id, entityType, entityID)
	return err
}

// UnheartEntity removes a heart from an entity
func (r *Repository) UnheartEntity(ctx context.Context, entityType, entityID string) error {
	query := `DELETE FROM hearts WHERE entity_type = ? AND entity_id = ?`
	_, err := r.db.ExecContext(ctx, query, entityType, entityID)
	return err
}

// GetAllHearts retrieves all hearts for the frontend context
func (r *Repository) GetAllHearts(ctx context.Context) ([]models.Heart, error) {
	query := `SELECT id, entity_type, entity_id, created_at FROM hearts`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hearts []models.Heart
	for rows.Next() {
		var h models.Heart
		if err := rows.Scan(&h.ID, &h.EntityType, &h.EntityID, &h.CreatedAt); err != nil {
			return nil, err
		}
		hearts = append(hearts, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hearts, nil
}

// ExportHearts performs a robust JOIN to export permanent file_paths rather than temporary UUIDs
func (r *Repository) ExportHearts(ctx context.Context) ([]models.HeartBackup, error) {
	query := `
		SELECT h.entity_type, 
		       CASE 
		           WHEN h.entity_type = 'track' THEN t.file_path 
		           WHEN h.entity_type = 'album' THEN a.title 
		           ELSE '' 
		       END as reference,
		       h.created_at
		FROM hearts h
		LEFT JOIN tracks t ON h.entity_type = 'track' AND h.entity_id = t.id
		LEFT JOIN albums a ON h.entity_type = 'album' AND h.entity_id = a.id
		WHERE reference != '' AND reference IS NOT NULL
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []models.HeartBackup
	for rows.Next() {
		var b models.HeartBackup
		if err := rows.Scan(&b.EntityType, &b.Reference, &b.CreatedAt); err != nil {
			return nil, err
		}
		backups = append(backups, b)
	}
	return backups, nil
}

// ImportHeartBackup safely resolves the permanent reference back to the local database UUID
func (r *Repository) ImportHeartBackup(ctx context.Context, b models.HeartBackup) error {
	var entityID string
	if b.EntityType == "track" {
		err := r.db.QueryRowContext(ctx, `SELECT id FROM tracks WHERE file_path = ?`, b.Reference).Scan(&entityID)
		if err != nil {
			return err // Silently skip if track no longer exists in this user's library
		}
	} else if b.EntityType == "album" {
		err := r.db.QueryRowContext(ctx, `SELECT id FROM albums WHERE title = ?`, b.Reference).Scan(&entityID)
		if err != nil {
			return err
		}
	} else {
		return nil
	}

	return r.HeartEntity(ctx, b.EntityType, entityID)
}

// ScrobbleTrack records a successful track play into the user's history
func (r *Repository) ScrobbleTrack(ctx context.Context, trackID string) error {
	id := generateUUID()
	query := `INSERT INTO scrobbles (id, track_id) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, trackID)
	return err
}

// GetRecentScrobbles returns the user's most recently listened to tracks
func (r *Repository) GetRecentScrobbles(ctx context.Context, limit int) ([]models.Track, error) {
	query := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate
		FROM tracks t
		JOIN scrobbles s ON t.id = s.track_id
		ORDER BY s.listened_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}
