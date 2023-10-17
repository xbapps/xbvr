package scrape

import (
	"html"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/go-resty/resty/v2"
)

func ScrapeR18D(out *[]models.ScrapedScene, queryString string) error {
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"

		r, _ := resty.New().R().Get("https://r18.dev/videos/vod/movies/detail/-/combined=" + v + "/json")
		JsonMetadata := r.String()

		content_id := gjson.Get(JsonMetadata, "content_id").String()
		sc.HomepageURL = "https://www.dmm.co.jp/en/digital/videoa/-/detail/=/cid=" + content_id + "/"

		// Title
		if gjson.Get(JsonMetadata, "title_en_is_machine_translation").String() == "false" {
			sc.Title = strings.Replace(strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, "title_en").String())), "[VR] ", "", -1)
		} else {
			sc.Title = gjson.Get(JsonMetadata, "content_id").String()
		}

		// Studio
		sc.Studio = gjson.Get(JsonMetadata, "maker_name_en").String()

		// Date
		sc.Released = gjson.Get(JsonMetadata, "release_date").String()

		// Time
		tmpDuration, err := strconv.Atoi(gjson.Get(JsonMetadata, "runtime_mins").String())
		if err == nil {
			sc.Duration = tmpDuration
		}

		// Covers
		coverimgs := gjson.Get(JsonMetadata, "jacket_full_url")
		sc.Covers = append(sc.Covers, strings.TrimSpace(html.UnescapeString(coverimgs.String())))

		// Gallery
		galleryimgs := gjson.Get(JsonMetadata, "gallery.#.image_full")
		for _, name := range galleryimgs.Array() {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(html.UnescapeString(name.String())))
		}

		// Cast
		actornames := gjson.Get(JsonMetadata, "actresses.#.name_romaji")
		for _, name := range actornames.Array() {
			sc.Cast = append(sc.Cast, strings.TrimSpace(html.UnescapeString(name.String())))
		}

		// Tags
		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"Featured Actress":       true,
			"VR Exclusive":           true,
			"High-Quality VR":        true,
			"Exclusive Distribution": true,
		}

		// JSON dump from R18's final days shows "女子校生" replaced with "Academy Uniform" instead of "Schoolgirl" - r18.dev seems to follow the same mapping for FANZA category 1018 - get your credit card compliance puritanism out of my porn, I don't need "Academy Uniform" and "Uniform" tagged together lmao
		joshikosei := "Academy Uniform"

		taglist := gjson.Get(JsonMetadata, "categories.#.name_en")
		for _, name := range taglist.Array() {
			if !skiptags[name.Str] {
				if name.Str == joshikosei {
					sc.Tags = append(sc.Tags, "schoolgirl")
				} else {
					sc.Tags = append(sc.Tags, strings.TrimSpace(html.UnescapeString(name.String())))
				}
			}
		}
		sc.Tags = append(sc.Tags, "JAVR")
		sc.Tags = append(sc.Tags, "R18.dev")

		// Scene ID and Site
		dvdID := gjson.Get(JsonMetadata, "dvd_id").String()
		if dvdID == "----" || dvdID == "" {
			sc.SceneID = content_id
			sc.SiteID = content_id
			sc.Site = gjson.Get(JsonMetadata, "label_name_en").String()
		} else {
			sc.SceneID = dvdID
			sc.SiteID = dvdID
			sc.Site = strings.Split(dvdID, "-")[0]
		}

		if sc.SceneID != "" {
			*out = append(*out, sc)
		}
	}
	return nil
}
