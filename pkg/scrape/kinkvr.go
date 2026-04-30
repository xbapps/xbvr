package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func KinkVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "kinkvr"
	siteID := "KinkVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.kink.com", "kink.com")
	siteCollector := createCollector("www.kink.com", "kink.com")

	setAgeGateCookie := func(r *colly.Request) {
		r.Headers.Set("Cookie", "age_gate_accepted=1")
	}
	sceneCollector.OnRequest(setAgeGateCookie)
	siteCollector.OnRequest(setAgeGateCookie)

	posterURLRegex := regexp.MustCompile(`"posterUrl":"([^"]+)"`)
	// Recover legacy kinkvr.com asset id from new-site img paths so existing user DBs stay matched.
	assetIDRegex := regexp.MustCompile(`(?:pov_|/galleryi[mM]age/)(\d+)_`)
	listingPageBase := "https://www.kink.com/shoots?channelIds=kinkvr&sort=published&thirdParty=false&page="

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "KinkVR"
		sc.Site = siteID
		sc.HomepageURL = e.Request.URL.String()
		sc.ActorDetails = make(map[string]models.ActorDetails)

		var thumbnailURL string
		e.ForEach(`script[type="application/ld+json"]`, func(id int, e *colly.HTMLElement) {
			jsonText := strings.TrimSpace(e.Text)
			if jsonText == "" {
				return
			}
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
				return
			}
			if t, ok := jsonData["@type"].(string); !ok || t != "VideoObject" {
				return
			}

			if name, ok := jsonData["name"].(string); ok {
				sc.Title = strings.TrimSpace(name)
			}
			if desc, ok := jsonData["description"].(string); ok {
				sc.Synopsis = strings.TrimSpace(desc)
			}
			if uploadDate, ok := jsonData["uploadDate"].(string); ok {
				if tmpDate, err := time.Parse(time.RFC3339, uploadDate); err == nil {
					sc.Released = tmpDate.Format("2006-01-02")
				}
			}
			if contentURL, ok := jsonData["contentUrl"].(string); ok && contentURL != "" {
				sc.TrailerType = "url"
				sc.TrailerSrc = contentURL
			}
			if thumb, ok := jsonData["thumbnailUrl"].(string); ok {
				thumbnailURL = thumb
			}
			if actors, ok := jsonData["actor"].([]interface{}); ok {
				for _, a := range actors {
					actor, ok := a.(map[string]interface{})
					if !ok {
						continue
					}
					name, _ := actor["name"].(string)
					name = strings.TrimSpace(name)
					if name == "" {
						continue
					}
					profileURL, _ := actor["url"].(string)
					profileURL = strings.Replace(profileURL, "https://kink.com/", "https://www.kink.com/", 1)
					if profileURL != "" {
						profileURL += "?ageverified=g"
					}
					sc.Cast = append(sc.Cast, name)
					sc.ActorDetails[name] = models.ActorDetails{
						Source:     scraperID + " scrape",
						ProfileUrl: profileURL,
					}
				}
			}
		})

		var posterURL string
		if dataSetup := e.ChildAttr(`div.kvjs-container`, "data-setup"); dataSetup != "" {
			if m := posterURLRegex.FindStringSubmatch(dataSetup); len(m) == 2 {
				posterURL = m[1]
			}
		}
		if posterURL != "" {
			sc.Covers = append(sc.Covers, posterURL)
		} else if thumbnailURL != "" {
			sc.Covers = append(sc.Covers, thumbnailURL)
		}

		e.ForEach(`#galleryImagesContainer img.gallery-img`, func(id int, e *colly.HTMLElement) {
			if src := e.Attr("data-image-file"); src != "" {
				sc.Gallery = append(sc.Gallery, src)
			}
		})

		// Tags from the Categories block (visible labels, e.g. "Brunette", "Flogging")
		e.ForEach(`a[href^="/tag/"][data-testid^="index-macros-link-"]`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			tag = strings.TrimSuffix(tag, ",")
			tag = strings.TrimSpace(tag)
			if tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// SiteID priority: legacy asset id from poster regex → same regex on
		// gallery URLs → title-match against existing kinkvr scenes (preserves
		// SceneID across the kinkvr.com → kink.com migration when no legacy id
		// is embedded in the new page) → new shoot id from URL.
		var siteIDStr string
		if posterURL != "" {
			if m := assetIDRegex.FindStringSubmatch(posterURL); len(m) == 2 {
				siteIDStr = m[1]
			}
		}
		if siteIDStr == "" {
			for _, gURL := range sc.Gallery {
				if m := assetIDRegex.FindStringSubmatch(gURL); len(m) == 2 {
					siteIDStr = m[1]
					break
				}
			}
		}
		if siteIDStr == "" && sc.Title != "" {
			if db, err := models.GetCommonDB(); err == nil {
				var existing models.Scene
				db.Where("scraper_id = ? AND title = ?", scraperID, sc.Title).First(&existing)
				if existing.ID != 0 {
					siteIDStr = strings.TrimPrefix(existing.SceneID, scraperID+"-")
				}
			}
		}
		if siteIDStr == "" {
			urlPath := strings.TrimSuffix(e.Request.URL.Path, "/")
			parts := strings.Split(urlPath, "/")
			if len(parts) > 0 {
				siteIDStr = parts[len(parts)-1]
			}
		}
		sc.SiteID = siteIDStr

		if singleSceneURL != "" && len(sc.Gallery) > 0 {
			sc.Covers = append(sc.Covers, sc.Gallery[0])
		}

		if sc.SiteID != "" {
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
			out <- sc
		}
	})

	var maxPage int
	siteCollector.OnHTML(`div.page-link[data-page]`, func(e *colly.HTMLElement) {
		if v, err := strconv.Atoi(e.Attr("data-page")); err == nil && v > maxPage {
			maxPage = v
		}
	})

	siteCollector.OnHTML(`a[href^="/shoot/"][data-testid$="-link-5"]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		// skip if exists
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(listingPageBase + "1")
		if !limitScraping {
			for p := 2; p <= maxPage; p++ {
				siteCollector.Visit(listingPageBase + strconv.Itoa(p))
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
	registerScraper("kinkvr", "KinkVR", "https://static.rlcontent.com/shared/KINK/skins/web-10/branding/favicon.png", "kink.com", KinkVR)
}
