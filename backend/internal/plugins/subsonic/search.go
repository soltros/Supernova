package subsonic

import (
	"github.com/soltros/Supernova/internal/media"
	"github.com/soltros/Supernova/internal/models"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseSearchPage(r *http.Request, countKey, offsetKey string, defaultCount int) (int,int,error) {
	count:=defaultCount
	offset:=0
	if raw:=r.FormValue(countKey);raw!="" {
		n,err:=strconv.Atoi(raw);if err!=nil||n<0||n>500{return 0,0,fmt.Errorf("invalid %s",countKey)}
		count=n
	}
	if raw:=r.FormValue(offsetKey);raw!="" {
		n,err:=strconv.Atoi(raw);if err!=nil||n<0{return 0,0,fmt.Errorf("invalid %s",offsetKey)}
		offset=n
	}
	return count,offset,nil
}

func (p *SubsonicPlugin) handleSearch3(w http.ResponseWriter,r *http.Request){
	query:=strings.TrimSpace(r.FormValue("query"))
	if query==""{p.writeError(w,r,10,"Required parameter is missing: query");return}
	artistCount,artistOffset,err:=parseSearchPage(r,"artistCount","artistOffset",20);if err!=nil{p.writeError(w,r,10,err.Error());return}
	albumCount,albumOffset,err:=parseSearchPage(r,"albumCount","albumOffset",20);if err!=nil{p.writeError(w,r,10,err.Error());return}
	songCount,songOffset,err:=parseSearchPage(r,"songCount","songOffset",20);if err!=nil{p.writeError(w,r,10,err.Error());return}
	results,err:=p.repo.SearchPaged(r.Context(),query,artistCount,artistOffset,albumCount,albumOffset,songCount,songOffset)
	if err!=nil{p.writeError(w,r,0,"Database error");return}

	artists:=[]map[string]interface{}{}
	for _,a:=range results["artists"].([]map[string]interface{}){
		artists=append(artists,map[string]interface{}{"id":a["id"],"name":a["name"],"albumCount":a["album_count"]})
	}
	albums:=[]map[string]interface{}{}
	for _,a:=range results["albums"].([]map[string]interface{}){
		albums=append(albums,map[string]interface{}{"id":a["id"],"title":a["title"],"name":a["title"],"artist":a["artist_name"],"coverArt":a["id"],"songCount":a["song_count"],"duration":a["duration"]})
	}
	songs:=[]map[string]interface{}{}
	for _,t:=range results["tracks"].([]map[string]interface{}){
		duration,_:=t["duration_ms"].(int)
		songs=append(songs,map[string]interface{}{"id":t["id"],"title":t["title"],"album":t["album_title"],"artist":t["artist_name"],"coverArt":t["album_id"],"duration":duration/1000,"parent":t["album_id"],"albumId":t["album_id"],"isDir":false,"suffix":t["format"],"bitRate":t["bitrate"],"track":t["track_number"],"discNumber":t["disc_number"]})
	}
	key:="searchResult3";if strings.HasPrefix(r.URL.Path,"/rest/search2"){key="searchResult2"}
	p.writeResponse(w,r,map[string]interface{}{key:map[string]interface{}{"artist":artists,"album":albums,"song":songs}})
}

func songNode(t models.Track) map[string]interface{} {
	return map[string]interface{}{"id": t.ID, "parent": t.AlbumID, "albumId": t.AlbumID, "isDir": false, "title": t.Title, "artist": t.ArtistName, "coverArt": t.AlbumID, "duration": t.DurationMs / 1000, "suffix": t.Format, "contentType": media.ContentType(t.Format), "bitRate": t.Bitrate, "track": t.TrackNumber, "discNumber": t.DiscNumber}
}

func (p *SubsonicPlugin) handleGetSong(w http.ResponseWriter, r *http.Request) {
	track, err := p.repo.GetTrackByID(r.Context(), r.FormValue("id"))
	if err != nil {
		p.writeError(w, r, 70, "Song not found")
		return
	}
	p.writeResponse(w, r, map[string]interface{}{"song": songNode(*track)})
}

func (p *SubsonicPlugin) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	size := 10
	if v, err := strconv.Atoi(r.FormValue("size")); err == nil && v >= 0 && v <= 500 {
		size = v
	}
	tracks, err := p.repo.GetRandomTracks(r.Context(), size)
	if err != nil {
		p.writeError(w, r, 0, "Database error")
		return
	}
	songs := make([]map[string]interface{}, 0, len(tracks))
	for _, t := range tracks {
		songs = append(songs, songNode(t))
	}
	p.writeResponse(w, r, map[string]interface{}{"randomSongs": map[string]interface{}{"song": songs}})
}

func (p *SubsonicPlugin) handleScrobble(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*models.User)
	ids, times := r.Form["id"], r.Form["time"]
	if len(ids) == 0 || (len(times) != 0 && len(times) != len(ids)) {
		p.writeError(w, r, 10, "Invalid id/time parameters")
		return
	}
	submission := r.FormValue("submission")
	if submission != "" && submission != "true" && submission != "false" {
		p.writeError(w, r, 10, "Invalid submission")
		return
	}
	timestamps := make([]time.Time, len(ids))
	for i, id := range ids {
		if _, err := p.repo.GetTrackByID(r.Context(), id); err != nil {
			p.writeError(w, r, 70, "Song not found")
			return
		}
		timestamps[i] = time.Now()
		if len(times) != 0 {
			ms, err := strconv.ParseInt(times[i], 10, 64)
			if err != nil || ms <= 0 {
				p.writeError(w, r, 10, "Invalid time")
				return
			}
			timestamps[i] = time.UnixMilli(ms)
		}
	}
	if submission != "false" {
		if err := p.repo.ScrobbleBatch(r.Context(), user.ID, ids, timestamps); err != nil {
			p.writeError(w, r, 0, "Failed to scrobble")
			return
		}
	}
	p.writeResponse(w, r, nil)
}
