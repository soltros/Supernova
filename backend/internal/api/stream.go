package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/soltros/Supernova/internal/media"
)

func (s *Server) handleStreamTrack()http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
	trackID:=r.PathValue("id")
	if trackID==""{http.Error(w,"track ID required",400);return}
	track,err:=s.repo.GetTrackByID(r.Context(),trackID)
	if err!=nil{http.Error(w,"track not found",404);return}
	file,err:=media.Open(track.FilePath)
	if err!=nil{http.Error(w,"media file unavailable",404);return}
	defer file.Close()

	format:=r.URL.Query().Get("format")
	if format==""{
		info,err:=file.Stat();if err!=nil{http.Error(w,"media file unavailable",404);return}
		w.Header().Del("Content-Type")
		http.ServeContent(w,r,info.Name(),info.ModTime(),file)
		return
	}
	bitrate:=128
	if raw:=r.URL.Query().Get("bitrate");raw!=""{if n,err:=strconv.Atoi(raw);err==nil{bitrate=n}else{http.Error(w,"invalid bitrate",400);return}}
	seek:=0
	if raw:=r.URL.Query().Get("time");raw!=""{if n,err:=strconv.Atoi(raw);err==nil&&n>=0{seek=n}else{http.Error(w,"invalid time offset",400);return}}
	contentType:=map[string]string{"mp3":"audio/mpeg","aac":"audio/aac","ogg":"audio/ogg","opus":"audio/ogg; codecs=opus"}[format]
	if contentType==""{http.Error(w,"unsupported transcode format",400);return}
	err=media.StreamTranscode(r.Context(),file,media.TranscodeOptions{Format:format,BitrateKbps:bitrate,SeekSeconds:seek},w,func(){
		w.Header().Set("Content-Type",contentType)
		w.Header().Set("Cache-Control","no-cache, no-store, must-revalidate")
		w.Header().Set("Accept-Ranges","none")
		if f,ok:=w.(http.Flusher);ok{f.Flush()}
	})
	if err!=nil{log.Printf("transcode %s failed: %v",trackID,err)}
}}
