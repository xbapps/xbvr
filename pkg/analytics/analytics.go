package analytics

import (
	"net/http"
	"runtime"

	"github.com/posthog/posthog-go"
	uuid "github.com/satori/go.uuid"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var distinctID string
var client posthog.Client

func GenerateID() {
	db, _ := models.GetDB()
	defer db.Close()

	// Check if already exists, generate new if needed
	var obj models.KV
	err := db.Where(&models.KV{Key: "distinctid"}).First(&obj).Error
	if err == nil {
		distinctID = obj.Value
	} else {
		uuid, _ := uuid.NewV4()
		distinctID = uuid.String()

		obj = models.KV{Key: "distinctid", Value: distinctID}
		obj.Save()
	}

	UserData()
}

func UserData() {
	if !common.EnvConfig.DisableAnalytics {
		client.Enqueue(posthog.Identify{
			DistinctId: distinctID,
			Properties: posthog.NewProperties().
				Set("platform", runtime.GOOS).
				Set("arch", runtime.GOARCH).
				Set("version", common.CurrentVersion),
		})
	}
}

func Event(event string, prop posthog.Properties) {
	if !common.EnvConfig.DisableAnalytics {
		client.Enqueue(posthog.Capture{
			DistinctId: distinctID,
			Event:      event,
			Properties: prop,
		})
	}
}

func Pageview(req *http.Request) {
	Event("$pageview", posthog.NewProperties().Set("$current_url", "http://xbvr"+req.URL.Path))
}

func init() {
	var err error
	client, err = posthog.NewWithConfig(
		"AZdgwtG2txnWDCmG9LrldQUy1UxgbgMuvhWgOY3U-BE",
		posthog.Config{Endpoint: "https://updates.xbvr.app"},
	)
	if err != nil {
		common.Log.Info(err)
	}
}
