package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/soltros/Supernova/internal/models"
)

func (r *Repository) HeartEntityWithMetadata(ctx context.Context, userID, entityType, entityID string, metadata json.RawMessage) error {
	if err:=r.HeartEntity(ctx,userID,entityType,entityID);err!=nil{return err}
	if entityType!="radio" && entityType!="podcast" { return nil }
	if len(metadata)==0 { return nil }
	var decoded interface{}
	if err:=json.Unmarshal(metadata,&decoded);err!=nil{return fmt.Errorf("invalid favorite metadata: %w",err)}
	if len(metadata)>64*1024{return fmt.Errorf("favorite metadata too large")}
	_,err:=r.db.ExecContext(ctx,`UPDATE hearts SET metadata_json=? WHERE user_id=? AND entity_type=? AND entity_id=?`,string(metadata),userID,entityType,entityID)
	return err
}

func (r *Repository) GetExternalHeartMetadata(ctx context.Context,userID string)(radio []json.RawMessage,podcasts []json.RawMessage,err error){
	rows,err:=r.db.QueryContext(ctx,`SELECT entity_type,metadata_json FROM hearts WHERE user_id=? AND entity_type IN ('radio','podcast') AND metadata_json IS NOT NULL ORDER BY created_at DESC`,userID)
	if err!=nil{return nil,nil,err}
	defer rows.Close()
	for rows.Next(){
		var typ,raw string
		if err:=rows.Scan(&typ,&raw);err!=nil{return nil,nil,err}
		if !json.Valid([]byte(raw)){continue}
		if typ=="radio"{radio=append(radio,json.RawMessage(raw))}else{podcasts=append(podcasts,json.RawMessage(raw))}
	}
	return radio,podcasts,rows.Err()
}

func (r *Repository) ExportHeartsV2(ctx context.Context,userID string)([]models.HeartBackup,error){
	rows,err:=r.db.QueryContext(ctx,`
		SELECT h.entity_type,
		       CASE
		         WHEN h.entity_type='track' AND COALESCE(t.file_fingerprint,'')<>'' THEN t.file_fingerprint
		         WHEN h.entity_type='track' THEN t.file_path
		         WHEN h.entity_type='album' AND COALESCE(a.musicbrainz_id,'')<>'' THEN a.musicbrainz_id
		         WHEN h.entity_type='album' THEN a.id
		         WHEN h.entity_type='artist' AND COALESCE(art.musicbrainz_id,'')<>'' THEN art.musicbrainz_id
		         WHEN h.entity_type='artist' THEN art.id
		         WHEN h.entity_type='playlist' THEN p.id
		         ELSE h.entity_id
		       END,
		       CASE
		         WHEN h.entity_type='track' AND COALESCE(t.file_fingerprint,'')<>'' THEN 'fingerprint'
		         WHEN h.entity_type='track' THEN 'path'
		         WHEN h.entity_type IN ('album','artist') AND COALESCE(CASE WHEN h.entity_type='album' THEN a.musicbrainz_id ELSE art.musicbrainz_id END,'')<>'' THEN 'musicbrainz'
		         ELSE 'id'
		       END,
		       h.created_at,
		       COALESCE(h.metadata_json,'')
		FROM hearts h
		LEFT JOIN tracks t ON h.entity_type='track' AND h.entity_id=t.id
		LEFT JOIN albums a ON h.entity_type='album' AND h.entity_id=a.id
		LEFT JOIN artists art ON h.entity_type='artist' AND h.entity_id=art.id
		LEFT JOIN playlists p ON h.entity_type='playlist' AND h.entity_id=p.id AND p.user_id=h.user_id
		WHERE h.user_id=?
	`,userID)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=[]models.HeartBackup{}
	for rows.Next(){
		var b models.HeartBackup
		var raw string
		if err:=rows.Scan(&b.EntityType,&b.Reference,&b.ReferenceType,&b.CreatedAt,&raw);err!=nil{return nil,err}
		if raw!="" && json.Valid([]byte(raw)){b.Metadata=json.RawMessage(raw)}
		if b.Reference!=""{out=append(out,b)}
	}
	return out,rows.Err()
}

func resolveHeartReferenceTx(ctx context.Context,tx *sql.Tx,userID string,b models.HeartBackup)(string,error){
	switch b.ReferenceType {
	case "fingerprint":
		rows,err:=tx.QueryContext(ctx,`SELECT id FROM tracks WHERE file_fingerprint=? ORDER BY id LIMIT 2`,b.Reference)
		if err!=nil{return "",err}
		defer rows.Close()
		var ids []string
		for rows.Next(){var id string;if err:=rows.Scan(&id);err!=nil{return "",err};ids=append(ids,id)}
		if len(ids)==1{return ids[0],nil}
		if len(ids)>1{return "",fmt.Errorf("ambiguous track fingerprint")}
		return "",sql.ErrNoRows
	case "path":
		var id string;err:=tx.QueryRowContext(ctx,`SELECT id FROM tracks WHERE file_path=?`,b.Reference).Scan(&id);return id,err
	case "musicbrainz":
		var id string
		if b.EntityType=="album"{err:=tx.QueryRowContext(ctx,`SELECT id FROM albums WHERE musicbrainz_id=?`,b.Reference).Scan(&id);return id,err}
		if b.EntityType=="artist"{err:=tx.QueryRowContext(ctx,`SELECT id FROM artists WHERE musicbrainz_id=?`,b.Reference).Scan(&id);return id,err}
		return "",fmt.Errorf("musicbrainz reference unsupported for %s",b.EntityType)
	case "id":
		var one int
		switch b.EntityType {
		case "album": if err:=tx.QueryRowContext(ctx,`SELECT 1 FROM albums WHERE id=?`,b.Reference).Scan(&one);err!=nil{return "",err}
		case "artist": if err:=tx.QueryRowContext(ctx,`SELECT 1 FROM artists WHERE id=?`,b.Reference).Scan(&one);err!=nil{return "",err}
		case "playlist": if err:=tx.QueryRowContext(ctx,`SELECT 1 FROM playlists WHERE id=? AND user_id=?`,b.Reference,userID).Scan(&one);err!=nil{return "",err}
		case "radio","podcast": return b.Reference,nil
		default:return "",fmt.Errorf("id reference unsupported for %s",b.EntityType)
		}
		return b.Reference,nil
	case "":
		// Legacy backup semantics.
		var id string
		switch b.EntityType{
		case "track": err:=tx.QueryRowContext(ctx,`SELECT id FROM tracks WHERE file_path=?`,b.Reference).Scan(&id);return id,err
		case "album": err:=tx.QueryRowContext(ctx,`SELECT id FROM albums WHERE title=? ORDER BY id LIMIT 1`,b.Reference).Scan(&id);return id,err
		case "artist": err:=tx.QueryRowContext(ctx,`SELECT id FROM artists WHERE name=? ORDER BY id LIMIT 1`,b.Reference).Scan(&id);return id,err
		case "playlist": err:=tx.QueryRowContext(ctx,`SELECT id FROM playlists WHERE name=? AND user_id=? ORDER BY id LIMIT 1`,b.Reference,userID).Scan(&id);return id,err
		case "radio","podcast":return b.Reference,nil
		}
	}
	return "",fmt.Errorf("unsupported favorite reference")
}

func (r *Repository) ImportHeartBackupsV2(ctx context.Context,userID string,backups []models.HeartBackup)error{
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx,err:=r.db.BeginTx(ctx,nil);if err!=nil{return err}
	defer tx.Rollback()
	for i,b:=range backups{
		if b.Reference==""{return fmt.Errorf("favorite %d has empty reference",i)}
		entityID,resolveErr:=resolveHeartReferenceTx(ctx,tx,userID,b)
		if resolveErr!=nil {
			if errors.Is(resolveErr,sql.ErrNoRows){return fmt.Errorf("favorite %d %s reference not found",i,b.EntityType)}
			return fmt.Errorf("favorite %d: %w",i,resolveErr)
		}
		raw:=""
		if len(b.Metadata)>0{
			if !json.Valid(b.Metadata){return fmt.Errorf("favorite %d has invalid metadata",i)}
			raw=string(b.Metadata)
		}
		id:=generateUUID()
		if b.CreatedAt!=""{
			_,err=tx.ExecContext(ctx,`
				INSERT INTO hearts(id,user_id,entity_type,entity_id,created_at,metadata_json)
				VALUES(?,?,?,?,?,NULLIF(?,''))
				ON CONFLICT(user_id,entity_type,entity_id) DO UPDATE SET metadata_json=COALESCE(excluded.metadata_json,hearts.metadata_json)
			`,id,userID,b.EntityType,entityID,b.CreatedAt,raw)
		}else{
			_,err=tx.ExecContext(ctx,`
				INSERT INTO hearts(id,user_id,entity_type,entity_id,metadata_json)
				VALUES(?,?,?,?,NULLIF(?,''))
				ON CONFLICT(user_id,entity_type,entity_id) DO UPDATE SET metadata_json=COALESCE(excluded.metadata_json,hearts.metadata_json)
			`,id,userID,b.EntityType,entityID,raw)
		}
		if err!=nil{return err}
	}
	return tx.Commit()
}
