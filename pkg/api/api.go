package api

import (
	"github.com/emicklei/go-restful"
	"github.com/xbapps/xbvr/pkg/common"
)

var log = common.Log

func APIError(req *restful.Request, resp *restful.Response, status int, err error) {
	resp.WriteError(status, err)
}
