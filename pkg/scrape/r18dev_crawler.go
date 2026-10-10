package scrape

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// R18.dev crawler: one XBVR scraper per entry in the "r18dev" section of scrapers.json.
// Each entry is a list of R18.dev sources (labels and/or makers), crawled through the same
// list2 JSON endpoint the R18.dev site uses.
//
// STAGE 2: dry run only. Lists every source, works out which content IDs XBVR already has,
// logs what it would scrape. Nothing is written to the database.

const (
	r18devBaseURL  = "https://r18.dev/videos/vod/movies"
	r18devPageSize = 100
	r18devDelay    = 1 * time.Second
)

type r18devListItem struct {
	ContentID   string
	DvdID       string
	ReleaseDate string
}

// Content ID from the URLs XBVR scenes carry: R18.dev (id=, combined=) and FANZA / DMM (cid=)
var r18devURLContentIDRe = regexp.MustCompile(`(?i)(?:[/?&]cid=|/id=|/combined=)([a-z0-9_]+)`)

// r18devContentIDFromURL returns the content ID in an R18.dev or DMM scene URL, lowercased and
// without a reissue suffix, or "" if there isn't one.
func r18devContentIDFromURL(u string) string {
	m := r18devURLContentIDRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return r18devStripReissue(strings.ToLower(m[1]))
}

type r18devClient struct {
	http     *resty.Client
	requests int
	last     time.Time
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

// get fetches a URL, keeping at least r18devDelay between requests.
func (c *r18devClient) get(url string) (*resty.Response, error) {
	if wait := r18devDelay - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
	c.requests++
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

func r18devSiteID(cfg config.R18devScraperConfig) string {
	return "r18dev-" + strings.ToLower(cfg.ID)
}

// R18devCrawler is the scraper function registered for every r18dev entry in scrapers.json.
func R18devCrawler(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool, cfg config.R18devScraperConfig) error {
	defer wg.Done()
	scraperID := r18devSiteID(cfg)
	logScrapeStart(scraperID, cfg.Name)
	defer logScrapeFinished(scraperID, cfg.Name)

	if singleSceneURL != "" {
		log.Infof("[%s] single scene scraping is not implemented yet: %s", scraperID, singleSceneURL)
		return nil
	}

	// Content IDs XBVR already knows about, from scene URLs
	known := map[string]bool{}
	for _, u := range knownScenes {
		if cid := r18devContentIDFromURL(u); cid != "" {
			known[cid] = true
		}
	}

	client := newR18devClient()
	seen := map[string]bool{}
	var listed, alreadyKnown int
	var newItems []r18devListItem

	for _, src := range cfg.Sources {
		items, err := client.listSource(src, limitScraping)
		if err != nil {
			log.Errorf("[%s] %s %d: %v", scraperID, src.Type, src.ID, err)
		}
		log.Infof("[%s] %s %d: %d scenes listed", scraperID, src.Type, src.ID, len(items))
		for _, it := range items {
			key := r18devStripReissue(it.ContentID)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			listed++
			if known[key] {
				alreadyKnown++
				continue
			}
			newItems = append(newItems, it)
		}
	}

	log.Infof("[%s] DRY RUN: %d unique scenes listed, %d already in XBVR (matched by content ID in the scene URL), %d new. %d requests. Nothing written.",
		scraperID, listed, alreadyKnown, len(newItems), client.requests)
	for i, it := range newItems {
		if i == 25 {
			log.Infof("[%s] DRY RUN: ... and %d more", scraperID, len(newItems)-25)
			break
		}
		log.Infof("[%s] DRY RUN: would scrape %s  scene ID %s  display %s  released %s",
			scraperID, it.ContentID, r18devSceneID(it.ContentID, it.DvdID), r18devDisplayID(it.ContentID), it.ReleaseDate)
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
