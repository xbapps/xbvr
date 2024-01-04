package scrape

import (
	"html"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func ScrapeR18D(out *[]models.ScrapedScene, queryString string) error {
	scenes := strings.Split(queryString, ",")

	for _, v := range scenes {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"

		req := resty.New().R()
		res := getByContentId(req, v)

		if res.StatusCode() == 404 {
			res = getByDVDId(req, v)

			if res.StatusCode() == 200 {
				content_id := gjson.Get(res.String(), "content_id").String()
				res = getByContentId(req, content_id)
			} else {
				return nil
			}
		}

		JsonMetadata := res.String()

		content_id := gjson.Get(JsonMetadata, "content_id").String()
		sc.HomepageURL = "https://www.dmm.co.jp/en/digital/videoa/-/detail/=/cid=" + content_id + "/"

		// Title
		if gjson.Get(JsonMetadata, "title_en_is_machine_translation").String() == "false" {
			sc.Title = strings.Replace(strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, "title_en").String())), "[VR] ", "", -1)
		} else {
			sc.Title = gjson.Get(JsonMetadata, "content_id").String()
			sc.Synopsis = gjson.Get(JsonMetadata, "title_en").String()
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
		quality := "VR"
		has8KVR := false
		for _, name := range taglist.Array() {
			if name.Str == "8KVR" {
				has8KVR = true
				quality = "8K"
			} else if name.Str == "High-Quality VR" && !has8KVR {
				quality = "HQ"
			}
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

		// Filler Filenames
		resolutions := []string{"vmb"}
		if quality == "HQ" {
			resolutions = []string{"vrv1uhqe"}
		} else if quality == "8K" {
			resolutions = []string{"vrv1uhqf", "vrv18khia"}
		}
		for r := range resolutions {
			parts := []string{"", "1", "2", "3"}
			for p := range parts {
				fn := content_id + resolutions[r] + parts[p] + ".mp4"
				sc.Filenames = append(sc.Filenames, fn)
			}
		}

		if sc.SceneID != "" {
			*out = append(*out, sc)
		}
	}
	return nil
}

func getByContentId(req *resty.Request, content_id string) *resty.Response {
	res, _ := req.Get("https://r18.dev/videos/vod/movies/detail/-/combined=" + content_id + "/json")

	return res
}

func getByDVDId(req *resty.Request, dvd_id string) *resty.Response {
	res, _ := req.Get("https://r18.dev/videos/vod/movies/detail/-/dvd_id=" + dvd_id + "/json")

	return res
}
