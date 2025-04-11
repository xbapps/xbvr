package scrape

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func FuckPassVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "fuckpassvr-native"
	siteID := "FuckPassVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.fuckpassvr.com")
	siteCollector := createCollector("www.fuckpassvr.com")

	client := resty.New()
	client.SetHeader("User-Agent", UserAgent)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "FuckPassVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				url := strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0]
				re := regexp.MustCompile(`FPVR(\d+)`)
				matches := re.FindStringSubmatch(url)
				if len(matches) > 1 {
					sc.SiteID = matches[1]
					sc.SceneID = "fpvr-" + matches[1]
				}
			}
		})

		e.ForEach(`h2.video__title`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = strings.TrimSpace(e.Text)
			}
		})

		e.ForEach(`pornhall-player`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, strings.Trim(e.Attr("poster"), " '"))
		})

		e.ForEach(`div.profile__gallery a.profile__galleryElement`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(e.Attr("href")))
		})

		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.models a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Attr("title")))
			sc.ActorDetails[strings.TrimSpace(e.Attr("title"))] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		e.ForEach(`a.tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Attr("title")))
		})

		e.ForEach(`div.readMoreWrapper2`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		e.ForEach(`div.video__addons p.wrapper__text`, func(id int, e *colly.HTMLElement) {
			s := strings.TrimSpace(e.Text)
			if strings.HasPrefix(s, "Released:") {
				tmpDate, _ := goment.New(strings.TrimSpace(strings.TrimPrefix(s, "Released:")), "MMM DD YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
		})

		e.ForEach(`div.wrapper__download a.wrapper__downloadLink`, func(id int, e *colly.HTMLElement) {
			url, err := url.Parse(e.Attr("href"))
			if err == nil {
				parts := strings.Split(url.Path, "/")
				if len(parts) > 0 {
					fn := parts[len(parts)-1]
					fn = strings.Replace(fn, "2min", "FULL", -1)
					sc.Filenames = append(sc.Filenames, fn)
				}
			}
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "pornhall-player source", ContentPath: "src", QualityPath: "data-quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		out <- sc
	})

	siteCollector.OnHTML(`section.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			siteCollector.Visit(e.Attr("href"))
		}
	})

	siteCollector.OnHTML(`div.videos__element a.videos__videoTitle`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.fuckpassvr.com/destination")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("fuckpassvr-native", "FuckPassVR", "https://www.fuckpassvr.com/favicon.png", "fuckpassvr.com", FuckPassVR)
}
