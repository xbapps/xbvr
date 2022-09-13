package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	auth "github.com/abbot/go-http-auth"
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
	"github.com/xbapps/xbvr/pkg/analytics"
	"github.com/xbapps/xbvr/pkg/api"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/migrations"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
	"github.com/xbapps/xbvr/ui"
	"willnorris.com/go/imageproxy"
)

var (
	wsAddr = common.WsAddr
	log    = common.Log
)

func authHandle(pattern string, authEnabled bool, authSecret auth.SecretProvider, handler http.Handler) {
	if authEnabled {
		authenticator := auth.NewBasicAuthenticator("default", authSecret)
		http.HandleFunc(pattern, authenticator.Wrap(func(res http.ResponseWriter, req *auth.AuthenticatedRequest) {
			http.StripPrefix(pattern, handler).ServeHTTP(res, &req.Request)
		}))
	} else {
		http.Handle(pattern, http.StripPrefix(pattern, handler))
	}
}

func StartServer(version, commit, branch, date string) {
	common.CurrentVersion = version

	config.LoadConfig()

	// First setup
	migrations.Migrate()
	analytics.GenerateID()
	analytics.Event("app-start", nil)

	// Remove old locks
	models.RemoveLock("index")
	models.RemoveLock("scrape")
	models.RemoveLock("update-scenes")
	models.RemoveLock("previews")

	go tasks.CheckDependencies()
	models.CheckVolumes()

	models.InitSites()

	// API endpoints
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(func(req *restful.Request, resp *restful.Response) {
		resp.AddHeader("Location", "/ui/")
		resp.WriteHeader(http.StatusFound)
	}))

	restful.Add(ws)
	restful.Add(api.SceneResource{}.WebService())
	restful.Add(api.TaskResource{}.WebService())
	restful.Add(api.DMSResource{}.WebService())
	restful.Add(api.ConfigResource{}.WebService())
	restful.Add(api.FilesResource{}.WebService())
	restful.Add(api.DeoVRResource{}.WebService())
	restful.Add(api.HeresphereResource{}.WebService())
	restful.Add(api.PlaylistResource{}.WebService())
	restful.Add(api.AkaResource{}.WebService())

	restConfig := restfulspec.Config{
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
					Version: common.CurrentVersion,
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
				{
					TagProps: spec.TagProps{
						Name:        "HereSphere",
						Description: "Endpoints for interfacing with HereSphere player",
					},
				},
			}
		},
	}
	restful.Add(restfulspec.NewOpenAPIService(restConfig))

	// Static files
	authHandle("/ui/", common.IsUIAuthEnabled(), common.GetUISecret, http.FileServer(ui.GetFileSystem(common.EnvConfig.Debug)))

	// Imageproxy
	r := mux.NewRouter()
	p := imageproxy.NewProxy(nil, diskCache(filepath.Join(common.AppDir, "imageproxy")))
	p.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"
	r.PathPrefix("/img/").Handler(http.StripPrefix("/img", p))
	hmp := NewHeatmapThumbnailProxy(p, diskCache(filepath.Join(common.AppDir, "heatmapthumbnailproxy")))
	r.PathPrefix("/imghm/").Handler(http.StripPrefix("/imghm", hmp))
	downloadhandler := DownloadHandler{}
	r.PathPrefix("/download/").Handler(http.StripPrefix("/download/", downloadhandler))
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
	if config.Config.Interfaces.DLNA.Enabled {
		go tasks.StartDMS()
	}

	// DeoVR remote
	go session.DeoRemote()

	// Cron
	SetupCron()

	// List binding addresses
	addrs, _ := net.InterfaceAddrs()
	ips := []string{}
	for _, addr := range addrs {
		ip, _ := addr.(*net.IPNet)
		if ip.IP.To4() != nil {
			ips = append(ips, fmt.Sprintf("http://%v:%v/", ip.IP, config.Config.Server.Port))
		}
	}

	// Prepare state
	tasks.UpdateState()
	config.State.Server.BoundIP = ips
	config.SaveState()

	log.Infof("Web UI available at %s", strings.Join(ips, ", "))
	log.Infof("Web UI Authentication enabled: %v", common.IsUIAuthEnabled())
	log.Infof("Using database: %s", common.DATABASE_URL)

	httpAddr := fmt.Sprintf("%v:%v", config.Config.Server.BindAddress, config.Config.Server.Port)
	if common.EnvConfig.DebugRequests {
		log.Fatal(http.ListenAndServe(httpAddr, wwwlog.Handle(handler, &wwwlog.Options{Color: true})))
	} else {
		log.Fatal(http.ListenAndServe(httpAddr, handler))
	}
}

func diskCache(path string) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath:  path,
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}
