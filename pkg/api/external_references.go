package api

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

//var RequestBody []byte

type ExternalReference struct{}

func (i ExternalReference) WebService() *restful.WebService {
	tags := []string{"Extref"}

	ws := new(restful.WebService)

	ws.Path("/api/extref").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/stashdb/apply_scene_rules").To(i.stashSceneApplyRules).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/stashdb/match_akas").To(i.matchAkaPerformers).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/stashdb/update_performer_data").To(i.stashDbUpdateData).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/stashdb/run_all").To(i.stashRunAll).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/stashdb/refresh_performer/{performerid}").To(i.refreshStashPerformer).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/generic/scrape_all").To(i.genericActorScraper).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/generic/scrape_single").To(i.genericSingleActorScraper).
		Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/generic/scrape_by_site/{site-id}").To(i.genericActorScraperBySite).
		Metadata(restfulspec.KeyOpenAPITags, tags))
	return ws
}
