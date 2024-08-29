package api

import (
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

func (i ExternalReference) refreshStashPerformer(req *restful.Request, resp *restful.Response) {
	performerId := req.PathParameter("performerid")
	scrape.RefreshPerformer(performerId)
	resp.WriteHeader(http.StatusOK)
}

func (i ExternalReference) stashSceneApplyRules(req *restful.Request, resp *restful.Response) {
	go externalreference.ApplySceneRules()
}

func (i ExternalReference) matchAkaPerformers(req *restful.Request, resp *restful.Response) {
	go externalreference.MatchAkaPerformers()

}
func (i ExternalReference) stashDbUpdateData(req *restful.Request, resp *restful.Response) {
	go externalreference.UpdateAllPerformerData()

}
func (i ExternalReference) stashRunAll(req *restful.Request, resp *restful.Response) {
	StashdbRunAll()
}

func StashdbRunAll() {
	go func() {
		if !models.CheckLock("scrape") {
			models.CreateLock("scrape")
			defer models.RemoveLock("scrape")

			t0 := time.Now()
			tlog := log.WithField("task", "scrape")
			tlog.Infof("StashDB Refresh started at %s", t0.Format("Mon Jan _2 15:04:05 2006"))
			scrape.StashDb()

			externalreference.ApplySceneRules()
			externalreference.MatchAkaPerformers()
			externalreference.UpdateAllPerformerData()
			tlog = log.WithField("task", "scrape")
			tlog.Infof("Stashdb Refresh Complete in %s",
				time.Since(t0).Round(time.Second))
		}
	}()
}
