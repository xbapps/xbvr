package scrape

import (
	// "encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	// "net/http"
	// "io"
	// "fmt"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	// "github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/go-resty/resty/v2"
	"github.com/nleeper/goment"
	
)

func UpCloseVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	// this scraper is non-standard in that it gathers info via an api rather than scraping html pages
	defer wg.Done()
	scraperID := "upclosevr"
	siteID := "UpCloseVR"
	logScrapeStart(scraperID, siteID)
	// nextApiUrl := ""

	siteCollector := createCollector("www.upclosevr.com")
	// apiCollector := createCollector("site-api.project1service.com")
	// offset := 0

	// apiCollector.OnResponse(func(r *colly.Response) {
	// 	sceneListJson := gjson.ParseBytes(r.Body)

	// 	processScene := func(scene gjson.Result) {
	// 		sc := models.ScrapedScene{}
	// 		sc.ScraperID = scraperID
	// 		sc.SceneType = "VR"
	// 		sc.Studio = "BangBros"
	// 		sc.Site = siteID
	// 		id := strconv.Itoa(int(scene.Get("id").Int()))
	// 		sc.SceneID = "bvr-" + id

	// 		sc.Title = scene.Get("title").String()
	// 		sc.HomepageURL = "https://virtualporn.com/video/" + id + "/" + slugify.Slugify(strings.ReplaceAll(sc.Title, "'", ""))
	// 		sc.MembersUrl = "https://site-ma.virtualporn.com/scene/" + id + "/" + slugify.Slugify(strings.ReplaceAll(sc.Title, "'", ""))
	// 		sc.Synopsis = scene.Get("description").String()
	// 		dateParts := strings.Split(scene.Get("dateReleased").String(), "T")
	// 		sc.Released = dateParts[0]

	// 		scene.Get("images.poster").ForEach(func(key, imgGroup gjson.Result) bool {
	// 			if key.String() == "0" {
	// 				imgurl := imgGroup.Get("xl.urls.webp").String()
	// 				if imgurl != "" {
	// 					sc.Covers = append(sc.Covers, imgurl)
	// 				}

	// 			} else {
	// 				imgurl := imgGroup.Get("xl.urls.webp").String()
	// 				if imgurl != "" {
	// 					if len(sc.Covers) == 0 {
	// 						sc.Covers = append(sc.Covers, imgurl)
	// 					} else {
	// 						sc.Gallery = append(sc.Gallery, imgurl)
	// 					}
	// 				}
	// 			}
	// 			return true
	// 		})

	// 		// Cast
	// 		sc.ActorDetails = make(map[string]models.ActorDetails)
	// 		scene.Get("actors").ForEach(func(key, actor gjson.Result) bool {
	// 			name := actor.Get("name").String()
	// 			if actor.Get("gender").String() == "female" {
	// 				sc.Cast = append(sc.Cast, name)
	// 			}
	// 			sc.ActorDetails[actor.Get("name").String()] = models.ActorDetails{Source: scraperID + " scrape", ProfileUrl: "https://virtualporn.com/model/" + strconv.Itoa(int(actor.Get("id").Int())) + "/" + slugify.Slugify(name)}
	// 			return true
	// 		})

	// 		// Tags
	// 		scene.Get("tags").ForEach(func(key, tag gjson.Result) bool {
	// 			if tag.Get("isVisible").Bool() {
	// 				sc.Tags = append(sc.Tags, tag.Get("name").String())
	// 			}
	// 			return true
	// 		})

	// 		// trailer & filename details
	// 		sc.TrailerType = "urls"
	// 		var trailers []models.VideoSource
	// 		scene.Get("children").ForEach(func(key, child gjson.Result) bool {
	// 			child.Get("videos.full.files").ForEach(func(key, file gjson.Result) bool {
	// 				quality := file.Get("format").String()
	// 				url := file.Get("urls.view").String()
	// 				filename := file.Get("urls.download").String()
	// 				if url != "" {
	// 					trailers = append(trailers, models.VideoSource{URL: url, Quality: quality})
	// 				}
	// 				pos := strings.Index(filename, "?filename=")
	// 				if pos != -1 {
	// 					sc.Filenames = append(sc.Filenames, filename[pos+10:])
	// 				}
	// 				return true
	// 			})
	// 			return true
	// 		})
	// 		trailerJson, _ := json.Marshal(models.VideoSourceResponse{VideoSources: trailers})
	// 		sc.TrailerSrc = string(trailerJson)

	// 		out <- sc

	// 	}
	// 	total := int(sceneListJson.Get("meta.total").Int())
	// 	scenes := sceneListJson.Get("result")
	// 	if strings.Contains(r.Request.URL.RawQuery, "offset=") {
	// 		scenes.ForEach(func(key, scene gjson.Result) bool {
	// 			// check if we have the scene already
	// 			matches := funk.Filter(knownScenes, func(s string) bool {
	// 				return strings.Contains(s, scene.Get("id").String())
	// 			})
	// 			if funk.IsEmpty(matches) {
	// 				processScene(scene)
	// 			}
	// 			return true
	// 		})
	// 	} else {
	// 		processScene(scenes)
	// 	}

	// 	offset += 24
	// 	if offset < total {
	// 		if !limitScraping {
	// 			apiCollector.Visit("https://site-api.project1service.com/v2/releases?type=scene&limit=24&offset=" + strconv.Itoa(offset))
	// 		}
	// 	}
	// })

	siteCollector.OnHTML(`script`, func(e *colly.HTMLElement) {
		re := regexp.MustCompile(`"apiKey":"(.+)"}},"site`)
		apiKey := re.FindStringSubmatch(e.Text)
		re = regexp.MustCompile(`"applicationID":"(.+)","apiKey`)
		applicationID := re.FindStringSubmatch(e.Text)

		if len(apiKey) > 0 && len(applicationID) > 0{	
			var data = strings.NewReader(`{"requests":[{"indexName":"all_scenes_latest_desc","params":"analytics=true&analyticsTags=%5B%22component%3Asearchlisting%22%2C%22section%3Afreetour%22%2C%22site%3Aupclosevr%22%2C%22context%3Avideos%22%2C%22device%3Adesktop%22%5D&clickAnalytics=true&facetingAfterDistinct=true&facets=%5B%22categories.name%22%5D&filters=(upcoming%3A'0')%20AND%20availableOnSite%3Aupclosevr&highlightPostTag=__%2Fais-highlight__&highlightPreTag=__ais-highlight__&hitsPerPage=60&maxValuesPerFacet=1000&page=0&query=&tagFilters="}]}`)	
			resp, _ := resty.New().R().
			SetHeader("Origin", "https://www.upclosevr.com").
			SetHeader("Referer", "https://www.upclosevr.com/").
			SetHeader("User-Agent", UserAgent).
			SetHeader("x-algolia-api-key", apiKey[1]).
			SetHeader("x-algolia-application-id", applicationID[1]).
			SetBody(data).
			Post("https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20JavaScript%20(4.22.1)%3B%20Browser%3B%20instantsearch.js%20(4.64.3)%3B%20react%20(18.2.0)%3B%20react-instantsearch%20(7.5.5)%3B%20react-instantsearch-core%20(7.5.5)%3B%20JS%20Helper%20(3.16.2)")
			
			// Convert the resp into a json string for gjson usability
			jsonString := resp.String()

			// Determine the amount of Hits in the response to know array length. Index result of results.0.Hits is unreliable
			nbScenes := int(gjson.Get(jsonString, "results.0.nbHits").Int())
			
			for i:=0; i<nbScenes; i++{
				queryStr := `results.0.hits.` + strconv.Itoa(i) 
				sc := models.ScrapedScene{}
				sc.ScraperID = scraperID
				sc.SceneType = "VR"
				sc.Studio = siteID
				sc.Site = siteID

				sc.SiteID = gjson.Get(jsonString, queryStr + `.clip_id`).String()

				sc.HomepageURL = `https://www.upclosevr.com/en/video/upclosevr/` + gjson.Get(jsonString, queryStr + `.url_title`).String() + `/` + sc.SiteID
				

				// Scene ID
				sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

				// Date
				tmpDate, _ := goment.New(gjson.Get(jsonString, queryStr + `.release_date`).String(), "YYYY-MM-DD")
				sc.Released = tmpDate.Format("YYYY-MM-DD")

				// Cover
				sc.Covers = append(sc.Covers, `https://transform.gammacdn.com/movies/` + gjson.Get(jsonString, queryStr + `.pictures.1920x1080`).String())

				// Synopsis
				sc.Synopsis = strings.TrimSpace(strings.Replace(gjson.Get(jsonString, queryStr + `.description`).String(), "</br></br>", " ", -1))

				// Title
				sc.Title = strings.TrimSpace(gjson.Get(jsonString, queryStr + `.title`).String())
				log.Infoln(`Scraping ` + sc.Title)

				sc.ActorDetails = make(map[string]models.ActorDetails)
				for i, name := range gjson.Get(jsonString, queryStr + `.female_actors.#.name`).Array(){
					sc.Cast = append(sc.Cast, name.String())

					actorQuery := queryStr + `.female_actors.` + strconv.Itoa(i)
	
					sc.ActorDetails[name.String()] = models.ActorDetails{
						Source:     scraperID + " scrape",
						ProfileUrl: `https://www.upclosevr.com/en/pornstar/view/` + gjson.Get(jsonString, actorQuery + `.url_name`).String() + `/` + gjson.Get(jsonString, actorQuery + `.actor_id`).String(),
					}	
				}

				for _, name := range gjson.Get(jsonString, queryStr + `.categories.#.name`).Array(){
					sc.Tags = append(sc.Tags, name.String())	
				}

				//Duration
				sc.Duration = int(gjson.Get(jsonString, queryStr + `.length`).Int()) / 60

				out <- sc
			}

			// client := &http.Client{}
			// var data = strings.NewReader(`{"requests":[{"indexName":"all_scenes_latest_desc","params":"analytics=true&analyticsTags=%5B%22component%3Asearchlisting%22%2C%22section%3Afreetour%22%2C%22site%3Aupclosevr%22%2C%22context%3Avideos%22%2C%22device%3Adesktop%22%5D&clickAnalytics=true&facetingAfterDistinct=true&facets=%5B%22categories.name%22%5D&filters=(upcoming%3A'0')%20AND%20availableOnSite%3Aupclosevr&highlightPostTag=__%2Fais-highlight__&highlightPreTag=__ais-highlight__&hitsPerPage=60&maxValuesPerFacet=1000&page=0&query=&tagFilters="}]}`)
			// req, err := http.NewRequest("POST", "https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20JavaScript%20(4.22.1)%3B%20Browser%3B%20instantsearch.js%20(4.64.3)%3B%20react%20(18.2.0)%3B%20react-instantsearch%20(7.5.5)%3B%20react-instantsearch-core%20(7.5.5)%3B%20JS%20Helper%20(3.16.2)", data)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// req.Header.Set("Origin", "https://www.upclosevr.com")
			// req.Header.Set("Referer", "https://www.upclosevr.com/")
			// req.Header.Set("User-Agent", UserAgent)
			// req.Header.Set("x-algolia-api-key", apiKey[1])
			// req.Header.Set("x-algolia-application-id", applicationID[1])
			// resp, err := client.Do(req)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// defer resp.Body.Close()
			// var j interface{}
			// err = json.NewDecoder(resp.Body).Decode(&j)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// log.Infoln(j.results.hits[0])
		}
		
		// if len(matches) > 1 {
		// 	instanceJson := gjson.ParseBytes([]byte(matches[1]))
		// 	token := instanceJson.Get("jwt").String()
		// 	// set up api requests to use the token in the Instance Header
		// 	apiCollector.OnRequest(func(r *colly.Request) {
		// 		r.Headers.Set("Instance", token)
		// 	})
		// 	apiCollector.Visit(nextApiUrl)
		// }
	})
	if singleSceneURL != "" {
		// ctx := colly.NewContext()
		// ctx.Put("dur", "")
		// ctx.Put("date", "")
		// urlParts := strings.Split(singleSceneURL, "/")
		// id := urlParts[len(urlParts)-2]
		// offset = 9999 // do read more pages, we only need 1
		// nextApiUrl = "https://site-api.project1service.com/v2/releases/" + id
		// siteCollector.Visit("https://virtualporn.com/videos")

	} else {
		// call virtualporn.com, this is just to get the instance token to use the api for this session
		// nextApiUrl = "https://site-api.project1service.com/v2/releases?type=scene&limit=24&offset=" + strconv.Itoa(offset)
		siteCollector.Visit("https://www.upclosevr.com/en/videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("upclosevr", "Up Close VR", "https://static01-cms-fame.gammacdn.com/upclosevr/m/3ixx4xg65im880g8/UpClose-VR_Favicon_114x114.png", "upclosevr.com", UpCloseVR)
}