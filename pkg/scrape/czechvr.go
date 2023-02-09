package scrape

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func CzechVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, nwID string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.czechvrnetwork.com")
	siteCollector := createCollector("www.czechvrnetwork.com")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "CzechVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://www.czechvrnetwork.com/", "https://www.czechvrnetwork.com/members/", 1)

		// Title
		e.ForEach(`div.post div.nazev h1`, func(id int, e *colly.HTMLElement) {
			fullTitle := strings.TrimSpace(e.Text)
			sc.Title = strings.Split(fullTitle, " - ")[1]
			tmp := strings.Split(strings.Split(fullTitle, " - ")[0], " ")
			sc.SiteID = tmp[len(tmp)-1]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Covers
		e.ForEach(`div.post dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("poster")))
		})

		// Gallery
		e.ForEach(`div.post div.galerka a`, func(id int, e *colly.HTMLElement) {
			base := e.Request.AbsoluteURL(e.Attr("href"))
			base = strings.Split(base, "?")[0]
			sc.Gallery = append(sc.Gallery, base)
		})

		// Synopsis
		//		e.ForEach(`div.post div.textDetail`, func(id int, e *colly.HTMLElement) {
		//			sc.Synopsis = strings.TrimSpace(e.Text)
		e.ForEach(`meta[name="description"]`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Attr("content"))
		})

		// Tags
		e.ForEach(`div.post div.tagyall div.tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Cast
		e.ForEach(`div.post div.featuring > a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`div.post div.nazev div.datum`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.post div.nazev div.casDetail span.desktop`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.Split(e.Text, ":")[0])

			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// trailer details
		sc.TrailerType = "heresphere"
		//  extract internal id with (\d+)
		var re = regexp.MustCompile(`(?m)https:\/\/www.czechvrnetwork.com\/detail-(\d+)`)
		r := re.FindStringSubmatch(sc.HomepageURL)
		sc.TrailerSrc = "https://www.czechvrnetwork.com/heresphere/videoID" + r[1]

		// Filenames
		e.ForEach(`div.post div.download a.trailer`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				tmp := strings.Split(e.Attr("href"), "/")
				parts := strings.Split(tmp[len(tmp)-1], "-")
				base := parts[0] + "-" + parts[1] + "-" + parts[2]

				filenames := []string{
					"1920x960-30fps-smartphone_lq",
					"1920x960-60fps-psvr_med_h264_180_sbs",
					"1920x960-60fps-smartphone_hq",
					"2160x1080-60fps-smartphone_hq",
					"2880x1440-60fps-gearvr_hq_h264-180x180_3dh",
					"2880x1440-60fps-gearvr_lq_h264-180x180_3dh",
					"2880x1440-60fps-gearvr_med_h264-180x180_3dh",
					"2880x1440-60fps-oculusrift_hq_h264",
					"2880x1440-60fps-psvr_hq_h264_180_sbs",
					"3840x1920-60fps-gearvr_hq_h264-180x180_3dh",
					"3840x1920-60fps-gearvr_med_h264-180x180_3dh",
					"3840x1920-60fps-oculusrift_hq_h264",
					"3840x1920-60fps-oculusrift_med_h264",
					"5400x2700-60fps-gearvr_hq_h265-180x180_3dh",
					"5400x2700-60fps-oculusrift_hq_h265",
					"7680x3840-60fps-oculusrift_uhq_h265",
				}

				for i := range filenames {
					filenames[i] = base + "-" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div#StrankovaniDesktop span.stred a,div#StrankovaniDesktopHome span.stred a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href") + "&sites=" + nwID)
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.postTag div.foto a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.czechvrnetwork.com/vr-porn-videos&sites=" + nwID)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addCZVRScraper(id string, name string, nwid string, avatarURL string) {
	registerScraper(id, name, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return CzechVR(wg, updateSite, knownScenes, out, id, name, nwid)
	})
}

func init() {
	addCZVRScraper("czechvr", "Czech VR", "15", "https://www.czechvr.com/images/favicon/android-chrome-256x256.png")
	addCZVRScraper("czechvrfetish", "Czech VR Fetish", "16", "https://www.czechvrfetish.com/images/favicon/android-chrome-256x256.png")
	addCZVRScraper("czechvrcasting", "Czech VR Casting", "17", "https://www.czechvrcasting.com/images/favicon/android-chrome-256x256.png")
	addCZVRScraper("czechvrintimacy", "VR Intimacy", "18", "https://www.vrintimacy.com/images/favicon/android-chrome-256x256.png")
}
