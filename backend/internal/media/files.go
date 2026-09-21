package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func mediaRoot() (string,error) {
	root:=os.Getenv("MEDIA_PATH")
	if root==""{root="./music"}
	abs,err:=filepath.Abs(root)
	if err!=nil{return "",err}
	return abs,nil
}

// Open atomically constrains traversal to MEDIA_PATH and returns the already-open
// regular file. os.Root prevents symlink/path-component escapes while opening.
func Open(path string)(*os.File,error){
	rootPath,err:=mediaRoot();if err!=nil{return nil,err}
	abs,err:=filepath.Abs(path);if err!=nil{return nil,err}
	rel,err:=filepath.Rel(rootPath,abs)
	if err!=nil || rel==".." || filepath.IsAbs(rel) || strings.HasPrefix(rel,".."+string(filepath.Separator)){
		return nil,fmt.Errorf("file outside media directory")
	}
	root,err:=os.OpenRoot(rootPath);if err!=nil{return nil,err}
	defer root.Close()
	f,err:=root.Open(rel);if err!=nil{return nil,err}
	info,err:=f.Stat()
	if err!=nil{f.Close();return nil,err}
	if !info.Mode().IsRegular(){f.Close();return nil,fmt.Errorf("not a regular file")}
	return f,nil
}

func Resolve(path string)(string,error){
	f,err:=Open(path);if err!=nil{return "",err}
	defer f.Close()
	return f.Name(),nil
}

func ResolveWithin(root,path string)(string,error){
	absRoot,err:=filepath.Abs(root);if err!=nil{return "",err}
	absPath,err:=filepath.Abs(path);if err!=nil{return "",err}
	rel,err:=filepath.Rel(absRoot,absPath)
	if err!=nil || rel==".." || filepath.IsAbs(rel) || strings.HasPrefix(rel,".."+string(filepath.Separator)){return "",fmt.Errorf("file outside media directory")}
	r,err:=os.OpenRoot(absRoot);if err!=nil{return "",err}
	defer r.Close()
	f,err:=r.Open(rel);if err!=nil{return "",err}
	defer f.Close()
	info,err:=f.Stat();if err!=nil{return "",err}
	if !info.Mode().IsRegular(){return "",fmt.Errorf("not a regular file")}
	return f.Name(),nil
}

func ContentType(format string)string{
	switch strings.ToLower(format){
	case "mp3":return "audio/mpeg"
	case "aac":return "audio/aac"
	case "m4a","m4b","alac":return "audio/mp4"
	case "ogg","opus":return "audio/ogg"
	case "wav":return "audio/wav"
	case "flac":return "audio/flac"
	case "aiff":return "audio/aiff"
	case "wma":return "audio/x-ms-wma"
	default:return "application/octet-stream"
	}
}
