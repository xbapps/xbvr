package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xbapps/xbvr/pkg/common"
)

type MyFilesHandler struct {
}

func (h MyFilesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(common.MyFilesDir, r.URL.Path)
	fi, err := os.Stat(path)

	if os.IsNotExist(err) || strings.Contains(path, "..") { // check the path exists and not trying to go up directory levels
		// file does not exist
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	//copy the relevant headers. If you want to preserve the downloaded file name, extract it with go's url parser.
	if strings.HasSuffix(path, ".json") {
		w.Header().Set("Content-Type", r.Header.Get("application/json"))
	}
	w.Header().Set("Content-Length", fmt.Sprint(fi.Size())) // useful for download progress

	//stream the body to the client without fully loading it into memory
	reader, _ := os.Open(path)
	io.Copy(w, reader)
}
