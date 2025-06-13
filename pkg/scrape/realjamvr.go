package scrape

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

func RealJamSite(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, domain string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector(domain)
	siteCollector := createCollector(domain)

	c := siteCollector.Cookies(domain)
	cookie := http.Cookie{Name: "age_confirmed", Value: "Tru", Domain: domain, Path: "/", Expires: time.Now().Add(time.Hour)}
	c = append(c, &cookie)
	siteCollector.SetCookies("https://"+domain, c)
	sceneCollector.SetCookies("https://"+domain, c)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Real Jam Network"
		sc.Site = siteID
		sc.HomepageURL = strings.TrimSuffix(strings.Split(e.Request.URL.String(), "?")[0], "/")

		// web code split
		// PornCorn sources the scene_id from the trailer URL. RealJam sources the scene_id from the trailer data-id
		trailerId := ""
		if scraperID == "realjamvr" {
			trailerId = e.ChildAttr(`div.ms-5`, "data-id")
		} else {
			e.ForEach(`dl8-video source[src]`, func(id int, e *colly.HTMLElement) {
				re := regexp.MustCompile(`\/([0-9]+)\/`)
				match := re.FindStringSubmatch(e.Attr("src"))
				if len(match) > 0 {
					if trailerId != "" {
						if trailerId != match[1] {
							// don't trust trailer files, make sure they all return the same id
							trailerId = "mismatch"
						}
					}
					_, err := strconv.Atoi(match[1])
					if err == nil {
						// only assign the id if it's a valid number
						trailerId = match[1]
					}
				}
			})
		}
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + trailerId

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		// RealJamVR & PornCorn web code split
		sc.ActorDetails = make(map[string]models.ActorDetails)

		if scraperID == "realjamvr" {
			e.ForEach(`div.mb-1 > a[href^='/actor/']`, func(id int, e *colly.HTMLElement) {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
				sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
			})
		} else {
			e.ForEach(`div.scene-view > a[href^='/actor/']`, func(id int, e *colly.HTMLElement) {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
				sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
			})
		}
		// Released
		e.ForEach(`.bi-calendar3`, func(id int, e *colly.HTMLElement) {
			p := e.DOM.Parent().Next()
			d, err := goment.New(p.Text(), "MMM DD, YYYY")
			if err != nil {
				log.Infof("%v", err)
			}
			sc.Released = d.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`.bi-clock-history + span`, func(id int, e *colly.HTMLElement) {
			t, _ := time.Parse("15:04:05", e.Text)
			sc.Duration = t.Minute() + t.Hour()*60
		})

		// Title
		titleSelection := e.DOM.Find(`h1`)
		titleSelection.Children().Remove()
		sc.Title = strings.TrimSpace(titleSelection.Text())

		// Cover URL
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			coverURL := e.Attr("poster")
			if len(coverURL) > 0 {
				sc.Covers = append(sc.Covers, coverURL)
			}
		})

		// Gallery
		e.ForEach(`.img-wrapper`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(e.Attr("data-src")))
		})

		// Synopsis
		e.ForEach(`div.my-2`, func(id int, e *colly.HTMLElement) {
			if !strings.HasPrefix(strings.TrimSpace(e.Text), "Tags:") {
				sc.Synopsis = strings.TrimSpace(e.Text)
			}
		})

		// Tags
		e.ForEach(`div a.tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Filenames
		fileMasktmp := strings.Split(sc.HomepageURL, "/")
		fileMask := strings.Replace(sc.Site, " ", "", -1) + "-" + fileMasktmp[len(fileMasktmp)-1] + "-Full$res_$fps_LR_180.mp4"

		// any "/join/" links on the public site will be for for the full movie
		uniqueFilenames := make(map[string]bool)
		e.ForEach(`a[href='/join/']`, func(id int, e *colly.HTMLElement) {
			resolution := ""
			fps := "60"
			e.ForEach(`div div`, func(id int, e *colly.HTMLElement) {
				txt := strings.TrimSpace(e.Text)
				if strings.HasPrefix(txt, "Full ") {
					index := strings.Index(txt, "p")
					if index != -1 {
						resolution = "_" + txt[5:index]
					}
					if strings.HasSuffix(txt, "HBR") {
						resolution = "-HBR" + resolution
					}
				} else {
					if strings.HasSuffix(txt, "fps") {
						fps = strings.TrimSuffix(txt, "fps")
					}
				}
			})
			if resolution != "" && fps != "" {
				filename := strings.Replace(fileMask, "$res", resolution, 1)
				filename = strings.Replace(filename, "$fps", fps, 1)
				if filename != "" && !uniqueFilenames[filename] {
					uniqueFilenames[filename] = true
					sc.Filenames = append(sc.Filenames, filename)
				}
			}
		})

		switch trailerId {
		case "":
			log.Errorf("Could not determine Scene Id for %v, Id not found", sc.HomepageURL)
		case "mismatch":
			log.Errorf("Could not determine Scene Id for %v, inconsistent trailer filenames", sc.HomepageURL)
		default:
			out <- sc
		}
	})

	siteCollector.OnHTML(`a.page-link`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.panel a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		sceneURL = strings.TrimSuffix(sceneURL, "/")

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && strings.Contains(sceneURL, domain+"/scene/") {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://" + domain + "/scenes")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func RealJamVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return RealJamSite(wg, updateSite, knownScenes, out, singleSceneURL, "realjamvr", "RealJam VR", "realjamvr.com", singeScrapeAdditionalInfo, limitScraping)
}
func PornCornVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return RealJamSite(wg, updateSite, knownScenes, out, singleSceneURL, "porncornvr", "PornCorn VR", "porncornvr.com", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("realjamvr", "RealJam VR", "https://styles.redditmedia.com/t5_3iym1/styles/communityIcon_kqzp15xw0r361.png", "realjamvr.com", RealJamVR)
	registerScraper("porncornvr", "PornCorn VR", "https://pbs.twimg.com/profile_images/1700944751837458433/IWZqucQ__400x400.jpg", "porncornvr.com", PornCornVR)
}
