package scrape

import (
	"html"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/resty.v1"
)

func ScrapeR18(knownScenes []string, out *[]models.ScrapedScene, queryString string) error {
	sceneCollector := createCollector("www.r18.com")
	siteCollector := createCollector("www.r18.com")
	siteCollector.CacheDir = ""

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "JAVR"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.Site = strings.Split(e.ChildText("title"), "-")[0]

		content_id := strings.Split(strings.Split(sc.HomepageURL, "=")[1], "/")[0]

		r, _ := resty.R().Get("https://www.r18.com/api/v4f/contents/" + content_id)

		JsonMetadata := r.String()
		//if not VR, bye bye...
		if gjson.Get(JsonMetadata, "data.is_vr").String() == "false" {
			return
		}

		// Title
		sc.Title = strings.Replace(strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, "data.title").String())), "[VR] ", "", -1)

		// Studio
		sc.Studio = gjson.Get(JsonMetadata, "data.maker.name").String()
		// Date
		sc.Released = strings.Split(gjson.Get(JsonMetadata, "data.release_date").String(), " ")[0]

		// Time
		tmpDuration, err := strconv.Atoi(gjson.Get(JsonMetadata, "data.runtime_minutes").String())
		if err == nil {
			sc.Duration = tmpDuration
		}

		// Covers
		coverimgs := gjson.Get(JsonMetadata, "data.images.jacket_image.large")
		sc.Covers = append(sc.Covers, strings.TrimSpace(html.UnescapeString(coverimgs.String())))

		// Gallery
		galleryimgs := gjson.Get(JsonMetadata, "data.gallery.#.large")
		for _, name := range galleryimgs.Array() {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(html.UnescapeString(name.String())))
		}

		// Cast
		actornames := gjson.Get(JsonMetadata, "data.actresses.#.name")
		for _, name := range actornames.Array() {
			sc.Cast = append(sc.Cast, strings.TrimSpace(html.UnescapeString(name.String())))
		}

		// Tags
		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"Featured Actress": true,
			"VR Exclusive":     true,
			"High-quality VR":  true,
			"Hi-Def":           true,
		}

		// weird... censored word?
		schoolgirltag := "S********l"

		taglist := gjson.Get(JsonMetadata, "data.categories.#.name")
		for _, name := range taglist.Array() {
			if !skiptags[name.Str] {
				if name.Str == schoolgirltag {
					sc.Tags = append(sc.Tags, "schoolgirl")
				} else {
					sc.Tags = append(sc.Tags, strings.TrimSpace(html.UnescapeString(name.String())))
				}
			}
		}
		sc.Tags = append(sc.Tags, "JAVR")

		// Scene ID
		dvdID := gjson.Get(JsonMetadata, "data.dvd_id").String()

		//		})

		if dvdID == "----" || dvdID == "" {
			sc.SceneID = content_id
			sc.SiteID = content_id
		} else {
			sc.SceneID = dvdID
			sc.SiteID = dvdID
		}

		*out = append(*out, sc)
	})

	siteCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sceneURL := ""

		e.ForEach(`li.item-list a.i3NLink`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sceneURL = strings.Split(e.Attr("href"), "?")[0]
			} else {
				sceneURL = ""
			}
		})

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if strings.Contains(queryString, "/movies/detail/") {
		return sceneCollector.Visit(queryString)
	} else {
		return siteCollector.Visit("https://www.r18.com/common/search/searchword=" + queryString + "/")
	}
}
