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

		// Cast
		e.ForEach(`div.idol-name`, func(id int, elem *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, elem.Text)
		})

		// Tags
		// Always add 'javr' as a tag
		sc.Tags = append(sc.Tags, `javr`)

		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"featured actress": true,
			"vr exclusive":     true,
			"high-quality vr":  true,
			"hi-def":           true,
		}

		e.ForEach(`div.movietable tr`, func(id int, tr *colly.HTMLElement) {
			label := tr.ChildText(`td.tablelabel > h3 > b`)

			if label == `Studio:` {
				// Studio
				sc.Studio = tr.ChildText(`td.tablevalue > span`)

			} else if label == `DVD ID:` {
				// Title, SceneID and SiteID all like 'VRKM-821' format
				dvdId := tr.ChildText(`td.tablevalue`)
				sc.Title = dvdId
				sc.SceneID = dvdId
				sc.SiteID = dvdId

				// Set 'Site' to first part of the ID (e.g. `VRKM for `vrkm-821`)
				siteParts := strings.Split(dvdId, `-`)
				if len(siteParts) > 0 {
					sc.Site = strings.ToUpper(siteParts[0])
				}

			} else if label == `Release Date:` {
				// Release date
				dateStr := tr.ChildText(`td.tablevalue`)
				tmpDate, _ := goment.New(strings.TrimSpace(dateStr), "YYYY-MM-DD")
				sc.Released = tmpDate.Format("YYYY-MM-DD")

			} else if label == `Genre(s):` {
				// Tags
				/* NOTE:
				   "Tags are technically incomplete vs. what you'd get translating dmm.co.jp
				   tags/correlating them back to their old equivalents on r18 using something
				   like Javinizer's tag CSV"
				*/
				tr.ForEach(`td.tablevalue > span.tags`, func(id2 int, span *colly.HTMLElement) {
					tag := strings.ToLower(span.Text)

					if !skiptags[tag] {
						sc.Tags = append(sc.Tags, tag)
					}
				})

			} else if label == `Translated Title:` {
				// Synopsis / description
				sc.Synopsis = tr.ChildText(`td.tablevalue`)

			} else if label == `Content ID:` {
				contentId := tr.ChildText(`td.tablevalue`)
				sc.HomepageURL = `https://www.dmm.co.jp/digital/videoa/-/detail/=/cid=` + contentId + `/`
				sc.Covers = append(sc.Covers, `https://pics.dmm.co.jp/digital/video/`+contentId+`/`+contentId+`pl.jpg`)
			}
		})

		// Screenshots
		e.ForEach("a", func(_ int, anchor *colly.HTMLElement) {
			linkHref := anchor.Attr(`href`)
			/* NOTE:
			   it only pulls 6 gallery images, but that appears to be a limitation
			   of how javdatabase.com is set up, they only pull 6 gallery images.
			*/
			if strings.HasPrefix(linkHref, "https://pics.dmm.co.jp/digital/video/") && strings.HasSuffix(linkHref, `.jpg`) {
				sc.Gallery = append(sc.Gallery, linkHref)
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
