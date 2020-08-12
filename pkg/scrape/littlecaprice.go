package scrape

import (
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

	// RegEx Patterns
	sceneRegEx := regexp.MustCompile(`^.+?'(.+?)'$`)
	coverRegEx := regexp.MustCompile(`\.vid_bg {\nbackground: url\('(.+?)'`)
	durationRegEx := regexp.MustCompile(`(\d+):(\d+)`)
	descriptionRegEx := regexp.MustCompile(`(?i)^e(?:nglish)?:`)

	sceneCollector.OnHTML(`article.project`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Little Caprice Media"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - Generate randomly
		sc.SiteID = strings.Split(e.Attr("id"), "-")[1]
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`.vid_title`))

		// Cover
		cover := e.Request.Ctx.GetAny("cover").(string)
		if len(cover) == 0 {
			cover = coverRegEx.FindStringSubmatch(e.DOM.Find(`style`).Text())[1]
		}
		cover = strings.Replace(cover, "media.", "", -1)
		sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(cover))

		// Duration
		minutes := durationRegEx.FindStringSubmatch(e.ChildText(`.vid_length`))[1]
		sc.Duration,_ = strconv.Atoi(minutes)

		// Released
		dt,_ := time.Parse("January 2, 2006", e.ChildText(`.vid_date`))
		sc.Released = dt.Format("2006-01-02")

		// Synopsis
		sc.Synopsis = strings.TrimSpace(
			descriptionRegEx.ReplaceAllString( // Some scene descriptions include a redundant prefix. We remove it.
				e.ChildText(`.vid_desc`), ""))

		// Cast
		e.ForEach(`.vid_infos .vid_info_content a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Gallery
		galleryPage,_ := e.DOM.Find(`.vid_buttons a[href*="project"]`).Attr("href")
		ctx := colly.NewContext()
		ctx.Put("scene", sc)

		galleryCollector.Request("GET", galleryPage, nil, ctx, nil)

		if galleryPage == "" {
			out <- sc
		}
	})

	galleryCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)

		e.ForEach(`.et_pb_gallery_items.et_post_gallery .et_pb_gallery_item a`, func(id int, e *colly.HTMLElement) {
			image := strings.Replace(e.Attr("href"), "media.", "www.", -1)
			sc.Gallery = append(sc.Gallery, image)
		})

		out <- sc
	})

	siteCollector.OnHTML(`.ct_video`, func(e *colly.HTMLElement) {
		sceneURL := sceneRegEx.FindStringSubmatch(e.Attr(`onclick`))[1]

		// If scene exists in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			ctx := colly.NewContext()
			ctx.Put("cover", e.ChildAttr("img.ct_video_cover", "data-src"))

			//sceneCollector.Visit(sceneURL)
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
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
	registerScraper("littlecaprice", "Little Caprice Dreams", "https://littlecaprice-dreams.com/wp-content/uploads/2019/03/cropped-lcd-heart-180x180.png", LittleCaprice)
}
