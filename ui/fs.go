package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed dist
var Assets embed.FS

func GetFileSystem(useOS bool) http.FileSystem {
	if useOS {
		return http.Dir("ui/dist")
	}

	fs, err := fs.Sub(Assets, "dist")
	if err != nil {
		log.Panic(err)
	}
	return http.FS(fs)
}
