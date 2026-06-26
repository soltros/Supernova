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
}

// Heart represents a user's favorite track, album, or artist
type Heart struct {
	ID         string `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	CreatedAt  string `json:"created_at"`
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
