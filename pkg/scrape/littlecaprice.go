package scrape

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func LittleCaprice(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "littlecaprice"
	siteID := "Little Caprice Dreams"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.littlecaprice-dreams.com")
	siteCollector := createCollector("www.littlecaprice-dreams.com")
	galleryCollector := cloneCollector(sceneCollector)

	sceneCollector.OnHTML(`.entry-content`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Little Caprice Media"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - Generate randomly
		sc.SiteID = strconv.Itoa(rand.Int())
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`.vid_title`))

		// Cover
		styleRe := regexp.MustCompile(`\.vid_bg {\nbackground: url\('(.+?)'`)
		image := styleRe.FindStringSubmatch(e.DOM.Find(`style`).Text())[1]
		sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(image))

		// Duration
		re := regexp.MustCompile(`(\d+):(\d+)`)
		minutes := re.FindStringSubmatch(e.ChildText(`.vid_length`))[1]
		sc.Duration,_ = strconv.Atoi(minutes)

		// Released
		dt,_ := time.Parse("January 2, 2006", e.ChildText(`.vid_date`))
		sc.Released = dt.Format("2006-01-02")

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`vid_desc`))

		// Cast
		e.ForEach(`.vid_infos .vid_info_content a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Gallery
		galleryPage,_ := e.DOM.Find(`.vid_buttons a[href*="project"]`).Attr("href")
		ctx := colly.NewContext()
		ctx.Put("scene", sc)

		galleryCollector.Request("GET", galleryPage, nil, ctx, nil)

		out <- sc
	})

	galleryCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)

		e.ForEach(`.et_pb_gallery_items.et_post_gallery .et_pb_gallery_item a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("href"))
		})

		out <- sc
	})

	siteCollector.OnHTML(`.ct_video`, func(e *colly.HTMLElement) {
		re := regexp.MustCompile(`^.+?'(.+?)'$`)
		sceneURL := re.FindStringSubmatch(e.Attr(`onclick`))[1]

		// If scene exists in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.littlecaprice-dreams.com/virtual-reality-little-caprice")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("littlecaprice", "Little Caprice Dreams", "https://media.littlecaprice-dreams.com/wp-content/uploads/2019/03/cropped-lcd-heart-32x32.png", LittleCaprice)
}
