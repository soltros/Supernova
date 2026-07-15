package models

// TrackMetadata holds the standardized metadata extracted from any audio file.
// This serves as the middle-layer between the raw file and the database schema.
type TrackMetadata struct {
	Title          string
	Album          string
	AlbumMBID      string // MusicBrainz Release ID
	Artist         string 
	ArtistMBID     string // MusicBrainz Artist ID
	TrackMBID      string // MusicBrainz Recording ID
	AlbumArtist    string
	TrackNumber    int
	DiscNumber     int
	Year           int
	DurationMs     int
	Format         string
	Bitrate        int
	FilePath       string // Crucial for database unique constraints and streaming
	CoverArtPath   string // Extracted embedded image or folder image path
	FileModifiedAt int64  // Used to prevent overwriting plugin changes during re-scans
}

// Artist represents a row in the artists table for JSON API responses
type Artist struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MusicBrainzID string `json:"musicbrainz_id"`
	ImageURL      string `json:"image_url"`
	Bio           string `json:"bio"`
}

// Album represents a row in the albums table for JSON API responses
type Album struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ReleaseYear   int    `json:"release_year"`
	MusicBrainzID string `json:"musicbrainz_id"`
	CoverArtPath  string `json:"cover_art_path"`
	Bio           string `json:"bio"`
	ArtistID      string `json:"artist_id,omitempty"`
	ArtistName    string `json:"artist_name,omitempty"`
}

// Track represents a row in the tracks table for JSON API responses
type Track struct {
	ID          string `json:"id"`
	AlbumID     string `json:"album_id"`
	Title       string `json:"title"`
	TrackNumber int    `json:"track_number"`
	DiscNumber  int    `json:"disc_number"`
	DurationMs  int    `json:"duration_ms"`
	FilePath    string `json:"-"`
	Format      string `json:"format"`
	Bitrate     int    `json:"bitrate"`
	ArtistID    string `json:"artist_id,omitempty"`
	ArtistName  string `json:"artist_name,omitempty"`
}

// Heart represents a user's favorite track, album, or artist
type Heart struct {
	ID         string `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	CreatedAt  string `json:"created_at"`
}

type PodcastSubscription struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	FeedID       string `json:"feed_id"`
	FeedURL      string `json:"feed_url"`
	Title        string `json:"title"`
	ImageURL     string `json:"image_url"`
	SubscribedAt string `json:"subscribed_at"`
}

type PodcastProgress struct {
	UserID     string `json:"user_id"`
	EpisodeID  string `json:"episode_id"`
	PositionMs int    `json:"position_ms"`
	Completed  bool   `json:"completed"`
	UpdatedAt  string `json:"updated_at"`
}

type RadioSubscription struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	StationID    string `json:"station_id"`
	URL          string `json:"url"`
	Name         string `json:"name"`
	Favicon      string `json:"favicon"`
	SubscribedAt string `json:"subscribed_at"`
}

// HeartBackup securely exports hearts by absolute file_path instead of volatile UUIDs
type HeartBackup struct {
	EntityType string `json:"entity_type"`
	Reference  string `json:"reference"`
	CreatedAt  string `json:"created_at"`
}

// User represents an authenticated account
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// Playlist represents a user-created collection of tracks
type Playlist struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// PlaylistBackup is used for exporting playlists robustly, matching tracks by file_path
type PlaylistBackup struct {
	Name      string   `json:"name"`
	CreatedAt string   `json:"created_at"`
	Tracks    []string `json:"tracks"` // file paths
}
