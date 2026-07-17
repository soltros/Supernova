package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	
	if artists == nil {
		return []models.Artist{}, nil
	}
	return artists, nil
}

// GetArtistsByLetter returns artists starting with a specific letter. '#' represents non-alphabetical.
func (r *Repository) GetArtistsByLetter(ctx context.Context, letter string, limit, offset int) ([]models.Artist, error) {
	var query string
	var args []interface{}

	if letter == "#" {
		query = `SELECT id, name, musicbrainz_id, image_url, bio FROM artists WHERE UPPER(SUBSTR(name, 1, 1)) < 'A' OR UPPER(SUBSTR(name, 1, 1)) > 'Z' ORDER BY name ASC LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	} else {
		query = `SELECT id, name, musicbrainz_id, image_url, bio FROM artists WHERE name LIKE ? ORDER BY name ASC LIMIT ? OFFSET ?`
		args = []interface{}{letter + "%", limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
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
 if err := rows.Err(); err != nil {
 	return nil, err
 }

	if artists == nil {
		return []models.Artist{}, nil
	}
	return artists, nil
}

// Search queries artists, albums, and tracks for the given query string.
func (r *Repository) Search(ctx context.Context, query string, limit int) (map[string]interface{}, error) {
	// Replace spaces with '%' to allow fuzzy matching (e.g., ignoring punctuation like apostrophes)
	fuzzyTerm := strings.ReplaceAll(query, " ", "%")
	likeQuery := "%" + fuzzyTerm + "%"
	
	// Search Artists
	artistRows, err := r.db.QueryContext(ctx, `SELECT id, name, image_url FROM artists WHERE name LIKE ? LIMIT ?`, likeQuery, limit)
	if err != nil {
		return nil, err
	}
	defer artistRows.Close()
	var artists []map[string]interface{}
	for artistRows.Next() {
		var id, name, img string
		if err := artistRows.Scan(&id, &name, &img); err == nil {
			artists = append(artists, map[string]interface{}{"id": id, "name": name, "image_url": img})
		}
	}
 if err := artistRows.Err(); err != nil {
 	return nil, err
 }

	// Search Albums
	albumRows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.title, ar.name as artist_name, a.cover_art_path
		FROM albums a
		LEFT JOIN album_artists aa ON a.id = aa.album_id
		LEFT JOIN artists ar ON aa.artist_id = ar.id
		WHERE a.title LIKE ? LIMIT ?
	`, likeQuery, limit)
	if err != nil {
		return nil, err
	}
	defer albumRows.Close()
	var albums []map[string]interface{}
	for albumRows.Next() {
		var id, title, coverArt string
		var artistName *string
		if err := albumRows.Scan(&id, &title, &artistName, &coverArt); err == nil {
			name := "Unknown Artist"
			if artistName != nil {
				name = *artistName
			}
			albums = append(albums, map[string]interface{}{
				"id": id, "title": title, "artist_name": name, "cover_art_url": coverArt,
			})
		}
	}
 if err := albumRows.Err(); err != nil {
 	return nil, err
 }

	// Search Tracks
	trackRows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.title, a.title as album_title, ar.name as artist_name, t.duration_ms, a.id as album_id, a.cover_art_path
		FROM tracks t
		JOIN albums a ON t.album_id = a.id
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists ar ON ta.artist_id = ar.id
		WHERE t.title LIKE ? GROUP BY t.id LIMIT ?
	`, likeQuery, limit)
	if err != nil {
		return nil, err
	}
	defer trackRows.Close()
	var tracks []map[string]interface{}
	for trackRows.Next() {
		var id, title, albumTitle, albumID, coverArt string
		var durationMs int
		var artistName *string
		if err := trackRows.Scan(&id, &title, &albumTitle, &artistName, &durationMs, &albumID, &coverArt); err == nil {
			name := "Unknown Artist"
			if artistName != nil {
				name = *artistName
			}
			tracks = append(tracks, map[string]interface{}{
				"id": id, "title": title, "album_title": albumTitle, 
				"artist_name": name, "duration_ms": durationMs,
				"album_id": albumID, "cover_art_url": coverArt,
			})
		}
	}
 if err := trackRows.Err(); err != nil {
 	return nil, err
 }

	if artists == nil { artists = []map[string]interface{}{} }
	if albums == nil { albums = []map[string]interface{}{} }
	if tracks == nil { tracks = []map[string]interface{}{} }

	return map[string]interface{}{
		"artists": artists,
		"albums": albums,
		"tracks": tracks,
	}, nil
}

// Podcast DB operations

func (r *Repository) GetPodcastSubscriptions(ctx context.Context, userID string) ([]models.PodcastSubscription, error) {
	query := `SELECT id, user_id, feed_id, feed_url, title, image_url, subscribed_at FROM podcast_subscriptions WHERE user_id = ? ORDER BY subscribed_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.PodcastSubscription
	for rows.Next() {
		var s models.PodcastSubscription
		var imageURL sql.NullString
		if err := rows.Scan(&s.ID, &s.UserID, &s.FeedID, &s.FeedURL, &s.Title, &imageURL, &s.SubscribedAt); err != nil {
			return nil, err
		}
		if imageURL.Valid {
			s.ImageURL = imageURL.String
		}
		subs = append(subs, s)
	}
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	return subs, nil
}

func (r *Repository) AddPodcastSubscription(ctx context.Context, sub models.PodcastSubscription) error {
	query := `
		INSERT INTO podcast_subscriptions (id, user_id, feed_id, feed_url, title, image_url)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, feed_id) DO UPDATE SET title=excluded.title, image_url=excluded.image_url
	`
	_, err := r.db.ExecContext(ctx, query, sub.ID, sub.UserID, sub.FeedID, sub.FeedURL, sub.Title, sub.ImageURL)
	return err
}

func (r *Repository) RemovePodcastSubscription(ctx context.Context, userID, feedID string) error {
	query := `DELETE FROM podcast_subscriptions WHERE user_id = ? AND feed_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID, feedID)
	return err
}

func (r *Repository) SavePodcastProgress(ctx context.Context, prog models.PodcastProgress) error {
	query := `
		INSERT INTO podcast_progress (user_id, episode_id, position_ms, completed, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, episode_id) DO UPDATE SET position_ms=excluded.position_ms, completed=excluded.completed, updated_at=CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, prog.UserID, prog.EpisodeID, prog.PositionMs, prog.Completed)
	return err
}

func (r *Repository) GetPodcastProgress(ctx context.Context, userID string, episodeIDs []string) (map[string]models.PodcastProgress, error) {
	if len(episodeIDs) == 0 {
		return make(map[string]models.PodcastProgress), nil
	}
	placeholders := make([]string, len(episodeIDs))
	args := make([]interface{}, len(episodeIDs)+1)
	args[0] = userID
	for i, id := range episodeIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}
	query := fmt.Sprintf(`SELECT episode_id, position_ms, completed, updated_at FROM podcast_progress WHERE user_id = ? AND episode_id IN (%s)`, strings.Join(placeholders, ","))
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progress := make(map[string]models.PodcastProgress)
	for rows.Next() {
		var p models.PodcastProgress
		p.UserID = userID
		if err := rows.Scan(&p.EpisodeID, &p.PositionMs, &p.Completed, &p.UpdatedAt); err != nil {
			return nil, err
		}
		progress[p.EpisodeID] = p
	}
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	return progress, nil
}


// Radio DB operations

func (r *Repository) GetRadioSubscriptions(ctx context.Context, userID string) ([]models.RadioSubscription, error) {
	query := `SELECT id, user_id, station_id, url, name, favicon, subscribed_at FROM radio_subscriptions WHERE user_id = ? ORDER BY subscribed_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.RadioSubscription
	for rows.Next() {
		var s models.RadioSubscription
		var favicon sql.NullString
		if err := rows.Scan(&s.ID, &s.UserID, &s.StationID, &s.URL, &s.Name, &favicon, &s.SubscribedAt); err != nil {
			return nil, err
		}
		if favicon.Valid {
			s.Favicon = favicon.String
		}
		subs = append(subs, s)
	}
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	return subs, nil
}

func (r *Repository) AddRadioSubscription(ctx context.Context, sub models.RadioSubscription) error {
	query := `
		INSERT INTO radio_subscriptions (id, user_id, station_id, url, name, favicon)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, station_id) DO UPDATE SET name=excluded.name, favicon=excluded.favicon
	`
	_, err := r.db.ExecContext(ctx, query, sub.ID, sub.UserID, sub.StationID, sub.URL, sub.Name, sub.Favicon)
	return err
}

func (r *Repository) RemoveRadioSubscription(ctx context.Context, userID, stationID string) error {
	query := `DELETE FROM radio_subscriptions WHERE user_id = ? AND station_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID, stationID)
	return err
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

// UpdateAlbumBio updates the bio/summary of an album.
func (r *Repository) UpdateAlbumBio(ctx context.Context, id, bio string) error {
	query := `UPDATE albums SET bio = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, bio, id)
	return err
}

// UpdateTrackDuration updates the duration of a track if it's currently 0.
// Matches by album_id and a case-insensitive track title.
func (r *Repository) UpdateTrackDuration(ctx context.Context, albumID string, title string, durationMs int) error {
	query := `UPDATE tracks SET duration_ms = ? WHERE album_id = ? AND title COLLATE NOCASE = ? AND (duration_ms = 0 OR duration_ms IS NULL)`
	_, err := r.db.ExecContext(ctx, query, durationMs, albumID, title)
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
 if err := rows.Err(); err != nil {
 	return nil, err
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
	query += ` GROUP BY a.id ORDER BY a.title ASC LIMIT ? OFFSET ?`
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
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	
	if albums == nil {
		return []models.Album{}, nil
	}
	return albums, nil
}

// GetTracks returns tracks from the library. It can optionally be filtered by a specific album.
func (r *Repository) GetTracks(ctx context.Context, albumID string, artistID string, limit, offset int) ([]models.Track, error) {
	query := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, t.file_path, art.id, art.name
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

	if artistID != "" && albumID == "" {
		query += ` GROUP BY t.id ORDER BY t.popularity DESC, t.disc_number ASC, t.track_number ASC LIMIT ? OFFSET ?`
	} else {
		query += ` GROUP BY t.id ORDER BY t.disc_number ASC, t.track_number ASC LIMIT ? OFFSET ?`
	}
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
		if err := rows.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate, &t.FilePath, &artID, &artName); err != nil {
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
	
	if tracks == nil {
		return []models.Track{}, nil
	}
	return tracks, nil
}

// UpdateArtistTracksPopularity updates the global popularity score for an artist's tracks
func (r *Repository) UpdateArtistTracksPopularity(ctx context.Context, artistID string, title string, popularity int) error {
	// We match by track title (case-insensitive approximation) and artist ID
	query := `
		UPDATE tracks 
		SET popularity = ? 
		WHERE title COLLATE NOCASE = ? 
		AND id IN (
			SELECT track_id FROM track_artists WHERE artist_id = ?
		)
	`
	_, err := r.db.ExecContext(ctx, query, popularity, title, artistID)
	return err
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
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	return albums, nil
}

// GetAlbumsMissingBio fetches albums that don't have a bio/summary fetched from Last.fm yet.
func (r *Repository) GetAlbumsMissingBio(ctx context.Context, limit int) ([]UnenrichedAlbum, error) {
	query := `
		SELECT 
			a.id, a.title, 
			art.id, art.name,
			t.title
		FROM albums a
		JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
		JOIN artists art ON aa.artist_id = art.id
		LEFT JOIN tracks t ON a.id = t.album_id
		WHERE (a.bio = '' OR a.bio IS NULL)
		AND COALESCE(a.bio, '') != 'NOT_FOUND'
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
		if err := rows.Scan(&a.AlbumID, &a.AlbumTitle, &a.ArtistID, &a.ArtistName, &a.TrackTitle); err == nil {
			albums = append(albums, a)
		}
	}
 if err := rows.Err(); err != nil {
 	return nil, err
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
		SELECT a.id, a.title, a.release_year, a.musicbrainz_id, a.cover_art_path, a.bio, art.id, art.name
		FROM albums a
		LEFT JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
		LEFT JOIN artists art ON aa.artist_id = art.id
		WHERE a.id = ?
	`
	var a models.Album
	var artID, artName, bio *string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath, &bio, &artID, &artName)
	if err != nil {
		return nil, err
	}
	if artID != nil {
		a.ArtistID = *artID
	}
	if artName != nil {
		a.ArtistName = *artName
	}
	if bio != nil {
		a.Bio = *bio
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

// GetHeartDetails retrieves the detailed track, album, artist, and playlist metadata for a user's favorites
func (r *Repository) GetHeartDetails(ctx context.Context, userID string) ([]models.Track, []models.Album, []models.Artist, []models.Playlist, error) {
	// 1. Fetch tracks
	queryTracks := `
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, art.id, art.name
		FROM tracks t
		JOIN hearts h ON t.id = h.entity_id AND h.entity_type = 'track'
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists art ON ta.artist_id = art.id
		WHERE h.user_id = ?
		GROUP BY t.id
		ORDER BY h.created_at DESC
	`
	rowsT, err := r.db.QueryContext(ctx, queryTracks, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rowsT.Close()

	var tracks []models.Track
	for rowsT.Next() {
		var t models.Track
		var artID, artName *string
		if err := rowsT.Scan(&t.ID, &t.AlbumID, &t.Title, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.Format, &t.Bitrate, &artID, &artName); err != nil {
			return nil, nil, nil, nil, err
		}
		if artID != nil {
			t.ArtistID = *artID
		}
		if artName != nil {
			t.ArtistName = *artName
		}
		tracks = append(tracks, t)
	}
 if err := rowsT.Err(); err != nil {
 	return nil, nil, nil, nil, err
 }

	// 2. Fetch albums
	queryAlbums := `
		SELECT a.id, a.title, a.release_year, a.musicbrainz_id, a.cover_art_path, art.id, art.name
		FROM albums a
		JOIN hearts h ON a.id = h.entity_id AND h.entity_type = 'album'
		LEFT JOIN album_artists aa ON a.id = aa.album_id AND aa.role = 'primary'
		LEFT JOIN artists art ON aa.artist_id = art.id
		WHERE h.user_id = ?
		GROUP BY a.id
		ORDER BY h.created_at DESC
	`
	rowsA, err := r.db.QueryContext(ctx, queryAlbums, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rowsA.Close()

	var albums []models.Album
	for rowsA.Next() {
		var a models.Album
		var artID, artName *string
		if err := rowsA.Scan(&a.ID, &a.Title, &a.ReleaseYear, &a.MusicBrainzID, &a.CoverArtPath, &artID, &artName); err != nil {
			return nil, nil, nil, nil, err
		}
		if artID != nil {
			a.ArtistID = *artID
		}
		if artName != nil {
			a.ArtistName = *artName
		}
		albums = append(albums, a)
	}
 if err := rowsA.Err(); err != nil {
 	return nil, nil, nil, nil, err
 }

	if tracks == nil {
		tracks = []models.Track{}
	}
	if albums == nil {
		albums = []models.Album{}
	}

	// 3. Fetch artists
	queryArtists := `
		SELECT a.id, a.name, a.musicbrainz_id, a.image_url, a.bio
		FROM artists a
		JOIN hearts h ON a.id = h.entity_id AND h.entity_type = 'artist'
		WHERE h.user_id = ?
		ORDER BY h.created_at DESC
	`
	rowsArt, err := r.db.QueryContext(ctx, queryArtists, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rowsArt.Close()

	var artists []models.Artist
	for rowsArt.Next() {
		var a models.Artist
		if err := rowsArt.Scan(&a.ID, &a.Name, &a.MusicBrainzID, &a.ImageURL, &a.Bio); err != nil {
			return nil, nil, nil, nil, err
		}
		artists = append(artists, a)
	}
 if err := rowsArt.Err(); err != nil {
 	return nil, nil, nil, nil, err
 }
	if artists == nil {
		artists = []models.Artist{}
	}

	// 4. Fetch playlists
	queryPlaylists := `
		SELECT p.id, p.user_id, p.name, p.created_at
		FROM playlists p
		JOIN hearts h ON p.id = h.entity_id AND h.entity_type = 'playlist'
		WHERE h.user_id = ?
		ORDER BY h.created_at DESC
	`
	rowsP, err := r.db.QueryContext(ctx, queryPlaylists, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rowsP.Close()

	var playlists []models.Playlist
	for rowsP.Next() {
		var p models.Playlist
		if err := rowsP.Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt); err != nil {
			return nil, nil, nil, nil, err
		}
		playlists = append(playlists, p)
	}
 if err := rowsP.Err(); err != nil {
 	return nil, nil, nil, nil, err
 }
	if playlists == nil {
		playlists = []models.Playlist{}
	}

	return tracks, albums, artists, playlists, nil
}

// ExportHearts performs a robust JOIN to export permanent file_paths rather than temporary UUIDs
func (r *Repository) ExportHearts(ctx context.Context, userID string) ([]models.HeartBackup, error) {
	query := `
		SELECT 
			h.entity_type,
			CASE 
				WHEN h.entity_type = 'track' THEN t.file_path
				WHEN h.entity_type = 'album' THEN a.title
				WHEN h.entity_type = 'artist' THEN art.name
				WHEN h.entity_type = 'playlist' THEN p.name
			END as reference,
			h.created_at
		FROM hearts h
		LEFT JOIN tracks t ON h.entity_type = 'track' AND h.entity_id = t.id
		LEFT JOIN albums a ON h.entity_type = 'album' AND h.entity_id = a.id
		LEFT JOIN artists art ON h.entity_type = 'artist' AND h.entity_id = art.id
		LEFT JOIN playlists p ON h.entity_type = 'playlist' AND h.entity_id = p.id
		WHERE h.user_id = ?
		AND CASE 
				WHEN h.entity_type = 'track' THEN t.file_path
				WHEN h.entity_type = 'album' THEN a.title
				WHEN h.entity_type = 'artist' THEN art.name
				WHEN h.entity_type = 'playlist' THEN p.name
			END IS NOT NULL
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
 if err := rows.Err(); err != nil {
 	return nil, err
 }
	return backups, nil
}

// ImportHeartBackups attempts to restore a user's favorites using their permanent references in a batch
func (r *Repository) ImportHeartBackups(ctx context.Context, userID string, backups []models.HeartBackup) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, backup := range backups {
		var query string
		id := generateUUID()

		if backup.EntityType == "track" {
			query = `
				INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id)
				SELECT ?, ?, 'track', id FROM tracks WHERE file_path = ?
			`
			_, _ = tx.ExecContext(ctx, query, id, userID, backup.Reference)
		} else if backup.EntityType == "album" {
			query = `
				INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id)
				SELECT ?, ?, 'album', id FROM albums WHERE title = ?
			`
			_, _ = tx.ExecContext(ctx, query, id, userID, backup.Reference)
		} else if backup.EntityType == "artist" {
			query = `
				INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id)
				SELECT ?, ?, 'artist', id FROM artists WHERE name = ?
			`
			_, _ = tx.ExecContext(ctx, query, id, userID, backup.Reference)
		} else if backup.EntityType == "playlist" {
			query = `
				INSERT OR IGNORE INTO hearts (id, user_id, entity_type, entity_id)
				SELECT ?, ?, 'playlist', id FROM playlists WHERE name = ? AND user_id = ?
			`
			_, _ = tx.ExecContext(ctx, query, id, userID, backup.Reference, userID)
		}
	}

	return tx.Commit()
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
		SELECT t.id, t.album_id, t.title, t.track_number, t.disc_number, t.duration_ms, t.format, t.bitrate, art.id, art.name
		FROM tracks t
		JOIN scrobbles s ON t.id = s.track_id
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN artists art ON ta.artist_id = art.id
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
