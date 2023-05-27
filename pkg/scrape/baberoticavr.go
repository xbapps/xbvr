package scrape

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func BaberoticaVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "baberoticavr"
	siteID := "BaberoticaVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("baberoticavr.com")
	siteCollector := createCollector("baberoticavr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Baberotica"
		sc.Site = siteID
		sc.SiteID = ""
		sc.HomepageURL = e.Request.URL.String()

		// Title
		e.ForEach(`h1[itemprop=name]`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`div[itemprop=video] meta[itemprop=thumbnailUrl]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				coverUrl := "https:" + e.Attr("content")
				sc.Covers = append(sc.Covers, coverUrl)
			}
		})

		// Gallery
		e.ForEach(`ul.caroussel li img`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, "https:"+e.Attr("src"))
		})

		// there are some weird categories like cup size or eye color, which don't make much sense without context
		ignoreTags := []string{"brown", "hazel", "blue", "black", "grey", "auburn", "categories", "green"}
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.video-info`, func(id int, e *colly.HTMLElement) {
			if id == 0 {

				// Cast
				e.ForEach(`span[itemprop=actor] span[itemprop=name]`, func(id int, e *colly.HTMLElement) {
					if strings.TrimSpace(e.Text) != "" {
						sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
					}
				})

				// Link to Cast page
				e.ForEach(`span[itemprop=actor]`, func(id int, e *colly.HTMLElement) {
					url := ""
					name := ""
					e.ForEach(`span[itemprop=name]`, func(id int, e *colly.HTMLElement) {
						name = strings.TrimSpace(e.Text)
					})
					e.ForEach(`span[itemprop=actor] a[itemprop=url]`, func(id int, e *colly.HTMLElement) {
						url = e.Attr("href")
					})
					sc.ActorDetails[name] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: url}
				})

				// Tags
				e.ForEach(`a[itemprop=genre]`, func(id int, e *colly.HTMLElement) {
					tag := strings.ToLower(strings.TrimSpace(e.Text))
					if len(tag) > 3 {
						ignore := false
						for i := range ignoreTags {
							if tag == ignoreTags[i] {
								ignore = true
								break
							}
						}
						if !ignore {
							sc.Tags = append(sc.Tags, tag)
						}
					}
				})

			}
		})

		// Synposis
		e.ForEach(`div.video-description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Release date
		e.ForEach(`meta[itemprop=datePublished]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				tmpDate, err := time.Parse(time.RFC3339, e.Attr("content"))
				if err == nil {
					sc.Released = tmpDate.Format("2006-01-02")
					sc.SiteID = tmpDate.Format("2006-01-02")
				}
			}
		})

		// Release date / Duration
		e.ForEach(`meta[itemprop=duration]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				r := regexp.MustCompile("T(\\d+)M")
				match := r.FindStringSubmatch(e.Attr("content"))
				if match != nil {
					tmpDuration, err := strconv.Atoi(match[1])
					if err == nil {
						sc.Duration = tmpDuration
					}
				}
			}
		})

		// Filenames (only a guess for now)
		suffixes := []string{"5k", "4k", "gear", "hd", "oculus"}
		e.ForEach(`div.video-downloads a`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				parts := strings.Split(e.Attr("href"), "/")
				basename := parts[len(parts)-1]
				parts = strings.Split(basename, "_trailer_")
				base := parts[0]
				for _, suffix := range suffixes {
					sc.Filenames = append(sc.Filenames, base+"_"+suffix+"_180x180_3dh.mp4")
				}
			}
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "video source", ContentPath: "src", QualityPath: "data-res", ContentBaseUrl: "https:"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		if sc.SiteID != "" {
			sc.SceneID = fmt.Sprintf("baberoticavr-%v", sc.SiteID)

			// save only if we got a SceneID
			out <- sc
		}
	})

	siteCollector.OnHTML(`div.pagination a.page-numbers`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.container-wide li div.rel a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://baberoticavr.com/solo-vr-porn/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("baberoticavr", "BaberoticaVR", "https://baberoticavr.com/wp-content/themes/baberoticavr/images/fav/android-chrome-192x192.png", BaberoticaVR)
}
