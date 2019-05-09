package xbase

import (
	"os"
	"runtime"

	"github.com/emicklei/go-restful"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
)

var log = logrus.New()

func APIError(req *restful.Request, resp *restful.Response, status int, err error) {
	log.Error(req.Request.URL.String(), err)
	resp.WriteError(status, err)
}

func init() {
	log.Out = os.Stdout
	log.SetLevel(logrus.InfoLevel)

	if runtime.GOOS == "windows" {
		log.Formatter = &prefixed.TextFormatter{
			DisableColors: true,
		}
	} else {
		log.Formatter = &prefixed.TextFormatter{}
	}
}