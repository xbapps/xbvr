package scrape

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/bregydoc/gtranslate"
	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"golang.org/x/text/language"
)

func CariVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "caribbeancomvr"
	siteID := "CaribbeanCom VR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("en.caribbeancom.com", "www.caribbeancom.com")
	siteCollector := createCollector("en.caribbeancom.com", "www.caribbeancom.com")
	sceneCollectorJap := cloneCollector(sceneCollector)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Caribbeancom"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from JavaScript
		e.ForEach(`script`, func(id int, e *colly.HTMLElement) {
			if !strings.Contains(e.Text, "movie_seq") {
				return
			}
			jsonData := e.Text[strings.Index(e.Text, "{") : len(e.Text)-3]
			movSeq := gjson.Get(jsonData, "movie_seq").String()
			if movSeq == "" {
				return
			}
			sc.SiteID = movSeq
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`h1[itemprop=name]`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(strings.Replace(e.Text, "[VR] ", "", 1))
		})

		// Cover
		coverURL := strings.Replace(strings.Replace(sc.HomepageURL, "eng/", "", 1), "index.html", "images/poster_en.jpg", 1)
		if len(coverURL) > 0 {
			sc.Covers = append(sc.Covers, coverURL)
		}

		// Filename  011421-001-carib-2160p.mp4
		sc.Filenames = append(sc.Filenames, strings.Split(coverURL, "/")[4]+"-carib-2160p.mp4")

		// Gallery
		e.ForEach(`div.movie-gallery a.fancy-gallery`, func(id int, e *colly.HTMLElement) {
			if strings.Compare(e.Attr(`data-is_sample`), "0") == 0 {
				return
			}
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// setup  trailers
		sc.TrailerType = "scrape_json"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "script", ExtractRegex: `Movie = (.+?})`, ContentPath: "sample_flash_url"}
		tmp, _ := json.Marshal(params)
		sc.TrailerSrc = string(tmp)

		// Cast & Tags
		e.ForEach(`div.movie-info a.spec__tag`, func(id int, e *colly.HTMLElement) {
			if strings.Compare(e.Attr(`itemprop`), "actor") == 0 {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			} else {
				if (strings.Compare(e.Attr(`itemprop`), "genre") == 0) || (strings.Compare(e.Attr(`itemprop`), "url") == 0) {
					sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
				}
			}
		})

		// Release Date
		e.ForEach(`div.movie-info span`, func(id int, e *colly.HTMLElement) {
			if e.Attr(`itemprop`) == "uploadDate" {
				sc.Released = strings.TrimSpace(strings.Replace(e.Text, "/", "-", -1))
			}
			// Duration
			if e.Attr(`itemprop`) == "duration" {
				tmpDuration := strings.Split(strings.Trim(e.Attr(`content`), "TS"), "M")[0]
				sc.Duration, _ = strconv.Atoi(strings.Split(tmpDuration, "H")[1])
			}
		})

		sceneURLJap := strings.Replace(strings.Replace(sc.HomepageURL, "eng/", "", 1), "en.", "www.", 1)
		ctx := colly.NewContext()
		ctx.Put("scene", sc)

		sceneCollectorJap.Request("GET", sceneURLJap, nil, ctx, nil)
	})

	// Synopsis - Pull from Japanese site & translate
	sceneCollectorJap.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)
		e.ForEach(`p[itemprop=description]`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis, _ = gtranslate.Translate(strings.TrimSpace(e.Text), language.Japanese, language.English)
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.media-thum a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		// If scene exists in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`.pagination-large .pagination__item[rel="next"]`, func(e *colly.HTMLElement) {
		if !limitScraping {
			// replace "all" with "vr" to allow for correct page navigation
			pageURL := strings.Replace(e.Request.AbsoluteURL(e.Attr("href")), "all", "vr", 1)
			siteCollector.Visit(pageURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://en.caribbeancom.com/eng/listpages/vr1.htm")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("caribbeancomvr", "CaribbeanCom VR", "https://mcdn.vrporn.com/files/20191217194900/baimudan-vr-porn-studio-logo-vrporn.com-virtual-reality-porn.jpg", "en.caribbeancom.com", CariVR)
}
