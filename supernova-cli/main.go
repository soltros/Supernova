package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
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

func saveConfig(c *Config) error {
	path := getConfigPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, data, 0600)
}

func doRequest(method, endpoint string, body io.Reader, token string) ([]byte, error) {
	c, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load config, please login first")
	}

	req, err := http.NewRequest(method, c.URL+endpoint, body)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Supernova CLI")
		fmt.Println("Usage: sn <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  login <url> <username> <password>")
		fmt.Println("  artists")
		fmt.Println("  albums")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "login":
		if len(os.Args) < 5 {
			fmt.Println("Usage: sn login <url> <username> <password>")
			os.Exit(1)
		}
		url := strings.TrimRight(os.Args[2], "/")
		username := os.Args[3]
		password := os.Args[4]

		payload, _ := json.Marshal(map[string]string{
			"username": username,
			"password": password,
		})

		req, err := http.NewRequest("POST", url+"/api/auth/login", bytes.NewBuffer(payload))
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Printf("Login failed: %v\n", result["error"])
			os.Exit(1)
		}

		token, ok := result["token"].(string)
		if !ok {
			fmt.Println("Token not found in response")
			os.Exit(1)
		}

		c := &Config{
			Token: token,
			URL:   url,
		}
		if err := saveConfig(c); err != nil {
			fmt.Println("Error saving config:", err)
			os.Exit(1)
		}
		fmt.Println("Logged in successfully. Credentials saved to ~/.config/supernova/credentials.json")

	case "artists":
		c, err := loadConfig()
		if err != nil {
			fmt.Println("Not logged in. Please run `sn login` first.")
			os.Exit(1)
		}
		data, err := doRequest("GET", "/api/artists?limit=100&offset=0", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		var result []map[string]interface{}
		json.Unmarshal(data, &result)
		
		fmt.Printf("Found %d artists:\n", len(result))
		for _, a := range result {
			fmt.Printf("- %s (ID: %s)\n", a["name"], a["id"])
		}

	case "albums":
		c, err := loadConfig()
		if err != nil {
			fmt.Println("Not logged in. Please run `sn login` first.")
			os.Exit(1)
		}
		data, err := doRequest("GET", "/api/albums?limit=100&offset=0", nil, c.Token)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		var result []map[string]interface{}
		json.Unmarshal(data, &result)
		
		fmt.Printf("Found %d albums:\n", len(result))
		for _, a := range result {
			fmt.Printf("- %s by %s (ID: %s)\n", a["title"], a["artist_name"], a["id"])
		}

	case "import-navidrome":
		if len(os.Args) < 3 {
			fmt.Println("Usage: sn import-navidrome <path-to-json> [optional-path-prefix]")
			os.Exit(1)
		}
		c, err := loadConfig()
		if err != nil {
			fmt.Println("Not logged in. Please run `sn login` first.")
			os.Exit(1)
		}

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

		payload, _ := json.Marshal(backups)
		_, err = doRequest("POST", "/api/hearts/import", bytes.NewBuffer(payload), c.Token)
		if err != nil {
			fmt.Println("Error importing favorites:", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully imported %d favorites!\n", len(backups))

	case "import-m3u":
		if len(os.Args) < 4 {
			fmt.Println("Usage: sn import-m3u <playlist-name> <path-to-m3u> [optional-path-prefix]")
			os.Exit(1)
		}
		c, err := loadConfig()
		if err != nil {
			fmt.Println("Not logged in.")
			os.Exit(1)
		}

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

		payload, _ := json.Marshal(backups)
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
