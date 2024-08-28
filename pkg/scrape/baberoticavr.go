package scrape

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/gosimple/slug"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func BaberoticaVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "baberoticavr"
	siteID := "BaberoticaVR"
	logScrapeStart(scraperID, siteID)
	additionalDetailCollector := createCollector("baberoticavr.com")

	additionalDetailCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)
		e.ForEach(`div.videoinfo>p`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})
		out <- sc
	})

	resp, err := resty.New().R().
		SetHeader("User-Agent", UserAgent).
		SetDoNotParseResponse(true).
		Get("https://baberoticavr.com/feed/csv/")

	if err != nil {
		log.Errorf("Error fetching BaberoticaVR feed: %s", err)
		logScrapeFinished(scraperID, siteID)
		return nil
	}

	csvreader := csv.NewReader(resp.RawBody())
	data, err := csvreader.ReadAll()
	if err != nil {
		log.Errorf("Error reading BaberoticaVR feed: %s", err)
		logScrapeFinished(scraperID, siteID)
		return nil
	}

	// Fields:
	// 0 Unique Item ID
	// 1 Username
	// 2 Publishing date
	// 3 Video URL
	// 4 Duration
	// 5 Title
	// 6 Categories
	// 7 Default thumbnail URL
	// 8 Preview image url
	// 9 Preview thumbnail URLs
	// 10 Tags/Channels
	// 11 Models
	// 12 Description
	// 13 HD
	// 14 4K
	// 15 VR
	// 16 UGC
	// 17 Premium
	// 18 PayPerClip
	// 19 PayPerView
	// 20 Fan subscription
	// 21 Studio Name
	// 22 Pay Site Name
	// 23 Preview available
	// 24 Preview duration
	// 25 Banned countries
	// 26 Under review
	// 27 Price

	// there are some weird categories like cup size or eye color, which don't make much sense without context
	ignoreTags := []string{"brown", "hazel", "blue", "black", "grey", "auburn", "categories", "green"}
	siteIdRegex := regexp.MustCompile(`baberoticavr-(\d+)`)
	for _, row := range data {
		sc := models.ScrapedScene{}
		sceneURL := row[3]
		if funk.ContainsString(knownScenes, sceneURL) && sceneURL != singleSceneURL {
			continue
		}

		match := siteIdRegex.FindStringSubmatch(row[0])
		if match != nil {
			sc.SiteID = match[1]
		} else {
			continue
		}

		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Baberotica"
		sc.Site = siteID
		sc.HomepageURL = sceneURL
		sc.Title = row[5]
		sc.Synopsis = row[12]
		sc.Covers = append(sc.Covers, row[7])
		sc.Gallery = strings.Split(row[9], ",")
		sc.Released = row[2]
		duration, err := strconv.Atoi(row[4])
		if err == nil {
			sc.Duration = duration / 60
		}

		tags := strings.Split(row[6], ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
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
		}

		sc.ActorDetails = make(map[string]models.ActorDetails)
		actors := strings.Split(row[11], ",")
		for _, actor := range actors {
			actor := strings.TrimSpace(actor)

			if actor != "" {
				sc.Cast = append(sc.Cast, actor)
				url := "https://baberoticavr.com/model/" + slug.Make(actor) + "/"
				sc.ActorDetails[actor] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: url}
			}
		}

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "video source", ContentPath: "src", QualityPath: "data-res", ContentBaseUrl: "https:"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		if sc.SiteID != "" {
			sc.SceneID = fmt.Sprintf("baberoticavr-%v", sc.SiteID)
		}

		ctx := colly.NewContext()
		ctx.Put("scene", sc)
		additionalDetailCollector.Request("GET", sc.HomepageURL, nil, ctx, nil)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}
func init() {
	registerScraper("baberoticavr", "BaberoticaVR", "https://baberoticavr.com/wp-content/themes/baberoticavr/images/fav/android-chrome-192x192.png", "baberoticavr.com", BaberoticaVR)
}
