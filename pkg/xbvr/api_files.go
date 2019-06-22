package xbvr

import (
	"net/http"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type AssignSceneRequest struct {
	Path string `json:"path"`
}

type FilesResource struct{}

func (i FilesResource) WebService() *restful.WebService {
	tags := []string{"Files"}

	ws := new(restful.WebService)

	ws.Path("/api/files").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/list/unmatched").To(i.listUnmatchedFiles).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i FilesResource) listUnmatchedFiles(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Raw(`select files.* from files where files.scene_id = 0;`).Scan(&files)

	resp.WriteHeaderAndEntity(http.StatusOK, files)
}
