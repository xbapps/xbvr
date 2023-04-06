package scrape

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func RealJamVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "realjamvr"
	siteID := "RealJam VR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("realjamvr.com")
	siteCollector := createCollector("realjamvr.com")

	var c = siteCollector.Cookies("realjamvr.com")
	cookie := http.Cookie{Name: "age_confirmed", Value: "Tru", Domain: "realjamvr.com", Path: "/", Expires: time.Now().Add(time.Hour)}
	c = append(c, &cookie)
	siteCollector.SetCookies("https://realjamvr.com", c)
	sceneCollector.SetCookies("https://realjamvr.com", c)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Real Jam Network"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		if strings.HasSuffix(sc.HomepageURL, "/") {
			// make homepage url conistant
			sc.HomepageURL = sc.HomepageURL[0 : len(sc.HomepageURL)-1]
		}

		// source the scene_id from the trailer filename.  This is not the best appraoch but the only id source we have
		trailerId := ""
		e.ForEach(`dl8-video source[src]`, func(id int, e *colly.HTMLElement) {
			re := regexp.MustCompile(`/([0-9]+)_[0-9]+p.mp4.`)
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
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + trailerId

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		e.ForEach(`div.scene-view a[href^='/actor/']`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

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
		e.ForEach(`.bi-clock-history`, func(id int, e *colly.HTMLElement) {
			p := e.DOM.Parent()
			t, _ := time.Parse("15:04:05", p.Text())
			sc.Duration = t.Minute() + t.Hour()*60
		})

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`h1`))

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
		fileMask := ""
		// any "download/" links on the public site will be for trailers, use one trailer to get the basis of the scenes filenames
		e.ForEachWithBreak(`a[href^='download/']`, func(id int, e *colly.HTMLElement) bool {
			trailerurl := sc.HomepageURL + "/" + e.Attr("href")
			// url does not point directly to a file, need to resolve redirects with http.Head
			resp, err := http.Head(trailerurl)
			if err == nil {
				params, err := url.ParseQuery(resp.Request.URL.RawQuery)
				if err == nil {
					if fileMaskTmp, ok := params["bcdn_filename"]; ok {
						tmp := strings.Split(fileMaskTmp[0], "_")
						if len(tmp) > 4 {
							fileMask = strings.TrimSuffix(tmp[0], "-Trailer") + "-Full_$res_$fps_" + tmp[3] + "_" + tmp[4]
							return false
						}
					}
				}
			}
			return true
		})

		// any "/join/" links on the public site will be for for the full movie
		uniqueFilenames := make(map[string]bool)
		e.ForEach(`a[href='/join/']`, func(id int, e *colly.HTMLElement) {
			resolution := ""
			fps := ""
			e.ForEach(`div div`, func(id int, e *colly.HTMLElement) {
				txt := strings.TrimSpace(e.Text)
				if strings.HasPrefix(txt, "Full ") {
					index := strings.Index(txt, "p")
					if index != -1 {
						resolution = txt[5:index]
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
			log.Errorf("Could not determine Scene Id for %, Id not found", sc.HomepageURL)
		case "mismatch":
			log.Errorf("Could not determine Scene Id for %, inconsistent trailer filenames", sc.HomepageURL)
		default:
			out <- sc
		}
	})

	siteCollector.OnHTML(`a.page-link`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.panel a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if strings.HasSuffix(sceneURL, "/") {
			// make a consistent URL
			sceneURL = sceneURL[0 : len(sceneURL)-1]
		}
		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && strings.Contains(sceneURL, "realjamvr.com/scene/") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://realjamvr.com/scenes")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("realjamvr", "RealJam VR", "https://styles.redditmedia.com/t5_3iym1/styles/communityIcon_kqzp15xw0r361.png", RealJamVR)
}
