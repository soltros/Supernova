package database

import (
	"context"

	"github.com/soltros/Supernova/internal/models"
)

type DashboardData struct {
	RecentlyAddedAlbums []models.Album `json:"recently_added_albums"`
	RecentlyPlayedTracks []models.Track `json:"recently_played_tracks"`
	FavoriteTracks      []models.Track `json:"favorite_tracks"`
}

func (r *Repository) GetDashboard(ctx context.Context, userID string) (*DashboardData, error) {
	dashboard := &DashboardData{
		RecentlyAddedAlbums:  []models.Album{},
		RecentlyPlayedTracks: []models.Track{},
		FavoriteTracks:       []models.Track{},
	}

	// 1. Recently Added Albums
	queryAlbums := `
		SELECT a.id, a.title, a.release_year, a.musicbrainz_id, a.cover_art_path, art.id, art.name
		FROM albums a
		LEFT JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
		LEFT JOIN artists art ON aa.artist_id = art.id
		ORDER BY a.created_at DESC
		LIMIT 10
	`
	rowsAlbums, err := r.db.QueryContext(ctx, queryAlbums)
	if err == nil {
		defer rowsAlbums.Close()
		for rowsAlbums.Next() {
			var a models.Album
			var artID, artName *string
			if err := rowsAlbums.Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath, &artID, &artName); err == nil {
				if artID != nil { a.ArtistID = *artID }
				if artName != nil { a.ArtistName = *artName }
				dashboard.RecentlyAddedAlbums = append(dashboard.RecentlyAddedAlbums, a)
			}
		}
	}

	// 2. Recently Played Tracks
	queryRecent := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, art.id, art.name
		FROM scrobbles s
		JOIN tracks t ON s.track_id = t.id
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists art ON ta.artist_id = art.id
		WHERE s.user_id = ?
		GROUP BY t.id
		ORDER BY MAX(s.listened_at) DESC
		LIMIT 10
	`
	rowsRecent, err := r.db.QueryContext(ctx, queryRecent, userID)
	if err == nil {
		defer rowsRecent.Close()
		for rowsRecent.Next() {
			var t models.Track
			var artID, artName *string
			if err := rowsRecent.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate, &artID, &artName); err == nil {
				if artID != nil { t.ArtistID = *artID }
				if artName != nil { t.ArtistName = *artName }
				dashboard.RecentlyPlayedTracks = append(dashboard.RecentlyPlayedTracks, t)
			}
		}
	}

	// 3. Favorite Tracks
	queryFavs := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, art.id, art.name
		FROM hearts h
		JOIN tracks t ON h.entity_id = t.id
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists art ON ta.artist_id = art.id
		WHERE h.user_id = ? AND h.entity_type = 'track'
		GROUP BY t.id
		ORDER BY h.created_at DESC
		LIMIT 10
	`
	rowsFavs, err := r.db.QueryContext(ctx, queryFavs, userID)
	if err == nil {
		defer rowsFavs.Close()
		for rowsFavs.Next() {
			var t models.Track
			var artID, artName *string
			if err := rowsFavs.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate, &artID, &artName); err == nil {
				if artID != nil { t.ArtistID = *artID }
				if artName != nil { t.ArtistName = *artName }
				dashboard.FavoriteTracks = append(dashboard.FavoriteTracks, t)
			}
		}
	}

	return dashboard, nil
}
