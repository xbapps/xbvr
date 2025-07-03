package scrape

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

const (
	scraperID = "vrspy"
	siteID    = "VRSpy"
	domain    = "vrspy.com"
	baseURL   = "https://www." + domain
)

func VRSpy(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singleScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	allowedDomains := []string{domain, "www." + domain}
	sceneCollector := createCollector(allowedDomains...)
	siteCollector := createCollector(allowedDomains...)

	cookies := []*http.Cookie{
		{
			Name:    "age",
			Value:   "true",
			Path:    "/",
			Expires: time.Now().Add(365 * 24 * time.Hour),
		},
	}

	sceneCollector.OnRequest(func(r *colly.Request) {
		for _, c := range cookies {
			r.Headers.Set("Cookie", c.Name+"="+c.Value)
		}
	})

	siteCollector.OnRequest(func(r *colly.Request) {
		for _, c := range cookies {
			r.Headers.Set("Cookie", c.Name+"="+c.Value)
		}
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = siteID
		sc.Site = siteID
		sc.HomepageURL = e.Request.URL.String()

		// Extract scene ID using the most reliable method first
		pageHTML, _ := e.DOM.Html()

		// 1. Try image URLs with pattern cdn.vrspy.com/videos/{id}/ or cdn.vrspy.com/films/{id}/
		imageRegex := regexp.MustCompile(`cdn\.vrspy\.com/(?:videos|films)/(\d+)/`)
		imageMatches := imageRegex.FindStringSubmatch(pageHTML)
		if len(imageMatches) > 1 {
			sc.SiteID = imageMatches[1]
		}

		// 2. Try og:image extraction if ID not found
		if sc.SiteID == "" {
			e.ForEach(`meta[property="og:image"][content*="vrspy.com/videos"], meta[property="og:image"][content*="vrspy.com/films"]`, func(id int, e *colly.HTMLElement) {
				if sc.SiteID == "" {
					ogimage := e.Attr("content")
					if ogimage != "" {
						ogimageURL, err := url.Parse(ogimage)
						if err == nil {
							parts := strings.Split(ogimageURL.Path, "/")
							if len(parts) > 2 {
								_, err := strconv.Atoi(parts[2])
								if err == nil {
									sc.SiteID = parts[2]
								}
							}
						}
					}
				}
			})
		}

		if sc.SiteID == "" {
			log.Infof("Unable to determine a Scene Id for %s", e.Request.URL)
			return
		}

		sc.SceneID = scraperID + "-" + sc.SiteID

		// Updated title selector based on reference scrapers
		title := e.ChildText(`h1.section-header-container`)
		if title == "" {
			title = e.ChildText(`div.video-title .section-header-container`)
		}
		// Clean up the title
		title = strings.TrimSpace(title)
		title = strings.TrimSuffix(title, " Scene")
		title = strings.TrimSuffix(title, " - VR Porn")
		title = strings.TrimSuffix(title, " - Vr Porn")
		sc.Title = title

		// Updated synopsis selector
		synopsis := e.ChildText(`.show-more-text p`)
		if synopsis == "" {
			synopsis = e.ChildText(`.video-description-container`)
		}
		sc.Synopsis = synopsis

		// Updated tags selector
		tags := e.ChildTexts(`.video-categories a`)
		if len(tags) == 0 {
			tags = e.ChildTexts(`.video-categories .chip`)
		}
		sc.Tags = tags

		// Updated cast selector
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`.video-actor-item`, func(id int, e *colly.HTMLElement) {
			actorName := strings.TrimSpace(e.Text)
			if actorName != "" {
				sc.Cast = append(sc.Cast, actorName)
				e.ForEach(`a`, func(id int, a *colly.HTMLElement) {
					sc.ActorDetails[actorName] = models.ActorDetails{
						Source:     scraperID + " scrape",
						ProfileUrl: e.Request.AbsoluteURL(a.Attr(`href`)),
					}
				})
			}
		})

		// Updated date and duration extraction
		e.ForEach(`.video-details-info-item`, func(id int, e *colly.HTMLElement) {
			infoText := e.Text

			// Check for release date
			if strings.Contains(infoText, "Release date") {
				dateText := e.ChildText("span")
				if dateText == "" && strings.Contains(infoText, ":") {
					dateText = strings.TrimSpace(strings.Split(infoText, ":")[1])
				}

				if dateText != "" {
					// Try most common date format first
					tmpDate, err := goment.New(dateText, "DD MMMM YYYY")
					if err == nil {
						sc.Released = tmpDate.Format("YYYY-MM-DD")
					}
				}
			}

			// Check for duration
			if strings.Contains(infoText, "Duration") {
				durationText := e.ChildText("span")
				if durationText == "" && strings.Contains(infoText, ":") {
					durationText = strings.TrimSpace(strings.Split(infoText, ":")[1])
				}

				if durationText != "" {
					// Simplified duration extraction
					parts := strings.Split(durationText, ":")
					var hours, mins, secs int
					if len(parts) == 3 {
						hours, _ = strconv.Atoi(parts[0])
						mins, _ = strconv.Atoi(parts[1])
						secs, _ = strconv.Atoi(parts[2])
					} else if len(parts) == 2 {
						mins, _ = strconv.Atoi(parts[0])
						secs, _ = strconv.Atoi(parts[1])
					}
					sc.Duration = (hours*3600 + mins*60 + secs) / 60
				}
			}
		})

		// Set up CDN URL for covers and images
		cdnSceneURL := e.Request.URL
		cdnSceneURL.Host = "cdn." + domain
		cdnSceneURL.Path = "/videos/" + sc.SiteID

		// Look for gallery images in page HTML
		pageHTMLStr := string(e.Response.Body)

		// Extract cover images
		cover := cdnSceneURL.JoinPath("images", "cover.jpg").String()

		// Find cover images directly in HTML to get the correct URLs
		coverRegex := regexp.MustCompile(`https://cdn\.vrspy\.com/(?:videos|films)/\d+/images/cover\.jpg`)

		coverMatches := coverRegex.FindAllString(pageHTMLStr, -1)

		// Use found URLs if available, otherwise fall back to constructed URLs
		if len(coverMatches) > 0 {
			// Clean up URL - extract just the base URL without any parameters or fragments
			parsedURL, err := url.Parse(coverMatches[0])
			if err == nil {
				parsedURL.Fragment = ""
				parsedURL.RawQuery = ""
				cover = parsedURL.String()
			}
		}

		sc.Covers = []string{cover}

		// Try to find gallery images with the CDN image processing format and direct CDN URLs
		// Updated to support both /videos/ and /films/ paths
		cdnImageRegex := regexp.MustCompile(`https://vrspy\.com/cdn-cgi/image/w=\d+/https://cdn\.vrspy\.com/(?:videos|films)/\d+/photos/([^"\s]+\.jpg)`)
		directImageRegex := regexp.MustCompile(`https://cdn\.vrspy\.com/(?:videos|films)/\d+/photos/([^"\s]+\.jpg)`)

		// Extract all image filenames and their source URLs
		type imageSource struct {
			filename string
			fullURL  string
			isdirect bool
		}

		images := make([]imageSource, 0)

		// Find CDN processed images
		cdnMatches := cdnImageRegex.FindAllStringSubmatch(pageHTMLStr, -1)
		for _, match := range cdnMatches {
			if len(match) >= 2 {
				images = append(images, imageSource{
					filename: match[1],
					fullURL:  match[0],
					isdirect: false,
				})
			}
		}

		// Find direct CDN images
		directMatches := directImageRegex.FindAllStringSubmatch(pageHTMLStr, -1)
		for _, match := range directMatches {
			if len(match) >= 2 {
				images = append(images, imageSource{
					filename: match[1],
					fullURL:  match[0],
					isdirect: true,
				})
			}
		}

		// Deduplicate images by filename and standardize to w=960
		cleanGallery := make([]string, 0)
		seenFilenames := make(map[string]bool)

		// Process all images with a single loop
		for _, img := range images {
			if !seenFilenames[img.filename] {
				seenFilenames[img.filename] = true

				var directURL string
				if !img.isdirect {
					// Extract the direct URL part from CDN URL
					parts := strings.Split(img.fullURL, "/https://")
					if len(parts) < 2 {
						continue
					}
					directURL = "https://" + parts[1]
				} else {
					directURL = img.fullURL
				}

				// Remove any fragments
				directURL = strings.Split(directURL, "#")[0]

				// Add both direct URL and CDN-transformed URL as fallback if CDN transformation fails
				cleanGallery = append(cleanGallery, directURL)
			}
		}

		sc.Gallery = cleanGallery

		// Extract trailer URLs - Updated to support both /videos/ and /films/ paths
		// First try to find trailer URLs directly in the HTML
		trailerRegex := regexp.MustCompile(`https://cdn\.vrspy\.com/(?:videos|films)/\d+/trailers/\w+\.mp4\?token=[^"&]+`)
		trailerMatches := trailerRegex.FindAllString(pageHTMLStr, -1)

		if len(trailerMatches) > 0 {
			// If we found trailer URLs directly, use the first one
			sc.TrailerType = "url"
			sc.TrailerSrc = trailerMatches[0]
		} else {
			// Fallback to the scrape_html method
			sc.TrailerType = "scrape_html"
			paramsdata := models.TrailerScrape{
				SceneUrl:     sc.HomepageURL,
				HtmlElement:  "body", // Search the entire body since NUXT_DATA might be in different locations
				ExtractRegex: `(https://cdn\.vrspy\.com/(?:videos|films)/\d+/trailers/\w+\.mp4\?token=[^"&]+)`,
			}
			jsonStr, _ := json.Marshal(paramsdata)
			sc.TrailerSrc = string(jsonStr)
		}

		out <- sc
	})

	// Handle pagination
	siteCollector.OnHTML(`body`, func(e *colly.HTMLElement) {
		// Check if we need to go to the next page
		currentPage, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
		if currentPage == 0 {
			currentPage = 1
		}

		if !limitScraping {
			// Check if there are videos on this page before going to the next page
			hasVideos := false
			e.ForEach(`.item-wrapper .photo a`, func(id int, e *colly.HTMLElement) {
				hasVideos = true
			})

			if hasVideos {
				// Visit the next page
				nextPage := currentPage + 1
				siteCollector.Visit(fmt.Sprintf("%s/videos?sort=all&page=%d", baseURL, nextPage))
			}
		}
	})

	// Find and visit scene pages
	siteCollector.OnHTML(`.item-wrapper .photo a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	// Fallback selector for scene links
	siteCollector.OnHTML(`.video-section a.photo-preview`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		// Ensure single scene URL uses www subdomain
		if !strings.Contains(singleSceneURL, "www.") && strings.Contains(singleSceneURL, "://") {
			parts := strings.Split(singleSceneURL, "://")
			if len(parts) > 1 {
				singleSceneURL = parts[0] + "://www." + strings.TrimPrefix(parts[1], "www.")
			}
		}
		log.Infof("visiting %s", singleSceneURL)
		sceneCollector.Visit(singleSceneURL)
	} else {
		listingURL := baseURL + "/videos?sort=all&page=1"
		log.Infof("visiting %s", listingURL)
		siteCollector.Visit(listingURL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper(scraperID, siteID, baseURL+"/favicon.ico", domain, VRSpy)
}
