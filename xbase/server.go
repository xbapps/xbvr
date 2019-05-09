package xbase

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cld9x/xbvr/xbase/assets"
	"github.com/emicklei/go-restful"
	wwwlog "github.com/gowww/log"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"github.com/rs/cors"
	"willnorris.com/go/imageproxy"
)

var (
	DEBUG = os.Getenv("DEBUG")
)

func StartServer() {
	CheckVolumes()

	// API endpoints
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(redirectUI))

	restful.Add(ws)
	restful.Add(ExtResource{}.WebService())
	restful.Add(SceneResource{}.WebService())
	restful.Add(TaskResource{}.WebService())
	restful.Add(DMSResource{}.WebService())

	// Static files
	if DEBUG == "" {
		http.Handle("/static/", http.FileServer(assets.HTTP))
	} else {
		http.Handle("/static/", http.FileServer(http.Dir("ui/public")))
	}

	// SPA
	http.HandleFunc("/ui/", func(resp http.ResponseWriter, req *http.Request) {
		b, _ := assets.ReadFile("index.html")
		resp.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(resp, string(b))
	})

	// Imageproxy
	p := imageproxy.NewProxy(nil, diskCache(filepath.Join(appDir, "imageproxy")))
	http.Handle("/img/", http.StripPrefix("/img", p))

	// CORS
	handler := cors.Default().Handler(http.DefaultServeMux)

	log.Infof("XBVR starting...")

	go StartDMS()

	log.Infof("Web UI available at http://127.0.0.1:9999/")
	log.Infof("Database file stored at %s", appDir)

	if DEBUG == "" {
		log.Fatal(http.ListenAndServe(":9999", handler))
	} else {
		log.Infof("Running in DEBUG mode")
		log.Fatal(http.ListenAndServe(":9999", wwwlog.Handle(handler, &wwwlog.Options{Color: true})))
	}
}

func redirectUI(req *restful.Request, resp *restful.Response) {
	resp.AddHeader("Location", "/ui/")
	resp.WriteHeader(http.StatusFound)
}

func diskCache(path string) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath:  path,
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}
