package api

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type RequestScrapeJAVR struct {
	Query string `json:"q"`
}

type TaskResource struct{}

func (i TaskResource) WebService() *restful.WebService {
	tags := []string{"Task"}

	ws := new(restful.WebService)

	ws.Path("/api/task").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/rescan").To(i.rescan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/clean-tags").To(i.cleanTags).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scrape").To(i.scrape).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/index").To(i.index).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/bundle/import").To(i.importBundle).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/preview/generate").To(i.previewGenerate).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/bundle/export").To(i.exportBundle).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scrape-javr").To(i.scrapeJAVR).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i TaskResource) rescan(req *restful.Request, resp *restful.Response) {
	go tasks.RescanVolumes()
}

func (i TaskResource) cleanTags(req *restful.Request, resp *restful.Response) {
	go tasks.CleanTags()
}

func (i TaskResource) index(req *restful.Request, resp *restful.Response) {
	go tasks.SearchIndex()
}

func (i TaskResource) scrape(req *restful.Request, resp *restful.Response) {
	qSiteID := req.QueryParameter("site")
	if qSiteID == "" {
		qSiteID = "_enabled"
	}
	go tasks.Scrape(qSiteID)
}

func (i TaskResource) importBundle(req *restful.Request, resp *restful.Response) {
	url := req.QueryParameter("url")
	go tasks.ImportBundle(url)
}

func (i TaskResource) exportBundle(req *restful.Request, resp *restful.Response) {
	go tasks.ExportBundle()
}

func (i TaskResource) previewGenerate(req *restful.Request, resp *restful.Response) {
	go tasks.GeneratePreviews()
}

func (i TaskResource) scrapeJAVR(req *restful.Request, resp *restful.Response) {
	var r RequestScrapeJAVR
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.Query != "" {
		go tasks.ScrapeJAVR(r.Query)
	}
}
