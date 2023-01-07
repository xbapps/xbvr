package scrape

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/xbapps/xbvr/pkg/models"
)

func ScrapeJavBus(out *[]models.ScrapedScene, queryString string) {
	sceneCollector := createCollector("www.javbus.com")

	sceneCollector.OnHTML(`html`, func(html *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		contentIdRegex := regexp.MustCompile("//pics.dmm.co.jp/digital/video/([^/]+)/")
		contentId := ""
		num := ""

		// Always add 'javr' as a tag
		sc.Tags = append(sc.Tags, `javr`)

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
						num = match[2]
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

				// Extract the content ID from the image url
				if len(contentId) == 0 {
					// Find content ID from image url
					match := contentIdRegex.FindStringSubmatch(linkHref)
					if match != nil && len(match) > 1 {
						contentId = match[1]
					}
				}
			}
		})

		// If we didn't get the contentId from the screenshots, use the guessed one
		if len(contentId) == 0 {
			// Guess contentId based on dvdId, as javbus simply doesn't have it otherwise.
			// 3DSVR-0878 and FSDSS-335 are examples of scenes that really has no contentId there
			i, _ := strconv.ParseInt(num, 10, 32)
			s := strings.ToLower(sc.Site)
			// what's the point of having nice simple unique id's if we have to use 3 different
			// versions of them...
			nameMap := map[string]bool{
				"3dsvr": true,
				"fsdss": true,
			}
			if nameMap[s] == true {
				s = "1" + s
			}
			contentId = fmt.Sprintf("%s%05d", s, i)
		}

		// Set Homepage and Covers based on the content id
		sc.HomepageURL = `https://www.dmm.co.jp/digital/videoa/-/detail/=/cid=` + contentId + `/`
		sc.Covers = append(sc.Covers, `https://pics.dmm.co.jp/digital/video/`+contentId+`/`+contentId+`pl.jpg`)

		// Some scenes have no gallery (i.e. 3DSVR-0878, FSDSS-335), so fill some stuff. Hopefully not needed much.
		if len(sc.Gallery) == 0 {
			for i := 1; i < 7; i++ {
				url := fmt.Sprintf("https:/pics.dmm.co.jp/digital/video/%s/%sjp-%d.jpg", contentId, contentId, i)
				sc.Covers = append(sc.Covers, url)
			}
		}

		*out = append(*out, sc)
	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sceneCollector.Visit("https://www.javbus.com/en/" + strings.ToUpper(v) + "/")
	}

	sceneCollector.Wait()
}
