package scrape

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func TwoWebMediaSite(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("wankitnowvr.com", "zexyvr.com")
	siteCollector := createCollector("wankitnowvr.com", "zexyvr.com")

	// Regex preparation
	reDateDuration := regexp.MustCompile(`Released\son\s(.*)\n+\s+Duration\s+:\s+(\d+):\d+`)
	reCastTags := regexp.MustCompile(`(?:zexyvr|wankitnowvr)\.com\/(models|videos)\/+`)
	reTagCat := regexp.MustCompile(`(.*)\s+\((.*)\)`)
	reFilename := regexp.MustCompile(`videos\/(?U:([a-z\d\-]+))(?:(?:-|_)preview)?(_\d{4}.*\.mp4)`)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "2WebMedia"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// SiteID, Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover / ID
		e.ForEach(`deo-video`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Attr("cover-image"))
		})
		// Note: not all scenes have a deo-video element, only a regular img cover instead
		if len(sc.Covers) == 0 {
			e.ForEach(`div.container.pt-5 > div > div > img`, func(id int, e *colly.HTMLElement) {
				sc.Covers = append(sc.Covers, e.Attr("src"))
			})
		}

		// Gallery
		e.ForEach(`div.gallery > div`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, e.ChildAttr("div.view > a > img", "src"))
			}
		})

		// Title
		e.ForEach(`div.container.pt-5 h2`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Synopsis
		e.ForEach(`div.container.pt-5 h2 + p`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		//Note: Date/Duration info is currently all inside the same div element...
		e.ForEach(`div.container.pt-5 p.text-muted`, func(id int, e *colly.HTMLElement) {
			tmpDateDurationParts := reDateDuration.FindStringSubmatch(e.Text)

			// Date
			if len(tmpDateDurationParts[1]) > 0 {
				tmpDate, _ := goment.New(tmpDateDurationParts[1], "MMM DD, YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}

			// Duration
			if len(tmpDateDurationParts[2]) > 0 {
				tmpDuration, err := strconv.Atoi(tmpDateDurationParts[2])
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// trailer details
		sc.TrailerType = "deovr"
		sc.TrailerSrc = strings.Replace(URL, `/videos/`, ``, 1) + `/deovr/video/` + sc.SiteID

		// Cast & Tags
		// Note: Cast/Tags links are currently all inside the same div element...
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.container.pt-5 p.text-muted > a`, func(id int, e *colly.HTMLElement) {
			tmpURLParts := reCastTags.FindStringSubmatch(e.Attr("href"))
			if len(tmpURLParts[1]) > 0 {
				if tmpURLParts[1] == "models" {
					// Cast
					sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
					sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Attr("href")}
				} else if tmpURLParts[1] == "videos" {
					// Tags
					tmpTagParts := reTagCat.FindStringSubmatch(e.Text)
					// Format is "tag (tag-category)" and we're removing the category part but some tags need fixing
					switch strings.ToLower(tmpTagParts[2]) {
					case "breasts":
						//only has tags like "enhanced/natural/small/medium/large/huge"
						if strings.ToLower(tmpTagParts[1]) == "large" {
							tmpTagParts[1] = "big tits"
						} else {
							tmpTagParts[1] = tmpTagParts[1] + " tits"
						}
					case "eyes":
						//some are like "gray eyes" while others are like "blue", but tag must include "eyes"
						if !strings.Contains(strings.ToLower(tmpTagParts[1]), "eyes") {
							tmpTagParts[1] = tmpTagParts[1] + " eyes"
						}
					case "lingerie":
						//only has the lingerie color so just use "lingerie" instead
						tmpTagParts[1] = "lingerie"
					case "nationality":
						//only change "english" to "british"
						if strings.ToLower(tmpTagParts[1]) == "english" {
							tmpTagParts[1] = "british"
						}
						//all other tags are fine to use as is.
					}

					if tmpTagParts[1] != "" {
						sc.Tags = append(sc.Tags, strings.TrimSpace(strings.ToLower(tmpTagParts[1])))
					}
				}
			}
		})

		// Filenames
		// Best guess, using trailer filenames and removing the preview-part of it.
		e.ForEach(`deo-video source`, func(id int, e *colly.HTMLElement) {
			tmpFilename := reFilename.FindStringSubmatch(e.Attr("src"))
			if len(tmpFilename) > 1 {
				sc.Filenames = append(sc.Filenames, tmpFilename[1]+tmpFilename[2])
			}
		})
		// Note: not all scenes have a deo-video element, in which case do a best guess from URL instead:
		if len(sc.Filenames) == 0 {
			tmpURLParts := strings.Split(e.Request.URL.Path, "/")
			if len(tmpURLParts) > 1 {
				baseStart := strings.Replace(tmpURLParts[2], "+", "_", -1)
				filenames := []string{"_1920", "_2160", "_2880", "_3840", "_5760"}
				baseEnd := "_180x180_3dh_180_sbs.mp4"
				for i := range filenames {
					filenames[i] = baseStart + filenames[i] + baseEnd
				}
				sc.Filenames = filenames
			}
		}

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination a.page-link`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.container div.card > a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(URL + "?order=newest")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func WankitNowVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return TwoWebMediaSite(wg, updateSite, knownScenes, out, singleSceneURL, "wankitnowvr", "WankitNowVR", "https://wankitnowvr.com/videos/", limitScraping)
}

func ZexyVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return TwoWebMediaSite(wg, updateSite, knownScenes, out, singleSceneURL, "zexyvr", "ZexyVR", "https://zexyvr.com/videos/", limitScraping)
}

func init() {
	registerScraper("wankitnowvr", "WankitNowVR", "https://mcdn.vrporn.com/files/20190103150250/wankitnow-profile.jpg", "wankitnowvr.com", WankitNowVR)
	registerScraper("zexyvr", "ZexyVR", "https://mcdn.vrporn.com/files/20210617065837/zexyvr-profile-400x400.jpg", "wankitnowvr.com", ZexyVR)
}
