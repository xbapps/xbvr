package scrape

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly"
	"github.com/xbapps/xbvr/pkg/models"
)

func ScrapeJavBus(out *[]models.ScrapedScene, queryString string) {
	sceneCollector := createCollector("www.javbus.com")

	sceneCollector.OnHTML(`html`, func(html *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"

		// Always add 'javr' as a tag
		sc.Tags = append(sc.Tags, `javr`)

		// Always add 'javbus' as a tag
		sc.Tags = append(sc.Tags, `javbus`)

		html.ForEach(`div.row.movie div.info > p`, func(id int, p *colly.HTMLElement) {
			label := p.ChildText(`span.header`)

			if label == `Studio:` {
				// Studio
				sc.Studio = p.ChildText(`a`)

			} else if label == `ID:` {
				// Title, SceneID and SiteID all like 'VRKM-821' format
				idRegex := regexp.MustCompile("^([A-Za-z0-9]+)-([0-9]+)$")
				p.ForEach("span", func(_ int, span *colly.HTMLElement) {
					match := idRegex.FindStringSubmatch(span.Text)
					if match != nil && len(match) > 2 {
						dvdId := match[1] + "-" + match[2]
						sc.Title = dvdId
						sc.SceneID = dvdId
						sc.SiteID = dvdId
						sc.Site = match[1]
					}
				})

			} else if label == `Release Date:` {
				// Release date
				dateStr := p.Text
				dateRegex := regexp.MustCompile("(\\d\\d\\d\\d-\\d\\d-\\d\\d)")
				match := dateRegex.FindStringSubmatch(dateStr)
				if match != nil && len(match) > 1 {
					sc.Released = match[1]
				}
			}
		})

		// Tags
		html.ForEach("div.row.movie span.genre > label > a", func(id int, anchor *colly.HTMLElement) {
			href := anchor.Attr("href")
			if strings.Contains(href, "javbus.com/en/genre/") {
				// Tags
				tag := ProcessJavrTag(anchor.Text)

				if tag != "" {
					sc.Tags = append(sc.Tags, tag)
				}
			}
		})

		// Cast
		html.ForEach("div.row.movie div.star-name > a", func(id int, anchor *colly.HTMLElement) {
			href := anchor.Attr("href")
			if strings.Contains(href, "javbus.com/en/star/") {
				sc.Cast = append(sc.Cast, anchor.Text)
			}
		})

		// Screenshots
		html.ForEach("a[href]", func(_ int, anchor *colly.HTMLElement) {
			linkHref := anchor.Attr(`href`)
			if strings.HasPrefix(linkHref, "https://pics.dmm.co.jp/digital/video/") && strings.HasSuffix(linkHref, `.jpg`) {
				sc.Gallery = append(sc.Gallery, linkHref)
			}
		})

		// Apply post-processing for error-correcting code
		PostProcessJavScene(&sc, "")

		if sc.SceneID != "" {
			*out = append(*out, sc)
		}
	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sceneCollector.Visit("https://www.javbus.com/en/" + strings.ToUpper(v) + "/")
	}

	sceneCollector.Wait()
}
