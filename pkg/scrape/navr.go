package scrape

import (
	"html"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/robertkrimen/otto"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func NaughtyAmericaVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "naughtyamericavr"
	siteID := "NaughtyAmerica VR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.naughtyamerica.com")
	siteCollector := createCollector("www.naughtyamerica.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "NaughtyAmerica"
		sc.Site = siteID
		sc.Title = ""
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://www.naughtyamerica.com/", "https://members.naughtyamerica.com/", 1)

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`div.scene-info a.site-title`)) + " - " + strings.TrimSpace(e.ChildText(`Title`))

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

		// trailer details
		sc.TrailerType = "url"

		// Filenames & Covers
		// There's a different video element for the four most recent scenes
		// New video element
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			// images5.naughtycdn.com/cms/nacmscontent/v1/scenes/2cst/nikkijaclynmarco/scene/horizontal/1252x708c.jpg
			base := strings.Split(strings.Replace(e.Attr("poster"), "//", "", -1), "/")
			if len(base) < 7 {
				return
			}
			baseName := base[5] + base[6]
			defaultBaseName := "nam" + base[6]

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4", "_8kvr265.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName+filenames[i], defaultBaseName+filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
			sc.TrailerSrc = `https://videos.naughtycdn.com/` + base[5] + `/trailers/vr/` + base[5] + base[6] + `/` + base[5] + base[6] + `teaser_vrdesktophd.mp4`
		})
		// Old video element
		e.ForEach(`a.play-trailer img.start-card.desktop-only`, func(id int, e *colly.HTMLElement) {
			// images5.naughtycdn.com/cms/nacmscontent/v1/scenes/2cst/nikkijaclynmarco/scene/horizontal/1252x708c.jpg
			srcset := e.Attr("data-srcset")
			if srcset == "" {
				srcset = e.Attr("srcset")
			}
			base := strings.Split(strings.Replace(srcset, "//", "", -1), "/")
			if len(base) < 7 {
				return
			}
			baseName := base[5] + base[6]
			defaultBaseName := "nam" + base[6]

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4", "_8kvr265.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName+filenames[i], defaultBaseName+filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
			sc.TrailerSrc = `https://videos.naughtycdn.com/` + base[5] + `/trailers/vr/` + base[5] + base[6] + `/` + base[5] + base[6] + `teaser_vrdesktophd.mp4`
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
				script = strings.ReplaceAll(script, "nanalytics.trackExperiment('scene_page_viewed', 'streaming_option_join_button');", "")
				script = strings.ReplaceAll(script, "nanalytics.trackExperiment('scene_page_viewed', 'yearly_option_first');", "")
				script = strings.Replace(script, "window.dataLayer", "dataLayer", -1)
				script = strings.Replace(script, "dataLayer = dataLayer || []", "dataLayer = []", -1)
				script = script + "\nout = []; dataLayer.forEach(function(v) { if (v.femaleStar) { out.push(v.femaleStar); } });"
				vm.Run(script)

				out, _ := vm.Get("out")
				outs, _ := out.ToString()

				sc.Cast = strings.Split(html.UnescapeString(outs), ",")
			}
		})
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a.scene-title`, func(id int, e *colly.HTMLElement) {
			for _, actor := range sc.Cast {
				if strings.EqualFold(actor, e.Text) {
					sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: strings.SplitN(e.Request.AbsoluteURL(e.Attr("href")), "?", 2)[0]}
				}
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul[class=pagination] li a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div[class=site-list] div[class=scene-item] a.contain-img`, func(e *colly.HTMLElement) {
		sceneURL := strings.Split(e.Request.AbsoluteURL(e.Attr("href")), "?")[0]

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.naughtyamerica.com/vr-porn")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("naughtyamericavr", "NaughtyAmerica VR", "https://mcdn.vrporn.com/files/20170718100937/naughtyamericavr-vr-porn-studio-vrporn.com-virtual-reality.png", "naughtyamerica.com", NaughtyAmericaVR)
}
