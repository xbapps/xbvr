package scrape

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func SexLikeReal(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, company string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.sexlikereal.com")
	siteCollector := createCollector("www.sexlikereal.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		re := regexp.MustCompile(`background(?:-image)?\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
		coverURL := re.FindStringSubmatch(strings.TrimSpace(e.ChildAttr(`.splash-screen`, "style")))[1]
		if len(coverURL) > 0 {
			sc.Covers = append(sc.Covers, coverURL)
		}

		// Gallery
		e.ForEach(`div#tabs-photos figure a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis
		e.ForEach(`div#tabs-about div.u-mb--four`, func(id int, e *colly.HTMLElement) {
			if !strings.Contains(e.Text, "Released:") {
				sc.Synopsis = strings.TrimSpace(e.Text)
			}
		})

		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"vr porn": true,
			"3D":      true, // Everything gets tagged 3D on SLR, even mono 360
		}

		// Tags
		// Note: known issue with SLR, they use a lot of combined tags like "cheerleader / college / school"
		// ...a lot of these are shared with RealJamVR which uses the same tags though.
		// Could split by / but would run into issues with "f/f/m" and "shorts / skirts"
		var videotype string
		e.ForEach(`ul.c-meta--scene-tags li a`, func(id int, e *colly.HTMLElement) {
			if !skiptags[e.Attr("title")] {
				sc.Tags = append(sc.Tags, e.Attr("title"))
			}

			// To determine filenames
			if e.Attr("title") == "180°" || e.Attr("title") == "360°" {
				videotype = e.Attr("title")
			}

		})

		// Extract from JSON meta data
		// NOTE: SLR only provides certain information like duration as json metadata inside a script element
		// The page code also changes often and is difficult to traverse, best to get as much as possible from metadata
		e.ForEach(`script[type="application/ld+json"]`, func(id int, e *colly.HTMLElement) {
			JsonMetadata := strings.TrimSpace(e.Text)

			// Title
			if gjson.Get(JsonMetadata, "name").Exists() {
				sc.Title = strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, "name").String()))
			}

			// Date
			if gjson.Get(JsonMetadata, "datePublished").Exists() {
				sc.Released = gjson.Get(JsonMetadata, "datePublished").String()
			}

			// Cast
			actornames := gjson.Get(JsonMetadata, "actor.#.name")
			for _, name := range actornames.Array() {
				sc.Cast = append(sc.Cast, strings.TrimSpace(html.UnescapeString(name.String())))
			}

			// Duration
			// NOTE: SLR fails to include hours (1h55m30s shows up as T55M30S)
			// ...but this is ready for the format of T01H55M30S should SLR fix that
			reGetDuration := regexp.MustCompile(`^T(\d{0,2})H?(\d{2})M(\d{2})S$`)
			duration := 0
			if gjson.Get(JsonMetadata, "duration").Exists() {
				tmpParts := reGetDuration.FindStringSubmatch(gjson.Get(JsonMetadata, "duration").String())
				if len(tmpParts[1]) > 0 {
					if h, err := strconv.Atoi(tmpParts[1]); err == nil {
						hrs := h
						if m, err := strconv.Atoi(tmpParts[2]); err == nil {
							mins := m
							duration = (hrs * 60) + mins
						}
					}
				} else {
					if m, err := strconv.Atoi(tmpParts[2]); err == nil {
						duration = m
					}
				}
				sc.Duration = duration
			}

			// Filenames
			// Only shown for logged in users so need to generate them
			// Format: SLR_siteID_Title_<Resolutions>_SceneID_<LR/TB>_<180/360>.mp4
			resolutions := []string{"_6400p_", "_2880p_", "_2700p_", "_1440p_", "_1080p_", "_original_"}
			baseName := "SLR_" + siteID + "_" + sc.Title
			if videotype == "360°" { // Sadly can't determine if TB or MONO so have to add both
				filenames := make([]string, 0, 2*len(resolutions))
				for i := range resolutions {
					filenames = append(filenames, baseName+resolutions[i]+sc.SiteID+"_MONO_360.mp4")
					filenames = append(filenames, baseName+resolutions[i]+sc.SiteID+"_TB_360.mp4")
					sc.Filenames = filenames
				}
			} else { // Assuming everything else is 180 and LR, yet to find a TB_180
				filenames := make([]string, 0, len(resolutions))
				for i := range resolutions {
					filenames = append(filenames, baseName+resolutions[i]+sc.SiteID+"_LR_180.mp4")
				}
				sc.Filenames = filenames
			}

		})

		out <- sc
	})

	siteCollector.OnHTML(`div.c-pagination ul li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.c-grid--scenes article a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		if strings.Contains(sceneURL, "scene") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				sceneCollector.Visit(sceneURL)
			}
		}
	})

	siteCollector.Visit("https://www.sexlikereal.com/studios/" + scraperID + "?sort=most_recent")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

// SLR Originals - SexLikeReal own productions
func SLROriginals(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "slr-originals", "SLR Originals", "SexLikeReal")
}

// iStripper - Has a site for 2D desktop app, but doesn't even mention they do VR scenes: https://www.istripper.com/
func iStripper(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "istripper", "iStripper", "TotemCore Ltd")
}

// EmilyBloom - does have vertical covers on her site but no scene info to scrape: https://theemilybloom.com/virtual-reality/
func EmilyBloom(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "emilybloom", "EmilyBloom", "Emily Bloom")
}

// VRSexperts - does have large covers on their blog but they appear very delayed: http://www.vrsexperts.com/
func VRSexperts(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "vrsexperts", "VRSexperts", "VRSexperts")
}

// VReXtasy - Can't find a site/twitter at all
func VReXtasy(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "vrextasy", "VReXtasy", "VReXtasy")
}

// VRSolos - https://twitter.com/VRsolos/
func VRSolos(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "vrsolos", "VRSolos", "VRSolos")
}

// Jimmy Draws - https://twitter.com/ukpornmaker
func JimmyDraws(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "jimmydraws", "JimmyDraws", "Jimmy Draws")
}

// POVcentralVR - Has a site with mixed 2D/VR content, doesn't seem very scrapeable: http://povcentral.com/home.html
// Does have a blog for VR scenes but no useful covers: http://blog.povcentralmembers.com/category/3d/
func POVcentralVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "povcentralvr", "POVcentralVR", "POV Central")
}

// OnlyTease - Has a site for their 2D scenes, only started doing VR since Oct 2019: https://www.onlytease.com/
func OnlyTease(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "onlytease", "OnlyTease", "OT Publishing Ltd")
}

// perVRt/Terrible - Likely to change to Terrible brand, is working on their own website here: http://terrible.porn/
// Publishes on SLR as perVRt, includes brands: Juggs, Babygirl, Sappho
// https://twitter.com/terribledotporn & https://twitter.com/perVRtPORN
func perVRt(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "pervrt", "perVRt", "Terrible")
}

// LeninaCrowne - Wife of https://twitter.com/DickTerrible from the perVRt/Terrible Studio.
func LeninaCrowne(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "leninacrowne", "LeninaCrowne", "Terrible")
}

// StripzVR.com doesn't have pagination or a model/scene index that's scrapeable
func StripzVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "stripzvr", "StripzVR", "N1ck Inc.")
}

// RealHotVR.com doesn't have complete scene index, pagination stops after two pages
func RealHotVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return SexLikeReal(wg, updateSite, knownScenes, out, "realhotvr", "RealHotVR", "RealHotVR")
}

func init() {
	registerScraper("slr-originals", "SLR Originals", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png", SLROriginals)
	registerScraper("istripper", "iStripper (SLR)", "https://www.istripper.com/favicons/istripper/apple-icon-120x120.png", iStripper)
	registerScraper("emilybloom", "EmilyBloom (SLR)", "https://theemilybloom.com/wp-content/uploads/2017/05/FlowerHeaderLogo.png", EmilyBloom)
	registerScraper("vrsexperts", "VRSexperts (SLR)", "https://mcdn.vrporn.com/files/20190812141431/vrsexpertslogo2.jpg", VRSexperts)
	registerScraper("vrextasy", "VReXtasy (SLR)", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png", VReXtasy)
	registerScraper("vrsolos", "VRSolos (SLR)", "https://mcdn.vrporn.com/files/20191226092954/VRSolos_Logo.jpg", VRSolos)
	registerScraper("jimmydraws", "JimmyDraws (SLR)", "https://mcdn.vrporn.com/files/20190821145930/iLPJW6J7_400x400.png", JimmyDraws)
	registerScraper("povcentralvr", "POVcentralVR (SLR)", "https://mcdn.vrporn.com/files/20191125091909/POVCentralLogo.jpg", POVcentralVR)
	registerScraper("onlytease", "OnlyTease (SLR)", "https://www.onlytease.com/assets/img/favicons/ot/apple-touch-icon.png", OnlyTease)
	registerScraper("pervrt", "perVRt/Terrible (SLR)", "https://mcdn.vrporn.com/files/20181218151630/pervrt-logo.jpg", perVRt)
	registerScraper("leninacrowne", "LeninaCrowne (SLR)", "https://mcdn.vrporn.com/files/20190711135807/terrible_logo-e1562878668857_400x400_acf_cropped.jpg", LeninaCrowne)
	registerScraper("stripzvr", "StripzVR (SLR)", "https://www.stripzvr.com/wp-content/uploads/2018/09/cropped-favicon-192x192.jpg", StripzVR)
	registerScraper("realhotvr", "RealHotVR (SLR)", "https://g8iek4luc8.ent-cdn.com/templates/realhotvr/images/favicon.jpg", RealHotVR)
}
