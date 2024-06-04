package scrape

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/xbapps/xbvr/pkg/models"
	"strconv"
	"strings"
)

func ScrapeJavDB(out *[]models.ScrapedScene, queryString string) {
	sceneCollector := createCollector("www.javdatabase.com")

	sceneCollector.OnHTML(`html`, func(html *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		contentId := ""

		// Always add 'javr' as a tag
		sc.Tags = append(sc.Tags, `javr`)

		// Always add 'javdatabase' as a tag
		sc.Tags = append(sc.Tags, `javdatabase`)

		// Cast
		html.ForEach("h2.subhead", func(id int, h2 *colly.HTMLElement) {
			if strings.HasSuffix(h2.Text, "Actress/Idols") {
				h2.DOM.Parent().Find("div.card-body a.cut-text").Each(func(i int, anchor *goquery.Selection) {
					href, exists := anchor.Attr("href")
					if exists && strings.Contains(href, "javdatabase.com/idols/") && anchor.Text() != "" {
						sc.Cast = append(sc.Cast, strings.TrimSpace(anchor.Text()))
					}
				})
			}
		})

		html.ForEach(`div.movietable tr`, func(id int, tr *colly.HTMLElement) {
			label := tr.ChildText(`td.tablelabel`)

			if label == `Studio:` {
				// Studio
				sc.Studio = tr.ChildText(`td.tablevalue > span`)

			} else if label == `DVD ID:` {
				// Title, SceneID and SiteID all like 'VRKM-821' format
				dvdId := strings.ToUpper(tr.ChildText(`td.tablevalue`))
				sc.Title = dvdId
				sc.SceneID = dvdId
				sc.SiteID = dvdId

				// Set 'Site' to first part of the ID (e.g. `VRKM for `vrkm-821`)
				siteParts := strings.Split(dvdId, `-`)
				if len(siteParts) > 0 {
					sc.Site = siteParts[0]
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
				tr.ForEach("a", func(id int, anchor *colly.HTMLElement) {
					href := anchor.Attr("href")
					if strings.Contains(href, "javdatabase.com/genres/") {
						// Tags
						tag := ProcessJavrTag(anchor.Text)

						if tag != "" {
							sc.Tags = append(sc.Tags, tag)
						}
					}
				})

			} else if label == `Translated Title:` {
				// Synopsis / description
				sc.Synopsis = tr.ChildText(`td.tablevalue`)

			} else if label == `Content ID:` {
				contentId = tr.ChildText(`td.tablevalue`)

			} else if label == "Runtime:" {
				// Duration
				sc.Duration, _ = strconv.Atoi(strings.Split(tr.ChildText(`td.tablevalue`), " ")[0])
			}

		})

		html.ForEach(`p.mb-1`, func(id int, p *colly.HTMLElement) {
			tr := strings.Split(p.Text, ": ")
			label := tr[0]

			if label == `Studio` {
				// Studio
				sc.Studio = tr[1]

			} else if label == `DVD ID` {
				// Title, SceneID and SiteID all like 'VRKM-821' format
				dvdId := strings.ToUpper(tr[1])
				sc.Title = dvdId
				sc.SceneID = dvdId
				sc.SiteID = dvdId

				// Set 'Site' to first part of the ID (e.g. `VRKM for `vrkm-821`)
				siteParts := strings.Split(dvdId, `-`)
				if len(siteParts) > 0 {
					sc.Site = siteParts[0]
				}

			} else if label == `Release Date` {
				// Release date
				dateStr := tr[1]
				tmpDate, _ := goment.New(strings.TrimSpace(dateStr), "YYYY-MM-DD")
				sc.Released = tmpDate.Format("YYYY-MM-DD")

			} else if label == `Genre(s)` {
				// Tags
				/* NOTE:
				   "Tags are technically incomplete vs. what you'd get translating dmm.co.jp
				   tags/correlating them back to their old equivalents on r18 using something
				   like Javinizer's tag CSV"
				*/
				p.ForEach("a", func(id int, anchor *colly.HTMLElement) {
					href := anchor.Attr("href")
					if strings.Contains(href, "javdatabase.com/genres/") {
						// Tags
						tag := ProcessJavrTag(anchor.Text)

						if tag != "" {
							sc.Tags = append(sc.Tags, tag)
						}
					}
				})

			} else if label == `Translated Title` {
				// Synopsis / description
				sc.Synopsis = tr[1]

			} else if label == `Content ID` {
				contentId = tr[1]

			} else if label == "Runtime" {
				// Duration
				sc.Duration, _ = strconv.Atoi(strings.Split(tr[1], " ")[0])
			}

		})

		// Screenshots
		html.ForEach("a[href]", func(_ int, anchor *colly.HTMLElement) {
			linkHref := anchor.Attr(`href`)
			/* NOTE:
			   it only pulls 6 gallery images, but that appears to be a limitation
			   of how javdatabase.com is set up, they only pull 6 gallery images.
			*/
			if strings.HasPrefix(linkHref, "https://pics.dmm.co.jp/digital/video/") && strings.HasSuffix(linkHref, `.jpg`) {
				sc.Gallery = append(sc.Gallery, linkHref)
			}
		})

		// Apply post-processing for error-correcting code
		PostProcessJavScene(&sc, contentId)

		if sc.SceneID != "" {
			*out = append(*out, sc)
		}
	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sceneCollector.Visit("https://www.javdatabase.com/movies/" + strings.ToLower(v) + "/")
	}

	sceneCollector.Wait()
}
