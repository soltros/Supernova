package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Resolve(path string) (string, error) {
	root := os.Getenv("MEDIA_PATH")
	if root == "" {
		root = "./music"
	}
	return ResolveWithin(root, path)
}

// ResolveWithin permits only regular files beneath the resolved root.
func ResolveWithin(root, path string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file outside media directory")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	return path, nil
}

func ContentType(format string) string {
	switch strings.ToLower(format) {
	case "mp3":
		return "audio/mpeg"
	case "aac":
		return "audio/aac"
	case "m4a", "m4b", "alac":
		return "audio/mp4"
	case "ogg", "opus":
		return "audio/ogg"
	case "wav":
		return "audio/wav"
	case "flac":
		return "audio/flac"
	case "aiff":
		return "audio/aiff"
	case "wma":
		return "audio/x-ms-wma"
	default:
		return "application/octet-stream"
	}
}
