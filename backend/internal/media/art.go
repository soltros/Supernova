package media

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
)

// ServeArt validates legacy and new art before serving any bytes.
// False means the caller should return its protocol-specific not-found response.
func ServeArt(w http.ResponseWriter, r *http.Request, coverPath string) bool {
	root := os.Getenv("ART_CACHE_PATH")
	if root == "" {
		root = "./data/art_cache"
	}
	path, err := ResolveWithin(root, coverPath)
	if err != nil {
		path, err = Resolve(coverPath)
	}
	if err != nil {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() > 15*1024*1024 {
		return false
	}
	cfg, format, err := image.DecodeConfig(file)
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 25_000_000 {
		return false
	}
	contentType := map[string]string{"jpeg": "image/jpeg", "png": "image/png", "gif": "image/gif"}[format]
	if contentType == "" {
		return false
	}
	if _, err := file.Seek(0, 0); err != nil {
		return false
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return true
}
