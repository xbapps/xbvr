package api

import (
	"net/http"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/models"
)

// var RequestBody []byte
type RequestEditExtRefLink struct {
	ID                  uint      `json:"id"`
	ExternalReferenceID uint      `json:"external_reference_id"`
	ExternalSource      string    `json:"external_source"`
	ExternalId          string    `json:"external_id"`
	MatchType           int       `json:"match_type"`
	InternalTable       string    `json:"internal_table"`
	InternalDbId        uint      `json:"internal_db_id"`
	InternalNameId      string    `json:"internal_name_id"`
	DeleteDate          time.Time `json:"delete_date"`
}

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
	ws.Route(ws.POST("/edit_link").To(i.editExtRefLink).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes())
	ws.Route(ws.DELETE("/delete_extref").To(i.deleteExtRefLink).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes())
	ws.Route(ws.DELETE("/delete_extref_source").To(i.deleteExtRefSource).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes())
	ws.Route(ws.DELETE("/delete_extref_source_links/all").To(i.deleteExtRefSourceLinks).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes())
	ws.Route(ws.DELETE("/delete_extref_source_links/keep_manual").To(i.deleteExtRefSourceLinksKeepManualMatches).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes())
	return ws
}

func (i ExternalReference) editExtRefLink(req *restful.Request, resp *restful.Response) {
	var r RequestEditExtRefLink
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var extreflink models.ExternalReferenceLink
	if r.ID > 0 {
		extreflink.ExternalReference.GetIfExist(r.ID)
	} else {
		extreflink.FindByExternaID(r.ExternalSource, r.ExternalId)
	}
	extreflink.InternalTable = r.InternalTable
	extreflink.InternalDbId = r.InternalDbId
	extreflink.InternalNameId = r.InternalNameId
	extreflink.MatchType = r.MatchType
	extreflink.Save()
	resp.WriteHeaderAndEntity(http.StatusOK, extreflink)
}
func (i ExternalReference) deleteExtRefLink(req *restful.Request, resp *restful.Response) {
	// delete a single external_reference_link
	var r RequestEditExtRefLink
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var extreflink models.ExternalReferenceLink
	if r.ID > 0 {
		extreflink.ExternalReference.GetIfExist(r.ID)
	} else {
		extreflink.FindByExternaID(r.ExternalSource, r.ExternalId)
	}
	extreflink.ExternalReference.Delete()
	extreflink.Delete()
	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}
func (i ExternalReference) deleteExtRefSource(req *restful.Request, resp *restful.Response) {
	// deletes all external_reference_links and external_references for a source
	var r RequestEditExtRefLink
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}
	commonDb, _ := models.GetCommonDB()

	commonDb.Where("external_source = ?", r.ExternalSource).Delete(models.ExternalReferenceLink{})
	commonDb.Where("external_source = ?", r.ExternalSource).Delete(models.ExternalReference{})

	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}
func (i ExternalReference) deleteExtRefSourceLinks(req *restful.Request, resp *restful.Response) {
	// deletes external_reference_links for a source
	var r RequestEditExtRefLink
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}
	commonDb, _ := models.GetCommonDB()
	commonDb.Where("external_source like ?", r.ExternalSource).Delete(models.ExternalReferenceLink{})

	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}
func (i ExternalReference) deleteExtRefSourceLinksKeepManualMatches(req *restful.Request, resp *restful.Response) {
	// deletes external_reference_links for a source, but keeps links the user has manually set, ie match_type = 99999
	var r RequestEditExtRefLink
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}
	db, _ := models.GetDB()
	defer db.Close()

	if r.DeleteDate.IsZero() {
		db.Where("external_source like ? and match_type not in (99999, -1)", r.ExternalSource).Delete(models.ExternalReferenceLink{})
	} else {
		// Fetch records to delete
		var recordsToDelete []models.ExternalReferenceLink
		db.Debug().Joins("JOIN external_references ON external_reference_links.external_reference_id = external_references.id").
			Where("external_reference_links.external_source LIKE ? AND match_type NOT IN (99999, -1) AND external_references.external_date >= ?", r.ExternalSource, r.DeleteDate).
			Find(&recordsToDelete)
		for _, record := range recordsToDelete {
			db.Debug().Delete(&record)
		}

	}

	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}
