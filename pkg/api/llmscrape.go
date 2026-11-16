package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/llmdraft"
	"github.com/xbapps/xbvr/pkg/llmscrape"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
	"github.com/xbapps/xbvr/pkg/tasks"
)

// secretMask stands in for a saved secret in responses; saving it back leaves the secret unchanged.
const secretMask = "********"

type LLMScrapeResource struct{}

type RequestLLMScrapeConfig struct {
	BaseURL              string  `json:"baseUrl"`
	Model                string  `json:"model"`
	APIKey               string  `json:"apiKey"`
	DisableThinking      bool    `json:"disableThinking"`
	StructuredOutput     string  `json:"structuredOutput"`
	TimeoutSeconds       int     `json:"timeoutSeconds"`
	MaxPageChars         int     `json:"maxPageChars"`
	BraveAPIKey          string  `json:"braveApiKey"`
	ResultsPerFile       int     `json:"resultsPerFile"`
	MinConfidence        float64 `json:"minConfidence"`
	Concurrency          int     `json:"concurrency"`
	MaxFollowRounds      int     `json:"maxFollowRounds"`
	SkipDownloadSites    bool    `json:"skipDownloadSites"`
	BlockedDomains       string  `json:"blockedDomains"`
	AllowPrivateNetworks bool    `json:"allowPrivateNetworks"`
}

type RequestFileContext struct {
	Context string `json:"context"`
}

type RequestScrapeURL struct {
	URL    string `json:"url"`
	FileID uint   `json:"file_id"`
}

type RequestSaveDraft struct {
	FileID uint `json:"file_id"`
}

type RequestLearnedDomain struct {
	Domain  string `json:"domain"`
	Allowed bool   `json:"allowed"`
}

type RequestLLMBatch struct {
	Force bool `json:"force"`
	Limit int  `json:"limit"`
}

func (i LLMScrapeResource) WebService() *restful.WebService {
	tags := []string{"LLM scraper"}
	ws := new(restful.WebService)
	ws.Path("/api/llmscrape").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/config").To(i.getConfig).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/config").To(i.saveConfig).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/test").Consumes("*/*").To(i.test).Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/context/{file-id}").To(i.getContext).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/context/{file-id}").To(i.setContext).Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/suggest/{file-id}").Consumes("*/*").To(i.suggest).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/scrape").To(i.scrapeURL).Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/drafts").To(i.listDrafts).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/draft/{draft-id}").To(i.getDraft).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/draft/{draft-id}/save").To(i.saveDraft).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.DELETE("/draft/{draft-id}").To(i.rejectDraft).Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/learned-domains").To(i.learnedDomains).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/learned-domains").To(i.setLearnedDomain).Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/batch/start").To(i.batchStart).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/batch/stop").Consumes("*/*").To(i.batchStop).Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.GET("/batch/status").To(i.batchStatus).Metadata(restfulspec.KeyOpenAPITags, tags))
	return ws
}

func mask(secret string) string {
	if secret == "" {
		return ""
	}
	return secretMask
}

func (i LLMScrapeResource) getConfig(req *restful.Request, resp *restful.Response) {
	c := config.Config.LLMScraper
	c.APIKey = mask(c.APIKey)
	c.BraveAPIKey = mask(c.BraveAPIKey)
	resp.WriteHeaderAndEntity(http.StatusOK, c)
}

func (i LLMScrapeResource) saveConfig(req *restful.Request, resp *restful.Response) {
	var r RequestLLMScrapeConfig
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	c := &config.Config.LLMScraper
	c.BaseURL = r.BaseURL
	c.Model = r.Model
	if r.APIKey != secretMask {
		c.APIKey = r.APIKey
	}
	c.DisableThinking = r.DisableThinking
	if r.StructuredOutput == "json_object" {
		c.StructuredOutput = "json_object"
	} else {
		c.StructuredOutput = "json_schema"
	}
	c.TimeoutSeconds = r.TimeoutSeconds
	c.MaxPageChars = r.MaxPageChars
	if r.BraveAPIKey != secretMask {
		c.BraveAPIKey = r.BraveAPIKey
	}
	c.ResultsPerFile = r.ResultsPerFile
	c.MinConfidence = r.MinConfidence
	c.Concurrency = r.Concurrency
	c.MaxFollowRounds = r.MaxFollowRounds
	c.SkipDownloadSites = r.SkipDownloadSites
	c.BlockedDomains = r.BlockedDomains
	c.AllowPrivateNetworks = r.AllowPrivateNetworks
	config.SaveConfig()
	i.getConfig(req, resp)
}

// test sends a minimal structured request, so a misconfigured endpoint fails here rather than
// half way through a batch.
func (i LLMScrapeResource) test(req *restful.Request, resp *restful.Response) {
	c := config.Config.LLMScraper
	client, err := llmscrape.NewClient(llmscrape.LLMConfig{
		BaseURL: c.BaseURL, Model: c.Model, APIKey: c.APIKey,
		DisableThinking: c.DisableThinking, StructuredOutput: c.StructuredOutput,
		Timeout: time.Duration(c.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"ok": map[string]any{"type": "boolean"}},
		"required":             []string{"ok"},
		"additionalProperties": false,
	}
	start := time.Now()
	_, err = client.Complete(req.Request.Context(),
		[]llmscrape.Message{{Role: "user", Content: `Reply with {"ok": true}.`}}, "test", schema, 50)
	out := map[string]any{
		"ok":              err == nil,
		"model":           c.Model,
		"latency_ms":      time.Since(start).Milliseconds(),
		"search_provider": map[bool]string{true: "brave", false: ""}[c.BraveAPIKey != ""],
	}
	if err != nil {
		out["error"] = err.Error()
	}
	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func pathUint(req *restful.Request, name string) (uint, bool) {
	v, err := strconv.ParseUint(req.PathParameter(name), 10, 64)
	return uint(v), err == nil
}

func (i LLMScrapeResource) getContext(req *restful.Request, resp *restful.Response) {
	id, ok := pathUint(req, "file-id")
	if !ok {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "invalid file id"})
		return
	}
	out := map[string]string{"context": models.GetFileMatchContext(id)}
	// The match screen adds these to its own search, so both searches use the scene's folder.
	if file, err := llmdraft.LoadFileInfo(id); err == nil {
		out["folder_terms"] = file.FolderTerms()
	}
	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i LLMScrapeResource) setContext(req *restful.Request, resp *restful.Response) {
	id, ok := pathUint(req, "file-id")
	var r RequestFileContext
	if !ok || req.ReadEntity(&r) != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := models.SetFileMatchContext(id, r.Context); err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"context": r.Context})
}

func (i LLMScrapeResource) suggest(req *restful.Request, resp *restful.Response) {
	id, ok := pathUint(req, "file-id")
	if !ok {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "invalid file id"})
		return
	}
	svc, err := llmdraft.NewService(scrape.UserAgent)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(req.Request.Context(), 10*time.Minute)
	defer cancel()
	res, err := svc.SuggestForFile(ctx, id, req.QueryParameter("refresh") == "true")
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, res)
}

func (i LLMScrapeResource) scrapeURL(req *restful.Request, resp *restful.Response) {
	var r RequestScrapeURL
	if err := req.ReadEntity(&r); err != nil || r.URL == "" {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}
	svc, err := llmdraft.NewService(scrape.UserAgent)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var file *llmscrape.FileInfo
	if r.FileID != 0 {
		if file, err = llmdraft.LoadFileInfo(r.FileID); err != nil {
			resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	ctx, cancel := context.WithTimeout(req.Request.Context(), 5*time.Minute)
	defer cancel()
	res := svc.ScrapeURL(ctx, r.URL, r.FileID, file, "", true)
	resp.WriteHeaderAndEntity(http.StatusOK, res)
}

func (i LLMScrapeResource) listDrafts(req *restful.Request, resp *restful.Response) {
	fileID, _ := strconv.ParseUint(req.QueryParameter("file_id"), 10, 64)
	status := req.QueryParameter("status")
	resp.WriteHeaderAndEntity(http.StatusOK, llmdraft.ListDrafts(uint(fileID), status))
}

func (i LLMScrapeResource) getDraft(req *restful.Request, resp *restful.Response) {
	id, ok := pathUint(req, "draft-id")
	if !ok {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "invalid draft id"})
		return
	}
	d, err := llmdraft.GetDraft(id)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, d)
}

func (i LLMScrapeResource) saveDraft(req *restful.Request, resp *restful.Response) {
	id, ok := pathUint(req, "draft-id")
	var r RequestSaveDraft
	if !ok || req.ReadEntity(&r) != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	sceneID, err := llmdraft.SaveDraft(id, r.FileID)
	if sceneID != "" {
		// New scenes are otherwise only indexed by the next full scrape, so without this the scene
		// would not show up in the match screen or library search.
		var saved models.Scene
		if saved.GetIfExist(sceneID) == nil {
			tasks.IndexScenes(&[]models.Scene{saved})
		}
	}
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error(), "scene_id": sceneID})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"scene_id": sceneID})
}

func (i LLMScrapeResource) rejectDraft(req *restful.Request, resp *restful.Response) {
	id, ok := pathUint(req, "draft-id")
	if !ok {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "invalid draft id"})
		return
	}
	if err := llmdraft.RejectDraft(id); err != nil {
		resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": models.DraftStatusRejected})
}

func (i LLMScrapeResource) batchStart(req *restful.Request, resp *restful.Response) {
	var r RequestLLMBatch
	_ = req.ReadEntity(&r)
	started, err := llmdraft.StartBatch(scrape.UserAgent, r.Force, r.Limit)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !started {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "already-running"})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
}

func (i LLMScrapeResource) batchStop(req *restful.Request, resp *restful.Response) {
	llmdraft.StopBatch()
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "stopping"})
}

func (i LLMScrapeResource) batchStatus(req *restful.Request, resp *restful.Response) {
	resp.WriteHeaderAndEntity(http.StatusOK, llmdraft.Status())
}

func (i LLMScrapeResource) learnedDomains(req *restful.Request, resp *restful.Response) {
	resp.WriteHeaderAndEntity(http.StatusOK, llmdraft.ListLearned())
}

func (i LLMScrapeResource) setLearnedDomain(req *restful.Request, resp *restful.Response) {
	var r RequestLearnedDomain
	if err := req.ReadEntity(&r); err != nil || r.Domain == "" {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": "domain is required"})
		return
	}
	if !llmdraft.SetLearnedAllowed(r.Domain, r.Allowed) {
		resp.WriteHeaderAndEntity(http.StatusNotFound, map[string]string{"error": "not a learned domain"})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, llmdraft.ListLearned())
}
