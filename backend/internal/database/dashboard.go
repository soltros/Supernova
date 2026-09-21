package database

import (
	"context"
	"fmt"

	"github.com/soltros/Supernova/internal/models"
)

type DashboardData struct {
	RecentlyAddedAlbums  []models.Album `json:"recently_added_albums"`
	RecentlyPlayedTracks []models.Track `json:"recently_played_tracks"`
	FavoriteTracks       []models.Track `json:"favorite_tracks"`
}

func (r *Repository) GetDashboard(ctx context.Context, userID string) (*DashboardData, error) {
	dashboard := &DashboardData{
		RecentlyAddedAlbums: []models.Album{},
		RecentlyPlayedTracks: []models.Track{},
		FavoriteTracks: []models.Track{},
	}

	rowsAlbums, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.title, COALESCE(a.release_year,0), COALESCE(a.musicbrainz_id,''),
		       COALESCE(a.cover_art_path,''), COALESCE(art.id,''), COALESCE(art.name,'')
		FROM albums a
		LEFT JOIN album_artists aa ON a.id=aa.album_id AND aa.role='primary'
		LEFT JOIN artists art ON aa.artist_id=art.id
		ORDER BY a.created_at DESC LIMIT 10
	`)
	if err != nil { return nil, fmt.Errorf("recent albums: %w",err) }
	for rowsAlbums.Next() {
		var a models.Album
		if err:=rowsAlbums.Scan(&a.ID,&a.Title,&a.ReleaseYear,&a.MusicBrainzID,&a.CoverArtPath,&a.ArtistID,&a.ArtistName);err!=nil {
			rowsAlbums.Close(); return nil,fmt.Errorf("recent albums scan: %w",err)
		}
		dashboard.RecentlyAddedAlbums=append(dashboard.RecentlyAddedAlbums,a)
	}
	if err:=rowsAlbums.Close();err!=nil{return nil,err}
	if err:=rowsAlbums.Err();err!=nil{return nil,fmt.Errorf("recent albums rows: %w",err)}

	rowsRecent,err:=r.db.QueryContext(ctx,`
		SELECT t.id,t.album_id,t.title,COALESCE(t.track_number,0),COALESCE(t.disc_number,0),
		       COALESCE(t.duration_ms,0),COALESCE(t.format,''),COALESCE(t.bitrate,0),
		       COALESCE(art.id,''),COALESCE(art.name,'')
		FROM scrobbles s
		JOIN tracks t ON s.track_id=t.id
		LEFT JOIN track_artists ta ON t.id=ta.track_id
		LEFT JOIN artists art ON ta.artist_id=art.id
		WHERE s.user_id=?
		GROUP BY t.id ORDER BY MAX(s.listened_at) DESC LIMIT 10
	`,userID)
	if err!=nil{return nil,fmt.Errorf("recent tracks: %w",err)}
	for rowsRecent.Next(){
		var t models.Track
		if err:=rowsRecent.Scan(&t.ID,&t.AlbumID,&t.Title,&t.TrackNumber,&t.DiscNumber,&t.DurationMs,&t.Format,&t.Bitrate,&t.ArtistID,&t.ArtistName);err!=nil{
			rowsRecent.Close();return nil,fmt.Errorf("recent tracks scan: %w",err)
		}
		dashboard.RecentlyPlayedTracks=append(dashboard.RecentlyPlayedTracks,t)
	}
	if err:=rowsRecent.Close();err!=nil{return nil,err}
	if err:=rowsRecent.Err();err!=nil{return nil,fmt.Errorf("recent tracks rows: %w",err)}

	rowsFavs,err:=r.db.QueryContext(ctx,`
		SELECT t.id,t.album_id,t.title,COALESCE(t.track_number,0),COALESCE(t.disc_number,0),
		       COALESCE(t.duration_ms,0),COALESCE(t.format,''),COALESCE(t.bitrate,0),
		       COALESCE(art.id,''),COALESCE(art.name,'')
		FROM hearts h
		JOIN tracks t ON h.entity_id=t.id
		LEFT JOIN track_artists ta ON t.id=ta.track_id
		LEFT JOIN artists art ON ta.artist_id=art.id
		WHERE h.user_id=? AND h.entity_type='track'
		GROUP BY t.id ORDER BY h.created_at DESC LIMIT 10
	`,userID)
	if err!=nil{return nil,fmt.Errorf("favorite tracks: %w",err)}
	for rowsFavs.Next(){
		var t models.Track
		if err:=rowsFavs.Scan(&t.ID,&t.AlbumID,&t.Title,&t.TrackNumber,&t.DiscNumber,&t.DurationMs,&t.Format,&t.Bitrate,&t.ArtistID,&t.ArtistName);err!=nil{
			rowsFavs.Close();return nil,fmt.Errorf("favorite tracks scan: %w",err)
		}
		dashboard.FavoriteTracks=append(dashboard.FavoriteTracks,t)
	}
	if err:=rowsFavs.Close();err!=nil{return nil,err}
	if err:=rowsFavs.Err();err!=nil{return nil,fmt.Errorf("favorite tracks rows: %w",err)}
	return dashboard,nil
}
