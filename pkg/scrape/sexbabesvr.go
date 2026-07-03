package scrape

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

func SexBabesVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "sexbabesvr"
	siteID := "SexBabesVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("sexbabesvr.com")
	siteCollector := createCollector("sexbabesvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "SexBabesVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			posterURL := e.Request.AbsoluteURL(e.Attr("poster"))
			tmp := strings.Split(posterURL, "/")
			sc.SiteID = tmp[len(tmp)-2]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
			// Cover Url
			sc.Covers = append(sc.Covers, strings.Replace(e.Attr("poster"), "/videoDetail2x", "", -1))
		})

		// Title
		e.ForEach(`div.video-detail__description--container h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Gallery
		e.ForEach(`.gallery-slider a[data-fancybox=gallery]`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis — older scenes put the description as direct text in
		// .sbvr-scene-about__prose; newer scenes wrap each paragraph in a
		// child <div>. Take the whole subtree text and collapse whitespace.
		e.ForEach(`.sbvr-scene-about__prose`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.Join(strings.Fields(e.Text), " ")
		})

		// Tags
		e.ForEach(`.sbvr-scene-about__chips a.sbvr-scene-about__chip`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.video-detail__description--author a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Title, date, duration from JSON-LD
		durationRegex := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
		e.ForEach(`script[type="application/ld+json"]`, func(id int, e *colly.HTMLElement) {
			jsonText := strings.TrimSpace(e.Text)
			if jsonText == "" {
				return
			}
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
				return
			}
			if title, ok := jsonData["name"].(string); ok && strings.TrimSpace(title) != "" {
				sc.Title = strings.TrimSpace(title)
			}
			if duration, ok := jsonData["duration"].(string); ok {
				if m := durationRegex.FindStringSubmatch(duration); len(m) == 4 {
					hours, _ := strconv.Atoi(m[1])
					minutes, _ := strconv.Atoi(m[2])
					seconds, _ := strconv.Atoi(m[3])
					sc.Duration = (hours*3600 + minutes*60 + seconds) / 60
				}
			}
			if uploadDate, ok := jsonData["uploadDate"].(string); ok {
				if tmpDate, err := time.Parse(time.RFC3339, uploadDate); err == nil {
					sc.Released = tmpDate.Format("2006-01-02")
				}
			}
		})

		// Original Filenames — sceneSlug + suffix from SBVR-DEBUG
		sceneSlug := ""
		if u, err := url.Parse(sc.HomepageURL); err == nil {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) > 0 {
				sceneSlug = parts[len(parts)-1]
			}
		}
		// Postfixes used by the inline player are streaming sources, never real downloads.
		streamPostfixes := map[string]bool{}
		e.ForEach(`dl8-video source`, func(_ int, src *colly.HTMLElement) {
			u, err := url.Parse(src.Attr("src"))
			if err != nil {
				return
			}
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) == 0 {
				return
			}
			base := parts[len(parts)-1]
			if i := strings.Index(base, "_"); i > 0 {
				streamPostfixes["_"+base[i+1:]] = true
			}
		})
		if sceneSlug != "" {
			if m := regexp.MustCompile(`postfixes=\[([^\]]*)\]`).FindSubmatch(e.Response.Body); len(m) == 2 {
				for _, postfix := range strings.Split(string(m[1]), ";") {
					postfix = strings.TrimSpace(postfix)
					if postfix == "" || strings.Contains(postfix, "_trailer") || streamPostfixes[postfix] {
						continue
					}
					// Ignore sub-1080 res
					firstTok := strings.SplitN(strings.TrimPrefix(postfix, "_"), "_", 2)[0]
					res := 0
					if num, ok := strings.CutSuffix(firstTok, "k"); ok {
						if n, err := strconv.Atoi(num); err == nil {
							res = n * 1000
						}
					} else {
						res, _ = strconv.Atoi(firstTok)
					}
					if res > 0 && res < 1080 {
						continue
					}
					sc.Filenames = append(sc.Filenames, sceneSlug+postfix)
				}
			}
		}

		out <- sc
	})

	siteCollector.OnHTML(`a.pagination__button`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.videos__content`, func(e *colly.HTMLElement) {
		e.ForEach(`a.video-container__image`, func(cnt int, e *colly.HTMLElement) {
			sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
			if !funk.ContainsString(knownScenes, sceneURL) {
				sceneCollector.Visit(sceneURL)
			}
		})
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://sexbabesvr.com/vr-porn-videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("sexbabesvr", "SexBabesVR", "https://sexbabesvr.com/static/images/favicons/favicon-32x32.png", "sexbabesvr.com", SexBabesVR)
}
