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

func (r *Repository) GetArtistByID(ctx context.Context, id string) (models.Artist, error) {
	query := `SELECT id, name, musicbrainz_id, image_url, bio FROM artists WHERE id = ?`
	var a models.Artist
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.Name, &a.MusicBrainzID, &a.ImageURL, &a.Bio)
	return a, err
}

func (r *Repository) UpdateArtistInfo(ctx context.Context, id, imageUrl, bio string) error {
	query := `UPDATE artists SET image_url = ?, bio = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, imageUrl, bio, id)
	return err
}

func (r *Repository) GetUnenrichedArtists(ctx context.Context, limit int) ([]models.Artist, error) {
	query := `SELECT id, name, musicbrainz_id, image_url, bio FROM artists WHERE image_url = '' OR image_url IS NULL LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, limit)
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
	return artists, nil
}

// GetAlbums returns all albums in the library sorted alphabetically, with pagination.
func (r *Repository) GetAlbums(ctx context.Context, artistID string, limit, offset int) ([]models.Album, error) {
	query := `
		SELECT a.id, a.title, a.release_year, a.musicbrainz_id, a.cover_art_path, art.id, art.name
		FROM albums a
		LEFT JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
		LEFT JOIN artists art ON aa.artist_id = art.id
	`
	args := []any{}
	if artistID != "" {
		query += ` WHERE aa.artist_id = ?`
		args = append(args, artistID)
	}
	query += ` ORDER BY a.title ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var a models.Album
		var artID, artName *string
		if err := rows.Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath, &artID, &artName); err != nil {
			return nil, err
		}
		if artID != nil {
			a.ArtistID = *artID
		}
		if artName != nil {
			a.ArtistName = *artName
		}
		albums = append(albums, a)
	}
	
	if albums == nil {
		return []models.Album{}, nil
	}
	return albums, nil
}

// GetTracks returns tracks from the library. It can optionally be filtered by a specific album.
func (r *Repository) GetTracks(ctx context.Context, albumID string, artistID string, limit, offset int) ([]models.Track, error) {
	query := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, art.id, art.name
		FROM tracks t
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists art ON ta.artist_id = art.id
	`
	args := []any{}
	
	where := []string{}
	if albumID != "" {
		where = append(where, `t.album_id = ?`)
		args = append(args, albumID)
	}
	if artistID != "" {
		where = append(where, `ta.artist_id = ?`)
		args = append(args, artistID)
	}

	if len(where) > 0 {
		query += " WHERE " + where[0]
		if len(where) > 1 {
			query += " AND " + where[1]
		}
	}

	query += ` GROUP BY t.id ORDER BY t.disc_number ASC, t.track_number ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		var artID, artName *string
		if err := rows.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate, &artID, &artName); err != nil {
			return nil, err
		}
		if artID != nil {
			t.ArtistID = *artID
		}
		if artName != nil {
			t.ArtistName = *artName
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
	query := `
		SELECT a.id, a.title, a.release_year, a.musicbrainz_id, a.cover_art_path, art.id, art.name
		FROM albums a
		LEFT JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
		LEFT JOIN artists art ON aa.artist_id = art.id
		WHERE a.id = ?
	`
	var a models.Album
	var artID, artName *string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath, &artID, &artName)
	if err != nil {
		return nil, err
	}
	if artID != nil {
		a.ArtistID = *artID
	}
	if artName != nil {
		a.ArtistName = *artName
	}
	return &a, nil
}

// HeartEntity saves a favorite for a specific user
func (r *Repository) HeartEntity(ctx context.Context, userID, entityType, entityID string) error {
	id := generateUUID()
	query := `INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, userID, entityType, entityID)
	return err
}

// UnheartEntity removes a favorite for a specific user
func (r *Repository) UnheartEntity(ctx context.Context, userID, entityType, entityID string) error {
	query := `DELETE FROM hearts WHERE user_id = ? AND entity_type = ? AND entity_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID, entityType, entityID)
	return err
}

// GetAllHearts retrieves all favorites for a specific user
func (r *Repository) GetAllHearts(ctx context.Context, userID string) ([]models.Heart, error) {
	query := `SELECT id, entity_type, entity_id, created_at FROM hearts WHERE user_id = ?`
	rows, err := r.db.QueryContext(ctx, query, userID)
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
func (r *Repository) ExportHearts(ctx context.Context, userID string) ([]models.HeartBackup, error) {
	query := `
		SELECT 
			h.entity_type,
			CASE 
				WHEN h.entity_type = 'track' THEN t.file_path
				WHEN h.entity_type = 'album' THEN a.title
			END as reference,
			h.created_at
		FROM hearts h
		LEFT JOIN tracks t ON h.entity_type = 'track' AND h.entity_id = t.id
		LEFT JOIN albums a ON h.entity_type = 'album' AND h.entity_id = a.id
		WHERE h.user_id = ?
		AND reference IS NOT NULL
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
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

// ImportHeartBackup attempts to restore a user's favorite using its permanent reference
func (r *Repository) ImportHeartBackup(ctx context.Context, userID string, backup models.HeartBackup) error {
	var query string
	var err error
	id := generateUUID()

	if backup.EntityType == "track" {
		query = `
			INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id)
			SELECT ?, ?, 'track', id FROM tracks WHERE file_path = ?
		`
		_, err = r.db.ExecContext(ctx, query, id, userID, backup.Reference)
	} else if backup.EntityType == "album" {
		query = `
			INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id)
			SELECT ?, ?, 'album', id FROM albums WHERE title = ?
		`
		_, err = r.db.ExecContext(ctx, query, id, userID, backup.Reference)
	}

	return err
}

// ScrobbleTrack records a successful track play into the user's history
func (r *Repository) ScrobbleTrack(ctx context.Context, userID, trackID string) error {
	id := generateUUID()
	query := `INSERT INTO scrobbles (id, user_id, track_id) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, userID, trackID)
	return err
}

// GetRecentScrobbles returns the user's most recently listened to tracks
func (r *Repository) GetRecentScrobbles(ctx context.Context, userID string, limit int) ([]models.Track, error) {
	query := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate
		FROM tracks t
		JOIN scrobbles s ON t.id = s.track_id
		WHERE s.user_id = ?
		ORDER BY s.listened_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
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
