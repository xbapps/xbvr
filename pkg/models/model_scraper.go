package models

import (
	"encoding/json"
	"sync"
)

var scrapers []Scraper

type ScraperFunc func(*sync.WaitGroup, bool, []string, chan<- ScrapedScene) error

type Scraper struct {
	ID        string
	Name      string
	AvatarURL string
	Scrape    ScraperFunc
}

type ScrapedActor struct {
	ActorID      string `json:"_id"`
	Aliases      string `json:"aliases"`
	Bio          string `json:"bio"`
	Birthday     string `json:"birthdate"`
	Ethnicity    string `json:"ethnicity"`
	EyeColor     string `json:"eye_color"`
	Facebook     string `json:"facebook"`
	HairColor    string `json:"hair_color"`
	Height       string `json:"height"`
	HomepageURL  string `json:"homepage_url"`
	ImageURL     string `json:"image_url"`
	Instagram    string `json:"instagram"`
	Measurements string `json:"measurements"`
	Name         string `json:"name"`
	Nationality  string `json:"nationality"`
	Reddit       string `json:"reddit"`
	Twitter      string `json:"twitter"`
	Weight       string `json:"weight"`
}

type ScrapedScene struct {
	SceneID     string   `json:"_id"`
	SiteID      string   `json:"scene_id"`
	SceneType   string   `json:"scene_type"`
	Title       string   `json:"title"`
	Studio      string   `json:"studio"`
	Site        string   `json:"site"`
	Covers      []string `json:"covers"`
	Gallery     []string `json:"gallery"`
	Tags        []string `json:"tags"`
	Cast        []string `json:"cast"`
	Filenames   []string `json:"filename"`
	Duration    int      `json:"duration"`
	Synopsis    string   `json:"synopsis"`
	Released    string   `json:"released"`
	HomepageURL string   `json:"homepage_url"`
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

func RegisterScraper(id string, name string, avatarURL string, f ScraperFunc) {
	s := Scraper{}
	s.ID = id
	s.Name = name
	s.AvatarURL = avatarURL
	s.Scrape = f
	scrapers = append(scrapers, s)
}
