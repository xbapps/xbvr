package scrape

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func RealityLoversSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, domain string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("realitylovers.com", "engine.realitylovers.com", "tsvirtuallovers.com", "engine.tsvirtuallovers.com")

	sceneCollector.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			return
		}
		json := gjson.ParseBytes(r.Body)

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "RealityLovers"
		sc.Site = siteID
		sc.HomepageURL = r.Request.Ctx.Get("sceneURL")

		// Scene ID
		sc.SiteID = json.Get("contentId").String()
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		sc.Title = json.Get("title").String()
		sc.Synopsis = json.Get("description").String()

		covers := json.Get("mainImages.0.imgSrcSet").String()
		sc.Covers = append(sc.Covers, strings.Fields(covers)[0])

		sc.Released = json.Get("releaseDate").String()

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		json.Get("starring").ForEach(func(_, star gjson.Result) bool {
			name := star.Get("name").String()
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: "https://" + domain + "/" + star.Get("uri").String()}
			return true
		})

		// Gallery
		json.Get("screenshots").ForEach(func(_, screenshot gjson.Result) bool {
			imgset := screenshot.Get("galleryImgSrcSet").String()
			images := strings.Split(imgset, ",")
			selectedImage := ""
			for _, image := range images {
				parts := strings.Fields(image)
				if selectedImage == "" {
					selectedImage = parts[0]
				}
				if parts[1] == "1920w" {
					selectedImage = parts[0]
				}
			}
			sc.Gallery = append(sc.Gallery, selectedImage)
			return true
		})

		// Tags
		json.Get("categories").ForEach(func(_, category gjson.Result) bool {
			sc.Tags = append(sc.Tags, category.Get("name").String())
			return true
		})

		sc.TrailerType = "url"
		sc.TrailerSrc = json.Get("trailerUrl").String()

		out <- sc
	})

	// Request scenes via REST API
	if singleSceneURL == "" {
		page := 0
		for {
			url := fmt.Sprintf("https://engine.%s/content/videos?max=12&page=%v&pornstar=&category=&perspective=&sort=NEWEST", domain, page)
			log.Infoln("visiting", url)
			r, err := resty.New().R().
				SetHeader("User-Agent", UserAgent).
				Get(url)

			if err != nil {
				log.Errorf("Error fetching BaberoticaVR feed: %s", err)
				logScrapeFinished(scraperID, siteID)
				return nil
			}

			scenecnt := 0
			if err == nil || r.StatusCode() == 200 {
				result := gjson.Get(r.String(), "contents")
				result.ForEach(func(key, value gjson.Result) bool {
					scenecnt++
					sceneURL := "https://" + domain + "/" + value.Get("videoUri").String()
					sceneID := value.Get("id").String()
					if !funk.ContainsString(knownScenes, sceneURL) {
						ctx := colly.NewContext()
						ctx.Put("sceneURL", sceneURL)
						sceneCollector.Request("GET", "https://engine."+domain+"/content/videoDetail?contentId="+sceneID, nil, ctx, nil)
					}
					return true
				})
			}
			if err != nil {
				log.Errorf("Error visiting %s %s", url, err)
			}
			if r.StatusCode() != 200 {
				log.Errorf("Return code visiting %s %v", url, r.StatusCode())
			}

			if scenecnt < 12 {
				break
			}
			page++
			if limitScraping {
				break
			}
			// have seen instances of status 404, so make sure we don't span will calls
			time.Sleep(time.Second)
		}
	} else {
		re := regexp.MustCompile(`.com\/vd\/(\d+)\/`)
		match := re.FindStringSubmatch(singleSceneURL)
		if len(match) >= 2 {
			ctx := colly.NewContext()
			ctx.Put("sceneURL", singleSceneURL)
			sceneCollector.Request("GET", "https://engine."+domain+"/content/videoDetail?contentId="+match[1], nil, ctx, nil)
		}

	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func RealityLovers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return RealityLoversSite(wg, updateSite, knownScenes, out, singleSceneURL, "realitylovers", "RealityLovers", "realitylovers.com", singeScrapeAdditionalInfo, limitScraping)
}

func TSVirtualLovers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return RealityLoversSite(wg, updateSite, knownScenes, out, singleSceneURL, "tsvirtuallovers", "TSVirtualLovers", "tsvirtuallovers.com", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("realitylovers", "RealityLovers", "http://static.rlcontent.com/shared/VR/common/favicons/apple-icon-180x180.png", "realitylovers.com", RealityLovers)
	registerScraper("tsvirtuallovers", "TSVirtualLovers", "http://static.rlcontent.com/shared/TS/common/favicons/apple-icon-180x180.png", "tsvirtuallovers.com", TSVirtualLovers)
}
