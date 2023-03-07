package scrape

import (
	"encoding/json"
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

func FuckPassVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "fuckpassvr-native"
	siteID := "FuckPassVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.fuckpassvr.com")

	sceneCollector.OnResponse(func(r *colly.Response) {
		scenedata := r.Ctx.GetAny("scenedata").(gjson.Result)
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

		scenedata.Get("porn_star_lead.#.name").ForEach(func(_, name gjson.Result) bool {
			sc.Cast = append(sc.Cast, name.String())
			return true
		})
		scenedata.Get("porn_star.#.name").ForEach(func(_, name gjson.Result) bool {
			sc.Cast = append(sc.Cast, name.String())
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

		fileNameBase := path.Base(scenedata.Get("preview_video").String())
		if strings.HasSuffix(strings.ToLower(fileNameBase), "_rollover.mp4") {
			scenedata.Get("videos.#.resolution").ForEach(func(_, resolution gjson.Result) bool {
				fn := fileNameBase[:len(fileNameBase)-len("_rollover.mp4")] + "-FULL_" + resolution.String() + ".mp4"
				sc.Filenames = append(sc.Filenames, fn)
				return true
			})
		}

		if r.StatusCode == 200 {
			res := gjson.ParseBytes(r.Body)
			res.Get("data.storyboards.#.image_origin_url").ForEach(func(_, url gjson.Result) bool {
				sc.Gallery = append(sc.Gallery, url.String())
				return true
			})
		}

		out <- sc
	})

	client := resty.New()
	client.SetHeader("User-Agent", UserAgent)

	var page int64 = 1
	var lastPage int64 = 1

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
				ctx.Put("scenedata", scenedata)

				sceneURL := "https://www.fuckpassvr.com/video/" + scenedata.Get("slug").String()
				galleryURL := "https://www.fuckpassvr.com/api/api/storyboard/show?scene_id=" + scenedata.Get("id").String()

				if !funk.ContainsString(knownScenes, sceneURL) {
					sceneCollector.Request("GET", galleryURL, nil, ctx, nil)
				}

				return true
			})
			lastPage = res.Get("data.scenes.last_page").Int()
			page = page + 1
		} else {
			log.Error(err)
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("fuckpassvr-native", "FuckPassVR", "https://www.fuckpassvr.com/_nuxt/img/logo_bw.1fac7d1.png", FuckPassVR)
}
