package server

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xbapps/xbvr/pkg/common"
)

type MyFilesHandler struct {
}

// commonMimeTypes is an explicit fallback map. On Windows, mime.TypeByExtension
// reads from the registry and may return an empty string for common types,
// which breaks clients (like DeoVR) that require a correct Content-Type.
var commonMimeTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".json": "application/json",
}

func (h MyFilesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "..") { // prevent directory traversal
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	path := filepath.Join(common.MyFilesDir, r.URL.Path)
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// set Content-Type based on the file extension so clients (browsers, DeoVR)
	// correctly interpret images, json, etc. mime.TypeByExtension is unreliable
	// on Windows (reads from the registry), so use an explicit map first.
	ext := strings.ToLower(filepath.Ext(path))
	ctype := commonMimeTypes[ext]
	if ctype == "" {
		ctype = mime.TypeByExtension(ext)
	}
	if ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}

	// http.ServeContent handles Range requests, conditional GETs and sets
	// Accept-Ranges/Last-Modified headers. Strict clients like DeoVR require
	// these to load thumbnails reliably.
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
