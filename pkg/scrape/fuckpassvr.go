package scrape

import (
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func FuckPassVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	scraperID := "fuckpassvr-native"
	siteID := "FuckPassVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.fuckpassvr.com")

	client := resty.New()
	client.SetHeader("User-Agent", UserAgent)

	sceneCollector.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			return
		}
		res := gjson.ParseBytes(r.Body)
		scenedata := res.Get("data.scene")
		previewVideoURL := r.Ctx.Get("preview_video_url")

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "FuckPassVR"
		sc.Site = siteID

		slug := scenedata.Get("slug").String()
		sc.HomepageURL = "https://www.fuckpassvr.com/video/" + slug

		sc.SiteID = scenedata.Get("cms_id").String()
		if sc.SiteID == "" || !strings.HasPrefix(sc.SiteID, "FPVR") {
			return
		}
		sc.SceneID = "fpvr-" + strings.Replace(sc.SiteID, "FPVR", "", 1)

		sc.Released = scenedata.Get("active_schedule").String()[:10]
		sc.Title = scenedata.Get("name").String()
		sc.Duration = int(scenedata.Get("duration").Int())
		sc.Covers = append(sc.Covers, scenedata.Get("thumbnail_url").String())

		desc := scenedata.Get("description").String()
		desc = strings.ReplaceAll(desc, "<p>", "")
		desc = strings.ReplaceAll(desc, "</p>", "\n\n")
		re := regexp.MustCompile(`<(.|\n)*?>`) // strip_tags
		sc.Synopsis = re.ReplaceAllString(desc, "")

		sc.ActorDetails = make(map[string]models.ActorDetails)
		scenedata.Get("porn_star_lead").ForEach(func(_, star gjson.Result) bool {
			name := star.Get("name").String()
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: "https://www.fuckpassvr.com/api/api/seo?porn_star_slug=" + star.Get("slug").String()}
			return true
		})
		scenedata.Get("porn_star").ForEach(func(_, star gjson.Result) bool {
			name := star.Get("name").String()
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: "https://www.fuckpassvr.com/api/api/seo?porn_star_slug=" + star.Get("slug").String()}
			return true
		})

		scenedata.Get("tag_input").ForEach(func(_, tag gjson.Result) bool {
			sc.Tags = append(sc.Tags, tag.String())
			return true
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "web-vr-video-player source", ContentPath: "src", QualityPath: "data-quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		resolutions := []string{"8kUHD", "8kHD", "4k", "2k", "1k"}
		parsedFileNameURL, err := url.Parse(previewVideoURL)
		if err == nil {
			fileNameBase := path.Base(parsedFileNameURL.Path)
			if strings.HasSuffix(strings.ToLower(fileNameBase), "_rollover.mp4") {
				for i := range resolutions {
					fn := fileNameBase[:len(fileNameBase)-len("_rollover.mp4")] + "-FULL_" + resolutions[i] + ".mp4"
					sc.Filenames = append(sc.Filenames, fn)
				}
			}
		} else {
			log.Error(err)
		}

		resp, err := client.R().
			SetQueryParams(map[string]string{
				"scene_id": scenedata.Get("id").String(),
			}).
			Get("https://www.fuckpassvr.com/api/api/storyboard/show")

		if err == nil {
			res := gjson.ParseBytes(resp.Body())
			res.Get("data.storyboards.#.image_origin_url").ForEach(func(_, url gjson.Result) bool {
				sc.Gallery = append(sc.Gallery, url.String())
				return true
			})
		} else {
			log.Error(err)
		}

		out <- sc
	})

	var page int64 = 1
	var lastPage int64 = 1

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("preview_video_url", "")
		slug := strings.Replace(singleSceneURL, "https://www.fuckpassvr.com/video/", "", 1)
		sceneDetail := "https://www.fuckpassvr.com/api/api/scene/show?slug=" + slug
		sceneCollector.Request("GET", sceneDetail, nil, ctx, nil)
	} else {
		for page <= lastPage {
			resp, err := client.R().
				SetQueryParams(map[string]string{
					"size":   "24",
					"sortBy": "newest",
					"page":   strconv.FormatInt(page, 10),
				}).
				Get("https://www.fuckpassvr.com/api/api/scene")

			if err == nil {
				res := gjson.ParseBytes(resp.Body())
				res.Get("data.scenes.data").ForEach(func(_, scenedata gjson.Result) bool {
					ctx := colly.NewContext()
					ctx.Put("preview_video_url", scenedata.Get("preview_video_url").String())

					sceneURL := "https://www.fuckpassvr.com/video/" + scenedata.Get("slug").String()
					sceneDetail := "https://www.fuckpassvr.com/api/api/scene/show?slug=" + scenedata.Get("slug").String()

					if !funk.ContainsString(knownScenes, sceneURL) {
						sceneCollector.Request("GET", sceneDetail, nil, ctx, nil)
					}

					return true
				})
				lastPage = res.Get("data.scenes.last_page").Int()
				page = page + 1
			} else {
				log.Error(err)
			}
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("fuckpassvr-native", "FuckPassVR", "https://www.fuckpassvr.com/_nuxt/img/logo_bw.1fac7d1.png", "fuckpassvr.com", FuckPassVR)
}
