package common

import (
	"context"
	"os"
	"runtime"

	"github.com/gammazero/nexus/v3/client"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
)

var Log = *logrus.New()

type WampHook struct {
	publisher *client.Client
}

func NewWampHook() *WampHook {
	wh := &WampHook{}

	publisher, _ := client.ConnectNet(context.Background(), "ws://"+WsAddr+"/ws", client.Config{
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
	Log.Out = os.Stdout
	Log.SetLevel(logrus.InfoLevel)
	if EnvConfig.Debug {
		Log.SetLevel(logrus.DebugLevel)
	}

	if runtime.GOOS == "windows" {
		Log.Formatter = &prefixed.TextFormatter{
			DisableColors: true,
		}
	} else {
		Log.Formatter = &prefixed.TextFormatter{}
	}
}
