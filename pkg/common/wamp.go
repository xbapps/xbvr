package common

import (
	"context"

	"github.com/gammazero/nexus/v3/client"
)

func PublishWS(topic string, message map[string]interface{}) error {
	publisher, err := client.ConnectNet(context.Background(), "ws://"+WsAddr+"/ws", client.Config{Realm: "default"})
	if err == nil {
		if EnvConfig.DebugWS {
			Log.Debugf("Sending WAMP message: %v %v", topic, message)
		}
		publisher.Publish(topic, nil, nil, message)
		publisher.Close()
	}
	return err
}
