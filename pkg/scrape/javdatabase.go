package scrape

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/nleeper/goment"
	"github.com/xbapps/xbvr/pkg/models"
)

func ScrapeJavDB(knownScenes []string, out *[]models.ScrapedScene, queryString string) {
	sceneCollector := createCollector("www.javdatabase.com")

	sceneCollector.OnHTML(`html`, func(html *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		contentId := ""

		// Always add 'javr' as a tag
		sc.Tags = append(sc.Tags, `javr`)

		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"featured actress":       true,
			"vr exclusive":           true,
			"high-quality vr":        true,
			"hi-def":                 true,
			"exclusive distribution": true,
		}

		// Cast
		html.ForEach("h2.subhead", func(id int, h2 *colly.HTMLElement) {
			if h2.Text == "Featured Idols" {
				dom := h2.DOM
				parent := dom.Parent()
				if parent != nil {
					parent.Find("a").Each(func(i int, anchor *goquery.Selection) {
						if anchor.Text() != "" {
							sc.Cast = append(sc.Cast, anchor.Text())
						}
					})
				}
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
						tag := strings.ToLower(anchor.Text)

						if !skiptags[tag] {
							sc.Tags = append(sc.Tags, tag)
						}
					}
				})

			} else if label == `Translated Title:` {
				// Synopsis / description
				sc.Synopsis = tr.ChildText(`td.tablevalue`)

			} else if label == `Content ID:` {
				contentId = tr.ChildText(`td.tablevalue`)
				sc.HomepageURL = `https://www.dmm.co.jp/digital/videoa/-/detail/=/cid=` + contentId + `/`
				sc.Covers = append(sc.Covers, `https://pics.dmm.co.jp/digital/video/`+contentId+`/`+contentId+`pl.jpg`)
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

		// Some specific postprocessing for error-correcting 3DSVR scenes
		if len(contentId) > 0 && sc.Site == "DSVR" {
			r := regexp.MustCompile("13dsvr0(\\d{4})")
			match := r.FindStringSubmatch(contentId)
			if match != nil && len(match) > 1 {
				// Found a 3DSVR scene that is being wrongly categorized as DSVR
				log.Println("Applying DSVR->3DSVR workaround")
				sid := match[1]
				sc.Site = "3DSVR"
				sc.SceneID = "3DSVR-" + sid
				sc.Title = sc.SceneID
				sc.SiteID = sc.SceneID
			}
		}

		*out = append(*out, sc)
	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sceneCollector.Visit("https://www.javdatabase.com/movies/" + strings.ToLower(v) + "/")
	}

	sceneCollector.Wait()
}
