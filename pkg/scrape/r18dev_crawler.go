package scrape

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// R18.dev crawler: one XBVR scraper per entry in the "r18dev" section of scrapers.json.
// Each entry is a list of R18.dev sources (labels and/or makers), crawled through the same
// list2 JSON endpoint the R18.dev site uses. Detail JSON is only fetched for content IDs
// that aren't already in XBVR.

const (
	r18devBaseURL     = "https://r18.dev/videos/vod/movies"
	r18devPageSize    = 100
	r18devDelay       = 1 * time.Second
	r18devCatVR       = 6793 // VR Exclusive, required
	r18devCatCompil   = 6003 // compilation, skipped by the crawler
	r18devMakerGlobal = 40150
	r18devCoverDays   = 30 // check covers for scenes released in the last N days (or not released yet)
)

// Labels that are never scraped, even when a maker source pulls them in.
// 26286 = OVVR (SD card releases, compilations), shows up under S1, MOODYZ, Idea Pocket, AQUA, ...
var r18devExcludedLabels = map[int64]bool{
	26286: true,
}

type r18devListItem struct {
	ContentID   string
	DvdID       string
	ReleaseDate string
}

// Content ID from the URLs XBVR scenes carry: R18.dev (id=, combined=) and FANZA / DMM (cid=)
var r18devURLContentIDRe = regexp.MustCompile(`(?i)(?:[/?&]cid=|/id=|/combined=)([a-z0-9_]+)`)

// Content ID from a DMM image URL: .../digital/video/<cid>/<cid>pl.jpg
var r18devCoverContentIDRe = regexp.MustCompile(`(?i)/digital/video/([a-z0-9_]+)/`)

// r18devContentIDFromURL returns the content ID in an R18.dev or DMM scene URL, lowercased and
// without a reissue suffix, or "" if there isn't one.
func r18devContentIDFromURL(u string) string {
	m := r18devURLContentIDRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return r18devStripReissue(strings.ToLower(m[1]))
}

func r18devContentIDFromCover(u string) string {
	m := r18devCoverContentIDRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return r18devStripReissue(strings.ToLower(m[1]))
}

// One shared pace for every R18.dev scraper: XBVR runs scrapers in parallel, R18.dev should
// still only see one request per r18devDelay from us.
var (
	r18devPaceLock sync.Mutex
	r18devLastReq  time.Time
)

type r18devClient struct {
	http     *resty.Client
	requests int
}

func newR18devClient() *r18devClient {
	c := resty.New()
	c.SetHeader("User-Agent", UserAgent)
	c.SetTimeout(30 * time.Second)
	if config.Config.Advanced.ScraperProxy != "" {
		c.SetProxy(config.Config.Advanced.ScraperProxy)
	}
	return &r18devClient{http: c}
}

func (c *r18devClient) pace() {
	r18devPaceLock.Lock()
	defer r18devPaceLock.Unlock()
	if wait := r18devDelay - time.Since(r18devLastReq); wait > 0 {
		time.Sleep(wait)
	}
	r18devLastReq = time.Now()
	c.requests++
}

func (c *r18devClient) get(url string) (*resty.Response, error) {
	c.pace()
	return c.http.R().Get(url)
}

// listPage fetches one page of a label / studio list. Returns the items and total_results.
func (c *r18devClient) listPage(srcType string, id int64, page int) ([]r18devListItem, int, error) {
	url := fmt.Sprintf("%s/list2/json?id=%d&type=%s&page=%d", r18devBaseURL, id, srcType, page)
	res, err := c.get(url)
	if err != nil {
		return nil, 0, err
	}
	if res.StatusCode() != 200 {
		return nil, 0, fmt.Errorf("HTTP %d for %s", res.StatusCode(), url)
	}
	body := res.String()
	var items []r18devListItem
	for _, r := range gjson.Get(body, "results").Array() {
		items = append(items, r18devListItem{
			ContentID:   strings.ToLower(r.Get("content_id").String()),
			DvdID:       r.Get("dvd_id").String(),
			ReleaseDate: r.Get("release_date").String(),
		})
	}
	return items, int(gjson.Get(body, "total_results").Int()), nil
}

// listSource fetches every page of one source (only page 1 when limitScraping is set).
func (c *r18devClient) listSource(src config.R18devSource, limitScraping bool) ([]r18devListItem, error) {
	items, total, err := c.listPage(src.Type, src.ID, 1)
	if err != nil {
		return nil, err
	}
	pages := (total + r18devPageSize - 1) / r18devPageSize
	if limitScraping {
		pages = 1
	}
	for p := 2; p <= pages; p++ {
		more, _, err := c.listPage(src.Type, src.ID, p)
		if err != nil {
			return items, err
		}
		items = append(items, more...)
	}
	return items, nil
}

// detail fetches the detail JSON for a content ID. Returns "" on 404.
func (c *r18devClient) detail(contentID string) (string, error) {
	res, err := c.get(r18devBaseURL + "/detail/-/combined=" + contentID + "/json")
	if err != nil {
		return "", err
	}
	if res.StatusCode() == 404 {
		return "", nil
	}
	if res.StatusCode() != 200 {
		return "", fmt.Errorf("HTTP %d for %s", res.StatusCode(), contentID)
	}
	return res.String(), nil
}

// coverReady is false while DMM serves its "NOW PRINTING" placeholder, which comes back as a
// 404 with an image body at the same URL the real cover will use later.
func (c *r18devClient) coverReady(url string) bool {
	if url == "" {
		return false
	}
	c.pace()
	res, err := c.http.R().Head(url)
	if err != nil {
		return false
	}
	return res.StatusCode() == 200
}

func r18devSiteID(cfg config.R18devScraperConfig) string {
	return "r18dev-" + strings.ToLower(cfg.ID)
}

func r18devIsOurs(scraperID string) bool {
	return strings.HasPrefix(scraperID, "r18dev-")
}

// ---- existing scenes, for the write guards ----

type r18devExisting struct {
	SceneID     string
	ScraperID   string
	Studio      string
	ReleaseDate string
	NeedsUpdate bool
}

type r18devIndex struct {
	byContentID map[string][]*r18devExisting
	bySceneID   map[string]*r18devExisting
}

func r18devLoadIndex() *r18devIndex {
	idx := &r18devIndex{byContentID: map[string][]*r18devExisting{}, bySceneID: map[string]*r18devExisting{}}
	db, _ := models.GetCommonDB()
	var scenes []models.Scene
	db.Select("scene_id, scraper_id, scene_url, cover_url, studio, release_date, needs_update").Find(&scenes)
	for _, s := range scenes {
		e := &r18devExisting{SceneID: s.SceneID, ScraperID: s.ScraperId, Studio: s.Studio, NeedsUpdate: s.NeedsUpdate}
		if !s.ReleaseDate.IsZero() {
			e.ReleaseDate = s.ReleaseDate.Format("2006-01-02")
		}
		idx.bySceneID[s.SceneID] = e
		seen := map[string]bool{}
		for _, cid := range []string{r18devContentIDFromURL(s.SceneURL), r18devContentIDFromCover(s.CoverURL)} {
			if cid != "" && !seen[cid] {
				seen[cid] = true
				idx.byContentID[cid] = append(idx.byContentID[cid], e)
			}
		}
	}
	return idx
}

// ---- rejected content IDs (not VR, compilation, excluded label), remembered between runs ----

func r18devRejectedKey(scraperID string) string {
	return "r18dev-rejected-" + scraperID
}

func r18devLoadRejected(scraperID string) map[string]bool {
	rejected := map[string]bool{}
	var kv models.KV
	db, _ := models.GetCommonDB()
	if db.Where(&models.KV{Key: r18devRejectedKey(scraperID)}).First(&kv).Error == nil {
		for _, cid := range strings.Split(kv.Value, ",") {
			if cid != "" {
				rejected[cid] = true
			}
		}
	}
	return rejected
}

func r18devSaveRejected(scraperID string, rejected map[string]bool) {
	list := make([]string, 0, len(rejected))
	for cid := range rejected {
		list = append(list, cid)
	}
	sort.Strings(list)
	kv := models.KV{Key: r18devRejectedKey(scraperID), Value: strings.Join(list, ",")}
	kv.Save()
}

// ---- scene building ----

// r18devBuildScene turns detail JSON into a scene for this scraper, or returns a reason to skip.
// reject = true means the content ID should never be fetched again (not VR, compilation, ...).
func r18devBuildScene(c *r18devClient, body string, scraperID string) (sc models.ScrapedScene, skip string, reject bool) {
	cats := map[int64]bool{}
	for _, cat := range gjson.Get(body, "categories").Array() {
		cats[cat.Get("id").Int()] = true
	}
	if !cats[r18devCatVR] {
		return sc, "not VR", true
	}
	if cats[r18devCatCompil] {
		return sc, "compilation", true
	}
	if r18devExcludedLabels[gjson.Get(body, "label_id").Int()] {
		return sc, "excluded label", true
	}

	// NOW PRINTING: only recent / upcoming scenes; old ones can have permanent 404 covers
	released := gjson.Get(body, "release_date").String()
	if t, err := time.Parse("2006-01-02", released); err == nil && time.Since(t) < r18devCoverDays*24*time.Hour {
		if !c.coverReady(gjson.Get(body, "jacket_full_url").String()) {
			return sc, "cover not ready yet", false
		}
	}

	sc = r18dSceneFromJSON(body)
	contentID := sc.SiteID
	dvdID := gjson.Get(body, "dvd_id").String()
	if dvdID == "----" {
		dvdID = ""
	}

	sc.ScraperID = scraperID
	sc.SceneID = r18devSceneID(contentID, dvdID)
	sc.SiteID = sc.SceneID
	if sc.Studio == "" {
		sc.Studio = gjson.Get(body, "maker_name_ja").String()
	}

	// Display ID: the normalized dvd_id, else built from the content ID
	display := ""
	if dvdID != "" {
		display = r18devNormalizeDVDID(dvdID)
	} else {
		display = r18devDisplayID(contentID)
	}
	if display == "" {
		display = r18devStripReissue(contentID)
	}
	sc.Site = r18devDisplayPrefix(display)
	if gjson.Get(body, "maker_id").Int() == r18devMakerGlobal {
		// Global Media "3D Mature" reuses KMP's DPVR numbers; keep the raw content ID in the title
		display = r18devStripReissue(contentID)
	}

	// Title: display ID + English title; machine translated titles stay in the synopsis
	if gjson.Get(body, "title_en_is_machine_translation").String() == "false" && sc.Title != "" && sc.Title != contentID {
		sc.Title = display + " " + sc.Title
	} else {
		sc.Title = display
	}
	return sc, "", false
}

// ---- the scraper ----

// R18devCrawler is the scraper function registered for every r18dev entry in scrapers.json.
func R18devCrawler(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool, cfg config.R18devScraperConfig) error {
	defer wg.Done()
	scraperID := r18devSiteID(cfg)
	logScrapeStart(scraperID, cfg.Name)
	defer logScrapeFinished(scraperID, cfg.Name)

	client := newR18devClient()
	idx := r18devLoadIndex()
	rejected := r18devLoadRejected(scraperID)
	rejectedBefore := len(rejected)

	// What to look at: every source, or the one scene from the rescrape button
	var items []r18devListItem
	if singleSceneURL != "" {
		cid := r18devContentIDFromURL(singleSceneURL)
		if cid == "" {
			log.Errorf("[%s] no content ID in %s", scraperID, singleSceneURL)
			return nil
		}
		items = append(items, r18devListItem{ContentID: cid})
		delete(rejected, cid) // explicit request: try again
	} else {
		for _, src := range cfg.Sources {
			list, err := client.listSource(src, limitScraping)
			if err != nil {
				log.Errorf("[%s] %s %d: %v", scraperID, src.Type, src.ID, err)
			}
			log.Infof("[%s] %s %d: %d scenes listed", scraperID, src.Type, src.ID, len(list))
			items = append(items, list...)
		}
	}

	var nKnown, nForeign, nRejected, nNew, nUpdated, nSkipped int
	seen := map[string]bool{}
	todo := 0
	for _, it := range items {
		key := r18devStripReissue(it.ContentID)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true

		// Write guard 1: already in XBVR under this content ID
		updateSceneID := ""
		if rows := idx.byContentID[key]; len(rows) > 0 {
			foreign, needsUpdate := false, false
			for _, r := range rows {
				if !r18devIsOurs(r.ScraperID) {
					foreign = true
				}
				if r.NeedsUpdate {
					needsUpdate = true
				}
			}
			if foreign {
				nForeign++ // someone else's scene (old R18.com, JavDatabase, comma box, ...): hands off
				continue
			}
			if !needsUpdate && singleSceneURL == "" {
				nKnown++
				continue
			}
			updateSceneID = rows[0].SceneID
		}
		if rejected[key] {
			nRejected++
			continue
		}

		todo++
		if todo%25 == 0 {
			log.Infof("[%s] %d scenes fetched so far", scraperID, todo)
		}
		body, err := client.detail(it.ContentID)
		if err != nil {
			log.Errorf("[%s] %s: %v", scraperID, it.ContentID, err)
			continue
		}
		if body == "" && it.ContentID != key {
			body, err = client.detail(key) // reissue gone, try the original
		}
		if body == "" || err != nil {
			nSkipped++
			continue
		}

		sc, why, reject := r18devBuildScene(client, body, scraperID)
		if why != "" {
			if reject {
				rejected[key] = true
				nRejected++
			} else {
				nSkipped++
				log.Debugf("[%s] %s skipped: %s", scraperID, it.ContentID, why)
			}
			continue
		}

		if updateSceneID != "" {
			sc.SceneID, sc.SiteID = updateSceneID, updateSceneID
		} else if e := idx.bySceneID[sc.SceneID]; e != nil && !r18devIsOurs(e.ScraperID) {
			// Write guard 2: the scene ID is taken by a scene that isn't ours
			sameScene := strings.EqualFold(e.Studio, sc.Studio) || (e.ReleaseDate != "" && e.ReleaseDate == sc.Released)
			if sameScene {
				nForeign++ // same scene under another scraper (content ID changed, etc.): hands off
				continue
			}
			// Different scene with a clashing dvd_id: fall back to the unique content ID
			raw := r18devStripReissue(key)
			if e2 := idx.bySceneID[raw]; e2 != nil && !r18devIsOurs(e2.ScraperID) {
				nForeign++
				continue
			}
			log.Infof("[%s] %s: scene ID %s belongs to a different scene, using %s", scraperID, it.ContentID, sc.SceneID, raw)
			sc.SceneID, sc.SiteID = raw, raw
		}

		if e := idx.bySceneID[sc.SceneID]; e != nil {
			nUpdated++
		} else {
			nNew++
		}
		e := &r18devExisting{SceneID: sc.SceneID, ScraperID: scraperID, Studio: sc.Studio, ReleaseDate: sc.Released}
		idx.bySceneID[sc.SceneID] = e
		idx.byContentID[key] = append(idx.byContentID[key], e)
		out <- sc
	}

	if len(rejected) != rejectedBefore || singleSceneURL != "" {
		r18devSaveRejected(scraperID, rejected)
	}
	log.Infof("[%s] done: %d new, %d updated, %d already scraped, %d belong to other scrapers, %d rejected (not VR / compilation / excluded), %d skipped for now. %d requests.",
		scraperID, nNew, nUpdated, nKnown, nForeign, nRejected, nSkipped, client.requests)

	if updateSite && singleSceneURL == "" {
		updateSiteLastUpdate(scraperID)
	}
	return nil
}

func addR18devScraper(cfg config.R18devScraperConfig, custom bool) {
	if cfg.ID == "" || len(cfg.Sources) == 0 {
		log.Warnf("R18.dev scraper %q skipped: needs an id and at least one source", cfg.Name)
		return
	}
	name := cfg.Name + " (R18.dev)"
	if custom {
		name = cfg.Name + " (Custom R18.dev)"
	}
	registerScraper(r18devSiteID(cfg), name, cfg.AvatarUrl, "r18.dev",
		func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return R18devCrawler(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, limitScraping, cfg)
		})
}

func init() {
	var scrapers config.ScraperList
	scrapers.Load()
	for _, cfg := range scrapers.XbvrScrapers.R18devScrapers {
		addR18devScraper(cfg, false)
	}
	for _, cfg := range scrapers.CustomScrapers.R18devScrapers {
		addR18devScraper(cfg, true)
	}
}
