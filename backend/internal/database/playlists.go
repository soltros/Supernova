package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/soltros/Supernova/internal/models"
)

func (r *Repository) createPlaylistUnlocked(ctx context.Context, userID, name string) (*models.Playlist, error) {
	id := generateUUID()
	_, err := r.db.ExecContext(ctx, `INSERT INTO playlists (id, user_id, name) VALUES (?, ?, ?)`, id, userID, name)
	if err != nil {
		return nil, err
	}
	return &models.Playlist{ID: id, UserID: userID, Name: name}, nil
}

func (r *Repository) CreatePlaylist(ctx context.Context, userID, name string) (*models.Playlist, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return r.createPlaylistUnlocked(ctx, userID, name)
}

func (r *Repository) GetPlaylists(ctx context.Context, userID string) ([]models.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, COALESCE(created_at, '')
		FROM playlists WHERE user_id = ? ORDER BY name ASC, id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	playlists := []models.Playlist{}
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

func (r *Repository) DeletePlaylist(ctx context.Context, userID, playlistID string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	result, err := r.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = ? AND user_id = ?`, playlistID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("playlist not found or unauthorized")
	}
	return nil
}

func playlistOwnedTx(ctx context.Context, tx *sql.Tx, userID, playlistID string) error {
	var one int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM playlists WHERE id = ? AND user_id = ?`, playlistID, userID).Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("playlist not found or unauthorized")
		}
		return err
	}
	return nil
}

func validateTrackIDsTx(ctx context.Context, tx *sql.Tx, trackIDs []string) error {
	for _, id := range trackIDs {
		if strings.TrimSpace(id) == "" {
			return errors.New("track id is required")
		}
		var one int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM tracks WHERE id = ?`, id).Scan(&one); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("track not found: %s", id)
			}
			return err
		}
	}
	return nil
}

func insertPlaylistTracksTx(ctx context.Context, tx *sql.Tx, playlistID string, start int, trackIDs []string) error {
	for i, trackID := range trackIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO playlist_tracks(entry_id, playlist_id, track_id, position)
			VALUES (?, ?, ?, ?)
		`, generateUUID(), playlistID, trackID, start+i); err != nil {
			return err
		}
	}
	return nil
}

func compactPlaylistPositionsTx(ctx context.Context, tx *sql.Tx, playlistID string) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT entry_id FROM playlist_tracks
		WHERE playlist_id = ? ORDER BY position ASC, added_at ASC, entry_id ASC
	`, playlistID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE playlist_tracks SET position = ? WHERE entry_id = ?`, -(i + 1), id); err != nil {
			return err
		}
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE playlist_tracks SET position = ? WHERE entry_id = ?`, i, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) CreatePlaylistWithTracks(ctx context.Context, userID, name string, trackIDs []string) (*models.Playlist, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()
	if err := validateTrackIDsTx(ctx, tx, trackIDs); err != nil { return nil, err }
	p := &models.Playlist{ID: generateUUID(), UserID: userID, Name: name}
	if _, err := tx.ExecContext(ctx, `INSERT INTO playlists(id,user_id,name) VALUES(?,?,?)`, p.ID, userID, name); err != nil {
		return nil, err
	}
	if err := insertPlaylistTracksTx(ctx, tx, p.ID, 0, trackIDs); err != nil { return nil, err }
	if err := tx.Commit(); err != nil { return nil, err }
	return p, nil
}

func (r *Repository) ReplacePlaylist(ctx context.Context, userID, playlistID, name string, trackIDs []string) (*models.Playlist, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()
	if err := playlistOwnedTx(ctx, tx, userID, playlistID); err != nil { return nil, err }
	if err := validateTrackIDsTx(ctx, tx, trackIDs); err != nil { return nil, err }
	if strings.TrimSpace(name) != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE playlists SET name = ? WHERE id = ? AND user_id = ?`, name, playlistID, userID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM playlist_tracks WHERE playlist_id = ?`, playlistID); err != nil { return nil, err }
	if err := insertPlaylistTracksTx(ctx, tx, playlistID, 0, trackIDs); err != nil { return nil, err }
	var p models.Playlist
	if err := tx.QueryRowContext(ctx, `SELECT id,user_id,name,COALESCE(created_at,'') FROM playlists WHERE id=?`, playlistID).
		Scan(&p.ID,&p.UserID,&p.Name,&p.CreatedAt); err != nil { return nil, err }
	if err := tx.Commit(); err != nil { return nil, err }
	return &p, nil
}

func (r *Repository) UpdatePlaylist(ctx context.Context, userID, playlistID string, name *string, addTrackIDs []string, removeIndexes []int) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	if err := playlistOwnedTx(ctx, tx, userID, playlistID); err != nil { return err }
	if err := validateTrackIDsTx(ctx, tx, addTrackIDs); err != nil { return err }

	rows, err := tx.QueryContext(ctx, `SELECT entry_id FROM playlist_tracks WHERE playlist_id=? ORDER BY position ASC, entry_id ASC`, playlistID)
	if err != nil { return err }
	var entryIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { rows.Close(); return err }
		entryIDs = append(entryIDs, id)
	}
	if err := rows.Close(); err != nil { return err }

	seen := map[int]bool{}
	for _, idx := range removeIndexes {
		if idx < 0 || idx >= len(entryIDs) { return fmt.Errorf("playlist index out of range: %d", idx) }
		if seen[idx] { return fmt.Errorf("duplicate playlist removal index: %d", idx) }
		seen[idx] = true
	}
	sort.Sort(sort.Reverse(sort.IntSlice(removeIndexes)))
	for _, idx := range removeIndexes {
		if _, err := tx.ExecContext(ctx, `DELETE FROM playlist_tracks WHERE entry_id=?`, entryIDs[idx]); err != nil { return err }
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" { return errors.New("playlist name cannot be empty") }
		if _, err := tx.ExecContext(ctx, `UPDATE playlists SET name=? WHERE id=? AND user_id=?`, trimmed, playlistID, userID); err != nil { return err }
	}
	if err := compactPlaylistPositionsTx(ctx, tx, playlistID); err != nil { return err }
	var next int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position)+1,0) FROM playlist_tracks WHERE playlist_id=?`, playlistID).Scan(&next); err != nil { return err }
	if err := insertPlaylistTracksTx(ctx, tx, playlistID, next, addTrackIDs); err != nil { return err }
	return tx.Commit()
}

func (r *Repository) AddTrackToPlaylist(ctx context.Context, userID, playlistID, trackID string) error {
	return r.UpdatePlaylist(ctx, userID, playlistID, nil, []string{trackID}, nil)
}

func (r *Repository) RemoveTrackFromPlaylist(ctx context.Context, userID, playlistID, trackID string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	if err := playlistOwnedTx(ctx, tx, userID, playlistID); err != nil { return err }
	var entryID string
	if err := tx.QueryRowContext(ctx, `
		SELECT entry_id FROM playlist_tracks WHERE playlist_id=? AND track_id=?
		ORDER BY position ASC, entry_id ASC LIMIT 1
	`, playlistID, trackID).Scan(&entryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return errors.New("track is not in playlist") }
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM playlist_tracks WHERE entry_id=?`, entryID); err != nil { return err }
	if err := compactPlaylistPositionsTx(ctx, tx, playlistID); err != nil { return err }
	return tx.Commit()
}

func (r *Repository) GetPlaylistTracks(ctx context.Context, userID, playlistID string) ([]models.Track, error) {
	var valid int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM playlists WHERE id=? AND user_id=?`, playlistID, userID).Scan(&valid); err != nil {
		return nil, errors.New("playlist not found or unauthorized")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.album_id, t.title,
		       COALESCE(t.track_number,0), COALESCE(t.disc_number,0),
		       COALESCE(t.duration_ms,0), COALESCE(t.format,''), COALESCE(t.bitrate,0),
		       COALESCE((SELECT a.id FROM track_artists ta JOIN artists a ON a.id=ta.artist_id WHERE ta.track_id=t.id ORDER BY a.name LIMIT 1),''),
		       COALESCE((SELECT a.name FROM track_artists ta JOIN artists a ON a.id=ta.artist_id WHERE ta.track_id=t.id ORDER BY a.name LIMIT 1),'')
		FROM playlist_tracks pt JOIN tracks t ON t.id=pt.track_id
		WHERE pt.playlist_id=? ORDER BY pt.position ASC, pt.entry_id ASC
	`, playlistID)
	if err != nil { return nil, err }
	defer rows.Close()
	tracks := []models.Track{}
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID,&t.AlbumID,&t.Title,&t.TrackNumber,&t.DiscNumber,&t.DurationMs,&t.Format,&t.Bitrate,&t.ArtistID,&t.ArtistName); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (r *Repository) PlaylistStats(ctx context.Context, userID, playlistID string) (count int, durationSeconds int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(COALESCE(t.duration_ms,0))/1000,0)
		FROM playlists p
		LEFT JOIN playlist_tracks pt ON pt.playlist_id=p.id
		LEFT JOIN tracks t ON t.id=pt.track_id
		WHERE p.id=? AND p.user_id=?
	`, playlistID, userID).Scan(&count,&durationSeconds)
	return
}

func (r *Repository) ExportPlaylists(ctx context.Context, userID string) ([]models.PlaylistBackup, error) {
	playlists, err := r.GetPlaylists(ctx, userID)
	if err != nil { return nil, err }
	backups := []models.PlaylistBackup{}
	for _, p := range playlists {
		rows, err := r.db.QueryContext(ctx, \`
			SELECT t.file_path, COALESCE(t.file_fingerprint,'')
			FROM playlist_tracks pt JOIN tracks t ON t.id=pt.track_id
			WHERE pt.playlist_id=? ORDER BY pt.position ASC, pt.entry_id ASC
		\`, p.ID)
		if err != nil { return nil, err }
		var paths []string
		var refs []models.PlaylistTrackBackup
		for rows.Next() {
			var path, fingerprint string
			if err := rows.Scan(&path, &fingerprint); err != nil { rows.Close(); return nil, err }
			paths = append(paths, path)
			refs = append(refs, models.PlaylistTrackBackup{FilePath:path, Fingerprint:fingerprint})
		}
		if err := rows.Close(); err != nil { return nil, err }
		if err := rows.Err(); err != nil { return nil, err }
		backups = append(backups, models.PlaylistBackup{Name:p.Name, CreatedAt:p.CreatedAt, Tracks:paths, TrackRefs:refs})
	}
	return backups, nil
}

func resolveBackupTrackTx(ctx context.Context, tx *sql.Tx, ref models.PlaylistTrackBackup) (string, error) {
	if ref.Fingerprint != "" {
		rows, err := tx.QueryContext(ctx, \`SELECT id FROM tracks WHERE file_fingerprint=? ORDER BY id LIMIT 2\`, ref.Fingerprint)
		if err != nil { return "", err }
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil { return "", err }
			ids = append(ids,id)
		}
		if err := rows.Err(); err != nil { return "", err }
		if len(ids) == 1 { return ids[0], nil }
		if len(ids) > 1 { return "", fmt.Errorf("ambiguous track fingerprint %s", ref.Fingerprint) }
	}
	if ref.FilePath != "" {
		var id string
		if err := tx.QueryRowContext(ctx,\`SELECT id FROM tracks WHERE file_path=?\`,ref.FilePath).Scan(&id); err != nil {
			if errors.Is(err,sql.ErrNoRows){return "",fmt.Errorf("track not found: %s",ref.FilePath)}
			return "",err
		}
		return id,nil
	}
	return "",errors.New("backup track has no usable reference")
}

func (r *Repository) ImportPlaylistBackups(ctx context.Context, userID string, backups []models.PlaylistBackup) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx, err := r.db.BeginTx(ctx,nil)
	if err != nil { return err }
	defer tx.Rollback()

	for _, backup := range backups {
		if strings.TrimSpace(backup.Name)=="" { return errors.New("playlist backup has empty name") }
		id := generateUUID()
		if backup.CreatedAt != "" {
			if _,err=tx.ExecContext(ctx,\`INSERT INTO playlists(id,user_id,name,created_at) VALUES(?,?,?,?)\`,id,userID,backup.Name,backup.CreatedAt);err!=nil{return err}
		} else {
			if _,err=tx.ExecContext(ctx,\`INSERT INTO playlists(id,user_id,name) VALUES(?,?,?)\`,id,userID,backup.Name);err!=nil{return err}
		}
		refs:=backup.TrackRefs
		if len(refs)==0 {
			refs=make([]models.PlaylistTrackBackup,0,len(backup.Tracks))
			for _,path:=range backup.Tracks{refs=append(refs,models.PlaylistTrackBackup{FilePath:path})}
		}
		for position,ref:=range refs{
			trackID,resolveErr:=resolveBackupTrackTx(ctx,tx,ref)
			if resolveErr!=nil{return fmt.Errorf("playlist %q entry %d: %w",backup.Name,position,resolveErr)}
			if _,err=tx.ExecContext(ctx,\`INSERT INTO playlist_tracks(entry_id,playlist_id,track_id,position) VALUES(?,?,?,?)\`,generateUUID(),id,trackID,position);err!=nil{return err}
		}
	}
	return tx.Commit()
}

func (r *Repository) ImportPlaylistBackup(ctx context.Context, userID string, backup models.PlaylistBackup) error {
	return r.ImportPlaylistBackups(ctx,userID,[]models.PlaylistBackup{backup})
}
