package database

import (
	"context"
	"fmt"

	"github.com/soltros/Supernova/internal/models"
)

// CreatePlaylist creates a new playlist for the user
func (r *Repository) CreatePlaylist(ctx context.Context, userID, name string) (*models.Playlist, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	id := generateUUID()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO playlists (id, user_id, name)
		VALUES (?, ?, ?)
	`, id, userID, name)
	if err != nil {
		return nil, err
	}

	return &models.Playlist{
		ID:     id,
		UserID: userID,
		Name:   name,
	}, nil
}

// GetPlaylists returns all playlists for a user
func (r *Repository) GetPlaylists(ctx context.Context, userID string) ([]models.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, created_at
		FROM playlists
		WHERE user_id = ?
		ORDER BY name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []models.Playlist
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, nil
}

// DeletePlaylist deletes a playlist
func (r *Repository) DeletePlaylist(ctx context.Context, userID, playlistID string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	_, err := r.db.ExecContext(ctx, `
		DELETE FROM playlists
		WHERE id = ? AND user_id = ?
	`, playlistID, userID)
	return err
}

// AddTrackToPlaylist appends a track to a playlist
func (r *Repository) AddTrackToPlaylist(ctx context.Context, userID, playlistID, trackID string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	// Verify ownership
	var valid int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM playlists WHERE id = ? AND user_id = ?`, playlistID, userID).Scan(&valid)
	if err != nil {
		return fmt.Errorf("playlist not found or unauthorized")
	}

	// Get max position
	var maxPos int
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) FROM playlist_tracks WHERE playlist_id = ?`, playlistID).Scan(&maxPos)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO playlist_tracks (playlist_id, track_id, position)
		VALUES (?, ?, ?)
	`, playlistID, trackID, maxPos+1)

	return err
}

// RemoveTrackFromPlaylist removes a track from a playlist
func (r *Repository) RemoveTrackFromPlaylist(ctx context.Context, userID, playlistID, trackID string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	// Verify ownership
	var valid int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM playlists WHERE id = ? AND user_id = ?`, playlistID, userID).Scan(&valid)
	if err != nil {
		return fmt.Errorf("playlist not found or unauthorized")
	}

	_, err = r.db.ExecContext(ctx, `
		DELETE FROM playlist_tracks
		WHERE playlist_id = ? AND track_id = ?
	`, playlistID, trackID)

	return err
}

// GetPlaylistTracks returns all tracks in a playlist, ordered by position
func (r *Repository) GetPlaylistTracks(ctx context.Context, userID, playlistID string) ([]models.Track, error) {
	// Verify ownership
	var valid int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM playlists WHERE id = ? AND user_id = ?`, playlistID, userID).Scan(&valid)
	if err != nil {
		return nil, fmt.Errorf("playlist not found or unauthorized")
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate
		FROM tracks t
		JOIN playlist_tracks pt ON t.id = pt.track_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.position ASC
	`, playlistID)
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

// ExportPlaylists exports all playlists and their tracks' absolute paths
func (r *Repository) ExportPlaylists(ctx context.Context, userID string) ([]models.PlaylistBackup, error) {
	playlists, err := r.GetPlaylists(ctx, userID)
	if err != nil {
		return nil, err
	}

	var backups []models.PlaylistBackup
	for _, p := range playlists {
		rows, err := r.db.QueryContext(ctx, `
			SELECT t.file_path
			FROM tracks t
			JOIN playlist_tracks pt ON t.id = pt.track_id
			WHERE pt.playlist_id = ?
			ORDER BY pt.position ASC
		`, p.ID)
		if err != nil {
			continue
		}
		
		var tracks []string
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err == nil {
				tracks = append(tracks, path)
			}
		}
		rows.Close()

		backups = append(backups, models.PlaylistBackup{
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
			Tracks:    tracks,
		})
	}
	return backups, nil
}

// ImportPlaylistBackup safely creates a playlist from paths
func (r *Repository) ImportPlaylistBackup(ctx context.Context, userID string, backup models.PlaylistBackup) error {
	p, err := r.CreatePlaylist(ctx, userID, backup.Name)
	if err != nil {
		return err
	}
	
	// Add tracks one by one using file path matching
	for _, path := range backup.Tracks {
		var trackID string
		err := r.db.QueryRowContext(ctx, `SELECT id FROM tracks WHERE file_path = ?`, path).Scan(&trackID)
		if err == nil {
			_ = r.AddTrackToPlaylist(ctx, userID, p.ID, trackID)
		}
	}
	return nil
}
