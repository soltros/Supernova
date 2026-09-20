package database

import (
	"context"
	"github.com/soltros/Supernova/internal/models"
	"time"
)

func (r *Repository) GetRandomTracks(ctx context.Context, size int) ([]models.Track, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, COALESCE(ar.name, '') FROM tracks t LEFT JOIN track_artists ta ON ta.track_id=t.id LEFT JOIN artists ar ON ar.id=ta.artist_id GROUP BY t.id ORDER BY RANDOM() LIMIT ?`, size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := []models.Track{}
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate, &t.ArtistName); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}
func (r *Repository) ScrobbleBatch(ctx context.Context, userID string, ids []string, times []time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `INSERT INTO scrobbles (id, user_id, track_id, listened_at) VALUES (?, ?, ?, ?)`, generateUUID(), userID, id, times[i].UTC().Format("2006-01-02 15:04:05")); err != nil {
			return err
		}
	}
	return tx.Commit()
}
