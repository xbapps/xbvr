package api

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/recommend"
)

type RecommendationResource struct{}

// RequestRecommendationConfig mirrors config.Config.Recommendation for the UI.
type RequestRecommendationConfig struct {
	Enabled                bool    `json:"enabled"`
	UseLearnedModel        bool    `json:"useLearnedModel"`
	ModelType              string  `json:"modelType"`
	UseVisualEmbeddings    bool    `json:"useVisualEmbeddings"`
	WatchListSize          int     `json:"watchListSize"`
	DeleteListSize         int     `json:"deleteListSize"`
	ProtectRating          float64 `json:"protectRating"`
	GraceDays              int     `json:"graceDays"`
	ExcludeRecentlyWatched bool    `json:"excludeRecentlyWatched"`
	DiversityDecay         float64 `json:"diversityDecay"`
	WActor                 float64 `json:"wActor"`
	WTag                   float64 `json:"wTag"`
	WSite                  float64 `json:"wSite"`
	WQuality               float64 `json:"wQuality"`
	WFreshness             float64 `json:"wFreshness"`
	WSize                  float64 `json:"wSize"`
	WVisualQuality         float64 `json:"wVisualQuality"`
	VQMaxSamples           int     `json:"vqMaxSamples"`
	NoiseWeight            float64 `json:"noiseWeight"`
}

func (i RecommendationResource) WebService() *restful.WebService {
	tags := []string{"Recommendations"}

	ws := new(restful.WebService)
	ws.Path("/api/recommendations").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/config").To(i.getConfig).
		Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/config").To(i.saveConfig).
		Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/recompute").Consumes("*/*").To(i.recompute).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i RecommendationResource) getConfig(req *restful.Request, resp *restful.Response) {
	resp.WriteHeaderAndEntity(http.StatusOK, config.Config.Recommendation)
}

func (i RecommendationResource) saveConfig(req *restful.Request, resp *restful.Response) {
	var r RequestRecommendationConfig
	if err := req.ReadEntity(&r); err != nil {
		log.Error(err)
		resp.WriteHeaderAndEntity(http.StatusBadRequest, nil)
		return
	}

	config.Config.Recommendation.Enabled = r.Enabled
	config.Config.Recommendation.UseLearnedModel = r.UseLearnedModel
	config.Config.Recommendation.ModelType = r.ModelType
	config.Config.Recommendation.UseVisualEmbeddings = r.UseVisualEmbeddings
	config.Config.Recommendation.WatchListSize = r.WatchListSize
	config.Config.Recommendation.DeleteListSize = r.DeleteListSize
	config.Config.Recommendation.ProtectRating = r.ProtectRating
	config.Config.Recommendation.GraceDays = r.GraceDays
	config.Config.Recommendation.ExcludeRecentlyWatched = r.ExcludeRecentlyWatched
	config.Config.Recommendation.DiversityDecay = r.DiversityDecay
	config.Config.Recommendation.WActor = r.WActor
	config.Config.Recommendation.WTag = r.WTag
	config.Config.Recommendation.WSite = r.WSite
	config.Config.Recommendation.WQuality = r.WQuality
	config.Config.Recommendation.WFreshness = r.WFreshness
	config.Config.Recommendation.WSize = r.WSize
	config.Config.Recommendation.WVisualQuality = r.WVisualQuality
	config.Config.Recommendation.VQMaxSamples = r.VQMaxSamples
	config.Config.Recommendation.NoiseWeight = r.NoiseWeight
	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, config.Config.Recommendation)
}

func (i RecommendationResource) recompute(req *restful.Request, resp *restful.Response) {
	go recommend.Generate()
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
}
