package xbvr

import (
	"github.com/emicklei/go-restful"
)

func APIError(req *restful.Request, resp *restful.Response, status int, err error) {
	resp.WriteError(status, err)
}
