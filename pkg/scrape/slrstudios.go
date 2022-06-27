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

func SexLikeReal(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, company string, kink string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.sexlikereal.com")
	siteCollector := createCollector("www.sexlikereal.com")

	// RegEx Patterns
	coverRegEx := regexp.MustCompile(`background(?:-image)?\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
	durationRegExForSceneCard := regexp.MustCompile(`^(?:(\d{2}):)?(\d{2}):(\d{2})$`)
	durationRegExForScenePage := regexp.MustCompile(`^T(\d{0,2})H?(\d{2})M(\d{2})S$`)
	filenameRegEx := regexp.MustCompile(`[:?]|( & )|( \\u0026 )`)

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
		coverURL := e.ChildAttr(`.splash-screen > img`, "src")
		if len(coverURL) > 0 {
			sc.Covers = append(sc.Covers, coverURL)
		} else {
			m := coverRegEx.FindStringSubmatch(strings.TrimSpace(e.ChildAttr(`.splash-screen`, "style")))
			if len(m) > 0 && len(m[1]) > 0 {
				sc.Covers = append(sc.Covers, m[1])
			}
		}

		// Gallery
		e.ForEach(`meta[name^="twitter:image"]`, func(id int, e *colly.HTMLElement) {
			if e.Attr("name") != "twitter:image" { // we need image1, image2...
				sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("content")))
			}
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(
			e.DOM.Find(`div#tabs-about > div > div.u-px--four > div.u-wh`).First().Text())

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
		var FB360 string
		e.ForEach(`ul.c-meta--scene-tags li a`, func(id int, e *colly.HTMLElement) {
			if !skiptags[e.Attr("title")] {
				sc.Tags = append(sc.Tags, e.Attr("title"))
			}

			// To determine filenames
			if e.Attr("title") == "Fisheye" || e.Attr("title") == "360°" {
				videotype = e.Attr("title")
			}
			if e.Attr("title") == "Spatial audio" {
				FB360 = "_FB360.MKV"
			}

		})

		// Duration
		sc.Duration = e.Request.Ctx.GetAny("duration").(int)

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
				// NOTE: We should already have the duration from the scene list, but if we don't (happens for at least
				// one scene where SLR fails to include it), we try to get it from here
				// We don't always get it from here, because SLR fails to include hours (1h55m30s shows up as T55M30S)
				// ...but this is ready for the format of T01H55M30S should SLR fix that
				if sc.Duration == 0 {
					duration := 0
					if gjson.Get(JsonMetadata, "duration").Exists() {
						tmpParts := durationRegExForScenePage.FindStringSubmatch(gjson.Get(JsonMetadata, "duration").String())
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
				}

				// Filenames
				// Only shown for logged in users so need to generate them
				// Format: SLR_siteID_Title_<Resolutions>_SceneID_<LR/TB>_<180/360>.mp4
				resolutions := []string{"_6400p_", "_3840p_", "_3360p_", "_3160p_", "_3072p_", "_2900p_", "_2880p_", "_2700p_", "_2650p_", "_2160p_", "_1920p_", "_1440p_", "_1080p_", "_original_"}
				baseName := "SLR_" + siteID + "_" + filenameRegEx.ReplaceAllString(sc.Title, "_")
				switch videotype {
				case "360°": // Sadly can't determine if TB or MONO so have to add both
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MONO_360.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_TB_360.mp4")
					}
				case "Fisheye": // 200° videos named with MKX200
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX200.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX220.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_VRCA220.mp4")
					}
				default: // Assuming everything else is 180 and LR, yet to find a TB_180
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_LR_180.mp4")
					}
				}
				if FB360 != "" {
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_LR_180"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MKX200"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MKX220"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_VRCA220"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MONO_360"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_TB_360"+FB360)
				}
			}

		})

		out <- sc
	})

	siteCollector.OnHTML(`div.c-pagination ul li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.c-grid--scenes article`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a", "href"))
		if strings.Contains(sceneURL, "scene") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				durationText := e.ChildText("div.c-grid-ratio-bottom.u-z--two")
				m := durationRegExForSceneCard.FindStringSubmatch(durationText)
				duration := 0
				if len(m) == 4 {
					hours, _ := strconv.Atoi("0" + m[1])
					minutes, _ := strconv.Atoi(m[2])
					duration = hours*60 + minutes
				}
				ctx := colly.NewContext()
				ctx.Put("duration", duration)
				sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
			}
		}
	})

	// Fix for duplicate entries removing -slr suffix to match web studio id
	scraperIdUrlPath := scraperID
	if strings.HasSuffix(scraperID, "-slr") {
		scraperIdUrlPath = strings.Replace(scraperID, "-slr", "", 1)
	}

	switch kink {
	case "g":
		siteCollector.Visit("https://www.sexlikereal.com/gay/studios/" + scraperIdUrlPath + "?sort=most_recent")
	case "t":
		siteCollector.Visit("https://www.sexlikereal.com/trans/studios/" + scraperIdUrlPath + "?sort=most_recent")
	default:
		siteCollector.Visit("https://www.sexlikereal.com/studios/" + scraperIdUrlPath + "?sort=most_recent")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addSLRScraper(id string, name string, company string, avatarURL string, kink string) {
	suffixedName := name
	if company != "SexLikeReal" {
		suffixedName += " (SLR)"
	}
	registerScraper(id, suffixedName, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return SexLikeReal(wg, updateSite, knownScenes, out, id, name, company, kink)
	})
}

func init() {
	// id suffix "-slr" means there exists a dedicated grabber. this string is added to avoid id collision

	// Hetero
	addSLRScraper("33", "+33", "+33", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/343/logo_crop_1628188580.png", "")
	addSLRScraper("100vr", "100％VR", "100％VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/156/logo_crop_1606868336.png", "")
	addSLRScraper("3dvr", "3DV&R", "3DV&R", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/205/logo_crop_1606869147.png", "")
	addSLRScraper("4k-vr", "4KVR", "4KVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/204/logo_crop_1606868871.png", "")
	addSLRScraper("ad4x", "AD4X", "AD4X", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/230/logo_crop_1606755231.png", "")
	addSLRScraper("ainovdo", "Ainovdo", "Ainovdo", "", "")
	addSLRScraper("alternativevrgirls", "AlternativeVRGirls", "AlternativeVRGirls", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/301/logo_crop_1617013880.png", "")
	addSLRScraper("amateurcouplesvr", "AmateurCouplesVR", "AmateurCouplesVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/272/logo_crop_1648490342.png", "")
	addSLRScraper("amateurvr3d", "AmateurVR3D", "AmateurVR3D", "", "")
	addSLRScraper("anal-delight", "Anal Delight", "AnalDelight", "", "")
	addSLRScraper("aromaplanning", "AromaPlanning", "AromaPlanning", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/192/logo_crop_1606868399.png", "")
	addSLRScraper("art-vr", "Art VR", "ArtVR", "", "")
	addSLRScraper("asari-vr", "Asari VR", "AsariVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/148/logo_crop_1606868320.png", "")
	addSLRScraper("mongercash", "AsiansexdiaryVR", "AsiansexdiaryVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/219/logo_crop_1628500033.png", "")
	addSLRScraper("avervr", "AverVR", "AverVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/369/logo_crop_1642559026.png", "")
	addSLRScraper("baberoticavr-slr", "BaberoticaVR", "BaberoticaVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/247/logo_crop_1609889316.png", "") // Dedicated scraper exists
	addSLRScraper("babykxtten", "Babykxtten", "Babykxtten", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/321/logo_crop_1624979737.png", "")
	addSLRScraper("bakunouvr", "BakunouVR", "BakunouVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/339/logo_crop_1627446437.png", "")
	addSLRScraper("blush-erotica", "Blush Erotica", "BlushErotica", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/385/logo_crop_1649724830.png", "")
	addSLRScraper("bravomodelsmedia", "BravoModelsMedia", "BravoModelsMedia", "", "")
	addSLRScraper("bullitt", "Bullitt", "Bullitt", "", "")
	addSLRScraper("burningangelvr", "BurningAngelVR", "BurningAngelVR", "", "")
	addSLRScraper("carivr", "Caribbeancom", "Caribbeancom", "", "")
	addSLRScraper("casanova", "CasanovA", "CasanovA", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/184/logo_crop_1606868350.png", "")
	addSLRScraper("chinchinvr", "ChinChinVR", "ChinChinVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/188/logo_crop_1606868368.png", "")
	addSLRScraper("cosmoplanetsvr", "CosmoPlanetsVR", "CosmoPlanetsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/252/logo_crop_1606869377.png", "")
	addSLRScraper("cosplay-with-me", "Cosplay With Me", "CosplayWithMe", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/274/logo_crop_1609165310.png", "")
	addSLRScraper("covert-japan", "CovertJapan", "CovertJapan", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/221/logo_crop_1607605022.png", "")
	addSLRScraper("cumcoders", "Cumcoders", "Cumcoders", "", "")
	addSLRScraper("cumtoon", "Cumtoon", "Cumtoon", "", "")
	addSLRScraper("czechvrcasting-slr", "CzechVRCasting", "CzechVRCasting", "", "") // Dedicated scraper exists
	addSLRScraper("czechvrfetish-slr", "CzechVRFetish", "CzechVRFetish", "", "")    // Dedicated scraper exists
	addSLRScraper("dandy", "DANDY", "DANDY", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/355/logo_crop_1634611924.png", "")
	addSLRScraper("ddfnetworkvr-slr", "DDFNetworkVR", "DDFNetworkVR", "", "") // Dedicated scraper exists
	addSLRScraper("deepinsex", "Deepinsex", "Deepinsex", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/266/logo_crop_1610126420.png", "")
	addSLRScraper("deviantsvr", "DeviantsVR", "DeviantsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/351/logo_crop_1638539790.png", "")
	addSLRScraper("dezyred", "Dezyred", "Dezyred", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/356/logo_crop_1636053912.png", "")
	addSLRScraper("dreamticketvr", "DreamTicketVR", "DreamTicketVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/119/logo_crop_1606868303.png", "")
	addSLRScraper("dreamcam", "Dreamcam", "Dreamcam", "", "")
	addSLRScraper("dynamiceyes", "DynamicEyes", "DynamicEyes", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/263/logo_crop_1606870478.png", "")
	addSLRScraper("ebony-vr-solos", "EBONY VR SOLOS", "EBONYVRSOLOS", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/387/logo_crop_1651471829.png", "")
	addSLRScraper("ebonyvr", "EbonyVR", "EbonyVR", "", "")
	addSLRScraper("ellielouisevr", "EllieLouiseVR", "EllieLouiseVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/265/logo_crop_1607603680.png", "")
	addSLRScraper("emilybloom", "EmilyBloom", "EmilyBloom", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/42/logo_crop_1608166932.png", "")
	addSLRScraper("erikalust", "ErikaLust", "ErikaLust", "", "")
	addSLRScraper("erotimevr", "EroTimeVR", "EroTimeVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/208/logo_crop_1606869186.png", "")
	addSLRScraper("erotic-sinners", "Erotic Sinners", "EroticSinners", "", "")
	addSLRScraper("evileyevr-slr", "EvilEyeVR", "EvilEyeVR", "", "") // Dedicated scraper exists
	addSLRScraper("fantastica", "FANTASTICA", "FANTASTICA", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/197/logo_crop_1606868466.png", "")
	addSLRScraper("fatp", "FATP", "FATP", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/382/logo_crop_1648196512.png", "")
	addSLRScraper("ffstockings", "FFStockings", "FFStockings", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/63/logo_crop_1607992231.png", "")
	addSLRScraper("fsknightsvisual", "FS.KnightsVisual", "FS.KnightsVisual", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/341/logo_crop_1627446822.png", "")
	addSLRScraper("fakingsvr", "FaKingsVR", "FaKingsVR", "", "")
	addSLRScraper("fantasista-vr", "Fantasista VR", "FantasistaVR", "", "")
	addSLRScraper("fapp3d", "Fapp3D", "Fapp3D", "", "")
	addSLRScraper("fleshy-body-vr", "Fleshy BODY VR", "FleshyBODYVR", "", "")
	addSLRScraper("fuckpassvr", "FuckPassVR", "FuckPassVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/352/logo_crop_1635153994.png", "")
	addSLRScraper("fucktruck-vr", "FuckTruck VR", "FuckTruckVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/398/logo_crop_1654289252.png", "")
	addSLRScraper("glosstightsglamourvr", "GlossTightsGlamourVR", "GlossTightsGlamourVR", "", "")
	addSLRScraper("gokunyu", "Gokunyu", "Gokunyu", "", "")
	addSLRScraper("grannies-vr", "GranniesVR", "GranniesVR", "", "")
	addSLRScraper("hot-babes-vr", "HOT BABES VR", "HOTBABESVR", "", "")
	addSLRScraper("heathering", "Heathering", "Heathering", "", "")
	addSLRScraper("hentaivr", "HentaiVR", "HentaiVR", "", "")
	addSLRScraper("herfirstvr", "HerFirstVR", "HerFirstVR", "", "")
	addSLRScraper("holivr", "HoliVR", "HoliVR", "", "")
	addSLRScraper("hologirlsvr-slr", "HoloGirlsVR", "HoloGirlsVR", "", "") // Dedicated scraper exists
	addSLRScraper("hookfer", "Hookfer", "Hookfer", "", "")
	addSLRScraper("hotentertainment", "HotEntertainment", "HotEntertainment", "", "")
	addSLRScraper("iloveitshiny", "ILoveItShiny", "ILoveItShiny", "", "")
	addSLRScraper("ichimonkai", "IchiMonKai", "IchiMonKai", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/318/logo_crop_1627446974.png", "")
	addSLRScraper("intimovr", "IntimoVR", "IntimoVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/264/logo_crop_1608437997.png", "")
	addSLRScraper("jvrporn", "JVRPorn", "JVRPorn", "", "")
	addSLRScraper("jackandjillvr", "JackandJillVR", "JackandJillVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/367/logo_crop_1645997567.png", "")
	addSLRScraper("jimmydraws", "JimmyDraws", "JimmyDraws", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/173/logo_crop_1607603641.png", "")
	addSLRScraper("justvr", "JustVR", "JustVR", "", "")
	addSLRScraper("kmpvr", "KMPVR", "KMPVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/209/logo_crop_1606869200.png", "")
	addSLRScraper("kinkygirlsberlin", "KinkyGirlsBerlin", "KinkyGirlsBerlin", "", "")
	addSLRScraper("koalavr", "KoalaVR", "KoalaVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/189/logo_crop_1606868385.png", "")
	addSLRScraper("leninacrowne", "LeninaCrowne", "LeninaCrowne", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/223/logo_crop_1607550578.png", "")
	addSLRScraper("lethalhardcorevr-slr", "LethalHardcoreVR", "LethalHardcoreVR", "", "") // Dedicated scraper exists
	addSLRScraper("lezvr", "LezVR", "LezVR", "", "")
	addSLRScraper("libertinevr", "LibertineVR", "LibertineVR", "", "")
	addSLRScraper("lightsouthern", "Lightsouthern", "Lightsouthern", "", "")
	addSLRScraper("littlecapricedreamsvr", "LittleCapriceVR", "LittleCapriceVR", "", "") // Dedicated scraper exists, no id collision
	addSLRScraper("loveherfeetvr", "LoveHerFeetVR", "LoveHerFeetVR", "", "")
	addSLRScraper("lustreality", "LustReality", "LustReality", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/233/logo_crop_1611129820.png", "")
	addSLRScraper("lustyvr", "LustyVR", "LustyVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/371/logo_crop_1644057581.png", "")
	addSLRScraper("max-a", "MAX-A", "MAX-A", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/198/logo_crop_1606868853.png", "")
	addSLRScraper("maxing", "MAXING", "MAXING", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/345/logo_crop_1633940793.png", "")
	addSLRScraper("milfvr-slr", "MILFVR", "MILFVR", "", "") // Dedicated scraper exists
	addSLRScraper("mmm100", "MMM100", "MMM100", "", "")
	addSLRScraper("mondelde-vr", "MONDELDE VR", "MONDELDEVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/359/logo_crop_1636078868.png", "")
	addSLRScraper("mugon-vr", "MUGON VR", "MUGONVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/344/logo_crop_1634613016.png", "")
	addSLRScraper("majamagic", "MajaMagic", "MajaMagic", "", "")
	addSLRScraper("markohs", "Markohs", "Markohs", "", "")
	addSLRScraper("marrionvr", "MarrionVR", "MarrionVR", "", "")
	addSLRScraper("maturesvr", "MaturesVR", "MaturesVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/314/logo_crop_1620737406.png", "")
	addSLRScraper("mayashandjobsvr", "MayasHandjobsVR", "MayasHandjobsVR", "", "")
	addSLRScraper("meangirlsvr", "MeanGirlsVR", "MeanGirlsVR", "", "")
	addSLRScraper("mike-and-kate", "Mike and Kate", "MikeandKate", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/309/logo_crop_1620307648.png", "")
	addSLRScraper("miluvr", "MiluVR", "MiluVR", "", "")
	addSLRScraper("mousouvr", "MousouVR", "MousouVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/317/logo_crop_1623114497.png", "")
	addSLRScraper("mrmichiru", "Mr.Michiru", "Mr.Michiru", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/307/logo_crop_1621216369.png", "")
	addSLRScraper("mugur-porn-vr", "Mugur Porn VR", "MugurPornVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/375/logo_crop_1649424076.png", "")
	addSLRScraper("munakuso", "MunaKuso", "MunaKuso", "", "")
	addSLRScraper("muscleworshipvr", "MuscleWorshipVR", "MuscleWorshipVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/347/logo_crop_1633378952.png", "")
	addSLRScraper("natural-high", "NATURAL HIGH", "NATURALHIGH", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/331/logo_crop_1623047952.png", "")
	addSLRScraper("naughtyfreyavr", "NaughtyFreyaVR", "NaughtyFreyaVR", "", "")
	addSLRScraper("net69vr", "Net69VR", "Net69VR", "", "")
	addSLRScraper("never-lonely-vr", "Never Lonely VR", "NeverLonelyVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/353/logo_crop_1638474774.png", "")
	addSLRScraper("no2studiovr", "No2StudioVR", "No2StudioVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/257/logo_crop_1607614044.png", "")
	addSLRScraper("noir", "Noir", "Noir", "", "")
	addSLRScraper("nylon-xtreme", "NylonXtreme", "NylonXtreme", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/227/logo_crop_1641243932.png", "")
	addSLRScraper("office-ks-vr", "OFFICE K’S VR", "OFFICEK’SVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/248/logo_crop_1606869252.png", "")
	addSLRScraper("olipsvr", "OlipsVR", "OlipsVR", "", "")
	addSLRScraper("only3xvr", "Only3xVR", "Only3xVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/183/logo_crop_1609761306.png", "")
	addSLRScraper("onlytease", "OnlyTease", "OnlyTease", "", "")
	addSLRScraper("p-box-vr", "P-BOX VR", "P-BOXVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/250/logo_crop_1606869346.png", "")
	addSLRScraper("pip-vr", "PIP VR", "PIPVR", "", "")
	addSLRScraper("povcentralvr", "POVcentralVR", "POVcentralVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/193/logo_crop_1618115816.png", "")
	addSLRScraper("prestige", "PRESTIGE", "PRESTIGE", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/302/logo_crop_1621216541.png", "")
	addSLRScraper("ps-porn", "PS-Porn", "PS-Porn", "", "")
	addSLRScraper("pvrstudio", "PVRStudio", "PVRStudio", "", "")
	addSLRScraper("parellalwordle", "ParellalWordle", "ParellalWordle", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/196/logo_crop_1606868450.png", "")
	addSLRScraper("peeping-thom", "Peeping Thom", "PeepingThom", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/303/logo_crop_1619656190.png", "")
	addSLRScraper("pegasproductions", "PegasProductions", "PegasProductions", "", "")
	addSLRScraper("petersmax", "PetersMAX", "PetersMAX", "", "")
	addSLRScraper("petersprimovr", "PetersPrimoVR", "PetersPrimoVR", "", "")
	addSLRScraper("plushiesvr", "PlushiesVR", "PlushiesVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/122/logo_crop_1615877579.png", "")
	addSLRScraper("pornbcn", "Pornbcn", "Pornbcn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/163/logo_crop_1621982426.png", "")
	addSLRScraper("pornodanvr", "PornoDanVR", "PornoDanVR", "", "")
	addSLRScraper("propertysexvr", "PropertySexVR", "PropertySexVR", "", "")
	addSLRScraper("radical", "RADICAL", "RADICAL", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/275/logo_crop_1610511622.png", "")
	addSLRScraper("rocket", "ROCKET", "ROCKET", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/357/logo_crop_1636689676.png", "")
	addSLRScraper("realhotvr", "RealHotVR", "RealHotVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/160/logo_crop_1624292036.png", "")
	addSLRScraper("realjamvr-slr", "RealJamVR", "RealJamVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/53/logo_crop_1606820837.png", "")             // Dedicated scraper exists
	addSLRScraper("realitylovers-slr", "RealityLovers", "RealityLovers", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/27/logo_crop_1606823905.png", "") // Dedicated scraper exists
	addSLRScraper("red-star-berlin-vr", "Red Star Berlin VR", "RedStarBerlinVR", "", "")
	addSLRScraper("reevr", "ReeVR", "ReeVR", "", "")
	addSLRScraper("rezowoods", "Rezowoods", "Rezowoods", "", "")
	addSLRScraper("salome-prologue", "SALOME Prologue", "SALOMEPrologue", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/251/logo_crop_1606869361.png", "")
	addSLRScraper("scoop-vr", "SCOOP VR", "SCOOPVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/262/logo_crop_1606869393.png", "")
	addSLRScraper("slr-labs", "SLR Labs", "SLRLabs", "", "")
	addSLRScraper("slr-originals", "SLR Originals", "SLROriginals", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/224/logo_crop_1606319897.png", "")
	addSLRScraper("slr-originals-bts", "SLR Originals BTS", "SLROriginalsBTS", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/340/logo_crop_1626700476.png", "")
	addSLRScraper("screwboxvr", "ScrewBoxVR", "ScrewBoxVR", "", "")
	addSLRScraper("seducevr", "SeduceVR", "SeduceVR", "", "")
	addSLRScraper("sensesvrporn", "SensesVRporn", "SensesVRporn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/281/logo_crop_1628150671.png", "")
	addSLRScraper("sexbabesvr-slr", "SexBabesVR", "SexBabesVR", "", "") // Dedicated scraper exists
	addSLRScraper("sheer-experience", "SheerExperience", "SheerExperience", "", "")
	addSLRScraper("sinsdealers", "SinsDealers", "SinsDealers", "", "")
	addSLRScraper("sinsvr-slr", "SinsVR", "SinsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/112/logo_crop_1643894413.png", "") // Dedicated scraper exists
	addSLRScraper("skinrays", "SkinRays", "SkinRays", "", "")
	addSLRScraper("sodcreate", "SodCreate", "SodCreate", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/216/logo_crop_1606869217.png", "")
	addSLRScraper("splinevr", "SplineVR", "SplineVR", "", "")
	addSLRScraper("squeeze-vr", "Squeeze VR", "SqueezeVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/285/logo_crop_1625588555.png", "")
	addSLRScraper("stasyqvr-slr", "StasyQVR", "StasyQVR", "", "") // Dedicated scraper exists
	addSLRScraper("stockingsvr", "StockingsVR", "StockingsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/47/logo_crop_1607992206.png", "")
	addSLRScraper("strictlyglamourvr", "StrictlyGlamourVR", "StrictlyGlamourVR", "", "")
	addSLRScraper("stripzvr", "StripzVR", "StripzVR", "", "")
	addSLRScraper("suckmevr", "SuckMeVR", "SuckMeVR", "", "")
	addSLRScraper("swallowbay", "Swallowbay", "Swallowbay", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/298/logo_crop_1629722414.png", "")
	addSLRScraper("sweetlonglips", "Sweetlonglips", "Sweetlonglips", "", "")
	addSLRScraper("tmavr", "TMAVR", "TMAVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/207/logo_crop_1606869169.png", "")
	addSLRScraper("taboo-vr-porn", "Taboo VR Porn", "TabooVRPorn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/354/logo_crop_1643894389.png", "")
	addSLRScraper("taboovr", "TabooVR", "TabooVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/346/logo_crop_1637164502.png", "")
	addSLRScraper("tadpolexxxstudio", "TadPoleXXXStudio", "TadPoleXXXStudio", "", "")
	addSLRScraper("teppanvr", "TeppanVR", "TeppanVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/194/logo_crop_1606868415.png", "")
	addSLRScraper("thelockedcockchronicles", "TheLockedCockChronicles", "TheLockedCockChronicles", "", "")
	addSLRScraper("thevirtualpornwebsite", "TheVirtualPornWebsite", "TheVirtualPornWebsite", "", "")
	addSLRScraper("tmwvrnet-slr", "TmwVRNet", "TmwVRNet", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/26/logo_crop_1623330575.png", "") // Dedicated scraper exists
	addSLRScraper("unfinished-vr", "UnfinishedVR", "UnfinishedVR", "", "")
	addSLRScraper("v1vr", "V1VR", "V1VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/195/logo_crop_1606868432.png", "")
	addSLRScraper("vr-bangers", "VR Bangers", "VRBangers", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/21/logo_crop_1606744484.png", "") // Dedicated scraper exists, no id collision
	addSLRScraper("vr-japanese-pornstars-stay-home-routines", "VR Japanese Pornstars stay home routines", "VRJapanesePornstarsstayhomeroutines", "", "")
	addSLRScraper("vr-japanese-idols-party", "VR Japanese idols Party", "VRJapaneseidolsParty", "", "")
	addSLRScraper("vr-pornnow", "VR Pornnow", "VRPornnow", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/378/logo_crop_1647344034.png", "")
	addSLRScraper("vr3000-slr", "VR3000", "VR3000", "", "")                                                                                                      // Dedicated scraper exists
	addSLRScraper("vrallure-slr", "VRAllure", "VRAllure", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/213/logo_crop_1606755181.png", "") // Dedicated scraper exists
	addSLRScraper("vranimeted", "VRAnimeTed", "VRAnimeTed", "", "")
	addSLRScraper("vrclubz-slr", "VRClubz", "VRClubz", "", "")                                                                                            // Dedicated scraper exists
	addSLRScraper("vrconk-slr", "VRConk", "VRConk", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/96/logo_crop_1636655397.png", "") // Dedicated scraper exists
	addSLRScraper("vr-fan-service", "VRFanService", "VRFanService", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/153/logo_crop_1619422412.png", "")
	addSLRScraper("vrfirsttimer", "VRFirsttimer", "VRFirsttimer", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/236/logo_crop_1607745084.png", "")
	addSLRScraper("vrfootfetish", "VRFootFetish", "VRFootFetish", "", "")
	addSLRScraper("vrhush-slr", "VRHush", "VRHush", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/64/logo_crop_1606755050.png", "") // Dedicated scraper exists
	addSLRScraper("vr-intimacy", "VRIntimacy", "VRIntimacy", "", "")                                                                                      // Dedicated scraper exists, no id collision
	addSLRScraper("vrjjproductions", "VRJJProductions", "VRJJProductions", "", "")
	addSLRScraper("vrlatina-slr", "VRLatina", "VRLatina", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/110/logo_crop_1606820822.png", "") // Dedicated scraper exists
	addSLRScraper("vrmodelphotography", "VRModelPhotography", "VRModelPhotography", "", "")
	addSLRScraper("vrpfilms", "VRPFilms", "VRPFilms", "", "")
	addSLRScraper("vrparadisexxx", "VRParadiseXXX", "VRParadiseXXX", "", "")
	addSLRScraper("vrsexperts", "VRSexperts", "VRSexperts", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/89/logo_crop_1611130026.png", "")
	addSLRScraper("vrsexygirlz", "VRSexyGirlz", "VRSexyGirlz", "", "")
	addSLRScraper("vrsmokers", "VRSmokers", "VRSmokers", "", "")
	addSLRScraper("vrstars", "VRStars", "VRStars", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/271/logo_crop_1648490662.png", "")
	addSLRScraper("vrteenrs-slr", "VRTeenrs", "VRTeenrs", "", "") // Dedicated scraper exists
	addSLRScraper("vrvids", "VRVids", "VRVids", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/261/logo_crop_1648490520.png", "")
	addSLRScraper("vrbuz", "VRbuz", "VRbuz", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/235/logo_crop_1606869235.png", "")
	addSLRScraper("vrextasy", "VReXtasy", "VReXtasy", "", "")
	addSLRScraper("vredging", "VRedging", "VRedging", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/245/logo_crop_1606774151.png", "")
	addSLRScraper("vrixxens", "VRixxens", "VRixxens", "", "")
	addSLRScraper("vrlab9division", "VRlab9division", "VRlab9division", "", "")
	addSLRScraper("vrmodels", "VRmodels", "VRmodels", "", "")
	addSLRScraper("vroomed", "VRoomed", "VRoomed", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/380/logo_crop_1647990015.png", "")
	addSLRScraper("vrpornjack", "VRpornjack", "VRpornjack", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/172/logo_crop_1617047058.png", "")
	addSLRScraper("vrsolos", "VRsolos", "VRsolos", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/182/logo_crop_1606774059.png", "")
	addSLRScraper("vippissy", "VipPissy", "VipPissy", "", "")
	addSLRScraper("viro-playspace", "Viro Playspace", "ViroPlayspace", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/304/logo_crop_1619563208.png", "")
	addSLRScraper("virtual-papi", "Virtual Papi", "VirtualPapi", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/381/logo_crop_1648046257.png", "")
	addSLRScraper("virtualexotica", "VirtualExotica", "VirtualExotica", "", "")
	addSLRScraper("virtualpee", "VirtualPee", "VirtualPee", "", "")
	addSLRScraper("virtualporn360", "VirtualPorn360", "VirtualPorn360", "", "")
	addSLRScraper("virtualporndesire", "VirtualPornDesire", "VirtualPornDesire", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/28/logo_crop_1606324685.png", "")
	addSLRScraper("virtualrealamateur-slr", "VirtualRealAmateur", "VirtualRealAmateur", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/71/logo_crop_1608196629.png", "") // Dedicated scraper exists
	addSLRScraper("virtualrealpassion-slr", "VirtualRealPassion", "VirtualRealPassion", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/57/logo_crop_1608196615.png", "") // Dedicated scraper exists
	addSLRScraper("virtualrealporn-slr", "VirtualRealPorn", "VirtualRealPorn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/2/logo_crop_1608196599.png", "")           // Dedicated scraper exists
	addSLRScraper("virtualxporn", "VirtualXPorn", "VirtualXPorn", "", "")
	addSLRScraper("vixenvr", "VixenVR", "VixenVR", "", "")
	addSLRScraper("waapvr", "WAAPVR", "WAAPVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/117/logo_crop_1606868278.png", "")
	addSLRScraper("wow", "WOW!", "WOW!", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/249/logo_crop_1606869271.png", "")
	addSLRScraper("wankitnowvr-slr", "WankitnowVR", "WankitnowVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/154/logo_crop_1610456190.png", "") // Dedicated scraper exists
	addSLRScraper("wankzvr-slr", "WankzVR", "WankzVR", "", "")                                                                                                            // Dedicated scraper exists
	addSLRScraper("wastelandvr", "WastelandVR", "WastelandVR", "", "")
	addSLRScraper("whorecraftvr-slr", "WhorecraftVR", "WhorecraftVR", "", "") // Dedicated scraper exists
	addSLRScraper("yanksvr", "YanksVR", "YanksVR", "", "")
	addSLRScraper("yellow-pinkman", "Yellow Pinkman", "YellowPinkman", "", "")
	addSLRScraper("zaawaadivr", "ZaawaadiVR", "ZaawaadiVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/268/logo_crop_1608553140.png", "")
	addSLRScraper("zexyvr-slr", "ZexyVR", "ZexyVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/157/logo_crop_1610456086.png", "") // Dedicated scraper exists
	addSLRScraper("istripper", "iStripper", "iStripper", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/171/logo_crop_1632299940.png", "")
	addSLRScraper("myvrsin", "myVRsin", "myVRsin", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/115/logo_crop_1646595730.png", "")
	addSLRScraper("pervrt", "perVRt", "perVRt", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/85/logo_crop_1607550545.png", "")
	addSLRScraper("xjellyfish", "xJellyFish", "xJellyFish", "", "")
	addSLRScraper("xvirtual", "xVirtual", "xVirtual", "", "")

	// Gay
	addSLRScraper("kinkvr-gay", "KinkVR Gay", "KinkVRGay", "", "g") // Dedicated scraper exists, no id collision
	addSLRScraper("pervrt-gay", "perVRt Male", "perVRtMale", "", "g")
	addSLRScraper("virtual-real-gay", "Virtual Real Gay", "VirtualRealGay", "", "g") // Dedicated scraper exists, no id collision
	addSLRScraper("vrb-gay", "VRB Gay", "VRB Gay", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/4/5/logo_crop_1606810548.png", "g")

	// Trans
	addSLRScraper("ebony-vr-solos-shemale", "EBONY VR SOLOS Shemale", "EBONYVRSOLOSShemale", "", "t")
	addSLRScraper("futavr", "FutaVR", "FutaVR", "", "t")
	addSLRScraper("groobyvr-slr", "GroobyVR", "GroobyVR", "", "t") // Dedicated scraper exists
	addSLRScraper("helloladyboy", "HelloLadyboy", "HelloLadyboy", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/3/18/logo_crop_1628776153.png", "t")
	addSLRScraper("kinkvrshemale", "KinkVRShemale", "KinkVRShemale", "", "t")
	addSLRScraper("never-lonely-vr-shemale", "Never Lonely VR Shemale", "NeverLonelyVRShemale", "", "t")
	addSLRScraper("peeping-thom-shemale", "Peeping Thom Shemale", "PeepingThomShemale", "", "t")
	addSLRScraper("salamonvr", "SalamonVR", "SalamonVR", "", "t")
	addSLRScraper("sodcreate-tranny", "SodCreate Tranny", "SodCreateTranny", "", "t")
	addSLRScraper("transexvr", "TransexVR", "TransexVR", "", "t")
	addSLRScraper("tsvirtuallovers-slr", "TSVirtualLovers", "TSVirtualLovers", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/3/6/logo_crop_1606824641.png", "t")    // Dedicated scraper exists
	addSLRScraper("virtualrealtrans-slr", "VirtualRealTrans", "VirtualRealTrans", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/3/1/logo_crop_1608197876.png", "t") // Dedicated scraper exists
	addSLRScraper("vrb-trans", "VRB Trans", "VRBTrans", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/3/7/logo_crop_1606810513.png", "t")                           // Dedicated scraper exists, no id collision
	addSLRScraper("vrsexpertsts", "VRSexpertsTS", "VRSexpertsTS", "", "t")
}
