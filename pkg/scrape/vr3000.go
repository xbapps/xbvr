package scrape

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VR3000(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vr3000"
	siteID := "VR3000"
	logScrapeStart(scraperID, siteID)

	siteCollector := createCollector("vr3000.com", "www.vr3000.com")

	siteCollector.OnHTML(`.row.no-gutter`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VR3000"
		sc.Site = siteID

		if e.ChildText(`.welldescription`) != "" {
			coverURL := e.ChildAttr(`.col-lg-12 img`, "src")
			altCover := strings.Replace(coverURL, "panel", "dvd", 1)
			sc.Covers = append(sc.Covers, coverURL, altCover)

			sc.Title = strings.TrimSpace(e.ChildText(`div.welldescription h4`))

			// Scene ID - get from coverURL
			tmp := strings.Split(coverURL, "/")
			sc.SiteID = tmp[len(tmp)-2]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

			comments := findComments(e.DOM.ChildrenFiltered("div"))

			// Release Date: 30 Sep 2016
			tmpDate := strings.Split(getTextFromHTMLWithSelector(comments[0], "div"), ": ")[1]
			releaseDate, _ := goment.New(strings.TrimSpace(tmpDate), "DD MMM YYYY")
			sc.Released = releaseDate.Format("YYYY-MM-DD")

			e.ForEach(`div.welldescription .text-success`, func(id int, e *colly.HTMLElement) {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			})

			e.ForEach(`div.welldescription div`, func(id int, e *colly.HTMLElement) {
				t := strings.TrimSpace(e.Text)

				if strings.HasPrefix(t, "Description:") {
					s := strings.Split(t, ": ")
					if len(s) > 1 {
						sc.Synopsis = strings.TrimSpace(s[1])
					}
				}
				if strings.HasPrefix(t, "Duration:") {
					s := strings.Split(t, ": ")
					if len(s) > 1 {
						tmpDuration, err := strconv.Atoi(strings.Split(strings.TrimSpace(s[1]), ":")[0])
						if err == nil {
							sc.Duration = tmpDuration
						}
					}
				}
			})

			sc.Gallery = e.ChildAttrs(`a[data-toggle="lightbox"]`, "href")

			downloadURLs := e.ChildAttrs(`div.welldescription ul.dropdown-menu a`, "href")
			funk.ForEach(downloadURLs, func(u string) {
				f := strings.Replace(getFilenameFromURL(u), "_TR", "", 1)
				// VR3000 only started adding real links for newer scenes
				if f != "join" {
					sc.Filenames = append(sc.Filenames, f)
				} else {
					// We're just gonna guess based on the newest filenames
					filenames := []string{"%s_1920_60fps_30mb_180x180_3dh.mp4",
						"%s_1440_60fps_30mb_180x180_3dh.mp4",
						"%s_960_60fps_15mb_180x180_3dh.mp4",
						"%s_960_30fps_10mb_180x180_3dh.mp4"}
					for i := range filenames {
						filenames[i] = fmt.Sprintf(filenames[i], sc.SiteID)
					}
					sc.Filenames = append(filenames)
				}
			})

			sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`div.welldescription a.btn-primary`, "href"))
			if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
				sc.HomepageURL = sceneURL

				if sc.Title != "" {
					out <- sc
				}
			}
		}
	})

	siteCollector.OnHTML(`script`, func(e *colly.HTMLElement) {
		// Generate the page URLs from the JavaScript pagination script
		if strings.Contains(e.Text, "#pagination-vr3000") {
			p := `(?s)totalPages:\s+(?P<pages>\d+),.*href:\s+'(?P<pageURL>.*?)'`
			r := regexp.MustCompile(p)
			m := r.FindStringSubmatch(e.Text)
			maxPages, err := strconv.Atoi(m[1])
			if err == nil {
				for i := 2; i < maxPages+1; i++ {
					pageURL := strings.Replace(m[2], "{{number}}", strconv.Itoa(i), 1)
					siteCollector.Visit(pageURL)
				}
			}
		}
	})

	siteCollector.Visit("https://www.vr3000.com")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vr3000", "VR3000", "https://pbs.twimg.com/profile_images/753992720348217344/-q03m_OT_200x200.jpg", VR3000)
}
