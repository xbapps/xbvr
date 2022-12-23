package scrape

import (
	"strings"

	"github.com/gocolly/colly"
	"github.com/nleeper/goment"
	"github.com/xbapps/xbvr/pkg/models"
)

func ScrapeJavDB(knownScenes []string, out *[]models.ScrapedScene, queryString string) {
	sceneCollector := createCollector("www.javdatabase.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Site = "JAVR"
		sc.HomepageURL = e.Request.URL.String()

		// Cover image
		e.ForEach(`td.moviepostermobile img`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Attr(`src`))
		})

		// Cast
		e.ForEach(`div.idol-name`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, e.Text)
		})

		// Tags
		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"featured actress": true,
			"vr exclusive":     true,
			"high-quality vr":  true,
			"hi-def":           true,
		}

		e.ForEach(`div.movietable tr`, func(id int, e *colly.HTMLElement) {
			label := e.ChildText(`td.tablelabel > h3 > b`)

			// Studio
			if label == `Studio:` {
				sc.Studio = e.ChildText(`td.tablevalue > span`)

			// Title, SceneID and SiteID all like 'IPVR-016' format
			} else if label == `DVD ID:` {
				sc.Title = e.ChildText(`td.tablevalue`)
				sc.SceneID = sc.Title
				sc.SiteID = sc.Title

			// Release date
			} else if label == `Release Date:` {
				dateStr := e.ChildText(`td.tablevalue`)
				tmpDate, _ := goment.New(strings.TrimSpace(dateStr), "YYYY-MM-DD")
				sc.Released = tmpDate.Format("YYYY-MM-DD")

			// Tags
			} else if label == `Genre(s):` {
				e.ForEach(`td.tablevalue > span.tags`, func(id2 int, e2 *colly.HTMLElement) {
					tag := strings.ToLower(e2.Text)

					if !skiptags[tag] {
						sc.Tags = append(sc.Tags, tag)
					}
				})

			// Synopsis / description
			} else if label == `Translated Title:` {
				sc.Synopsis = e.ChildText(`td.tablevalue`)
			}
		})

		// Screenshots
		e.ForEach(`img`, func(id int, e *colly.HTMLElement) {
			alt := e.Attr(`alt`)
			if strings.Contains(alt, "Screenshot") {
				sc.Gallery = append(sc.Gallery, e.Attr(`src`))
			}
		})

		*out = append(*out, sc)
	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sceneCollector.Visit("https://www.javdatabase.com/movies/" + strings.ToLower(v) + "/")
	}

	sceneCollector.Wait()
}
