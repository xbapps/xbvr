package xbvr

import (
	"os"
	"runtime"

	"github.com/emicklei/go-restful"
	"github.com/gammazero/nexus/client"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
)

var log = logrus.New()

func APIError(req *restful.Request, resp *restful.Response, status int, err error) {
	// log.Error(req.Request.URL.String(), err)
	resp.WriteError(status, err)
}

type WampHook struct {
	publisher *client.Client
}

func NewWampHook() *WampHook {
	wh := &WampHook{}

	publisher, _ := client.ConnectNet("ws://"+wsAddr+"/ws", client.Config{
		Realm: "default",
	})

	wh.publisher = publisher

	return wh
}

func (hook *WampHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *WampHook) Fire(entry *logrus.Entry) error {
	err := hook.publisher.Publish("service.log", nil, nil, map[string]interface{}{
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"data":      entry.Data,
		"timestamp": entry.Time.String(),
	})
	if err != nil {
		return err
	}
	return nil
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
