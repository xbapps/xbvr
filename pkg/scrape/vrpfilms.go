package scrape

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRPFilms(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vrpfilms"
	siteID := "VRP Films"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrpfilms.com", "www.vrpfilms.com")
	siteCollector := createCollector("vrpfilms.com", "www.vrpfilms.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRP Films"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from download link. It's the closest thing they have to a scene id
		sc.SiteID = e.ChildAttr(`a.member-download`, "data-main-product-id")
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		sc.Title = strings.TrimSpace(e.ChildText(`span.breadcrumb_last`))
		coverURL := e.ChildAttr(`meta[property="og:image"]`, "content")
		sc.Covers = append(sc.Covers, coverURL)

		// No release date anywhere, but we can approximate based on the wordpress date of the
		// cover image. It's at least better than nothing.
		//
		// https://vrpfilms.com/wp-content/uploads/2019/10/No-Boys-Just-Toys-Banner-1600x800.jpg
		t := strings.Split(coverURL, "/")
		tmpDate := fmt.Sprintf("%s-%s-01", t[5], t[6])
		date, _ := goment.New(tmpDate, "YYYY-MM-DD")
		sc.Released = date.Format("YYYY-MM-DD")

		sc.Gallery = e.ChildAttrs(`.movies-gallery a`, "href")

		unfilteredTags := []string{}
		e.ForEach(`.detail p`, func(id int, e *colly.HTMLElement) {
			if strings.Contains(e.Text, "Featuring:") {
				// Featuring: Amber Jayne, Selvaggia
				tmpCast := strings.Split(e.Text, ":")[1]
				cast := strings.Split(strings.TrimSpace(tmpCast), ",")
				funk.ForEach(cast, func(c string) {
					sc.Cast = append(sc.Cast, strings.TrimSpace(c))
				})

			}

			if strings.Contains(e.Text, "Length:") {
				// Length: 35 Minutes
				tmpDuration := strings.TrimSpace(strings.Split(e.Text, ":")[1])
				duration, err := strconv.Atoi(strings.Split(tmpDuration, " ")[0])
				if err == nil {
					sc.Duration = duration
				}
			}

			if strings.Contains(e.Text, "Tags:") {
				tmpTags := strings.Split(e.Text, ":")[1]
				tags := strings.Split(strings.TrimSpace(tmpTags), ",")
				funk.ForEach(tags, func(t string) {
					unfilteredTags = append(unfilteredTags, strings.TrimSpace(t))
				})
			}
		})

		// It pains me to have to do this
		garbageTags := []string{"pussy", "polly pons", "little cindy",
			"bass ass handy women", "hot",
			"estate agent sex pov", "real estate sex vr",
			"sandy's superstar escorts", "wet and wild",
		}
		sc.Tags = funk.FilterString(unfilteredTags, func(t string) bool {
			lt := strings.ToLower(t)
			if funk.ContainsString(garbageTags, lt) {
				return false
			}

			var badTag bool
			funk.ForEach(sc.Cast, func(c string) {
				if strings.ToLower(c) == lt {
					badTag = true
				}
			})

			if badTag {
				return false
			}

			if strings.ToLower(sc.Title) == lt {
				return false
			}
			return true
		})

		out <- sc
	})

	siteCollector.OnHTML(`article a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`a.page-numbers`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !strings.Contains(pageURL, "/join") {
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.Visit("https://vrpfilms.com/vrp-movies")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrpfilms", "VRP Films", "https://vrpfilms.com/storage/settings/March2021/Z0krYIQBMwSJ4R1eCnv1.png", VRPFilms)
}
