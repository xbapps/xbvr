package scrape

import (
	"encoding/json"
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

func VRBangersSiteNew(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	securityToken := ""

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com", "vrbtrans.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	ajaxCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com", "vrbtrans.com"),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com", "vrbtrans.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		e.ForEach(`link[rel=shortlink]`, func(id int, e *colly.HTMLElement) {
			tmp := strings.Split(e.Attr("href"), "?p=")
			sc.SiteID = tmp[1]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`h1.video-content__title`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = strings.TrimSpace(e.Text)
			}
		})

		// Date & Duration
		e.ForEach(`div.section__item-title-download-space`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, ":")
			if len(parts) > 1 {
				switch strings.TrimSpace(parts[0]) {
				case "Release date":
					tmpDate, _ := goment.New(strings.TrimSpace(parts[1]), "MMMM D, YYYY")
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
				baseName := strings.Replace(basePath[1], "vrb_", "", -1)

				filenames := []string{"6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

				for i := range filenames {
					filenames[i] = "VRBANGERS_" + baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		// Cover URLs
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("poster")))
		})

		// Gallery
		e.ForEach(`div.gallery-top a.fancybox.image`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis
		e.ForEach(`div.video-content__description div.less-text`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.video-item-info-tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast
		e.ForEach(`div.video-item-info--starring-download a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		out <- sc
	})

	siteCollector.OnHTML(`script`, func(e *colly.HTMLElement) {
		if strings.HasPrefix(e.Text, "var config=") {
			re := regexp.MustCompile("\"security\":\"(\\w+)\"")
			securityToken = re.FindStringSubmatch(e.Text)[1]
		}
	})

	siteCollector.OnHTML(`div.pagination`, func(e *colly.HTMLElement) {
		re := regexp.MustCompile("https://vrbangers.com/videos/page/(\\d+)/")
		maxPage := 0
		e.ForEach(`a.page-numbers`, func(id int, e *colly.HTMLElement) {
			matches := re.FindStringSubmatch(e.Attr("href"))
			page, err := strconv.Atoi(matches[1])
			if err == nil && page > maxPage {
				maxPage = page
			}
		})
		maxPage = (maxPage / 10) + 1
		fastAjaxUrl := "https://vrbangers.com/wp-content/themes/vrbangers/fastAjax/index.php"
		if securityToken != "" {
			for i := 1; i <= maxPage; i++ {
				var params = map[string]string{
					"security": securityToken,
					"action":   "ajaxSort",
					"tax":      "All videos",
					"taxType":  "video_category",
					"sortBy":   "latest",
					"page":     strconv.Itoa(i),
					"pageSlug": "videos",
					"perPage":  "120",
				}
				log.Println("visiting page " + strconv.Itoa(i))

				ajaxCollector.Post(fastAjaxUrl, params)
			}
		}
	})

	ajaxCollector.OnResponse(func(r *colly.Response) {
		var body string
		json.Unmarshal(r.Body, &body)
		r.Body = []byte("<html>" + body + "</html>")
		r.Headers.Set("Content-Type", "text/html; charset=UTF-8")
	})

	ajaxCollector.OnHTML(`div.video-item-info--title a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit(URL)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VRBangersSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com", "vrbtrans.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com", "vrbtrans.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		e.ForEach(`link[rel=shortlink]`, func(id int, e *colly.HTMLElement) {
			tmp := strings.Split(e.Attr("href"), "?p=")
			sc.SiteID = tmp[1]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`div.video-info-title h1 span`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = e.Text
			}
		})

		// Date
		e.ForEach(`p[itemprop=datePublished]`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "DD MMMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`p.minutes`, func(id int, e *colly.HTMLElement) {
			minutes := strings.Split(e.Text, ":")[0]
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(minutes))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames
		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				basePath := strings.Split(strings.Replace(e.Attr("src"), "//", "", -1), "/")
				baseName := strings.Replace(basePath[1], "vrb_", "", -1)

				filenames := []string{"6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

				for i := range filenames {
					filenames[i] = "VRBANGERS_" + baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		// Cover URLs
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("poster")))
		})

		e.ForEach(`img.girls_image`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("src")))
		})

		// Gallery
		e.ForEach(`div#single-video-gallery-free a,div.old-gallery a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis
		e.ForEach(`div.mainContent`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.video-tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast
		e.ForEach(`div.video-info-title h1 span a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.wp-pagenavi a.page`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.video-page-block a.model-foto`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit(URL)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VRBangers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSiteNew(wg, updateSite, knownScenes, out, "vrbangers", "VRBangers", "https://vrbangers.com/videos/")
}
func VRBTrans(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbtrans", "VRBTrans", "https://vrbtrans.com/videos/")
}

func init() {
	registerScraper("vrbangers", "VRBangers", VRBangers)
	registerScraper("vrbtrans", "VRBTrans", VRBTrans)
}
