package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/soltros/Supernova/internal/models"
)

func (r *Repository) HeartEntityWithMetadata(ctx context.Context, userID, entityType, entityID string, metadata json.RawMessage) error {
	if err := r.HeartEntity(ctx, userID, entityType, entityID); err != nil {
		return err
	}
	if entityType != "radio" && entityType != "podcast" {
		return nil
	}
	if len(metadata) == 0 {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal(metadata, &decoded); err != nil {
		return fmt.Errorf("invalid favorite metadata: %w", err)
	}
	if len(metadata) > 64*1024 {
		return fmt.Errorf("favorite metadata too large")
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE hearts SET metadata_json=? WHERE user_id=? AND entity_type=? AND entity_id=?",
		string(metadata), userID, entityType, entityID,
	)
	return err
}

func (r *Repository) GetExternalHeartMetadata(ctx context.Context, userID string) (radio []json.RawMessage, podcasts []json.RawMessage, err error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT entity_type, metadata_json FROM hearts WHERE user_id=? AND entity_type IN ('radio','podcast') AND metadata_json IS NOT NULL ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var typ, raw string
		if err := rows.Scan(&typ, &raw); err != nil {
			return nil, nil, err
		}
		if !json.Valid([]byte(raw)) {
			continue
		}
		if typ == "radio" {
			radio = append(radio, json.RawMessage(raw))
		} else {
			podcasts = append(podcasts, json.RawMessage(raw))
		}
	}
	return radio, podcasts, rows.Err()
}

func (r *Repository) ExportHeartsV2(ctx context.Context, userID string) ([]models.HeartBackup, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT h.entity_type,
		       CASE
		         WHEN h.entity_type='track' THEN t.file_path
		         WHEN h.entity_type='album' THEN a.title
		         WHEN h.entity_type='artist' THEN art.name
		         WHEN h.entity_type='playlist' THEN p.name
		         ELSE h.entity_id
		       END,
		       h.created_at,
		       COALESCE(h.metadata_json,'')
		FROM hearts h
		LEFT JOIN tracks t ON h.entity_type='track' AND h.entity_id=t.id
		LEFT JOIN albums a ON h.entity_type='album' AND h.entity_id=a.id
		LEFT JOIN artists art ON h.entity_type='artist' AND h.entity_id=art.id
		LEFT JOIN playlists p ON h.entity_type='playlist' AND h.entity_id=p.id AND p.user_id=h.user_id
		WHERE h.user_id=?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.HeartBackup{}
	for rows.Next() {
		var b models.HeartBackup
		var raw string
		if err := rows.Scan(&b.EntityType, &b.Reference, &b.CreatedAt, &raw); err != nil {
			return nil, err
		}
		if raw != "" && json.Valid([]byte(raw)) {
			b.Metadata = json.RawMessage(raw)
		}
		if b.Reference != "" {
			out = append(out, b)
		}
	}
	return out, rows.Err()
}

func (r *Repository) ImportHeartBackupsV2(ctx context.Context, userID string, backups []models.HeartBackup) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, b := range backups {
		id := generateUUID()
		switch b.EntityType {
		case "track":
			_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO hearts(id,user_id,entity_type,entity_id,metadata_json) SELECT ?,?,'track',id,NULL FROM tracks WHERE file_path=?", id, userID, b.Reference)
		case "album":
			_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO hearts(id,user_id,entity_type,entity_id,metadata_json) SELECT ?,?,'album',id,NULL FROM albums WHERE title=?", id, userID, b.Reference)
		case "artist":
			_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO hearts(id,user_id,entity_type,entity_id,metadata_json) SELECT ?,?,'artist',id,NULL FROM artists WHERE name=?", id, userID, b.Reference)
		case "playlist":
			_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO hearts(id,user_id,entity_type,entity_id,metadata_json) SELECT ?,?,'playlist',id,NULL FROM playlists WHERE name=? AND user_id=?", id, userID, b.Reference, userID)
		case "radio", "podcast":
			raw := ""
			if len(b.Metadata) > 0 {
				if !json.Valid(b.Metadata) {
					return fmt.Errorf("invalid %s favorite metadata", b.EntityType)
				}
				raw = string(b.Metadata)
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO hearts(id,user_id,entity_type,entity_id,metadata_json)
				VALUES(?,?,?,?,NULLIF(?,''))
				ON CONFLICT(user_id,entity_type,entity_id) DO UPDATE SET metadata_json=excluded.metadata_json
			`, id, userID, b.EntityType, b.Reference, raw)
		default:
			return fmt.Errorf("unsupported favorite type: %s", b.EntityType)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
