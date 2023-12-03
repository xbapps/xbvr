package scrape

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

const (
	scraperID = "vrspy"
	siteID    = "VRSpy"
	domain    = "vrspy.com"
	baseURL   = "https://" + domain
)

func VRSpy(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singleScrapeAdditionalInfo string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector(domain)
	siteCollector := createCollector(domain)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = siteID
		sc.Site = siteID
		sc.HomepageURL = e.Request.URL.String()

		ogimage := e.ChildAttr(`meta[property="og:image"]`, "content")
		if ogimage != "" {
			ogimageURL, err := url.Parse(ogimage)
			if err == nil {
				parts := strings.Split(ogimageURL.Path, "/")
				if len(parts) > 2 {
					sc.SiteID = parts[2]
				}
			}
		}

		if sc.SiteID == "" {
			return
		}

		sc.SceneID = scraperID + "-" + sc.SiteID

		sc.Title = e.ChildText(`.video-content .header-container .section-header-container`)
		sc.Synopsis = e.ChildText(`.video-description`)
		sc.Tags = e.ChildTexts(`.video-categories .v-chip__content`)

		e.ForEach(`.video-details-row`, func(id int, e *colly.HTMLElement) {
			parts := strings.SplitN(e.Text, ":", 2)
			key, value := parts[0], parts[1]
			switch strings.TrimSpace(key) {
			case "Stars":
				sc.ActorDetails = make(map[string]models.ActorDetails)
				e.ForEach(`.stars-list a`, func(id int, e *colly.HTMLElement) {
					sc.Cast = append(sc.Cast, e.Text)
					sc.ActorDetails[e.Text] = models.ActorDetails{
						Source:     scraperID + " scrape",
						ProfileUrl: e.Request.AbsoluteURL(e.Attr(`href`)),
					}
				})
			case "Duration":
				durationParts := strings.Split(strings.SplitN(strings.TrimSpace(value), " ", 2)[0], ":")
				if len(durationParts) == 3 {
					hours, _ := strconv.Atoi(durationParts[0])
					minutes, _ := strconv.Atoi(durationParts[1])
					sc.Duration = hours*60 + minutes
				}
			case "Release date":
				tmpDate, _ := goment.New(strings.TrimSpace(value), "DD MMM YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
		})

		var durationParts []string
		// Date & Duration
		e.ForEach(`div.single-video-info__list-item`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, ":")
			if len(parts) > 1 {
				switch strings.TrimSpace(parts[0]) {
				case "Release date":
					tmpDate, _ := goment.New(strings.TrimSpace(parts[1]), "MMM D, YYYY")
					sc.Released = tmpDate.Format("YYYY-MM-DD")
				case "Duration":
					durationParts = strings.Split(strings.TrimSpace(parts[1]), " ")
					tmpDuration, err := strconv.Atoi(durationParts[0])
					if err == nil {
						sc.Duration = tmpDuration
					}
				}
			}
		})

		cdnSceneURL := e.Request.URL
		cdnSceneURL.Host = "cdn." + domain
		cdnSceneURL.Path = "/videos/" + sc.SiteID

		sc.Covers = []string{
			cdnSceneURL.JoinPath("images", "cover.jpg").String(),
			cdnSceneURL.JoinPath("images", "poster.jpg").String(),
		}

		nuxtData := e.ChildText(`#__NUXT_DATA__`)
		imageRegex := regexp.MustCompile(regexp.QuoteMeta(cdnSceneURL.String()) + `/photos/[^?"]*\.jpg`)
		sc.Gallery = imageRegex.FindAllString(nuxtData, -1)

		// trailer details
		sc.TrailerType = "scrape_html"
		paramsdata := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "script[id=\"__NUXT_DATA__\"]", ExtractRegex: `(https:\/\/cdn.vrspy.com\/videos\/\d*\/trailers\/\dk\.mp4\?token.*?)"`}
		jsonStr, _ := json.Marshal(paramsdata)
		sc.TrailerSrc = string(jsonStr)

		out <- sc
	})

	siteCollector.OnHTML(`body`, func(e *colly.HTMLElement) {
		e.ForEachWithBreak(`.video-section a.photo-preview`, func(id int, e *colly.HTMLElement) bool {
			currentPage, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
			siteCollector.Visit(fmt.Sprintf("%s/videos?sort=new&page=%d", baseURL, currentPage+1))
			return false
		})
	})

	siteCollector.OnHTML(`.video-section a.photo-preview`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(baseURL + "/videos?sort=new&page=1")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper(scraperID, siteID, baseURL+"/favicon.ico", domain, VRSpy)
}
