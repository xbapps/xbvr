package scrape

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/resty.v1"
)

func ScrapeTPDB(knownScenes []string, out *[]models.ScrapedScene, apiToken string, sceneUrl string) error {
	sc := models.ScrapedScene{}
	sc.SceneType = "VR"

	// We accept 2 scene URL syntaxes:
	// https://metadataapi.net/scenes/scene-title-1
	// or
	// https://api.metadataapi.net/scenes/scene-title-1
	re := regexp.MustCompile("metadataapi.net/scenes/(.+)")
	subMatches := re.FindStringSubmatch(sceneUrl)
	if subMatches == nil || len(subMatches) != 2 {
		return errors.New("TPDB Url is malformed")
	}

	r, _ := resty.R().
		SetAuthToken(apiToken).
		Get(fmt.Sprintf("https://api.metadataapi.net/scenes/%v", subMatches[1]))

	tpdbMetadata := r.String()

	if r.StatusCode() >= 400 {
		errorMessage := errors.New(gjson.Get(tpdbMetadata, "message").String())
		return fmt.Errorf("TPDB Error: %v", errorMessage)
	}

	// Title
	sc.Title = gjson.Get(tpdbMetadata, "data.title").String()

	// Studio
	sc.Studio = gjson.Get(tpdbMetadata, "data.site.name").String()
	sc.Site = sc.Studio

	// Synopsis
	sc.Synopsis = gjson.Get(tpdbMetadata, "data.description").String()

	// Home Page URL
	sc.HomepageURL = gjson.Get(tpdbMetadata, "data.url").String()

	// Date
	sc.Released = gjson.Get(tpdbMetadata, "data.date").String()

	// Covers
	coverImage := gjson.Get(tpdbMetadata, "data.image").String()
	sc.Covers = append(sc.Covers, coverImage)

	// Cast
	performerNames := gjson.Get(tpdbMetadata, "data.performers.#.name")
	for _, name := range performerNames.Array() {
		sc.Cast = append(sc.Cast, name.String())
	}

	// Tags
	// Skipping some very generic and useless tags
	skipTags := map[string]bool{
		"Assorted Additional Tags": true,
	}
	tags := gjson.Get(tpdbMetadata, "data.tags.#.tag")
	for _, tag := range tags.Array() {
		if !skipTags[tag.String()] {
			sc.Tags = append(sc.Tags, tag.String())
		}
	}

	sc.SiteID = gjson.Get(tpdbMetadata, "data._id").String()
	siteShortName := gjson.Get(tpdbMetadata, "data.site.short_name")
	sc.SceneID = fmt.Sprintf("tpdb-%v-%v", siteShortName, sc.SiteID)

	*out = append(*out, sc)

	return nil
}
