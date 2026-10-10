package scrape

import (
	"html"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

// Tags
// R18.dev category ID -> re-write
// "" = drop

var r18dTagOverrides = map[int64]string{
	// Clutter/Junk
	4025: "", // Featured Actress (単体作品)
	6548: "", // Exclusive Distribution (独占配信)
	6793: "", // VR Exclusive (VR専用)
	6925: "", // High-Quality VR (ハイクオリティVR) - still used for FANZA filenames/quality below

	// Censored/Renames
	1018: "schoolgirl", // JSON dump from R18's final days shows "女子校生" replaced with "Academy Uniform" instead of "Schoolgirl" - r18.dev seems to follow the same mapping for FANZA category 1018 - get your credit card compliance puritanism out of my porn, I don't need "Academy Uniform" and "Uniform" tagged together lmao
	4121: "drunk girl", // shows up as "D***k Girl"
}

// Actress names

// R18.dev actress ID -> name. Wins over everything else. This potential bodge is only for names the filename fix described later cannot handle.
// Remove this map and the lookup at the top of r18dActressName to drop overrides entirely. Could be moved out to a JSON list like scrapers.json, with the priority flipped (user decision).
var r18dActressOverrides = map[int64]string{
	// 1234567: "GivenName FamilyName",
	1112912: "Yomogi Mochino",
}

func r18dActressName(actress gjson.Result) string {
	if fixed, ok := r18dActressOverrides[actress.Get("id").Int()]; ok {
		return fixed
	}

	name := strings.TrimSpace(html.UnescapeString(actress.Get("name_romaji").String()))
	if name == "" || strings.Contains(name, " ") {
		return name // empty, or already "FirstName LastName"
	}

	// Actresses who never had an official English name on R18.com (or a fix added by R18.dev themselves, Ozakierika is now correctly Erika Ozaki) come back from R18.dev as one word, family name first: 藍瀬ミナ (あいせみな) -> "Aisemina".
	// The image filename sometimes still has the split ("aise_mina.jpg"). If the filename with the underscore stripped match the name, rebuild it as "First/GivenName Last/FamilyName" -> "Mina Aise".
	// Filenames can have a number on the end ("adati_yuri2.jpg") and are often Kunrei-shiki while name_romaji is Hepburn: "adati" vs "Adachi", "tatibana" vs "Tachibana". Both sides are compared in Kunrei-shiki, the split point comes from the filename, the spelling from name_romaji -> "Yuri Adachi".
	// Anything that still doesn't match is left alone.
	img := path.Base(actress.Get("image_url").String())
	img = strings.TrimSuffix(img, path.Ext(img))
	img = strings.TrimRight(img, "0123456789")
	parts := strings.Split(img, "_")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return name // no clean family_given split (AIKA, JULIA, no image...)
	}

	family, given := r18dKunrei(parts[0]), r18dKunrei(parts[1])
	lower := strings.ToLower(name)
	if r18dKunrei(lower) != family+given {
		return name // letters don't match (some other romanization difference, etc.)
	}

	for i := 1; i < len(lower); i++ {
		if r18dKunrei(lower[:i]) == family && r18dKunrei(lower[i:]) == given {
			return r18dCapitalize(lower[i:]) + " " + r18dCapitalize(lower[:i])
		}
	}
	return name
}

// Hepburn -> Kunrei-shiki, only the spellings that differ in names. Already Kunrei-shiki text has none of these, so it comes back unchanged.
// shi -> si, sha -> sya, chi -> ti, cha -> tya, tsu -> tu, fu -> hu, ji -> zi, ja -> zya (matcha -> mattya works out on its own)
var r18dHepburnRe = regexp.MustCompile(`(sh|ch|j)([aiueo])|tsu|fu`)

func r18dKunrei(s string) string {
	return r18dHepburnRe.ReplaceAllStringFunc(strings.ToLower(s), func(m string) string {
		switch m {
		case "tsu":
			return "tu"
		case "fu":
			return "hu"
		}
		cons, vowel := m[:len(m)-1], m[len(m)-1:]
		c := map[string]string{"sh": "s", "ch": "t", "j": "z"}[cons]
		if vowel == "i" {
			return c + "i"
		}
		return c + "y" + vowel
	})
}

func r18dCapitalize(s string) string {
	s = strings.ToLower(s)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

const (
	r18dCatHQVR = 6925
	r18dCat8KVR = 307935
)

func ScrapeR18D(out *[]models.ScrapedScene, queryString string) error {
	client := resty.New()
	first := true
	for _, v := range strings.Split(queryString, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if !first {
			time.Sleep(1 * time.Second)
		}
		first = false

		res, err := getByContentId(client, v)
		if err != nil {
			log.Errorln("Failed to fetch R18.dev data for", v, ":", err)
			continue
		}
		if res.StatusCode() == 404 {
			res, err = getByDVDId(client, v)
			if err != nil || res.StatusCode() != 200 {
				log.Errorln("R18.dev: no match for", v, "by content ID or DVD ID")
				continue
			}
			content_id := gjson.Get(res.String(), "content_id").String()
			res, err = getByContentId(client, content_id)
			if err != nil {
				log.Errorln("Failed to fetch R18.dev data for", content_id, ":", err)
				continue
			}
		}
		if res.StatusCode() != 200 {
			log.Errorln("R18.dev returned", res.StatusCode(), "for", v)
			continue
		}

		JsonMetadata := res.String()
		sc := r18dSceneFromJSON(JsonMetadata)

		// Scene ID and Site (comma box keeps its original behavior)
		dvdID := gjson.Get(JsonMetadata, "dvd_id").String()
		if dvdID == "----" || dvdID == "" {
			sc.SceneID = sc.SiteID // content ID
			sc.Site = gjson.Get(JsonMetadata, "label_name_en").String()
		} else {
			sc.SceneID = dvdID
			sc.SiteID = dvdID
			sc.Site = strings.Split(dvdID, "-")[0]
		}

		if sc.SceneID != "" {
			log.Infoln("Scraping [" + sc.SceneID + "] " + sc.Title)
			*out = append(*out, sc)
		}
	}
	return nil
}

// r18dSceneFromJSON builds a scene from R18.dev detail JSON: title, studio, dates, images, cast,
// tags and filler filenames. Shared by the comma-box scraper and the R18.dev crawler; callers set
// SceneID / SiteID / Site (SiteID is pre-filled with the content ID).
func r18dSceneFromJSON(JsonMetadata string) models.ScrapedScene {
	sc := models.ScrapedScene{}
	sc.SceneType = "VR"

	content_id := gjson.Get(JsonMetadata, "content_id").String()
	sc.SiteID = content_id
	sc.HomepageURL = "https://r18.dev/videos/vod/movies/detail/-/id=" + content_id + "/"

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
	for _, actress := range gjson.Get(JsonMetadata, "actresses").Array() {
		if name := r18dActressName(actress); name != "" {
			sc.Cast = append(sc.Cast, name)
		}
	}

	quality := "VR"
	for _, cat := range gjson.Get(JsonMetadata, "categories").Array() {
		id := cat.Get("id").Int()

		switch {
		case id == r18dCat8KVR:
			quality = "8K"
		case id == r18dCatHQVR && quality != "8K":
			quality = "HQ"
		}

		name := strings.TrimSpace(html.UnescapeString(cat.Get("name_en").String()))
		if repl, ok := r18dTagOverrides[id]; ok {
			if repl == "" {
				continue
			}
			name = repl
		}
		sc.Tags = append(sc.Tags, name)
	}
	sc.Tags = append(sc.Tags, "JAVR")
	sc.Tags = append(sc.Tags, "R18.dev")

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

	return sc
}

func getByContentId(client *resty.Client, content_id string) (*resty.Response, error) {
	return client.R().SetHeader("User-Agent", UserAgent).
		Get("https://r18.dev/videos/vod/movies/detail/-/combined=" + content_id + "/json")
}

func getByDVDId(client *resty.Client, dvd_id string) (*resty.Response, error) {
	return client.R().SetHeader("User-Agent", UserAgent).
		Get("https://r18.dev/videos/vod/movies/detail/-/dvd_id=" + dvd_id + "/json")
}
