package xbase

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
)

type TaskResource struct{}

func (i TaskResource) WebService() *restful.WebService {
	tags := []string{"Task"}

	ws := new(restful.WebService)

	ws.Path("/api/task").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/rescan").To(i.rescan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i TaskResource) rescan(req *restful.Request, resp *restful.Response) {
	RescanVolumes()
}
