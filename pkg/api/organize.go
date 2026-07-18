package api

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/organize"
)

type OrganizeResource struct{}

type RequestOrganizeRun struct {
	DryRun bool `json:"dryRun"`
	Limit  int  `json:"limit"`
}

type RequestOrganizeConfig struct {
	Dedup          bool   `json:"dedup"`
	DeferDups      bool   `json:"deferDups"`
	IncomingDir    string `json:"incomingDir"`
	IncomingMinAge int    `json:"incomingMinAge"`
	TopFolder      string `json:"topFolder"`
	CastGender     string `json:"castGender"`
	SymlinkByActor bool   `json:"symlinkByActor"`
	ActorFolder    string `json:"actorFolder"`
}

func (i OrganizeResource) WebService() *restful.WebService {
	tags := []string{"Organize"}
	ws := new(restful.WebService)
	ws.Path("/api/organize").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/run").To(i.run).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/status").To(i.status).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/config").To(i.getConfig).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/config").To(i.saveConfig).Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/duplicates/analyze").Consumes("*/*").To(i.dupAnalyze).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/duplicates").To(i.dupList).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/duplicates/ignore").To(i.dupIgnore).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/duplicates/unignore").To(i.dupUnignore).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/duplicates/delete").To(i.dupDelete).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/duplicates/disassociate").To(i.dupDisassociate).Metadata(restfulspec.KeyOpenAPITags, tags))
	return ws
}

func (i OrganizeResource) dupAnalyze(req *restful.Request, resp *restful.Response) {
	force := req.QueryParameter("force") == "true"
	if organize.StartDupAnalysis(force) {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
	} else {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "already-running"})
	}
}

func (i OrganizeResource) dupList(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()
	running, done, total := organize.DupStatus()
	showIgnored := req.QueryParameter("showIgnored") == "true"
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]interface{}{
		"running": running, "done": done, "total": total,
		"groups": organize.ListDupGroups(db, showIgnored),
	})
}

func (i OrganizeResource) dupIgnore(req *restful.Request, resp *restful.Response) {
	i.dupSetIgnore(req, resp, true)
}
func (i OrganizeResource) dupUnignore(req *restful.Request, resp *restful.Response) {
	i.dupSetIgnore(req, resp, false)
}
func (i OrganizeResource) dupSetIgnore(req *restful.Request, resp *restful.Response, ignore bool) {
	var r struct {
		FileID uint `json:"fileId"`
	}
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}
	db, _ := models.GetDB()
	defer db.Close()
	if ignore {
		organize.IgnoreFile(db, r.FileID)
	} else {
		organize.UnignoreFile(db, r.FileID)
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]bool{"ignored": ignore})
}

func (i OrganizeResource) dupDelete(req *restful.Request, resp *restful.Response) {
	var r struct {
		FileIDs []uint `json:"fileIds"`
	}
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}
	db, _ := models.GetDB()
	defer db.Close()
	n := organize.DeleteFiles(db, r.FileIDs)
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]int{"deleted": n})
}

func (i OrganizeResource) dupDisassociate(req *restful.Request, resp *restful.Response) {
	var r struct {
		FileIDs []uint `json:"fileIds"`
	}
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}
	db, _ := models.GetDB()
	defer db.Close()
	n := organize.DisassociateFiles(db, r.FileIDs)
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]int{"disassociated": n})
}

// optionsFromConfig builds run options from the persisted organize config.
func optionsFromConfig(dryRun bool, limit int) organize.Options {
	c := config.Config.Organize
	return organize.Options{
		DryRun:         dryRun,
		Limit:          limit,
		Dedup:          c.Dedup,
		DeferDups:      c.DeferDups,
		IncomingDir:    c.IncomingDir,
		IncomingMinAge: c.IncomingMinAge,
		TopFolder:      c.TopFolder,
		CastGender:     c.CastGender,
		SymlinkByActor: c.SymlinkByActor,
		ActorFolder:    c.ActorFolder,
	}
}

func (i OrganizeResource) run(req *restful.Request, resp *restful.Response) {
	var r RequestOrganizeRun
	if err := req.ReadEntity(&r); err != nil {
		log.Error(err)
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}
	if organize.Start(optionsFromConfig(r.DryRun, r.Limit)) {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
	} else {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "already-running"})
	}
}

func (i OrganizeResource) status(req *restful.Request, resp *restful.Response) {
	running, result := organize.Status()
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]interface{}{"running": running, "result": result})
}

func (i OrganizeResource) getConfig(req *restful.Request, resp *restful.Response) {
	resp.WriteHeaderAndEntity(http.StatusOK, config.Config.Organize)
}

func (i OrganizeResource) saveConfig(req *restful.Request, resp *restful.Response) {
	var r RequestOrganizeConfig
	if err := req.ReadEntity(&r); err != nil {
		log.Error(err)
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}
	config.Config.Organize.Dedup = r.Dedup
	config.Config.Organize.DeferDups = r.DeferDups
	config.Config.Organize.IncomingDir = r.IncomingDir
	config.Config.Organize.IncomingMinAge = r.IncomingMinAge
	config.Config.Organize.TopFolder = r.TopFolder
	config.Config.Organize.CastGender = r.CastGender
	config.Config.Organize.SymlinkByActor = r.SymlinkByActor
	config.Config.Organize.ActorFolder = r.ActorFolder
	config.SaveConfig()
	resp.WriteHeaderAndEntity(http.StatusOK, config.Config.Organize)
}
