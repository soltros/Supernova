package media

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type TranscodeOptions struct {
	Format string
	BitrateKbps int
	SeekSeconds int
}

var transcodeSlots=make(chan struct{},4)
var transcodeBufferPool=sync.Pool{New:func()any{return make([]byte,32*1024)}}

func validateTranscode(opts *TranscodeOptions)(codec,muxer string,error){
	switch opts.Format {
	case "mp3":codec,muxer="libmp3lame","mp3"
	case "aac":codec,muxer="aac","adts"
	case "ogg":codec,muxer="libvorbis","ogg"
	case "opus":codec,muxer="libopus","opus"
	default:return "","",fmt.Errorf("unsupported transcode format")
	}
	if opts.BitrateKbps==0{opts.BitrateKbps=128}
	if opts.BitrateKbps<64{opts.BitrateKbps=64}
	if opts.BitrateKbps>320{opts.BitrateKbps=320}
	if opts.SeekSeconds<0{return "","",fmt.Errorf("invalid seek offset")}
	return codec,muxer,nil
}

// StreamTranscode runs FFmpeg against an already-open library descriptor.
// beforeCopy is called only after FFmpeg produces its first byte, allowing HTTP
// handlers to commit success headers without masking startup failures.
func StreamTranscode(ctx context.Context,input *os.File,opts TranscodeOptions,dst io.Writer,beforeCopy func())error{
	codec,muxer,err:=validateTranscode(&opts);if err!=nil{return err}
	select{
	case transcodeSlots<-struct{}{}:
		defer func(){<-transcodeSlots}()
	case <-ctx.Done():
		return ctx.Err()
	}

	args:=[]string{"-nostdin","-protocol_whitelist","file,pipe"}
	if opts.SeekSeconds>0{args=append(args,"-ss",strconv.Itoa(opts.SeekSeconds))}
	args=append(args,"-i","pipe:3","-map","0:a:0","-f",muxer,"-c:a",codec,"-b:a",strconv.Itoa(opts.BitrateKbps)+"k","-loglevel","error","pipe:1")
	cmd:=exec.CommandContext(ctx,"ffmpeg",args...)
	cmd.ExtraFiles=[]*os.File{input}
	stdout,err:=cmd.StdoutPipe();if err!=nil{return err}
	if err:=cmd.Start();err!=nil{return err}
	reader:=bufio.NewReader(stdout)
	if _,err:=reader.Peek(1);err!=nil{_ = cmd.Process.Kill();_ = cmd.Wait();return fmt.Errorf("transcoding failed: %w",err)}
	if beforeCopy!=nil{beforeCopy()}
	buf:=transcodeBufferPool.Get().([]byte);defer transcodeBufferPool.Put(buf)
	_,copyErr:=io.CopyBuffer(dst,reader,buf)
	if copyErr!=nil{_ = cmd.Process.Kill()}
	waitErr:=cmd.Wait()
	if copyErr!=nil{return copyErr}
	if waitErr!=nil{return waitErr}
	return nil
}
