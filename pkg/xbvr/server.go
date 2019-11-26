package xbvr

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/gammazero/nexus/v3/router"
	"github.com/gammazero/nexus/v3/wamp"
	"github.com/go-openapi/spec"
	"github.com/gorilla/mux"
	wwwlog "github.com/gowww/log"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/koding/websocketproxy"
	"github.com/peterbourgon/diskv"
	"github.com/rs/cors"
	"github.com/xbapps/xbvr/pkg/assets"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/migrations"
	"github.com/xbapps/xbvr/pkg/models"
	"willnorris.com/go/imageproxy"
)

var (
	DEBUG          = common.DEBUG
	DLNA           = common.DLNA
	httpAddr       = common.HttpAddr
	wsAddr         = common.WsAddr
	currentVersion = ""
)

func StartServer(version, commit, branch, date string) {
	currentVersion = version

	migrations.Migrate()

	// Remove old locks
	models.RemoveLock("index")
	models.RemoveLock("scrape")
	models.RemoveLock("update-scenes")

	go CheckDependencies()
	models.CheckVolumes()

	models.InitSites()

	// API endpoints
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(func(req *restful.Request, resp *restful.Response) {
		resp.AddHeader("Location", "/ui/")
		resp.WriteHeader(http.StatusFound)
	}))

	restful.Add(ws)
	restful.Add(SceneResource{}.WebService())
	restful.Add(TaskResource{}.WebService())
	restful.Add(DMSResource{}.WebService())
	restful.Add(ConfigResource{}.WebService())
	restful.Add(FilesResource{}.WebService())
	restful.Add(DeoVRResource{}.WebService())
	restful.Add(SecurityResource{}.WebService())

	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
		APIPath:     "/api.json",
		PostBuildSwaggerObjectHandler: func(swo *spec.Swagger) {
			var e = spec.VendorExtensible{}
			e.AddExtension("x-logo", map[string]interface{}{
				"url": "/ui/icons/xbvr-512.png",
			})

			swo.Info = &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "XBVR API",
					Version: currentVersion,
				},
				VendorExtensible: e,
			}
			swo.Tags = []spec.Tag{
				{
					TagProps: spec.TagProps{
						Name:        "Config",
						Description: "Endpoints used by options screen",
					},
				},
				{
					TagProps: spec.TagProps{
						Name:        "DeoVR",
						Description: "Endpoints for interfacing with DeoVR player",
					},
				},
			}
		},
	}
	restful.Add(restfulspec.NewOpenAPIService(config))

	// Static files
	if DEBUG == "" {
		http.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(assets.HTTP)))
	} else {
		http.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.Dir("ui/dist"))))
	}

	// Imageproxy
	r := mux.NewRouter()
	p := imageproxy.NewProxy(nil, diskCache(filepath.Join(common.AppDir, "imageproxy")))
	p.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"
	r.PathPrefix("/img/").Handler(http.StripPrefix("/img", p))
	r.SkipClean(true)

	r.PathPrefix("/").Handler(http.DefaultServeMux)

	// CORS
	handler := cors.Default().Handler(r)

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

	wampRouter, err := router.NewRouter(routerConfig, &log)
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
	wampHook := common.NewWampHook()
	log.AddHook(wampHook)

	log.Infof("XBVR %v (build date %v) starting...", version, date)

	// DMS
	log.Info("DLNA Enabled: ", DLNA)
	if DLNA {
		go StartDMS()
	}

	// Cron
	SetupCron()

	addrs, _ := net.InterfaceAddrs()
	ips := []string{}
	for _, addr := range addrs {
		ip, _ := addr.(*net.IPNet)
		if ip.IP.To4() != nil {
			ips = append(ips, fmt.Sprintf("http://%v:9999/", ip.IP))
		}
	}
	log.Infof("Web UI available at %s", strings.Join(ips, ", "))
	log.Infof("Database file stored at %s", common.AppDir)

	if DEBUG == "" {
		log.Fatal(http.ListenAndServe(httpAddr, handler))
	} else {
		log.Infof("Running in DEBUG mode")
		log.Fatal(http.ListenAndServe(httpAddr, wwwlog.Handle(handler, &wwwlog.Options{Color: true})))
	}
}

func diskCache(path string) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath:  path,
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}
