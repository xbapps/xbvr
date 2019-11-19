package xbvr

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/models"
)

type RequestActorList struct {
	Limit  optional.Int    `json:"limit"`
	Offset optional.Int    `json:"offset"`
	Sort   optional.String `json:"sort"`
}

type ResponseGetActors struct {
	Results int            `json:"results"`
	Actors  []models.Actor `json:"actors"`
}

type ActorResource struct{}

func (i ActorResource) WebService() *restful.WebService {
	tags := []string{"Actors"}

	ws := new(restful.WebService)

	ws.Path("/api/actor").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/list").To(i.getActors).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	return ws
}

func (i ActorResource) getActors(req *restful.Request, resp *restful.Response) {
	var r RequestActorList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var total = 0

	limit := r.Limit.OrElse(100)
	offset := r.Offset.OrElse(0)

	db, _ := models.GetDB()
	defer db.Close()

	var actors []models.Actor
	tx := db.Model(&actors)

	// We need some filters here... at the very least only display
	// actors with available scenes

	switch r.Sort.OrElse("") {
	default:
		tx = tx.Order("name asc")
	}

	// Count totals first
	tx.
		Group("actors.actor_id").
		Count(&total)

	// Get scenes
	tx.
		Group("actors.actor_id").
		Limit(limit).
		Offset(offset).
		Find(&actors)

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetActors{Results: total, Actors: actors})
}
