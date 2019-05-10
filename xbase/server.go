package xbase

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/cld9x/xbvr/xbase/assets"
	"github.com/emicklei/go-restful"
	"github.com/gammazero/nexus/router"
	"github.com/gammazero/nexus/wamp"
	wwwlog "github.com/gowww/log"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/koding/websocketproxy"
	"github.com/peterbourgon/diskv"
	"github.com/rs/cors"
	"willnorris.com/go/imageproxy"
)

var (
	DEBUG    = os.Getenv("DEBUG")
	httpAddr = "0.0.0.0:9999"
	wsAddr   = "0.0.0.0:9998"
)

func StartServer() {
	// Remove old locks
	RemoveLock("scrape")
	RemoveLock("update-scenes")

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

	// WAMP router
	routerConfig := &router.Config{
		Debug: false,
		RealmConfigs: []*router.RealmConfig{
			{
				URI:           wamp.URI("default"),
				AnonymousAuth: true,
				AllowDisclose: false,
			},
		},
	}

	wampRouter, err := router.NewRouter(routerConfig, log)
	if err != nil {
		log.Fatal(err)
	}
	defer wampRouter.Close()

	// Run websocket server.
	wss := router.NewWebsocketServer(wampRouter)
	wss.AllowOrigins([]string{"*"})
	wsCloser, err := wss.ListenAndServe(wsAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer wsCloser.Close()

	// Proxy websocket
	wsURL, err := url.Parse("ws://" + wsAddr)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/ws/", func(w http.ResponseWriter, req *http.Request) {
		req.Header["Origin"] = nil
		handler := websocketproxy.ProxyHandler(wsURL)
		handler.ServeHTTP(w, req)
	})

	// Attach logrus hook
	wampHook := NewWampHook()
	log.AddHook(wampHook)


	log.Infof("XBVR starting...")

	// DMS
	go StartDMS()

	log.Infof("Web UI available at http://%v/", httpAddr)
	log.Infof("Database file stored at %s", appDir)

	if DEBUG == "" {
		log.Fatal(http.ListenAndServe(httpAddr, handler))
	} else {
		log.Infof("Running in DEBUG mode")
		log.Fatal(http.ListenAndServe(httpAddr, wwwlog.Handle(handler, &wwwlog.Options{Color: true})))
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
