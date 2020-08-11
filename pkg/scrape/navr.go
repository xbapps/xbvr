package scrape

import (
	"html"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/robertkrimen/otto"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func NaughtyAmericaVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "naughtyamericavr"
	siteID := "NaughtyAmerica VR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.naughtyamerica.com")
	siteCollector := createCollector("www.naughtyamerica.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "NaughtyAmerica"
		sc.Site = siteID
		sc.Title = ""
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Date
		e.ForEach(`div.date-tags span.entry-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.date-tags div.duration`, func(id int, e *colly.HTMLElement) {
			r := strings.NewReplacer("|", "", "min", "")
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(r.Replace(e.Text)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames & Covers
		// There's a different video element for the four most recent scenes
		// New video element
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			// images5.naughtycdn.com/cms/nacmscontent/v1/scenes/2cst/nikkijaclynmarco/scene/horizontal/1252x708c.jpg
			base := strings.Split(strings.Replace(e.Attr("poster"), "//", "", -1), "/")
			baseName := base[5] + base[6]
			defaultBaseName := "nam" + base[6]

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_180x180_3dh.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName + filenames[i], defaultBaseName + filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
		})
		// Old video element
		e.ForEach(`a.play-trailer img.start-card`, func(id int, e *colly.HTMLElement) {
			// images5.naughtycdn.com/cms/nacmscontent/v1/scenes/2cst/nikkijaclynmarco/scene/horizontal/1252x708c.jpg
			base := strings.Split(strings.Replace(e.Attr("src"), "//", "", -1), "/")
			baseName := base[5] + base[6]
			defaultBaseName := "nam" + base[6]

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_180x180_3dh.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName + filenames[i], defaultBaseName + filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
		})

		// Gallery
		e.ForEach(`div.contain-scene-images.desktop-only a.thumbnail`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, strings.Replace(e.Request.AbsoluteURL(e.Attr("href")), "dynamic", "", -1))
			}
		})

		// Synopsis
		e.ForEach(`div.synopsis`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(strings.Replace(e.Text, "Synopsis", "", -1))
		})

		// Tags
		e.ForEach(`div.categories a.cat-tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast (extract from JavaScript)
		e.ForEach(`script`, func(id int, e *colly.HTMLElement) {
			if strings.Contains(e.Text, "femaleStar") {
				vm := otto.New()

				script := e.Text
				script = strings.Replace(script, "window.dataLayer", "dataLayer", -1)
				script = strings.Replace(script, "dataLayer = dataLayer || []", "dataLayer = []", -1)
				script = script + "\nout = []; dataLayer.forEach(function(v) { if (v.femaleStar) { out.push(v.femaleStar); } });"
				vm.Run(script)

				out, _ := vm.Get("out")
				outs, _ := out.ToString()

				sc.Cast = strings.Split(html.UnescapeString(outs), ",")
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul[class=pagination] li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div[class=site-list] div[class=scene-item] a[class=contain-img]`, func(e *colly.HTMLElement) {
		sceneURL := strings.Split(e.Request.AbsoluteURL(e.Attr("href")), "?")[0]

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.naughtyamerica.com/vr-porn")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("naughtyamericavr", "NaughtyAmerica VR", "https://twivatar.glitch.me/naughtyamerica", NaughtyAmericaVR)
}
