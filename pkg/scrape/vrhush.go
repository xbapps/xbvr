package scrape

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRHush(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	scraperID := "vrhush"
	siteID := "VRHush"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrhush.com")
	siteCollector := createCollector("vrhush.com")
	pageCnt := 1

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRHush"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://vrhush.com/scenes/", "https://ma.vrhush.com/scene/", 1)

		// get json data
		var jsonResult map[string]interface{}
		e.ForEach(`script[Id="__NEXT_DATA__"]`, func(id int, e *colly.HTMLElement) {
			json.Unmarshal([]byte(e.Text), &jsonResult)
		})
		jsonResult = jsonResult["props"].(map[string]interface{})
		jsonResult = jsonResult["pageProps"].(map[string]interface{})
		content := jsonResult["content"].(map[string]interface{})

		// Scene ID - get from json scene code (url no longer has the code)
		if _, ok := content["scene_code"]; ok {
			tmp := strings.Split(content["scene_code"].(string), "_")[0]
			sc.SiteID = strings.Replace(tmp, "vrh", "", -1)
		} else {
			log.Warnf("Unable to process %s - no scene code", e.Request.URL)
			return
		}
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title / Cover
		if _, ok := content["title"]; ok {
			sc.Title = content["title"].(string)
		}

		if _, ok := content["trailer_screencap"]; ok {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(content["trailer_screencap"].(string)))
		}

		// Synopsis
		if _, ok := content["description"]; ok {
			sc.Synopsis = content["description"].(string)
		}

		// Tags
		if _, ok := content["tags"]; ok {
			tagList := content["tags"].([]interface{})
			for _, tag := range tagList {
				sc.Tags = append(sc.Tags, tag.(string))
			}
		}

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		if _, ok := content["models"]; ok {
			modelList := jsonResult["models"].([]interface{})
			for _, model := range modelList {
				modelMap, _ := model.(map[string]interface{})
				if modelMap["gender"] == "Female" {
					sc.Cast = append(sc.Cast, modelMap["name"].(string))
					sc.ActorDetails[modelMap["name"].(string)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: "https://vrhush.com/models/" + modelMap["slug"].(string)}
				}
			}
		}

		// Date & duration
		if _, ok := content["publish_date"]; ok {
			tmpDate, _ := goment.New(content["publish_date"].(string), "YYYY/MM/DD")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		}
		if _, ok := content["videos_duration"]; ok {
			dur_str := content["videos_duration"].(string)
			if dur_str != "" {
				num, _ := strconv.ParseFloat(dur_str, 64)
				sc.Duration = int(num / 60)
			}
		}
		// trailer details

		sc.TrailerType = "scrape_json"
		var t models.TrailerScrape
		t.SceneUrl = sc.HomepageURL
		t.HtmlElement = `script[id="__NEXT_DATA__"]`
		t.RecordPath = "props.pageProps.content.trailers"
		t.ContentPath = "url"
		t.QualityPath = "label"
		t.ContentBaseUrl = "https:"
		tmpjson, _ := json.Marshal(t)
		sc.TrailerSrc = string(tmpjson)

		// Filenames
		if _, ok := content["videos"]; ok {
			videolList := content["videos"].(map[string]interface{})
			for _, video := range videolList {
				videoMap, _ := video.(map[string]interface{})
				if _, ok := videoMap["file"]; ok {
					tmp := strings.Split(videoMap["file"].(string), "/")
					sc.Filenames = append(sc.Filenames, tmp[len(tmp)-1])
				} else {
					if _, ok := videoMap["url"]; ok {
						parsedURL, _ := url.Parse(videoMap["url"].(string))
						baseName := path.Base(parsedURL.Path)
						sc.Filenames = append(sc.Filenames, baseName)
					}
				}
			}
		}

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination li`, func(e *colly.HTMLElement) {
		if strings.Contains(e.Attr("class"), "next") && !strings.Contains(e.Attr("class"), "disabled") {
			pageCnt += 1
			pageURL := e.Request.AbsoluteURL(`https://vrhush.com/scenes?page=` + fmt.Sprint(pageCnt) + `&order_by=publish_date&sort_by=desc`)
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.contentThumb__info__title A`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://vrhush.com/scenes?page=1&order_by=publish_date&sort_by=desc")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrhush", "VRHush", "https://cdn-nexpectation.secure.yourpornpartner.com/sites/vrh/favicon/apple-touch-icon-180x180.png", "vrhush.com", VRHush)
}
