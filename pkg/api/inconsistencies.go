package api

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/inconsistencies"
	"github.com/xbapps/xbvr/pkg/models"
)

type InconsistenciesResource struct{}

func (i InconsistenciesResource) WebService() *restful.WebService {
	tags := []string{"Inconsistencies"}
	ws := new(restful.WebService)
	ws.Path("/api/inconsistencies").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(i.list).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/fix").To(i.fix).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/fixall").Consumes("*/*").To(i.fixAll).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/status").To(i.status).Metadata(restfulspec.KeyOpenAPITags, tags))
	return ws
}

// list scans and returns the current inconsistencies without changing anything.
func (i InconsistenciesResource) list(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()
	items := inconsistencies.ScanInconsistencies(db)
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]interface{}{"count": len(items), "items": items})
}

// fixAll scans and applies every suggested fix in the background.
func (i InconsistenciesResource) fixAll(req *restful.Request, resp *restful.Response) {
	if inconsistencies.StartFixInconsistencies() {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
	} else {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "already-running"})
	}
}

// status reports fix-all progress and the most recent result.
func (i InconsistenciesResource) status(req *restful.Request, resp *restful.Response) {
	running, phase, done, total, result := inconsistencies.FixStatus()
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]interface{}{
		"running": running, "phase": phase, "done": done, "total": total, "result": result,
	})
}

// fix applies a single suggested fix.
func (i InconsistenciesResource) fix(req *restful.Request, resp *restful.Response) {
	var r struct {
		Action        string `json:"action"` // rematch | unmatch | refresh | recount-tag | recount-actor
		FileID        uint   `json:"fileId"`
		SceneID       uint   `json:"sceneId"`
		TargetSceneID uint   `json:"targetSceneId"`
		EntityID      uint   `json:"entityId"`
	}
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}
	db, _ := models.GetDB()
	defer db.Close()
	var err error
	switch r.Action {
	case "rematch":
		err = inconsistencies.RematchFile(db, r.FileID, r.TargetSceneID)
	case "unmatch":
		err = inconsistencies.UnmatchFile(db, r.FileID)
	case "refresh":
		err = inconsistencies.RefreshScene(db, r.SceneID)
	case "recount-tag":
		err = inconsistencies.RecountTag(db, r.EntityID)
	case "recount-actor":
		err = inconsistencies.RecountActor(db, r.EntityID)
	default:
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "unknown action"})
		return
	}
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]bool{"ok": true})
}
