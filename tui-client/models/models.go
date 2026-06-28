package models

type AuthResponse struct {
	Token string `json:"token"`
}

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Album struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ArtistID string `json:"artist_id"`
	Year     int    `json:"release_year"`
}

type Track struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	TrackNumber int    `json:"track_number"`
	Duration    int    `json:"duration_ms"`
}

type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HeartDetails struct {
	Tracks    []Track    `json:"tracks"`
	Albums    []Album    `json:"albums"`
	Artists   []Artist   `json:"artists"`
	Playlists []Playlist `json:"playlists"`
}
