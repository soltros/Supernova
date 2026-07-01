package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/soltros/Supernova/internal/models"
)

// createPlaylistUnlocked inserts a playlist row without acquiring writeMu (caller must hold the lock).
func (r *Repository) createPlaylistUnlocked(ctx context.Context, userID, name string) (*models.Playlist, error) {
	id := generateUUID()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO playlists (id, user_id, name)
		VALUES (?, ?, ?)
	`, id, userID, name)
	if err != nil {
		return nil, err
	}
	return &models.Playlist{ID: id, UserID: userID, Name: name}, nil
}

// CreatePlaylist creates a new playlist for the user
func (r *Repository) CreatePlaylist(ctx context.Context, userID, name string) (*models.Playlist, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return r.createPlaylistUnlocked(ctx, userID, name)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return playlists, nil
}

// DeletePlaylist deletes a playlist and returns NotFound if nothing was deleted (BUG-5)
func (r *Repository) DeletePlaylist(ctx context.Context, userID, playlistID string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM playlists
		WHERE id = ? AND user_id = ?
	`, playlistID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("playlist not found or unauthorized")
	}
	return nil
}

// AddTrackToPlaylist appends a track to a playlist (CONC-1: reads done outside mutex)
func (r *Repository) AddTrackToPlaylist(ctx context.Context, userID, playlistID, trackID string) error {
	// Verify ownership outside mutex — these are reads and don't need serialization
	var valid int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM playlists WHERE id = ? AND user_id = ?`, playlistID, userID).Scan(&valid)
	if err != nil {
		return fmt.Errorf("playlist not found or unauthorized")
	}
	var maxPos int
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) FROM playlist_tracks WHERE playlist_id = ?`, playlistID).Scan(&maxPos)

	// Only lock for the actual write
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_, err = r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id, position)
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
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, art.id, art.name
		FROM tracks t
		JOIN playlist_tracks pt ON t.id = pt.track_id
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists art ON ta.artist_id = art.id
		WHERE pt.playlist_id = ?
		GROUP BY pt.position, t.id
		ORDER BY pt.position ASC
	`, playlistID)
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
	if err := rows.Err(); err != nil {
		return nil, err
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
		defer rows.Close() // LEAK-1: defer instead of manual close

		var filePaths []string
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				continue
			}
			filePaths = append(filePaths, path)
		}

		backups = append(backups, models.PlaylistBackup{
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
			Tracks:    filePaths,
		})
	}
	return backups, nil
}

// ImportPlaylistBackup safely creates a playlist from paths.
// CONC-2 fix: acquires writeMu once for the whole operation instead of calling CreatePlaylist
// (which would try to lock the same non-reentrant mutex, causing a deadlock).
func (r *Repository) ImportPlaylistBackup(ctx context.Context, userID string, backup models.PlaylistBackup) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	p, err := r.createPlaylistUnlocked(ctx, userID, backup.Name)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtSelect, err := tx.PrepareContext(ctx, `SELECT id FROM tracks WHERE file_path = ?`)
	if err != nil {
		return err
	}
	defer stmtSelect.Close()

	stmtInsert, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id, position)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtInsert.Close()

	position := 1
	for _, path := range backup.Tracks {
		var trackID string
		if err := stmtSelect.QueryRowContext(ctx, path).Scan(&trackID); err == nil {
			if _, err := stmtInsert.ExecContext(ctx, p.ID, trackID, position); err == nil {
				position++
			}
		}
	}

	return tx.Commit()
}
