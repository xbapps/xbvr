package scrape

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/xbapps/xbvr/pkg/models"
)

func ScrapeJavLibraryJP(out *[]models.ScrapedScene, queryString string) {
	sceneCollector := createCollector("www.javlibrary.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		// This html page might be the redirected video details page, or the search results,
		// find out which by looking inside the DOM
		boxTitle := e.DOM.Find("div.boxtitle")
		if boxTitle != nil {
			r := regexp.MustCompile("「([^」]+)」品番検索結果")
			match := r.FindStringSubmatch(boxTitle.Text())
			if len(match) > 1 {
				// Found a search results page
				searchQuery := strings.ToLower(match[1])
				log.Printf("Search results page found for " + searchQuery)

				// Try to find exact match in the results
				videos := e.DOM.Find("div.videos div.video a")
				videos.Each(func(_ int, el *goquery.Selection) {
					sel := el.Find("div.id")
					if sel != nil {
						if strings.ToLower(sel.Text()) == searchQuery {
							href, exists := el.Attr("href")
							if exists {
								// Found matching search result, visit it
								baseURL := e.Request.URL
								hrefURL, err := url.Parse(href)
								if err == nil {
									linkURL := baseURL.ResolveReference(hrefURL)
									sceneCollector.Visit(linkURL.String())
								}
							}
						}
					}
				})

				return // end parsing html search results page
			}
		}

		// Begin parsing scene details
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"

		// Tags
		// Always add 'javr' as a tag
		sc.Tags = append(sc.Tags, `javr`)

		// Always add 'javlibrary' as a tag
		sc.Tags = append(sc.Tags, `javlibrary`)

		// ID
		videoIdSel := e.DOM.Find("div#video_id td.text")
		if videoIdSel != nil {
			dvdId := strings.ToUpper(videoIdSel.Text())
			sc.Title = dvdId
			sc.SiteID = dvdId
			sc.SceneID = dvdId

			// Set 'Site' to first part of the ID (e.g. `VRKM for `vrkm-821`)
			siteParts := strings.Split(dvdId, `-`)
			if len(siteParts) > 0 {
				sc.Site = siteParts[0]
			}
		}

		// Cover image
		coverImg := e.DOM.Find("img#video_jacket_img")
		if coverImg != nil {
			src, exists := coverImg.Attr("src")
			if exists {
				if strings.HasPrefix(src, "//") {
					// include protocol in image urls
					src = "https:" + src
				}
				sc.Covers = append(sc.Covers, src)
			}
		}

		// Gallery
		previewDiv := e.DOM.Find("div.previewthumbs")
		if previewDiv != nil {
			imgEls := previewDiv.Find("img")
			imgEls.Each(func(_ int, s *goquery.Selection) {
				src, exists := s.Attr("src")
				if exists {
					if strings.HasPrefix(src, "//") {
						// include protocol in image urls
						src = "https:" + src
					}

					// Replace low-res version with higher-res version for specific pics.dmm.co.jp images
					m := regexp.MustCompile("//pics.dmm.co.jp/digital/video/([^/]+)/(.+[0-9])-([0-9]+).jpg")
					res := m.ReplaceAllString(src, "//pics.dmm.co.jp/digital/video/${1}/${2}jp-${3}.jpg")
					sc.Gallery = append(sc.Gallery, res)
				}
			})
		}

		// Release date
		videoDateTd := e.DOM.Find("div#video_date td.text")
		if videoDateTd != nil {
			dateStr := videoDateTd.Text()
			tmpDate, _ := goment.New(strings.TrimSpace(dateStr), "YYYY-MM-DD")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		}

		// Cast
		videoCastSel := e.DOM.Find("span.star")
		videoCastSel.Each(func(_ int, s *goquery.Selection) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(s.Text()))
		})

		// Genre
		videoGenreSel := e.DOM.Find("span.genre")
		videoGenreSel.Each(func(_ int, s *goquery.Selection) {
			tag := ProcessJavrTag(s.Text())
			if tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Description
		videoTitleSel := e.DOM.Find("div#video_title h3")
		if videoTitleSel != nil {
			sc.Synopsis = videoTitleSel.Text()
			sc.Title = sc.Synopsis
		}

		// Studio
		videoStudioSel := e.DOM.Find("span.maker")
		if videoStudioSel != nil {
			sc.Studio = videoStudioSel.Text()
		}

		// Apply post-processing for error-correcting code
		PostProcessJavScene(&sc, "")

		if sc.SceneID != "" {
			*out = append(*out, sc)
		}
	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	for _, v := range scenes {
		sceneCollector.Visit("https://www.javlibrary.com/ja/vl_searchbyid.php?keyword=" + strings.ToLower(v))
	}

	sceneCollector.Wait()
}
