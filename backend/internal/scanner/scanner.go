package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/metadata"
	"github.com/soltros/Supernova/internal/models"
)

type realtimeEvent struct {
	path   string
	remove bool
	due    time.Time
}

// Scanner handles both bulk scans and real-time filesystem reconciliation.
type Scanner struct {
	mediaPath string
	watcher *fsnotify.Watcher
	repo *database.Repository
	enricher *Enricher

	ctx context.Context
	cancel context.CancelFunc
	realtimeEvents chan realtimeEvent
	realtimeJobs chan string
	wg sync.WaitGroup

	stateMu sync.RWMutex
	status string
	filesScanned int
}

func New(parent context.Context, mediaPath string, repo *database.Repository, enricher *Enricher) (*Scanner,error) {
	watcher,err:=fsnotify.NewWatcher()
	if err!=nil{return nil,err}
	ctx,cancel:=context.WithCancel(parent)
	s:=&Scanner{
		mediaPath:mediaPath,watcher:watcher,repo:repo,enricher:enricher,
		ctx:ctx,cancel:cancel,realtimeEvents:make(chan realtimeEvent,2048),
		realtimeJobs:make(chan string,512),status:"idle",
	}
	s.wg.Add(2)
	go s.realtimeWorker()
	go s.debounceWorker()
	return s,nil
}

func (s *Scanner) realtimeWorker(){
	defer s.wg.Done()
	for{
		select{
		case <-s.ctx.Done():return
		case path:=<-s.realtimeJobs:
			meta:=s.extractMetadata(path)
			if meta!=nil{
				if err:=s.repo.UpsertTrack(s.ctx,meta);err!=nil{log.Printf("DB insert failed (%s): %v",filepath.Base(path),err)}
				if s.enricher!=nil{s.enricher.Trigger()}
			}
		}
	}
}

func (s *Scanner) debounceWorker(){
	defer s.wg.Done()
	pending:=map[string]realtimeEvent{}
	ticker:=time.NewTicker(250*time.Millisecond)
	defer ticker.Stop()
	for{
		select{
		case <-s.ctx.Done():return
		case ev:=<-s.realtimeEvents:
			ev.due=time.Now().Add(1500*time.Millisecond)
			pending[ev.path]=ev
		case now:=<-ticker.C:
			for path,ev:=range pending{
				if now.Before(ev.due){continue}
				delete(pending,path)
				if ev.remove{
					if _,err:=os.Stat(path);errors.Is(err,os.ErrNotExist){
						if err:=s.repo.RemoveTrackByPath(s.ctx,path);err!=nil{log.Printf("failed to reconcile removed media %s: %v",path,err)}
					}
					continue
				}
				select{case s.realtimeJobs<-path:case <-s.ctx.Done():return}
			}
		}
	}
}

func (s *Scanner) FullScan() error {
	s.stateMu.Lock()
	if s.status=="scanning"{s.stateMu.Unlock();return nil}
	s.status="scanning";s.filesScanned=0
	s.stateMu.Unlock()
	defer func(){s.stateMu.Lock();s.status="idle";s.stateMu.Unlock()}()

	log.Printf("Starting library scan at: %s",s.mediaPath)
	start:=time.Now()
	jobs:=make(chan string,1024)
	dbJobs:=make(chan *models.TrackMetadata,1024)
	seen:=make(map[string]struct{})
	var seenMu sync.Mutex
	var workers,writer sync.WaitGroup

	writer.Add(1)
	go func(){
		defer writer.Done()
		for meta:=range dbJobs{
			if err:=s.repo.UpsertTrack(s.ctx,meta);err!=nil{
				log.Printf("DB insert failed (%s): %v",filepath.Base(meta.FilePath),err)
			}else{
				s.stateMu.Lock();s.filesScanned++;s.stateMu.Unlock()
			}
		}
	}()

	for i:=0;i<10;i++{
		workers.Add(1)
		go func(){
			defer workers.Done()
			for path:=range jobs{
				if meta:=s.extractMetadata(path);meta!=nil{
					select{case dbJobs<-meta:case <-s.ctx.Done():return}
				}
			}
		}()
	}

	walkErr:=filepath.WalkDir(s.mediaPath,func(path string,d os.DirEntry,err error)error{
		if err!=nil{log.Printf("scanner walk error %s: %v",path,err);return err}
		if d.IsDir(){
			if err:=s.watcher.Add(path);err!=nil{log.Printf("watch add failed for %s: %v",path,err)}
			return nil
		}
		if !isAudioFile(path){return nil}
		seenMu.Lock();seen[path]=struct{}{};seenMu.Unlock()
		select{
		case jobs<-path:return nil
		case <-s.ctx.Done():return s.ctx.Err()
		}
	})
	close(jobs);workers.Wait();close(dbJobs);writer.Wait()

	if walkErr==nil && s.ctx.Err()==nil{
		if err:=s.repo.ReconcileLibraryPaths(s.ctx,s.mediaPath,seen);err!=nil{
			log.Printf("library reconciliation failed: %v",err)
			walkErr=err
		}
	}
	log.Printf("Full scan completed in %v",time.Since(start))
	if s.enricher!=nil{s.enricher.Trigger()}
	return walkErr
}

func (s *Scanner) GetStatus()(string,int){
	s.stateMu.RLock();defer s.stateMu.RUnlock()
	return s.status,s.filesScanned
}

func (s *Scanner) enqueueEvent(path string,remove bool){
	select{
	case s.realtimeEvents<-realtimeEvent{path:path,remove:remove}:
	case <-s.ctx.Done():
	default:
		log.Printf("filesystem debounce queue full; scheduling full rescan for %s",path)
	}
}

func (s *Scanner) addDirectoryTree(root string){
	err:=filepath.WalkDir(root,func(path string,d os.DirEntry,err error)error{
		if err!=nil{return err}
		if d.IsDir(){
			if err:=s.watcher.Add(path);err!=nil{log.Printf("watch add failed for %s: %v",path,err)}
			return nil
		}
		if isAudioFile(path){s.enqueueEvent(path,false)}
		return nil
	})
	if err!=nil{log.Printf("watch tree add failed for %s: %v",root,err)}
}

func (s *Scanner) Watch(){
	log.Println("Starting real-time file watcher...")
	s.wg.Add(1)
	go func(){
		defer s.wg.Done()
		for{
			select{
			case <-s.ctx.Done():return
			case event,ok:=<-s.watcher.Events:
				if !ok{return}
				if event.Op&(fsnotify.Remove|fsnotify.Rename)!=0{
					s.enqueueEvent(event.Name,true)
				}
				if event.Op&(fsnotify.Create|fsnotify.Write)!=0{
					info,err:=os.Stat(event.Name)
					if err==nil && info.IsDir(){s.addDirectoryTree(event.Name);continue}
					if err==nil && isAudioFile(event.Name){s.enqueueEvent(event.Name,false)}
				}
			case err,ok:=<-s.watcher.Errors:
				if !ok{return}
				log.Printf("watcher error: %v",err)
			}
		}
	}()
}

func (s *Scanner) Close() error {
	s.cancel()
	err:=s.watcher.Close()
	s.wg.Wait()
	return err
}

func fingerprintFile(path string,size int64)(string,error){
	f,err:=os.Open(path)
	if err!=nil{return "",err}
	defer f.Close()
	h:=sha256.New()
	var sizeBuf [8]byte
	for i:=uint(0);i<8;i++{sizeBuf[i]=byte(uint64(size)>>(8*i))}
	_,_=h.Write(sizeBuf[:])
	const chunk int64=64*1024
	if _,err:=io.CopyN(h,f,min64(size,chunk));err!=nil && !errors.Is(err,io.EOF){return "",err}
	if size>chunk{
		offset:=size-chunk
		if _,err:=f.Seek(offset,io.SeekStart);err!=nil{return "",err}
		if _,err:=io.CopyN(h,f,min64(size,chunk));err!=nil && !errors.Is(err,io.EOF){return "",err}
	}
	return hex.EncodeToString(h.Sum(nil)),nil
}

func min64(a,b int64)int64{if a<b{return a};return b}

func (s *Scanner) extractMetadata(path string)*models.TrackMetadata{
	info,err:=os.Stat(path)
	if err!=nil || !info.Mode().IsRegular(){return nil}
	meta,err:=metadata.ExtractContext(s.ctx,path)
	if err!=nil{log.Printf("metadata extraction failed for %s: %v",path,err);return nil}
	fingerprint,err:=fingerprintFile(path,info.Size())
	if err!=nil{log.Printf("fingerprint failed for %s: %v",path,err);return nil}
	meta.FilePath=path
	meta.FileModifiedAt=info.ModTime().Unix()
	meta.FileModifiedNs=info.ModTime().UnixNano()
	meta.FileSize=info.Size()
	meta.FileFingerprint=fingerprint
	return meta
}

func isAudioFile(path string)bool{
	switch strings.ToLower(filepath.Ext(path)){
	case ".mp3",".flac",".ogg",".m4a",".aac",".opus",".wav",".alac",".wma",".aiff",".m4b":return true
	}
	return false
}
