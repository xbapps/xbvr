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

	// RegEx Patterns
	coverRegEx := regexp.MustCompile(`background(?:-image)?\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
	durationRegEx := regexp.MustCompile(`^T(\d{0,2})H?(\d{2})M(\d{2})S$`)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(scraperID) + "-" + sc.SiteID

		// Cover
		coverURL := coverRegEx.FindStringSubmatch(strings.TrimSpace(e.ChildAttr(`.splash-screen`, "style")))[1]
		if len(coverURL) > 0 {
			sc.Covers = append(sc.Covers, coverURL)
		}

		// Gallery
		e.ForEach(`div#tabs-photos figure a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(
			e.DOM.Find(`div#tabs-about div.u-mb--four`).First().Text())

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
			if e.Attr("title") == "Fisheye" || e.Attr("title") == "360°" {
				videotype = e.Attr("title")
			}

		})

		// Extract from JSON meta data
		// NOTE: SLR only provides certain information like duration as json metadata inside a script element
		// The page code also changes often and is difficult to traverse, best to get as much as possible from metadata
		e.ForEach(`script[type="application/ld+json"]`, func(id int, e *colly.HTMLElement) {
			JsonMetadata := strings.TrimSpace(e.Text)

			// skip non video Metadata
			if gjson.Get(JsonMetadata, "@type").String() == "VideoObject" {

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
				duration := 0
				if gjson.Get(JsonMetadata, "duration").Exists() {
					tmpParts := durationRegEx.FindStringSubmatch(gjson.Get(JsonMetadata, "duration").String())
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
				resolutions := []string{"_6400p_", "_3160p_", "_2900p_", "_2880p_", "_2700p_", "_2650p_", "_2160p_", "_1920p_", "_1440p_", "_1080p_", "_original_"}
				baseName := "SLR_" + siteID + "_" + sc.Title
				switch videotype {
				case "360°": // Sadly can't determine if TB or MONO so have to add both
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MONO_360.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_TB_360.mp4")
					}
				case "Fisheye": // 200° videos named with MKX200
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX200.mp4")
					}
				default: // Assuming everything else is 180 and LR, yet to find a TB_180
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_LR_180.mp4")
					}
				}
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

func addSLRScraper(id string, name string, company string, avatarURL string) {
	suffixedName := name
	if company != "SexLikeReal" {
		suffixedName += " (SLR)"
	}
	registerScraper(id, suffixedName, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return SexLikeReal(wg, updateSite, knownScenes, out, id, name, company)
	})
}

func init() {
	addSLRScraper("slr-originals", "SLR Originals", "SexLikeReal", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png")

	addSLRScraper("ad4x", "AD4X", "AD4X", "https://ad4x.com/ypp_theme_ad4x/images/logo.png")
	addSLRScraper("amateurvr3d", "AmateurVR3D", "AmateurVR3D", "http://amateurvr3d.com/assets/images/Nx50xlogo.png.pagespeed.ic.mr8RC-ybPl.webp")
	addSLRScraper("anal-delight", "Anal Delight", "AnalDelight", "https://mcdn.vrporn.com/files/20200907184611/AnalDelight_Logo.jpg")
	addSLRScraper("bravomodelsmedia", "BravoModelsMedia", "Bravo Models", "https://mcdn.vrporn.com/files/20181015142403/ohNFa81Q_400x400.png")
	addSLRScraper("burningangelvr", "BurningAngelVR", "BurningAngelVR", "https://mcdn.vrporn.com/files/20170830191746/burningangel-icon-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("emilybloom", "EmilyBloom", "Emily Bloom", "https://theemilybloom.com/wp-content/uploads/2017/05/FlowerHeaderLogo.png")
	addSLRScraper("grannies-vr", "GranniesVR", "GranniesVR", "https://mcdn.vrporn.com/files/20180222024100/itsmorti-logo-vr-porn-studio-vrporn.com-virtual-reality.jpg")
	addSLRScraper("herfirstvr", "HerFirstVR", "HerFirstVR", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png")
	addSLRScraper("holivr", "HoliVR", "HoliVR", "https://mcdn.vrporn.com/files/20170519145416/Holi_400x400.jpg")
	addSLRScraper("istripper", "iStripper", "TotemCore Ltd", "https://www.istripper.com/favicons/istripper/apple-icon-120x120.png")
	addSLRScraper("jimmydraws", "JimmyDraws", "Jimmy Draws", "https://mcdn.vrporn.com/files/20190821145930/iLPJW6J7_400x400.png")
	addSLRScraper("justvr", "JustVR", "JustVR", "https://mcdn.vrporn.com/files/20181023121629/logo.jpg")
	addSLRScraper("jvrporn", "JVRPorn", "JVRPorn", "https://mcdn.vrporn.com/files/20170710084815/jvrporn-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("leninacrowne", "LeninaCrowne", "Terrible", "https://mcdn.vrporn.com/files/20190711135807/terrible_logo-e1562878668857_400x400_acf_cropped.jpg")
	addSLRScraper("lustreality", "LustReality", "LustReality", "https://mcdn.vrporn.com/files/20200316102952/lustreality_logo2.png")
	addSLRScraper("mmm100", "MMM100", "MMM100", "https://mmm100.com/MMM100.png")
	addSLRScraper("no2studiovr", "No2StudioVR", "No2StudioVR", "https://mcdn.vrporn.com/files/20201021145654/No2StudioVR_400x400-1.jpg")
	addSLRScraper("only3xvr", "Only3xVR", "Only3xVR", "https://mcdn.vrporn.com/files/20190821140339/only3xvr-profile-pic.jpg")
	addSLRScraper("onlytease", "OnlyTease", "OT Publishing Ltd", "https://www.onlytease.com/assets/img/favicons/ot/apple-touch-icon.png")
	addSLRScraper("pervrt", "perVRt/Terrible", "Terrible", "https://mcdn.vrporn.com/files/20181218151630/pervrt-logo.jpg")
	addSLRScraper("povcentralvr", "POVcentralVR", "POV Central", "https://mcdn.vrporn.com/files/20191125091909/POVCentralLogo.jpg")
	addSLRScraper("pvrstudio", "PVRStudio", "PVRStudio", "https://pvr.fun/uploads/2019/10/08/084230gbctdepe7kovu4hs.jpg")
	addSLRScraper("realhotvr", "RealHotVR", "RealHotVR", "https://g8iek4luc8.ent-cdn.com/templates/realhotvr/images/favicon.jpg")
	addSLRScraper("screwboxvr", "ScrewBoxVR", "ScrewBox", "https://pbs.twimg.com/profile_images/1137432770936918016/ycL3ag5c_200x200.png")
	addSLRScraper("stockingsvr", "StockingsVR", "StockingsVR", "https://mcdn.vrporn.com/files/20171107092330/stockingsvr_logo_vr_porn_studio_vrporn.com_virtual_reality1-1.png")
	addSLRScraper("stripzvr", "StripzVR", "N1ck Inc.", "https://www.stripzvr.com/wp-content/uploads/2018/09/cropped-favicon-192x192.jpg")
	addSLRScraper("tadpolexxxstudio", "TadPoleXXXStudio", "TadPoleXXXStudio", "https://mcdn.vrporn.com/files/20190928101126/tadpolexxx-logo-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("vrsexperts", "VRSexperts", "VRSexperts", "https://mcdn.vrporn.com/files/20190812141431/vrsexpertslogo2.jpg")
	addSLRScraper("vrsolos", "VRSolos", "VRSolos", "https://mcdn.vrporn.com/files/20191226092954/VRSolos_Logo.jpg")
	addSLRScraper("vrextasy", "VReXtasy", "VReXtasy", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png")
	addSLRScraper("vredging", "VRedging", "VRedging", "https://mcdn.vrporn.com/files/20200630081500/VRedging_LOGO_v1-400x400.jpg")
	addSLRScraper("vrfirsttimer", "VRFirstTimer", "VRFirstTimer", "https://mcdn.vrporn.com/files/20200511115233/VRFirstTimers_Logo.jpg")
	addSLRScraper("virtualpee", "VirtualPee", "VirtualPee", "https://mcdn.vrporn.com/files/20180317104121/virtualpeeop-square-banner.jpg")
	addSLRScraper("virtualxporn", "VirtualXPorn", "VirtualXPorn", "https://www.virtualxporn.com/tour/custom_assets/favicons/android-chrome-192x192.png")
	addSLRScraper("xvirtual", "xVirtual", "xVirtual", "https://mcdn.vrporn.com/files/20181116133947/xvirtuallogo.jpg")
	addSLRScraper("sweetlonglips", "Sweetlonglips", "Sweetlonglips", "https://mcdn.vrporn.com/files/20200117105304/SLLVRlogo.png")
	addSLRScraper("pornbcn", "Pornbcn", "Pornbcn", "https://mcdn.vrporn.com/files/20190923110340/CHANNEL-LOGO-2.jpg")
	addSLRScraper("lezvr", "LezVR", "LezVR", "https://mcdn.vrporn.com/files/20170813184453/lezvr-icon-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("net69vr", "Net69VR", "Net69VR", "https://mcdn.vrporn.com/files/20171113233505/net69vr-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("VRpussyVision", "VRpussyVision", "VRpussyVision", "https://mcdn.vrporn.com/files/20180313160830/vrpussyvision-square-banner.png")
	addSLRScraper("altporn4u-vr", "AltPorn4uVR", "AltPorn4uVR", "https://www.altporn4u.com/favicon.ico")

}
