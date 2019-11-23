package xbvr

import (
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type RequestEnableDLNA struct {
	Enabled bool `json:"enabled"`
}

type SecurityResource struct{}

func (i SecurityResource) WebService() *restful.WebService {
	tags := []string{"Config"}

	ws := new(restful.WebService)

	ws.Path("/api/security").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.PUT("/enableDLNA").To(i.enableDLNA).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i SecurityResource) enableDLNA(req *restful.Request, resp *restful.Response) {
	var r RequestEnableDLNA
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.Enabled {
		StartDMS()
	} else {
		StopDMS()
	}
}
