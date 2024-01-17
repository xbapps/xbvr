package scrape

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func TNGFVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "tonightsgirlfriend"
	siteID := "Tonight's Girlfriend VR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.tonightsgirlfriend.com")
	siteCollector := createCollector("www.tonightsgirlfriend.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "NaughtyAmerica"
		sc.Site = siteID
		sc.Title = ""
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://www.tonightsgirlfriend.com/", "https://members.tonightsgirlfriend.com/", 1)

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(strings.Split(e.ChildText(`title`), "Porn")[0])

		// Date
		tmpDate, _ := goment.New(e.Request.Ctx.Get("date"), "MMM DD, YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		/*		// Duration
				e.ForEach(`div.date-tags div.duration`, func(id int, e *colly.HTMLElement) {
					r := strings.NewReplacer("|", "", "min", "")
					tmpDuration, err := strconv.Atoi(strings.TrimSpace(r.Replace(e.Text)))
					if err == nil {
						sc.Duration = tmpDuration
					}
				})
		*/

		// Filenames & Covers & Gallery
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

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4", "_8kvr264.mp4", "_8kvr265.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName+filenames[i], defaultBaseName+filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			// Gallery
			// /images3.naughtycdn.com/cms/nacmscontent/v1/scenes/tngf/anascotty/scene/image1/800x534cdynamic.jpg
			//		i := 1
			for i := 1; i < 5; i++ {
				imgTmp := "https://images3.naughtycdn.com/cms/nacmscontent/v1/scenes/tngf/" + base[6] + "/scene/image" + strconv.Itoa(i) + "/800x534cdynamic.jpg"
				sc.Gallery = append(sc.Gallery, strings.Replace(e.Request.AbsoluteURL(imgTmp), "dynamic", "", -1))
			}
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

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4", "_8kvr264.mp4", "_8kvr265.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName+filenames[i], defaultBaseName+filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`p[class="scene-description"]`))

		// Tags
		e.ForEach(`div.category.desktop-only a.cat-tag`, func(id int, e *colly.HTMLElement) {
			if e.Text != "Tonight's Girlfriend" {
				sc.Tags = append(sc.Tags, e.Text)
			}
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		e.ForEach(`p.grey-performers a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul[class=pagination] a.page-link[rel="next"]`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.panel-body`, func(e *colly.HTMLElement) {
		sceneURL := strings.Split(e.Request.AbsoluteURL(e.ChildAttr(`div.scene-thumbnail a`, "href")), "?")[0]

		ctx := colly.NewContext()
		ctx.Put("date", strings.TrimSpace(e.ChildText("div.scene-info span.scene-date")))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("date", "")

		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)

	} else {
		siteCollector.Visit("https://www.tonightsgirlfriend.com/scenes/vr")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("tonightsgirlfriend", "Tonight's Girlfriend VR", "https://mcdn.vrporn.com/files/20200404124349/TNGF_LOGO_BLK.jpg", "tonightsgirlfriend.com", TNGFVR)
}
