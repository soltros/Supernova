package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// externalRelease matches what we parse from iTunes
type externalRelease struct {
	CollectionName string    `json:"collectionName"`
	ArtistName     string    `json:"artistName"`
	ArtworkUrl100  string    `json:"artworkUrl100"`
	ReleaseDate    time.Time `json:"releaseDate"`
	CollectionViewUrl string `json:"collectionViewUrl"`
}

type iTunesResponse struct {
	Results []externalRelease `json:"results"`
}

func (s *Server) handleGetDiscovery() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Just pull 3 random artists from the DB to get their new releases
		query := `SELECT name FROM artists ORDER BY RANDOM() LIMIT 3`
		rows, err := s.repo.DB().QueryContext(r.Context(), query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var artists []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				artists = append(artists, name)
			}
		}
		rows.Close()

		var allReleases []externalRelease
		var similarArtists []map[string]interface{}
		
		client := &http.Client{Timeout: 5 * time.Second}
		lastFmKey := os.Getenv("LASTFM_API_KEY")

		for _, artist := range artists {
			// iTunes
			u := "https://itunes.apple.com/search?term=" + url.QueryEscape(artist) + "&entity=album&limit=2&sort=recent"
			resp, err := client.Get(u)
			if err == nil {
				var res iTunesResponse
				if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
					allReleases = append(allReleases, res.Results...)
				}
				resp.Body.Close()
			}

			// Last.fm Similar
			if lastFmKey != "" {
				lUrl := "http://ws.audioscrobbler.com/2.0/?method=artist.getsimilar&artist=" + url.QueryEscape(artist) + "&api_key=" + lastFmKey + "&format=json&limit=3"
				lResp, err := client.Get(lUrl)
				if err == nil {
					var lRes struct {
						SimilarArtists struct {
							Artist []struct {
								Name string `json:"name"`
								Image []struct {
									Text string `json:"#text"`
									Size string `json:"size"`
								} `json:"image"`
							} `json:"artist"`
						} `json:"similarartists"`
					}
					if err := json.NewDecoder(lResp.Body).Decode(&lRes); err == nil {
						for _, sim := range lRes.SimilarArtists.Artist {
							img := ""
							for _, imgObj := range sim.Image {
								if imgObj.Size == "extralarge" || imgObj.Size == "large" {
									img = imgObj.Text
								}
							}
							similarArtists = append(similarArtists, map[string]interface{}{
								"name": sim.Name,
								"basedOn": artist,
								"image": img,
							})
						}
					}
					lResp.Body.Close()
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"release_radar": allReleases,
			"similar_artists": similarArtists,
		})
	}
}
