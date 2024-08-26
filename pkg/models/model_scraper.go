package models

import (
	"encoding/json"
	"sync/atomic"
)

var scrapers []Scraper

type ScraperFunc func(*ScrapeWG, bool, []string, chan<- ScrapedScene, string, string, bool) error

type Scraper struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	AvatarURL    string      `json:"avaatarurl"`
	Domain       string      `json:"domain"`
	Scrape       ScraperFunc `json:"-"`
	MasterSiteId string      `json:"master_site_id"`
}

type ScrapedScene struct {
	SceneID           string   `json:"_id"`
	ScraperID         string   `json:"xbvr_site"`
	SiteID            string   `json:"scene_id"`
	SceneType         string   `json:"scene_type"`
	Title             string   `json:"title"`
	Studio            string   `json:"studio"`
	Site              string   `json:"site"`
	Covers            []string `json:"covers"`
	Gallery           []string `json:"gallery"`
	Tags              []string `json:"tags"`
	Cast              []string `json:"cast"`
	Filenames         []string `json:"filename"`
	Duration          int      `json:"duration"`
	Synopsis          string   `json:"synopsis"`
	Released          string   `json:"released"`
	HomepageURL       string   `json:"homepage_url"`
	MembersUrl        string   `json:"members_url"`
	TrailerType       string   `json:"trailer_type"`
	TrailerSrc        string   `json:"trailer_source"`
	ChromaKey         string   `json:"chromakey"`
	HasScriptDownload bool     `json:"has_script_Download"`
	AiScript          bool     `json:"ai_script"`
	HumanScript       bool     `json:"human_script"`

	OnlyUpdateScriptData bool `default:"false" json:"only_update_script_data"`
	InternalSceneId      uint `json:"internal_id"`

	ActorDetails map[string]ActorDetails `json:"actor_details"`
	MasterSiteId string                  `json:"master_site_id"`
}

type ActorDetails struct {
	ImageUrl   string
	ProfileUrl string
	Source     string
}
type TrailerScrape struct {
	SceneUrl       string `json:"scene_url"`        // url of the page to be scrapped
	HtmlElement    string `json:"html_element"`     // path to section of html (using colly)
	ExtractRegex   string `json:"extract_regex"`    // regex expression to extract the json, eg from a json variable assignment in javascript
	ContentBaseUrl string `json:"content_base_url"` // prefix for the url if the scrapped content urls are not abosolute
	RecordPath     string `json:"record_path"`      // points to a json array of video source (optional, there maybe a single video), uses jsonpath syntax
	ContentPath    string `json:"content_path"`     // points to the content url uses jsonpath syntax
	EncodingPath   string `json:"encoding_path"`    // optional, points to the encoding for the source using jsonpath syntax, eg h264, h265
	QualityPath    string `json:"quality_path"`     // points to the quality using jsonpath syntax, eg 1440p, 5k
}

func (s *ScrapedScene) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *ScrapedScene) Log() error {
	j, err := json.MarshalIndent(s, "", "  ")
	log.Debugf("\n%v", string(j))
	return err
}

func GetScrapers() []Scraper {
	return scrapers
}

func RegisterScraper(id string, name string, avatarURL string, domain string, f ScraperFunc, masterSiteId string) {
	s := Scraper{}
	s.ID = id
	s.Name = name
	s.AvatarURL = avatarURL
	s.Domain = domain
	s.Scrape = f
	s.MasterSiteId = masterSiteId
	scrapers = append(scrapers, s)
}

// Custom wg functions, to allow access to the current count of waitgroups. This allows running scrapers at max count always
type ScrapeWG struct {
	count int64
}

func (wg *ScrapeWG) Add(n int64) {
	atomic.AddInt64(&wg.count, n)
}

func (wg *ScrapeWG) Done() {
	wg.Add(-1)
	if atomic.LoadInt64(&wg.count) < 0 {
		panic("negative wait group counter")
	}
}

func (wg *ScrapeWG) Wait(n int64) {
	for atomic.LoadInt64(&wg.count) >= n && atomic.LoadInt64(&wg.count) != 0 {
		continue
	}
}

func (wg *ScrapeWG) Count() int64 {
	return atomic.LoadInt64(&wg.count)
}
