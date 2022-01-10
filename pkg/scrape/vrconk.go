package scrape

import (
	"math"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/resty.v1"
)

func VRCONK(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vrconk"
	siteID := "VRCONK"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrconk.com")
	siteCollector := createCollector("vrconk.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRCONK"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		content_id := strings.Split(strings.Replace(sc.HomepageURL, "//", "/", -1), "/")[3]

		//https://content.vrconk.com
		contentURL := strings.Replace("https://vrconk.com", "//", "//content.", 1)

		r, _ := resty.R().
			SetHeader("User-Agent", UserAgent).
			Get("https://content." + sc.Site + ".com/api/content/v1/videos/" + content_id)

		JsonMetadata := r.String()

		//if not valid scene...
		if gjson.Get(JsonMetadata, "status.message").String() != "Ok" {
			return
		}

		//Scene ID - back 8 of the"id" via api response
		sc.SiteID = strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.id").String()[15:])

		//Scene ID - use PlayaId for scene-id instead of "random" using id
		//		playaId := gjson.Get(JsonMetadata, "data.item.playaId").Int()
		//		sc.SiteID = strconv.Itoa(int(playaId))
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.title").String())

		// Filenames - VRCONK_Ballerina_8K_180x180_3dh.mp4
		baseName := sc.Site + "_" + strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.videoSettings.videoShortName").String()) + "_"
		filenames := []string{"8K_180x180_3dh", "6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

		for i := range filenames {
			filenames[i] = baseName + filenames[i] + ".mp4"
		}

		sc.Filenames = filenames

		// Date & Duration
		tmpDate, _ := goment.Unix(gjson.Get(JsonMetadata, "data.item.publishedAt").Int())
		sc.Released = tmpDate.Format("YYYY-MM-DD")
		tmpDuration := gjson.Get(JsonMetadata, "data.item.videoSettings.duration").Float()
		sc.Duration = int(math.Floor((tmpDuration / 60) + 0/5))

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			tmpCover := strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0]
			if tmpCover != "https://vrconk.com/wp-content/uploads/2020/03/VR-Conk-Logo.jpg" {
				sc.Covers = append(sc.Covers, tmpCover)
			}
		})

		sc.Covers = append(sc.Covers, e.ChildAttr(`section.banner picture img`, "src"))
		sc.Covers = append(sc.Covers, e.ChildAttr(`section.base-content__bg img[class="object-fit-cover base-border overflow-hidden hero-img"]`, "src"))

		// Gallery - https://content.vrconk.com/uploads/2021/08/611b4e0ca5c54351494706_XL.jpg
		gallerytmp := gjson.Get(JsonMetadata, "data.item.galleryImages.#.previews.#(sizeAlias==XL).permalink")
		for _, v := range gallerytmp.Array() {
			sc.Gallery = append(sc.Gallery, contentURL+v.Str)
		}

		// Synopsis
		sc.Synopsis = strings.TrimSpace(strings.Replace(e.ChildText(`div.video-item__description div.short-text`), `arrow_drop_up`, ``, -1))
		//sc.Synopsis = strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.description").String())

		// Tags
		tagstmp := gjson.Get(JsonMetadata, "data.item.categories.#.slug")
		for _, v := range tagstmp.Array() {
			sc.Tags = append(sc.Tags, v.Str)
		}

		// Positions - 1:"Sitting",2:"Missionary",3:"Standing",4:"Lying",5:"On the knees",6:"Close-up"
		var position string
		positions := gjson.Get(JsonMetadata, "data.item.videoTechBar.positions")
		for _, i := range positions.Array() {
			switch i.Int() {
			case 1:
				position = "sitting"
			case 2:
				position = "missionary"
			case 3:
				position = "standing"
			case 4:
				position = "lying"
			case 5:
				position = "on the knees"
			case 6:
				position = "close-up"
			}
			sc.Tags = append(sc.Tags, strings.TrimSpace(position))
		}

		// Cast
		casttmp := gjson.Get(JsonMetadata, "data.item.models.#.title")
		for _, v := range casttmp.Array() {
			sc.Cast = append(sc.Cast, strings.TrimSpace(v.Str))
		}

		out <- sc
	})

	siteCollector.OnHTML(`div.video-item-info-title a`, func(e *colly.HTMLElement) {
		url := strings.Split(e.Attr("href"), "?")[0]
		sceneURL := e.Request.AbsoluteURL(url)

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`.pagination-next a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))

		siteCollector.Visit(pageURL)
	})

	siteCollector.Visit("https://vrconk.com/videos/?sort=latest&bonus-video=1")

	// Edge-cases: Some early scenes are unlisted in both scenes and model index
	// #1-10 + 15 by FantAsia, #11-14, 19, 23 by Miss K. #22, 25 by Emi.
	// Unlisted but not added here: #86 by CumCoders (7 scenes on SLR) & some recent ones are WankzVR scenes from covid partnership.
	unlistedscenes := [19]string{"sex-with-slavic-chick-1", "only-for-your-eyes-2", "looking-for-your-cock-3",
		"finger-warm-up-4", "fun-with-sex-toy-5", "may-i-suck-it-6", "my-pleasure-in-your-hands-7", "take-me-baby-8",
		"breakfast-on-the-table-9", "united-boobs-of-desire-10", "i-change-my-lingerie-three-times-for-you-15",
		"take-care-of-the-bunny-11", "pussy-wide-open-12", "want-to-know-whats-for-dinner-13", "your-eastern-maid-14",
		"fun-with-real-vr-amateur-19", "juicy-holes-22", "rabbit-fuck-23", "amateur-chick-in-the-kitchen-25"}

	for _, scene := range unlistedscenes {
		sceneURL := "https://vrconk.com/videos/" + scene
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrconk", "VRCONK", "https://vrconk.com/s/favicon/apple-touch-icon.png", VRCONK)
}
