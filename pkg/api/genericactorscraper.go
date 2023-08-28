package api

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/tasks"
)

type RequestActorScrape struct {
	ActorId int    `json:"id"`
	URL     string `json:"url"`
}

func (i ExternalReference) genericActorScraper(req *restful.Request, resp *restful.Response) {
	go tasks.ScrapeActors()
}

func (i ExternalReference) genericSingleActorScraper(req *restful.Request, resp *restful.Response) {
	var r RequestActorScrape
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	tasks.ScrapeActor(uint(r.ActorId), r.URL)
	resp.WriteHeader(http.StatusOK)
}

func (i ExternalReference) genericActorScraperBySite(req *restful.Request, resp *restful.Response) {
	siteId := req.PathParameter("site-id")
	go tasks.ScrapeActorBySite(siteId)
}
