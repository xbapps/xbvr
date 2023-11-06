package scrape

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xbapps/xbvr/pkg/models"
)

/* Returns en empty string if this tag is to be skipped, or the
 * mapped tag if it should be included.
 */
func ProcessJavrTag(tag string) string {
	taglower := strings.TrimSpace(strings.ToLower(tag))

	// Skipping some very generic and useless tags
	skiptags := map[string]bool{
		"featured actress":       true,
		"vr exclusive":           true,
		"high quality vr":        true,
		"high-quality vr":        true,
		"vr":                     true,
		"vr only":                true,
		"hi-def":                 true,
		"exclusive distribution": true,
		"single work":            true,
		"solo work":              true,
		"solowork":               true,
		"dmm exclusive":          true,
		"over 4 hours":           true,
	}
	if skiptags[taglower] {
		return ""
	}

	// Map some tags to normalize so different sources match
	// TODO: this mapping is totally incomplete and needs help from community to fill
	maptags := map[string]string{
		"blow":               "blowjob",
		"blow job":           "blowjob",
		"kiss":               "kiss kiss",
		"kiss / kiss":        "kiss kiss",
		"prostitute":         "club hostess & sex worker",
		"prostitutes":        "club hostess & sex worker",
		"sun tan":            "suntan",
		"huge cock":          "huge dick - large dick",
		"other fetish":       "other fetishes",
		"threesome/foursome": "threesome / foursome",
		"3P, 4P":             "threesome / foursome",
	}
	if maptags[taglower] != "" {
		return maptags[taglower]
	}

	// Leave out some japanese text tags
	matched, err := regexp.Match("[^a-z0-9_\\- /&()\\+]", []byte(taglower))
	if matched || err != nil {
		return ""
	}

	// keep tag as-is (but lowercase)
	return taglower
}

func determineContentId(sc *models.ScrapedScene) string {
	contentId := ""
	contentIdRegex := regexp.MustCompile("//pics.dmm.co.jp/digital/video/([^/]+)/")

	// obtain from cover
	for i := range sc.Covers {
		href := sc.Covers[i]
		match := contentIdRegex.FindStringSubmatch(href)
		if len(match) > 1 {
			contentId = match[1]
			log.Println("Found content ID from cover image: " + contentId)
			break
		}
	}

	// obtain from gallery
	if len(contentId) == 0 {
		for i := range sc.Gallery {
			href := sc.Gallery[i]
			match := contentIdRegex.FindStringSubmatch(href)
			if len(match) > 1 {
				contentId = match[1]
				log.Println("Found content ID from gallery image: " + contentId)
				break
			}
		}
	}

	// last resort: build from dvd id
	if len(contentId) == 0 {
		// Guess contentId based on dvdId, as javbus simply doesn't have it otherwise.
		// 3DSVR-0878 and FSDSS-335 are examples of scenes that really has no contentId there
		parts := strings.Split(sc.SceneID, `-`)
		if len(parts) == 2 {
			site := strings.ToLower(parts[0])
			numstr := parts[1]
			i, _ := strconv.ParseInt(numstr, 10, 32)
			nameMap := map[string]bool{
				"3dsvr": true,
				"fsdss": true,
			}
			if nameMap[site] {
				site = "1" + site
			}
			contentId = fmt.Sprintf("%s%05d", site, i)
			log.Println("Fallback content ID from dvd ID: " + contentId)
		}
	}

	return contentId
}

func PostProcessJavScene(sc *models.ScrapedScene, contentId string) {
	if sc.SceneID == "" {
		log.Println("Scene not found.")
		return
	}

	if len(contentId) == 0 {
		contentId = determineContentId(sc)
	}

	// Set Homepage URL
	if sc.HomepageURL == "" {
		sc.HomepageURL = `https://www.dmm.co.jp/digital/videoa/-/detail/=/cid=` + contentId + `/`
	}

	// Set Cover URL
	if len(sc.Covers) == 0 {
		sc.Covers = append(sc.Covers, `https://pics.dmm.co.jp/digital/video/`+contentId+`/`+contentId+`pl.jpg`)
	}

	// Fallback gallery images if needed
	if len(sc.Gallery) == 0 {
		for i := 1; i < 7; i++ {
			url := fmt.Sprintf("https:/pics.dmm.co.jp/digital/video/%s/%sjp-%d.jpg", contentId, contentId, i)
			sc.Covers = append(sc.Covers, url)
		}
	}

	// Trim excess whitespace
	if sc.Studio != "" {
		sc.Studio = strings.TrimSpace(sc.Studio)
	}
	if sc.Synopsis != "" {
		sc.Synopsis = strings.TrimSpace(sc.Synopsis)
	}

	// Some specific postprocessing for error-correcting 3DSVR scenes
	if len(contentId) > 0 && sc.Site == "DSVR" {
		r := regexp.MustCompile(`13dsvr0(\d{4})`)
		match := r.FindStringSubmatch(contentId)
		if len(match) > 1 {
			// Found a 3DSVR scene that is being wrongly categorized as DSVR
			log.Println("Applying DSVR->3DSVR workaround")
			sid := match[1]
			sc.Site = "3DSVR"
			sc.SceneID = "3DSVR-" + sid
			sc.Title = sc.SceneID
			sc.SiteID = sc.SceneID
		}
	}
}
