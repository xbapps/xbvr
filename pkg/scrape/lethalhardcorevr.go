package scrape

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func LethalHardcoreVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "lethalhardcorevr"
	siteID := "LethalHardcoreVR"
	logScrapeStart(scraperID, siteID)

	siteCollector := createCollector("www.lethalhardcorevr.com")

	siteCollector.OnHTML(`script`, func(e *colly.HTMLElement) {
		apiKeyRegex := regexp.MustCompile(`"apiKey":"(.+)"}},"site`)
		applicationIDRegex := regexp.MustCompile(`"applicationID":"(.+)","apiKey`)
		apiKey := apiKeyRegex.FindStringSubmatch(e.Text)
		applicationID := applicationIDRegex.FindStringSubmatch(e.Text)

		if len(apiKey) > 0 && len(applicationID) > 0 {
			pageTotal := 1
			client := resty.New()

			for page := 0; page < pageTotal; page++ {

				var payloadStr string
				if singleSceneURL != "" {
					tmp := strings.Split(singleSceneURL, "/")
					sceneID := tmp[len(tmp)-1]
					payloadStr = `{"requests":[{"indexName":"all_scenes","params":"clickAnalytics=true&facetFilters=%5B%5B%22availableOnSite%3Alethalhardcorevr%22%5D%2C%5B%22clip_id%3A` + sceneID + `%22%5D%5D&facets=%5B%5D&hitsPerPage=1&tagFilters="}]}`
				} else {
					payloadStr = `{"requests":[{"indexName":"all_scenes_latest_desc","params":"analytics=true&analyticsTags=%5B%22component%3Asearchlisting%22%2C%22section%3Afreetour%22%2C%22site%3Alethalhardcorevr%22%2C%22context%3Avideos%22%2C%22device%3Adesktop%22%5D&clickAnalytics=true&facetingAfterDistinct=true&facets=%5B%22categories.name%22%5D&filters=(upcoming%3A'0')%20AND%20availableOnSite%3Alethalhardcorevr&highlightPostTag=__%2Fais-highlight__&highlightPreTag=__ais-highlight__&hitsPerPage=60&maxValuesPerFacet=1000&page=` + strconv.Itoa(page) + `&query=&tagFilters="}]}`
				}

				var payload = strings.NewReader(payloadStr)
				resp, err := client.R().
					SetHeader("Origin", "https://www.lethalhardcorevr.com").
					SetHeader("Referer", "https://www.lethalhardcorevr.com/").
					SetHeader("User-Agent", UserAgent).
					SetHeader("x-algolia-api-key", apiKey[1]).
					SetHeader("x-algolia-application-id", applicationID[1]).
					SetBody(payload).
					Post("https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20JavaScript%20(4.22.1)%3B%20Browser%3B%20instantsearch.js%20(4.64.3)%3B%20react%20(18.2.0)%3B%20react-instantsearch%20(7.5.5)%3B%20react-instantsearch-core%20(7.5.5)%3B%20JS%20Helper%20(3.16.2)")

				if err != nil {
					log.Errorln("lethalhardcorevr encountered an error on the API Call", err)
					return
				}

				// Convert the resp into a json string for gjson usability
				jsonString := resp.String()

				// Check to see if there are multiple pages of results
				if pageTotal == 1 && singleSceneURL == "" && !limitScraping {
					pageTotal = int(gjson.Get(jsonString, "results.0.nbPages").Int())
				}

				// Make sure we are getting valid response. If the hits array is zero something went wrong
				if len(gjson.Get(jsonString, "results.0.hits").Array()) == 0 {
					log.Errorln("No Results found for LethalHardcoreVR message:", gjson.Get(jsonString, "message").String(), "response code:", gjson.Get(jsonString, "status").String())
				}

				// iterate over each hit result
				for i, _ := range gjson.Get(jsonString, "results.0.hits").Array() {
					queryStr := `results.0.hits.` + strconv.Itoa(i)

					// Check to make sure we don't update scenes we have already collected
					sceneID := gjson.Get(jsonString, queryStr+`.clip_id`).String()
					sceneURL := `https://www.lethalhardcorevr.com/en/video/lethalhardcorevr/` + gjson.Get(jsonString, queryStr+`.url_title`).String() + `/` + sceneID
					if !funk.ContainsString(knownScenes, sceneURL) || singleSceneURL != "" {

						sc := models.ScrapedScene{}

						sc.ScraperID = scraperID
						sc.SceneType = "VR"
						sc.Studio = siteID
						sc.Site = siteID
						sc.SiteID = sceneID
						sc.HomepageURL = sceneURL

						// Scene ID
						sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

						// Date
						tmpDate, _ := goment.New(gjson.Get(jsonString, queryStr+`.release_date`).String(), "YYYY-MM-DD")
						sc.Released = tmpDate.Format("YYYY-MM-DD")

						// Cover
						sc.Covers = append(sc.Covers, `https://transform.gammacdn.com/movies/`+gjson.Get(jsonString, queryStr+`.pictures.1920x1080`).String())

						// Synopsis
						sc.Synopsis = strings.TrimSpace(strings.Replace(gjson.Get(jsonString, queryStr+`.description`).String(), "</br></br>", " ", -1))

						// Title
						sc.Title = strings.TrimSpace(gjson.Get(jsonString, queryStr+`.title`).String())
						log.Infoln(`Scraping ` + sc.Title)

						// Cast - Females Only can be updated to include males if wanted
						sc.ActorDetails = make(map[string]models.ActorDetails)
						for i, name := range gjson.Get(jsonString, queryStr+`.female_actors.#.name`).Array() {
							sc.Cast = append(sc.Cast, name.String())

							actorQuery := queryStr + `.female_actors.` + strconv.Itoa(i)

							sc.ActorDetails[name.String()] = models.ActorDetails{
								Source:     scraperID + " scrape",
								ProfileUrl: `https://www.lethalhardcorevr.com/en/pornstar/view/` + gjson.Get(jsonString, actorQuery+`.url_name`).String() + `/` + gjson.Get(jsonString, actorQuery+`.actor_id`).String(),
							}
						}

						// Junk Tags we don't want to add to scene data
						skiptags := map[string]bool{
							"Original Series":     true,
							"Adult Time Original": true,
						}

						// Tags
						for _, name := range gjson.Get(jsonString, queryStr+`.categories.#.name`).Array() {
							if !skiptags[name.String()] {
								sc.Tags = append(sc.Tags, name.String())
							}
						}

						// Duration is in total seconds
						sc.Duration = int(gjson.Get(jsonString, queryStr+`.length`).Int()) / 60

						out <- sc
					}
				}
			}
		}
	})

	siteCollector.Visit("https://www.lethalhardcorevr.com/en/videos/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("lethalhardcorevr", "LethalHardcoreVR", "https://imgs1cdn.adultempire.com/bn/Lethal-Hardcore-apple-touch-icon.png", "lethalhardcorevr.com", LethalHardcoreVR)
}
