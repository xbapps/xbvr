package models

import (
	"sync"
)

var scrapers []Scraper

type ScraperFunc func(*sync.WaitGroup, bool, []string, chan<- ScrapedScene) error

type Scraper struct {
	ID     string
	Name   string
	Scrape ScraperFunc
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

func GetScrapers() []Scraper {
	return scrapers
}

func RegisterScraper(id string, name string, f ScraperFunc) {
	s := Scraper{}
	s.ID = id
	s.Name = name
	s.Scrape = f
	scrapers = append(scrapers, s)
}
