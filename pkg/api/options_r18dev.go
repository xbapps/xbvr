package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

// Custom R18.dev Scrapers, for the "Create Custom Site (JAV)" tab. Saved to scrapers.json in the app dir,
// loaded at the next start like upstream's custom sites.

type r18devCustomEntry struct {
	config.R18devScraperConfig
	Status string `json:"status"` // loaded, restart
}

type r18devCustomList struct {
	Scrapers    []r18devCustomEntry       `json:"scrapers"`
	Removed     []string                  `json:"removed"`     // deleted, still loaded until a restart
	Suggestions []config.R18devSuggestion `json:"suggestions"` // shipped suggestions not saved yet
}

type requestR18devCustomSave struct {
	Config  config.R18devScraperConfig `json:"config"`
	Editing bool                       `json:"editing"` // false: refuse to overwrite an existing ID
}

func writeR18devScraperList(list config.ScraperList) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(common.AppDir, "scrapers.json"), data, 0644)
}

func (i ConfigResource) listR18devCustom(req *restful.Request, resp *restful.Response) {
	var list config.ScraperList
	list.Load()
	out := r18devCustomList{Scrapers: []r18devCustomEntry{}, Removed: []string{}, Suggestions: []config.R18devSuggestion{}}
	saved := map[string]bool{}
	for _, cfg := range list.CustomScrapers.R18devScrapers {
		saved[strings.ToLower(cfg.ID)] = true
		out.Scrapers = append(out.Scrapers, r18devCustomEntry{cfg, scrape.R18devCustomStatus(cfg)})
	}
	for _, id := range scrape.R18devLoadedCustomIDs() {
		if !saved[id] {
			out.Removed = append(out.Removed, id)
		}
	}
	for _, s := range list.XbvrScrapers.R18devSuggestions {
		if !saved[strings.ToLower(s.ID)] {
			out.Suggestions = append(out.Suggestions, s)
		}
	}
	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) previewR18devList(req *restful.Request, resp *restful.Response) {
	src, err := scrape.R18devSourceFromURL(req.QueryParameter("url"))
	if err != nil {
		resp.WriteErrorString(http.StatusBadRequest, err.Error())
		return
	}
	p, err := scrape.R18devPreview(src)
	if err != nil {
		resp.WriteErrorString(http.StatusBadGateway, err.Error())
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, p)
}

func (i ConfigResource) saveR18devCustom(req *restful.Request, resp *restful.Response) {
	var r requestR18devCustomSave
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteErrorString(http.StatusBadRequest, err.Error())
		return
	}
	var list config.ScraperList
	list.Load()
	cfg := r.Config
	if err := scrape.R18devCheckCustom(&cfg, list.XbvrScrapers.R18devScrapers); err != nil {
		resp.WriteErrorString(http.StatusBadRequest, err.Error())
		return
	}
	replaced := false
	for idx, old := range list.CustomScrapers.R18devScrapers {
		if strings.EqualFold(old.ID, cfg.ID) {
			if !r.Editing {
				resp.WriteErrorString(http.StatusConflict, "you already have a custom scraper with the ID \""+cfg.ID+"\", pick another name")
				return
			}
			list.CustomScrapers.R18devScrapers[idx] = cfg
			replaced = true
		}
	}
	if !replaced {
		list.CustomScrapers.R18devScrapers = append(list.CustomScrapers.R18devScrapers, cfg)
	}
	if err := writeR18devScraperList(list); err != nil {
		resp.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	}
	log.Infof("Custom R18.dev scraper %s (r18dev-%s) saved, restart XBVR to load it", cfg.Name, cfg.ID)
	resp.WriteHeaderAndEntity(http.StatusOK, r18devCustomEntry{cfg, scrape.R18devCustomStatus(cfg)})
}

func (i ConfigResource) deleteR18devCustom(req *restful.Request, resp *restful.Response) {
	id := strings.ToLower(req.PathParameter("id"))
	var list config.ScraperList
	list.Load()
	kept := []config.R18devScraperConfig{}
	found := false
	for _, cfg := range list.CustomScrapers.R18devScrapers {
		if strings.EqualFold(cfg.ID, id) {
			found = true
			continue
		}
		kept = append(kept, cfg)
	}
	if !found {
		resp.WriteErrorString(http.StatusNotFound, "no custom scraper \""+id+"\"")
		return
	}
	list.CustomScrapers.R18devScrapers = kept
	if err := writeR18devScraperList(list); err != nil {
		resp.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	}
	// The row goes now; its scenes stay. After the restart they're orphans, and the next Scraper
	// that lists them adopts them.
	db, _ := models.GetCommonDB()
	db.Where("id = ?", "r18dev-"+id).Delete(&models.Site{})
	log.Infof("Custom R18.dev scraper r18dev-%s deleted, its scenes stay", id)
	resp.WriteHeader(http.StatusOK)
}
