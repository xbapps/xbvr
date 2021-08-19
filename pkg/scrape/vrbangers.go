package scrape

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRBangersSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrbangers.com", "vrbtrans.com", "vrbgay.com")
	siteCollector := createCollector("vrbangers.com", "vrbtrans.com", "vrbgay.com")
	ajaxCollector := createCollector("vrbangers.com", "vrbtrans.com", "vrbgay.com")
	ajaxCollector.CacheDir = ""

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		if e.ChildAttr(`link[rel=shortlink]`, "href") == "" {
			log.Printf("Skipping %s because it is not a valid scene.", e.Request.URL.String())
			return
		}
		// Scene ID - get from URL
		sc.SiteID = strings.Split(e.ChildAttr(`link[rel=shortlink]`, "href"), "?p=")[1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`h1.video-content__title`))

		// Date & Duration
		e.ForEach(`div.section__item-title-download-space`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, ":")
			if len(parts) > 1 {
				switch strings.TrimSpace(parts[0]) {
				case "Release date":
					tmpDate, _ := goment.New(strings.TrimSpace(parts[1]), "MMM D, YYYY")
					sc.Released = tmpDate.Format("YYYY-MM-DD")
				case "Duration":
					durationParts := strings.Split(strings.TrimSpace(parts[1]), " ")
					tmpDuration, err := strconv.Atoi(durationParts[0])
					if err == nil {
						sc.Duration = tmpDuration
					}
				}
			}

		})

		// Filenames
		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				basePath := strings.Split(strings.Replace(e.Attr("src"), "//", "", -1), "/")
				baseName := basePath[1]
				baseName = strings.Replace(baseName, "vrb_", "VRBANGERS_", -1)
				baseName = strings.Replace(baseName, "vrbtrans_", "VRBTRANS_", -1)
				baseName = strings.Replace(baseName, "vrbgay_", "VRBGAY_", -1)

				filenames := []string{"8K_180x180_3dh", "6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

				for i := range filenames {
					filenames[i] = baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				// some pages are not referencing the "cover" but instead the VRB logo
				tmpCover := strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0]
				if tmpCover != "https://vrbangers.com/wp-content/uploads/2020/03/VR-Bangers-Logo.jpg" && tmpCover != "https://vrbgay.com/wp-content/uploads/2020/03/VRB-Gay-Logo.jpg" && tmpCover != "https://vrbtrans.com/wp-content/uploads/2020/03/VRB-Trans-Logo.jpg" {
					sc.Covers = append(sc.Covers, tmpCover)
				}
			}
		})

		sc.Covers = append(sc.Covers, e.ChildAttr(`section.banner picture source[type="image/jpeg"]`, "srcset"))

		if len(e.ChildAttrs(`section.base-content--bg div.base-border img`, "data-pagespeed-lazy-src")) > 0 {
			sc.Covers = append(sc.Covers, e.ChildAttrs(`section.base-content--bg img`, "data-pagespeed-lazy-src")[0])
		} else {
			sc.Covers = append(sc.Covers, e.ChildAttr(`section.base-content--bg div.base-border img`, "src"))
		}

		// Gallery
		sc.Gallery = e.ChildAttrs(`div.free-gallery a.fancybox`, "href")

		// Synopsis
		sc.Synopsis = strings.TrimSpace(strings.Replace(e.ChildText(`div.video-content__description div.less-text`), `arrow_drop_up`, ``, -1))

		// Tags
		e.ForEach(`div.video-item__tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})
		if scraperID == "vrbgay" {
			sc.Tags = append(sc.Tags, "Gay")
		}

		// Cast
		e.ForEach(`div.video-item-info--starring-download a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.video-item-info--title a`, func(e *colly.HTMLElement) {
		// Some index pages have links to german language scene pages:
		// https://vrbangers.com/video/sensual-seduction/?lang=de
		// This will strip out the query params... gross
		url := strings.Split(e.Attr("href"), "?")[0]
		sceneURL := e.Request.AbsoluteURL(url)

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`.pagination a`, func(e *colly.HTMLElement) {
		// Some index pages have links to german language scene pages:
		// https://vrbangers.com/video/sensual-seduction/?lang=de
		// This will strip out the query params... gross
		url := strings.Split(e.Attr("href"), "?")[0]
		pageURL := e.Request.AbsoluteURL(url)

		siteCollector.Visit(pageURL)
	})

	siteCollector.Visit(URL)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VRBangers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbangers", "VRBangers", "https://vrbangers.com/videos/")
}
func VRBTrans(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbtrans", "VRBTrans", "https://vrbtrans.com/videos/")
}
func VRBGay(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbgay", "VRBGay", "https://vrbgay.com/videos/")
}

func init() {
	registerScraper("vrbangers", "VRBangers", "https://pbs.twimg.com/profile_images/1115746243320246272/Tiaofu5P_200x200.png", VRBangers)
	registerScraper("vrbtrans", "VRBTrans", "https://pbs.twimg.com/profile_images/980851177557340160/eTnu1ZzO_200x200.jpg", VRBTrans)
	registerScraper("vrbgay", "VRBGay", "https://pbs.twimg.com/profile_images/916453413344313344/8pT50i9j_200x200.jpg", VRBGay)
}
