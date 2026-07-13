package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".config", "supernova", "credentials.json")
}

func loadConfig() (*Config, error) {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func requireConfig() *Config {
	c, err := loadConfig()
	if err != nil {
		fmt.Println("Not logged in. Please run `sn login` first.")
		os.Exit(1)
	}
	return c
}

func saveConfig(c *Config) error {
	path := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func doAuthRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func doRequest(method, endpoint string, body io.Reader, token string) ([]byte, error) {
	c := requireConfig()
	url := c.URL + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func downloadFile(endpoint, dest string) error {
	c := requireConfig()

	req, err := http.NewRequest("GET", c.URL+endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func printPrettyJSON(data []byte) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		fmt.Println(string(data))
		return
	}
	pretty, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Println(string(pretty))
}

func requireArgs(expected int, usage string) {
	if len(os.Args) < expected {
		fmt.Println("Usage:", usage)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Supernova CLI")
		fmt.Println("Usage: sn <command> [args]")
		fmt.Println("\nAuthentication:")
		fmt.Println("  register <url> <username> <password>")
		fmt.Println("  login <url> <username> <password>")
		fmt.Println("\nLibrary:")
		fmt.Println("  artists [id]")
		fmt.Println("  albums [id]")
		fmt.Println("  tracks [artist|album] [id]")
		fmt.Println("\nMedia:")
		fmt.Println("  play <track_id>")
		fmt.Println("  stream <id> <output_file>")
		fmt.Println("  art <album_id> <output_file>")
		fmt.Println("\nUser Data:")
		fmt.Println("  dashboard")
		fmt.Println("  hearts")
		fmt.Println("  hearts-details")
		fmt.Println("  heart <type> <id> (types: track, album, artist, playlist, radio, podcast)")
		fmt.Println("  unheart <type> <id>")
		fmt.Println("\nPlaylists:")
		fmt.Println("  playlists")
		fmt.Println("  playlist-create <name>")
		fmt.Println("  playlist-delete <id>")
		fmt.Println("  playlist-tracks <id>")
		fmt.Println("  playlist-add <playlist_id> <track_id>")
		fmt.Println("  playlist-remove <playlist_id> <track_id>")
		fmt.Println("  playlist-export")
		fmt.Println("\nScrobbling:")
		fmt.Println("  scrobble <track_id>")
		fmt.Println("  scrobbles")
		fmt.Println("\nImports:")
		fmt.Println("  import-navidrome <path-to-json> [optional-path-prefix]")
		fmt.Println("  import-m3u <playlist-name> <path-to-m3u> [optional-path-prefix]")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	// ---------------------------------------------------------
	// AUTHENTICATION
	// ---------------------------------------------------------
	case "register":
		requireArgs(5, "sn register <url> <username> <password>")
		url := strings.TrimRight(os.Args[2], "/")
		payload, err := json.Marshal(map[string]string{
			"username": os.Args[3],
			"password": os.Args[4],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doAuthRequest("POST", url+"/api/auth/register", bytes.NewBuffer(payload))
		if err != nil {
			fmt.Println("Registration failed:", err)
			os.Exit(1)
		}
		fmt.Println("Registered successfully. You can now login.")

	case "login":
		requireArgs(5, "sn login <url> <username> <password>")
		url := strings.TrimRight(os.Args[2], "/")
		payload, err := json.Marshal(map[string]string{
			"username": os.Args[3],
			"password": os.Args[4],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		data, err := doAuthRequest("POST", url+"/api/auth/login", bytes.NewBuffer(payload))
		if err != nil {
			fmt.Println("Login request failed:", err)
			os.Exit(1)
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)

		token, ok := result["token"].(string)
		if !ok {
			fmt.Println("Token not found in response:", result["error"])
			os.Exit(1)
		}

		c := &Config{Token: token, URL: url}
		if err := saveConfig(c); err != nil {
			fmt.Println("Error saving config:", err)
			os.Exit(1)
		}
		fmt.Println("Logged in successfully. Credentials saved.")

	// ---------------------------------------------------------
	// PUBLIC LIBRARY
	// ---------------------------------------------------------
	case "artists":
		c := requireConfig()
		if len(os.Args) >= 3 {
			data, err := doRequest("GET", "/api/artists/"+os.Args[2], nil, c.Token)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			printPrettyJSON(data)
			return
		}
		data, err := doRequest("GET", "/api/artists?limit=100&offset=0", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "albums":
		c := requireConfig()
		if len(os.Args) >= 3 {
			data, err := doRequest("GET", "/api/albums/"+os.Args[2], nil, c.Token)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			printPrettyJSON(data)
			return
		}
		data, err := doRequest("GET", "/api/albums?limit=100&offset=0", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "tracks":
		c := requireConfig()
		endpoint := "/api/tracks"
		if len(os.Args) == 4 {
			filterType := os.Args[2]
			filterID := os.Args[3]
			if filterType == "artist" {
				endpoint += "?artist_id=" + filterID
			} else if filterType == "album" {
				endpoint += "?album_id=" + filterID
			} else {
				fmt.Println("Unknown filter. Use 'artist' or 'album'.")
				os.Exit(1)
			}
		}
		data, err := doRequest("GET", endpoint, nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	// ---------------------------------------------------------
	// STREAMING & MEDIA
	// ---------------------------------------------------------
	case "play":
		requireArgs(3, "sn play <track_id>")
		c := requireConfig()
		
		if _, err := exec.LookPath("mpv"); err != nil {
			fmt.Println("Error: 'mpv' media player not found in PATH.")
			os.Exit(1)
		}
		
		trackID := os.Args[2]
		streamURL := c.URL + "/api/stream/" + trackID
		authHeader := "Authorization: Bearer " + c.Token

		fmt.Printf("Starting mpv for track %s...\n", trackID)
		
		cmd := exec.Command("mpv", "--http-header-fields="+authHeader, streamURL)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		
		if err := cmd.Run(); err != nil {
			fmt.Println("Playback failed:", err)
			os.Exit(1)
		}

	case "stream":
		requireArgs(4, "sn stream <id> <output_file>")
		id := os.Args[2]
		dest := os.Args[3]
		fmt.Printf("Downloading stream for track %s to %s...\n", id, dest)
		err := downloadFile("/api/stream/"+id, dest)
		if err != nil {
			fmt.Println("Error downloading stream:", err)
			os.Exit(1)
		}
		fmt.Println("Download complete.")

	case "art":
		requireArgs(4, "sn art <album_id> <output_file>")
		id := os.Args[2]
		dest := os.Args[3]
		fmt.Printf("Downloading art for album %s to %s...\n", id, dest)
		err := downloadFile("/api/art/album/"+id, dest)
		if err != nil {
			fmt.Println("Error downloading art:", err)
			os.Exit(1)
		}
		fmt.Println("Download complete.")

	// ---------------------------------------------------------
	// USER DATA & FAVORITES
	// ---------------------------------------------------------
	case "dashboard":
		c := requireConfig()
		data, err := doRequest("GET", "/api/dashboard", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "hearts":
		c := requireConfig()
		data, err := doRequest("GET", "/api/hearts", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "hearts-details":
		c := requireConfig()
		data, err := doRequest("GET", "/api/hearts/details", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "heart":
		requireArgs(4, "sn heart <entity_type> <entity_id>")
		c := requireConfig()
		payload, err := json.Marshal(map[string]string{
			"entity_type": os.Args[2],
			"entity_id":   os.Args[3],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doRequest("POST", "/api/hearts", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Added to favorites.")

	case "unheart":
		requireArgs(4, "sn unheart <entity_type> <entity_id>")
		c := requireConfig()
		payload, err := json.Marshal(map[string]string{
			"entity_type": os.Args[2],
			"entity_id":   os.Args[3],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doRequest("DELETE", "/api/hearts", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Removed from favorites.")

	// ---------------------------------------------------------
	// PLAYLISTS
	// ---------------------------------------------------------
	case "playlists":
		c := requireConfig()
		data, err := doRequest("GET", "/api/playlists", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "playlist-create":
		requireArgs(3, "sn playlist-create <name>")
		c := requireConfig()
		payload, err := json.Marshal(map[string]string{
			"name": os.Args[2],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		data, err := doRequest("POST", "/api/playlists", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "playlist-delete":
		requireArgs(3, "sn playlist-delete <id>")
		c := requireConfig()
		_, err := doRequest("DELETE", "/api/playlists/"+os.Args[2], nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Playlist deleted.")

	case "playlist-tracks":
		requireArgs(3, "sn playlist-tracks <id>")
		c := requireConfig()
		data, err := doRequest("GET", "/api/playlists/"+os.Args[2]+"/tracks", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	case "playlist-add":
		requireArgs(4, "sn playlist-add <playlist_id> <track_id>")
		c := requireConfig()
		payload, err := json.Marshal(map[string]string{
			"track_id": os.Args[3],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doRequest("POST", "/api/playlists/"+os.Args[2]+"/tracks", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Track added to playlist.")

	case "playlist-remove":
		requireArgs(4, "sn playlist-remove <playlist_id> <track_id>")
		c := requireConfig()
		_, err := doRequest("DELETE", fmt.Sprintf("/api/playlists/%s/tracks/%s", os.Args[2], os.Args[3]), nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Track removed from playlist.")

	case "playlist-export":
		c := requireConfig()
		data, err := doRequest("GET", "/api/playlists/export", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	// ---------------------------------------------------------
	// INTERNAL SCROBBLING
	// ---------------------------------------------------------
	case "scrobble":
		requireArgs(3, "sn scrobble <track_id>")
		c := requireConfig()
		payload, err := json.Marshal(map[string]string{
			"track_id": os.Args[2],
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doRequest("POST", "/api/scrobbles", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Track scrobbled successfully.")

	case "scrobbles":
		c := requireConfig()
		data, err := doRequest("GET", "/api/scrobbles/recent", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		printPrettyJSON(data)

	// ---------------------------------------------------------
	// CUSTOM IMPORTS (PRESERVED)
	// ---------------------------------------------------------
	case "import-navidrome":
		requireArgs(3, "sn import-navidrome <path-to-json> [optional-path-prefix]")
		c := requireConfig()

		filePath := os.Args[2]
		prefix := ""
		if len(os.Args) >= 4 {
			prefix = os.Args[3]
			if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			os.Exit(1)
		}

		var export struct {
			Album []struct {
				Name string `json:"name"`
			} `json:"album"`
			Song []struct {
				Path string `json:"path"`
			} `json:"song"`
		}

		if err := json.Unmarshal(data, &export); err != nil {
			fmt.Println("Error parsing JSON:", err)
			os.Exit(1)
		}

		type HeartBackup struct {
			EntityType string `json:"entityType"`
			Reference  string `json:"reference"`
			CreatedAt  string `json:"createdAt"`
		}

		var backups []HeartBackup

		for _, a := range export.Album {
			backups = append(backups, HeartBackup{
				EntityType: "album",
				Reference:  a.Name,
				CreatedAt:  "2026-01-01T00:00:00Z",
			})
		}
		for _, s := range export.Song {
			backups = append(backups, HeartBackup{
				EntityType: "track",
				Reference:  prefix + s.Path,
				CreatedAt:  "2026-01-01T00:00:00Z",
			})
		}

		payload, err := json.Marshal(backups)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doRequest("POST", "/api/hearts/import", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error importing favorites:", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully imported %d favorites!\n", len(backups))

	case "import-m3u":
		requireArgs(4, "sn import-m3u <playlist-name> <path-to-m3u> [optional-path-prefix]")
		c := requireConfig()

		playlistName := os.Args[2]
		filePath := os.Args[3]
		prefix := ""
		if len(os.Args) >= 5 {
			prefix = os.Args[4]
			if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading m3u:", err)
			os.Exit(1)
		}

		lines := strings.Split(string(data), "\n")
		var tracks []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			tracks = append(tracks, prefix+line)
		}

		type PlaylistBackup struct {
			Name      string   `json:"name"`
			CreatedAt string   `json:"createdAt"`
			Tracks    []string `json:"tracks"`
		}

		backups := []PlaylistBackup{
			{
				Name:      playlistName,
				CreatedAt: "2026-01-01T00:00:00Z",
				Tracks:    tracks,
			},
		}

		payload, err := json.Marshal(backups)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			os.Exit(1)
		}
		_, err = doRequest("POST", "/api/playlists/import", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error importing playlist:", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully imported playlist '%s' with %d tracks!\n", playlistName, len(tracks))

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
