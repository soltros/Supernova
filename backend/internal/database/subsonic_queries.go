package database

import (
	"context"
	"fmt"
	"strings"
)

func (r *Repository) SearchPaged(ctx context.Context, query string, artistCount, artistOffset, albumCount, albumOffset, songCount, songOffset int) (map[string]interface{}, error) {
	if artistCount < 0 || albumCount < 0 || songCount < 0 || artistOffset < 0 || albumOffset < 0 || songOffset < 0 {
		return nil, fmt.Errorf("invalid search pagination")
	}
	likeQuery := "%" + strings.ReplaceAll(query, " ", "%") + "%"

	artists := make([]map[string]interface{}, 0, artistCount)
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.name, COALESCE(a.image_url,''),
		       (SELECT COUNT(*) FROM album_artists aa WHERE aa.artist_id=a.id)
		FROM artists a WHERE a.name LIKE ?
		ORDER BY a.name COLLATE NOCASE, a.id LIMIT ? OFFSET ?
	`, likeQuery, artistCount, artistOffset)
	if err != nil { return nil, err }
	for rows.Next() {
		var id,name,image string
		var albumTotal int
		if err:=rows.Scan(&id,&name,&image,&albumTotal);err!=nil{rows.Close();return nil,err}
		artists=append(artists,map[string]interface{}{"id":id,"name":name,"image_url":image,"album_count":albumTotal})
	}
	if err:=rows.Close();err!=nil{return nil,err}
	if err:=rows.Err();err!=nil{return nil,err}

	albums:=make([]map[string]interface{},0,albumCount)
	rows,err=r.db.QueryContext(ctx,`
		SELECT a.id,a.title,COALESCE(ar.name,''),COALESCE(a.cover_art_path,''),
		       COUNT(DISTINCT t.id),COALESCE(SUM(COALESCE(t.duration_ms,0))/1000,0)
		FROM albums a
		LEFT JOIN album_artists aa ON aa.album_id=a.id AND aa.role='primary'
		LEFT JOIN artists ar ON ar.id=aa.artist_id
		LEFT JOIN tracks t ON t.album_id=a.id
		WHERE a.title LIKE ?
		GROUP BY a.id
		ORDER BY a.title COLLATE NOCASE,a.id LIMIT ? OFFSET ?
	`,likeQuery,albumCount,albumOffset)
	if err!=nil{return nil,err}
	for rows.Next(){
		var id,title,artist,cover string
		var songs,duration int
		if err:=rows.Scan(&id,&title,&artist,&cover,&songs,&duration);err!=nil{rows.Close();return nil,err}
		albums=append(albums,map[string]interface{}{"id":id,"title":title,"artist_name":artist,"cover_art_url":cover,"song_count":songs,"duration":duration})
	}
	if err:=rows.Close();err!=nil{return nil,err}
	if err:=rows.Err();err!=nil{return nil,err}

	tracks:=make([]map[string]interface{},0,songCount)
	rows,err=r.db.QueryContext(ctx,`
		SELECT t.id,t.title,a.title,COALESCE(ar.name,''),COALESCE(t.duration_ms,0),
		       a.id,COALESCE(a.cover_art_path,''),COALESCE(t.format,''),COALESCE(t.bitrate,0),
		       COALESCE(t.track_number,0),COALESCE(t.disc_number,0)
		FROM tracks t JOIN albums a ON a.id=t.album_id
		LEFT JOIN track_artists ta ON ta.track_id=t.id
		LEFT JOIN artists ar ON ar.id=ta.artist_id
		WHERE t.title LIKE ?
		GROUP BY t.id
		ORDER BY t.title COLLATE NOCASE,t.id LIMIT ? OFFSET ?
	`,likeQuery,songCount,songOffset)
	if err!=nil{return nil,err}
	for rows.Next(){
		var id,title,album,artist,albumID,cover,format string
		var duration,bitrate,trackNo,discNo int
		if err:=rows.Scan(&id,&title,&album,&artist,&duration,&albumID,&cover,&format,&bitrate,&trackNo,&discNo);err!=nil{rows.Close();return nil,err}
		tracks=append(tracks,map[string]interface{}{"id":id,"title":title,"album_title":album,"artist_name":artist,"duration_ms":duration,"album_id":albumID,"cover_art_url":cover,"format":format,"bitrate":bitrate,"track_number":trackNo,"disc_number":discNo})
	}
	if err:=rows.Close();err!=nil{return nil,err}
	if err:=rows.Err();err!=nil{return nil,err}
	return map[string]interface{}{"artists":artists,"albums":albums,"tracks":tracks},nil
}

func (r *Repository) GetSubsonicAlbumList(ctx context.Context, userID, listType string, size, offset int) ([]map[string]interface{}, error) {
	if size < 0 || offset < 0 { return nil, fmt.Errorf("invalid pagination") }
	order := "a.title COLLATE NOCASE ASC, a.id ASC"
	where := ""
	join := ""
	args := []interface{}{}
	switch listType {
	case "", "alphabeticalByName":
	case "alphabeticalByArtist":
		order = "COALESCE(ar.name,'') COLLATE NOCASE ASC, a.title COLLATE NOCASE ASC, a.id ASC"
	case "newest":
		order = "a.created_at DESC, a.id DESC"
	case "random":
		order = "RANDOM()"
	case "frequent", "highest":
		order = "COALESCE(SUM(t.popularity),0) DESC, a.title COLLATE NOCASE ASC"
	case "recent":
		join = "LEFT JOIN scrobbles s ON s.track_id=t.id AND s.user_id=?"
		args=append(args,userID)
		order = "COALESCE(MAX(s.listened_at),'') DESC, a.title COLLATE NOCASE ASC"
	case "starred":
		where = "WHERE EXISTS (SELECT 1 FROM hearts h WHERE h.user_id=? AND h.entity_type='album' AND h.entity_id=a.id)"
		args=append(args,userID)
	default:
		return nil, fmt.Errorf("unsupported album list type: %s",listType)
	}
	args=append(args,size,offset)
	sqlQuery:=fmt.Sprintf(`
		SELECT a.id,a.title,COALESCE(a.release_year,0),COALESCE(ar.id,''),COALESCE(ar.name,''),
		       COUNT(DISTINCT t.id),COALESCE(SUM(COALESCE(t.duration_ms,0))/1000,0)
		FROM albums a
		LEFT JOIN album_artists aa ON aa.album_id=a.id AND aa.role='primary'
		LEFT JOIN artists ar ON ar.id=aa.artist_id
		LEFT JOIN tracks t ON t.album_id=a.id
		%s
		%s
		GROUP BY a.id
		ORDER BY %s LIMIT ? OFFSET ?
	`,join,where,order)
	rows,err:=r.db.QueryContext(ctx,sqlQuery,args...)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]map[string]interface{},0,size)
	for rows.Next(){
		var id,title,artistID,artist string
		var year,count,duration int
		if err:=rows.Scan(&id,&title,&year,&artistID,&artist,&count,&duration);err!=nil{return nil,err}
		out=append(out,map[string]interface{}{"id":id,"title":title,"year":year,"artist_id":artistID,"artist_name":artist,"song_count":count,"duration":duration})
	}
	return out,rows.Err()
}
