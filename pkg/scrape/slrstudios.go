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

	siteCollector.Visit("https://www.sexlikereal.com/studios/" + scraperIdUrlPath + "?sort=most_recent")

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
	addSLRScraper("slr-originals-bts", "SLR Originals BTS", "SexLikeReal", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png")

	addSLRScraper("100vr", "100％ VR", "100％ VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/156/logo_crop_1606868336.png")
	addSLRScraper("33", "33+", "33+", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/343/logo_crop_1628188580.png")
	addSLRScraper("3dvr", "3DV+R", "3DV+R", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/205/logo_crop_1606869147.png")
	addSLRScraper("4k-vr", "4K VR", "4K VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/204/logo_crop_1606868871.png")
	addSLRScraper("ad4x", "AD4X", "AD4X", "https://ad4x.com/ypp_theme_ad4x/images/logo.png")
	addSLRScraper("ainovdo", "Ainovdo", "Ainovdo", "")
	addSLRScraper("alternativevrgirls", "Alternative VR Girls", "AlternativeVRGirls", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/301/logo_crop_1617013880.png")
	//addSLRScraper("altporn4u-vr", "AltPorn4uVR", "AltPorn4uVR", "https://www.altporn4u.com/favicon.ico") => now markohs
	addSLRScraper("amateurcouplesvr", "Amateur Couples VR", "AmateurCouplesVR", "https://www.sexlikereal.com/s/images/content/sexlikereal.png")
	addSLRScraper("amateurvr3d", "AmateurVR 3D", "AmateurVR3D", "http://amateurvr3d.com/assets/images/Nx50xlogo.png.pagespeed.ic.mr8RC-ybPl.webp")
	addSLRScraper("anal-delight", "Anal Delight", "AnalDelight", "https://mcdn.vrporn.com/files/20200907184611/AnalDelight_Logo.jpg")
	addSLRScraper("aromaplanning", "Aroma Planning", "AromaPlanning", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/192/logo_crop_1606868399.png")
	addSLRScraper("asari-vr", "Asari VR", "Asari VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/148/logo_crop_1606868320.png")
	addSLRScraper("avervr", "AverVR", "AverVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/369/logo_crop_1642559026.png")
	addSLRScraper("baberoticavr-slr", "Baberotiva VR", "BaberoticaVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/247/logo_crop_1609889316.png")
	addSLRScraper("babykxtten", "Babykxtten", "Babykxtten", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/321/logo_crop_1624979737.png")
	addSLRScraper("bakunouvr", "Bakunou VR", "BakunouVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/339/logo_crop_1627446437.png")
	addSLRScraper("bravomodelsmedia", "BravoModels Media", "Bravo Models", "https://mcdn.vrporn.com/files/20181015142403/ohNFa81Q_400x400.png")
	addSLRScraper("bullitt", "Bullitt", "Bullitt", "")
	addSLRScraper("burningangelvr", "Burning Angel VR", "BurningAngelVR", "https://mcdn.vrporn.com/files/20170830191746/burningangel-icon-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("carivr", "Caribbeancom", "Caribbeancom", "")
	addSLRScraper("casanova", "CasanovA", "CasanovA", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/184/logo_crop_1606868350.png")
	addSLRScraper("chinchinvr", "ChinChin VR", "ChinChinVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/188/logo_crop_1606868368.png")
	addSLRScraper("cosmoplanetsvr", "CosmoPlanets VR", "CosmoPlanetsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/252/logo_crop_1606869377.png")
	addSLRScraper("cosplay-with-me", "Cosplay with Me", "Cosplay with Me", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/274/logo_crop_1609165310.png")
	addSLRScraper("covert-japan", "Covert Japan", "CovertJapan", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/221/logo_crop_1607605022.png")
	addSLRScraper("cumcoders", "Cumcoders", "Cumcoders", "")
	addSLRScraper("cumtoon", "Cumtoon", "Cumtoon", "https://www.sexlikereal.com/s/images/content/sexlikereal.png")
	addSLRScraper("czechvrcasting-slr", "Czech VR Casting", "CzechVRCasting", "")
	addSLRScraper("czechvrfetish-slr", "Czech VR Fetish", "CzechVRFetish", "")
	addSLRScraper("dandy", "DANDY", "DANDY", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/355/logo_crop_1634611924.png")
	addSLRScraper("ddfnetworkvr-slr", "DDFNetworkVR", "DDFNetworkVR", "http://pbs.twimg.com/profile_images/1083417183722434560/Ur5xIhqG_200x200.jpg")
	addSLRScraper("deepinsex", "Deep in Sex", "DeepInSex", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/266/logo_crop_1610126420.png")
	addSLRScraper("deviantsvr", "Deviants VR", "DeviantsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/351/logo_crop_1638539790.png")
	addSLRScraper("dezyred", "Dezyred", "Dezyred", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/356/logo_crop_1636053912.png")
	addSLRScraper("dreamcam", "Dreamcam", "Dreamcam", "")
	addSLRScraper("dreamticketvr", "Dream Ticket VR", "DreamTicketVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/119/logo_crop_1606868303.png")
	addSLRScraper("dynamiceyes", "Dynamic Eyes", "DynamicEyes", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/263/logo_crop_1606870478.png")
	addSLRScraper("ebonyvr", "Ebony VR", "EbonyVR", "")
	addSLRScraper("ellielouisevr", "Ellie Louise VR", "EllieLouiseVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/265/logo_crop_1607603680.png")
	addSLRScraper("emilybloom", "Emily Bloom", "Emily Bloom", "https://i.imgur.com/LxYzAQX.png")
	addSLRScraper("erikalust", "Erika Lust", "ErikaLust", "")
	addSLRScraper("erotic-sinners", "Erotic Sinners", "Erotic Sinners", "")
	addSLRScraper("erotimevr", "Erotime VR", "ErotimeVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/208/logo_crop_1606869186.png")
	addSLRScraper("evileyevr", "EvilEye VR", "EvilEyeVR", "")
	addSLRScraper("fakingsvr", "FaKings VR", "FaKingsVR", "")
	addSLRScraper("fantastica", "FANTASTICA", "FANTASTICA", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/197/logo_crop_1606868466.png")
	addSLRScraper("fapp3d", "Fapp 3D", "Fapp3D", "")
	addSLRScraper("ffstockings", "FFStockings", "FFStockings", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/63/logo_crop_1607992231.png")
	addSLRScraper("fsknightsvisual", "FS.Knights Visual", "FS.KnightsVisual", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/341/logo_crop_1627446822.png")
	addSLRScraper("fuckpassvr", "FuckPass VR", "FuckPassVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/352/logo_crop_1635153994.png")
	addSLRScraper("glosstightsglamourvr", "Gloss Tights Glamour VR", "GlossTightsGlamourVR", "")
	addSLRScraper("gokunyu", "Gokunyu", "Gokunyu", "")
	addSLRScraper("grannies-vr", "Grannies VR", "GranniesVR", "https://mcdn.vrporn.com/files/20180222024100/itsmorti-logo-vr-porn-studio-vrporn.com-virtual-reality.jpg")
	addSLRScraper("hentaivr", "Hentai VR", "HentaiVR", "https://pbs.twimg.com/profile_images/1394712735854874632/ULktf61I_400x400.jpg")
	addSLRScraper("herfirstvr", "Her First VR", "HerFirstVR", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png")
	addSLRScraper("hiyoko", "HIYOKO", "HIYOKO", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/358/logo_crop_1636690145.png")
	addSLRScraper("holivr", "HoliVR", "HoliVR", "https://mcdn.vrporn.com/files/20170519145416/Holi_400x400.jpg")
	addSLRScraper("hologirlsvr-slr", "HoloGirlsVR", "HoloGirlsVR", "")
	addSLRScraper("hookfer", "Hookfer", "Hookfer", "https://mcdn.vrporn.com/files/20201116170637/400x400-Hookfer-logo.jpg")
	addSLRScraper("hotentertainment", "Hot Entertainment", "HotEntertainment", "")
	addSLRScraper("ichimonkai", "IchiMonKai", "IchiMonKai", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/318/logo_crop_1627446974.png")
	addSLRScraper("iloveitshiny", "I Love it Shiny", "ILoveItShiny", "")
	addSLRScraper("intimovr", "Intimo VR", "IntimoVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/264/logo_crop_1608437997.png")
	addSLRScraper("istripper", "iStripper", "TotemCore Ltd", "https://www.istripper.com/favicons/istripper/apple-icon-120x120.png")
	addSLRScraper("jimmydraws", "Jimmy Draws", "Jimmy Draws", "https://mcdn.vrporn.com/files/20190821145930/iLPJW6J7_400x400.png")
	addSLRScraper("justvr", "Just VR", "JustVR", "https://mcdn.vrporn.com/files/20181023121629/logo.jpg")
	addSLRScraper("jvrporn", "JVRPorn", "JVRPorn", "https://mcdn.vrporn.com/files/20170710084815/jvrporn-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("kinkygirlsberlin", "Kinky Girls Berlin", "KinkyGirlsBerlin", "")
	addSLRScraper("kmpvr", "KMP VR", "KMPVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/209/logo_crop_1606869200.png")
	addSLRScraper("koalavr", "Koala VR", "KoalaVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/189/logo_crop_1606868385.png")
	addSLRScraper("leninacrowne", "Lenina Crowne", "Terrible", "https://mcdn.vrporn.com/files/20190711135807/terrible_logo-e1562878668857_400x400_acf_cropped.jpg")
	addSLRScraper("lethalhardcorevr-slr", "LethalHardcoreVR", "LethalHardcoreVR", "")
	addSLRScraper("lezvr", "LezVR", "Lez VR", "https://mcdn.vrporn.com/files/20170813184453/lezvr-icon-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("libertinevr", "Libertine VR", "LibertineVR", "")
	addSLRScraper("lightsouthern", "Lightsouthern", "Lightsouthern", "")
	addSLRScraper("littlecapricedreamsvr", "Little Caprice VR", "LittleCapriceVR", "")
	addSLRScraper("loveherfeetvr", "Love her Feet VR", "LoveHerFeetVR", "")
	addSLRScraper("lustreality", "Lust Reality", "LustReality", "https://mcdn.vrporn.com/files/20200316102952/lustreality_logo2.png")
	addSLRScraper("majamagic", "Maja Magic", "MajaMagic", "")
	addSLRScraper("markohs", "Markohs", "Markohs", "")
	addSLRScraper("marrionvr", "Marrion VR", "MarrionVR", "https://www.sexlikereal.com/s/images/content/sexlikereal.png")
	addSLRScraper("maturesvr", "Matures VR", "MaturesVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/314/logo_crop_1620737406.png")
	addSLRScraper("max-a", "MAX-A", "MAX-A", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/198/logo_crop_1606868853.png")
	addSLRScraper("maxing", "MAXING", "MAXING", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/345/logo_crop_1633940793.png")
	addSLRScraper("mayashandjobsvr", "Mayas Handjobs VR", "MayasHandjobsVR", "")
	addSLRScraper("meangirlsvr", "MeanGirls VR", "MeanGirlsVR", "")
	addSLRScraper("mike-and-kate", "Mike and Kate", "Mike and Kate", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/309/logo_crop_1620307648.png")
	addSLRScraper("milfvr-slr", "MILFVR", "MILFVR", "")
	addSLRScraper("miluvr", "Milu VR", "MiluVR", "")
	addSLRScraper("mmm100", "MMM100", "MMM100", "https://mmm100.com/MMM100.png")
	addSLRScraper("mondelde-vr", "MONDELDE VR", "MONDELDE VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/359/logo_crop_1636078868.png")
	addSLRScraper("mongercash", "Asiansexdiary VR", "AsiansexdiaryVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/219/logo_crop_1628500033.png")
	addSLRScraper("mousouvr", "Mousou VR", "MousouVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/317/logo_crop_1623114497.png")
	addSLRScraper("mrmichiru", "Mr. Michiru", "Mr. Michiru", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/307/logo_crop_1621216369.png")
	addSLRScraper("mugon-vr", "MUGON VR", "MUGON VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/344/logo_crop_1634613016.png")
	addSLRScraper("munakuso", "MunaKuso", "MunaKuso", "")
	addSLRScraper("muscleworshipvr", "Muscle Worship VR", "MuscleWorshipVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/347/logo_crop_1633378952.png")
	addSLRScraper("myvrsin", "myVRsin", "myVRsin", "")
	addSLRScraper("natural-high", "NATURALHIGH", "NATURAL HIGH", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/331/logo_crop_1623047952.png")
	addSLRScraper("naughtyfreyavr", "Naughty Freya VR", "NaughtyFreyaVR", "")
	addSLRScraper("net69vr", "Net69 VR", "Net69VR", "https://mcdn.vrporn.com/files/20171113233505/net69vr-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("never-lonely-vr", "Never Lonely VR", "Never Lonely VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/353/logo_crop_1638474774.png")
	addSLRScraper("no2studiovr", "No2Studio VR", "No2StudioVR", "https://mcdn.vrporn.com/files/20201021145654/No2StudioVR_400x400-1.jpg")
	addSLRScraper("nylon-xtreme", "Nylon Xtreme", "NylonXtreme", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/227/logo_crop_1641243932.png")
	addSLRScraper("office-ks-vr", "OFFICE KS VR", "OFFICE KS VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/248/logo_crop_1606869252.png")
	addSLRScraper("olipsvr", "Olips VR", "OlipsVR", "")
	addSLRScraper("only3xvr", "Only3x VR", "Only3xVR", "https://mcdn.vrporn.com/files/20190821140339/only3xvr-profile-pic.jpg")
	addSLRScraper("onlytease", "Only Tease", "OT Publishing Ltd", "https://www.onlytease.com/assets/img/favicons/ot/apple-touch-icon.png")
	addSLRScraper("p-box-vr", "P-BOXVR", "P-BOX VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/250/logo_crop_1606869346.png")
	addSLRScraper("parellalwordle", "Parellal Wordle", "ParellalWordle", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/196/logo_crop_1606868450.png")
	addSLRScraper("peeping-thom", "Peeping Thom", "Peeping Thom", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/303/logo_crop_1619656190.png")
	addSLRScraper("pegasproductions", "Pegas Productions", "Pegas Productions", "")
	addSLRScraper("pervrt", "perVRt/Terrible", "Terrible", "https://mcdn.vrporn.com/files/20181218151630/pervrt-logo.jpg")
	addSLRScraper("petersmax", "Peters MAX", "Peters MAX", "")
	addSLRScraper("petersprimovr", "Peters Primo VR", "PetersPrimoVR", "")
	addSLRScraper("pip-vr", "PIP VR", "PIP VR", "https://www.sexlikereal.com/s/images/content/sexlikereal.png")
	addSLRScraper("plushiesvr", "Plushies VR", "PlushiesVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/122/logo_crop_1615877579.png")
	addSLRScraper("pornbcn", "Pornbcn", "Pornbcn", "https://mcdn.vrporn.com/files/20190923110340/CHANNEL-LOGO-2.jpg")
	addSLRScraper("pornodanvr", "Porno Dan VR", "PornoDanVR", "")
	addSLRScraper("povcentralvr", "POVcentral VR", "POV Central", "https://mcdn.vrporn.com/files/20191125091909/POVCentralLogo.jpg")
	addSLRScraper("prestige", "PRESTIGE", "PRESTIGE", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/302/logo_crop_1621216541.png")
	addSLRScraper("propertysexvr", "Property Sex VR", "PropertySexVR", "")
	addSLRScraper("ps-porn", "PS-Porn", "Paula Shy", "https://mcdn.vrporn.com/files/20201221090642/PS-Porn-400x400.jpg")
	addSLRScraper("pvrstudio", "PVRStudio", "PVRStudio", "https://pvr.fun/uploads/2019/10/08/084230gbctdepe7kovu4hs.jpg")
	addSLRScraper("radical", "RADICAL", "RADICAL", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/275/logo_crop_1610511622.png")
	addSLRScraper("realhotvr", "RealHot VR", "RealHotVR", "https://images.povr.com/assets/logos/channels/0/3/3835/200.svg")
	addSLRScraper("realitylovers-slr", "RealityLovers", "RealityLovers", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/27/logo_crop_1606823905.png")
	addSLRScraper("realjamvr", "Real Jam VR", "RealJamVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/53/logo_crop_1606820837.png")
	addSLRScraper("reevr", "ReeVR", "ReeVR", "")
	addSLRScraper("rezowoods", "Rezowoods", "Rezowoods", "")
	addSLRScraper("rocket", "ROCKET", "ROCKET", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/357/logo_crop_1636689676.png")
	addSLRScraper("salome-prologue", "SALOME Prologue", "SALOME Prologue", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/251/logo_crop_1606869361.png")
	addSLRScraper("scoop-vr", "SCOOP VR", "SCOOP VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/262/logo_crop_1606869393.png")
	addSLRScraper("screwboxvr", "Screw Box VR", "ScrewBox", "https://pbs.twimg.com/profile_images/1137432770936918016/ycL3ag5c_200x200.png")
	addSLRScraper("seducevr", "Seduce VR", "SeduceVR", "")
	addSLRScraper("sensesvrporn", "Senses VRporn", "SensesVRporn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/281/logo_crop_1628150671.png")
	addSLRScraper("sexbabesvr", "SexBabes VR", "SexBabesVR", "")
	addSLRScraper("sheer-experience", "Sheer Experience", "SheerExperience", "")
	addSLRScraper("sinsdealers", "SinsDealers", "SinsDealers", "")
	addSLRScraper("sinsvr", "Sins VR", "SinsVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/112/logo_crop_1643894413.png")
	addSLRScraper("skinrays", "Skin Rays", "SkinRays", "")
	addSLRScraper("sodcreate", "SodCreate", "SodCreate", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/216/logo_crop_1606869217.png")
	addSLRScraper("squeeze-vr", "Squeeze VR", "SqueezeVR", "https://mcdn.vrporn.com/files/20210322150700/squeezevr_logo.png")
	addSLRScraper("stasyqvr-slr", "StasyQVR", "StasyQVR", "")
	addSLRScraper("stockingsvr", "Stockings VR", "StockingsVR", "https://mcdn.vrporn.com/files/20171107092330/stockingsvr_logo_vr_porn_studio_vrporn.com_virtual_reality1-1.png")
	addSLRScraper("strictlyglamourvr", "Strictly Glamour VR", "StrictlyGlamourVR", "")
	addSLRScraper("stripzvr", "Stripz VR", "N1ck Inc.", "https://www.stripzvr.com/wp-content/uploads/2018/09/cropped-favicon-192x192.jpg")
	addSLRScraper("swallowbay", "Swallow Bay", "SwallowBay", "https://mcdn.vrporn.com/files/20210330092926/swallowbay-400x400.jpg")
	addSLRScraper("sweetlonglips", "Sweetlonglips", "Sweetlonglips", "https://mcdn.vrporn.com/files/20200117105304/SLLVRlogo.png")
	addSLRScraper("taboo-vr-porn", "Taboo VRPorn", "Taboo VR Porn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/354/logo_crop_1643894389.png")
	addSLRScraper("taboovr", "Taboo VR", "TabooVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/346/logo_crop_1637164502.png")
	addSLRScraper("tadpolexxxstudio", "TadPoleXXXStudio", "TadPoleXXXStudio", "https://mcdn.vrporn.com/files/20190928101126/tadpolexxx-logo-vr-porn-studio-vrporn.com-virtual-reality.png")
	addSLRScraper("teppanvr", "Teppan VR", "TeppanVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/194/logo_crop_1606868415.png")
	addSLRScraper("thelockedcockchronicles", "The Locked Cock Chronicles", "TheLockedCockChronicles", "")
	addSLRScraper("thevirtualpornwebsite", "The Virtual Porn Website", "TheVirtualPornWebsite", "")
	addSLRScraper("tmavr", "TMA VR", "TMAVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/207/logo_crop_1606869169.png")
	addSLRScraper("tmwvrnet", "TmwVRNet", "TmwVRNet", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/26/logo_crop_1623330575.png")
	addSLRScraper("unfinished-vr", "Unfinished VR", "UnfinishedVR", "")
	addSLRScraper("v1vr", "V1VR", "V1VR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/195/logo_crop_1606868432.png")
	addSLRScraper("vippissy", "Vip Pissy", "VipPissy", "")
	addSLRScraper("viro-playspace", "Viro Playspace", "Viro Playspace", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/304/logo_crop_1619563208.png")
	addSLRScraper("virtualexotica", "Virtual Exotica", "VirtualExotica", "")
	addSLRScraper("virtualpee", "Virtual Pee", "VirtualPee", "https://mcdn.vrporn.com/files/20180317104121/virtualpeeop-square-banner.jpg")
	addSLRScraper("virtualporn360", "VirtualPorn 360", "VirtualPorn360", "")
	addSLRScraper("virtualporndesire", "VirtualPorn Desire", "VirtualPornDesire", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/28/logo_crop_1606324685.png")
	addSLRScraper("virtualrealamateur-slr", "VirtualRealAmateur", "VirtualRealAmateur", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/71/logo_crop_1608196629.png")
	addSLRScraper("virtualrealpassion-slr", "VirtualRealPassion", "VirtualRealPassion", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/57/logo_crop_1608196615.png")
	addSLRScraper("virtualrealporn-slr", "VirtualRealPorn", "VirtualRealPorn", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/2/logo_crop_1608196599.png")
	addSLRScraper("virtualxporn", "VirtualX Porn", "VirtualXPorn", "https://www.virtualxporn.com/tour/custom_assets/favicons/android-chrome-192x192.png")
	addSLRScraper("vixenvr", "Vixen VR", "VixenVR", "")
	addSLRScraper("vr-bangers-slr", "VRBangers", "VR Bangers", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/21/logo_crop_1606744484.png")
	addSLRScraper("vr-fan-service", "VR FanService", "VRFanService", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/153/logo_crop_1619422412.png")
	addSLRScraper("vr-intimacy", "VR Intimacy", "VRIntimacy", "")
	addSLRScraper("vr-japanese-idols-party", "VR Japanese Idols Party", "VR Japanese Idols Party ", "")
	addSLRScraper("vr-japanese-pornstars-stay-home-routines", "VR Japanese Pornstars Stay Home Routines", "VR Japanese Pornstars stay home routines", "")
	addSLRScraper("vr3000-slr", "VR3000", "VR3000", "")
	addSLRScraper("vrallure-slr", "VRAllure", "VRAllure", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/213/logo_crop_1606755181.png")
	addSLRScraper("vranimeted", "VRAnimeTed", "VRAnimeTed", "")
	addSLRScraper("vrbuz", "VRbuz", "VRbuz", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/235/logo_crop_1606869235.png")
	addSLRScraper("vrclubz-slr", "VRClubz", "VRClubz", "")
	addSLRScraper("vrconk-slr", "VRConk", "VRConk", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/96/logo_crop_1636655397.png")
	addSLRScraper("vredging", "VRedging", "VRedging", "https://mcdn.vrporn.com/files/20200630081500/VRedging_LOGO_v1-400x400.jpg")
	addSLRScraper("vrextasy", "VReXtasy", "VReXtasy", "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png")
	addSLRScraper("vrfirsttimer", "VR FirstTimer", "VRFirstTimer", "https://mcdn.vrporn.com/files/20200511115233/VRFirstTimers_Logo.jpg")
	addSLRScraper("vrfootfetish", "VR Foot Fetish", "VRFootFetish", "")
	addSLRScraper("vrgoddess", "VRgoddess", "VRgoddess", "")
	addSLRScraper("vrhush-slr", "VRHush", "VRHush", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/64/logo_crop_1606755050.png")
	addSLRScraper("vrixxens", "VRixxens", "VRixxens", "https://mcdn.vrporn.com/files/20200511115233/VRFirstTimers_Logo.jpg")
	addSLRScraper("vrjjproductions", "VRJJ Productions", "VRJJProductions", "")
	addSLRScraper("vrlatina-slr", "VRLatina", "VRLatina", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/110/logo_crop_1606820822.png")
	addSLRScraper("vrmodelphotography", "VR Model Photography", "VRModelPhotography", "")
	addSLRScraper("vrmodels", "VRmodels", "VRmodels", "")
	addSLRScraper("vrparadisexxx", "VR ParadiseXXX", "VRParadiseXXX", "")
	addSLRScraper("vrpfilms", "VRPFilms", "VRPFilms", "https://vrpfilms.com/storage/settings/March2021/Z0krYIQBMwSJ4R1eCnv1.png")
	addSLRScraper("vrpornjack", "VR PornJack", "VRPornJack", "https://mcdn.vrporn.com/files/20210330121852/VRPORNJACK_Logo-400x400.png")
	//addSLRScraper("vrpussyvision", "VRpussyVision", "VRpussyVision", "https://mcdn.vrporn.com/files/20180313160830/vrpussyvision-square-banner.png") => deprecated
	addSLRScraper("vrsexperts", "VRSexperts", "VRSexperts", "https://mcdn.vrporn.com/files/20190812141431/vrsexpertslogo2.jpg")
	addSLRScraper("vrsexygirlz", "VR Sexy Girlz", "VRSexyGirlz", "")
	addSLRScraper("vrsmokers", "VR Smokers", "VRSmokers", "")
	addSLRScraper("vrsolos", "VR Solos", "VRSolos", "https://mcdn.vrporn.com/files/20191226092954/VRSolos_Logo.jpg")
	addSLRScraper("vrstars", "VR Stars", "VRStars", "")
	addSLRScraper("vrteenrs-slr", "VR Teenrs", "VRTeenrs", "")
	addSLRScraper("vrvids", "VR Vids", "VRVids", "https://www.sexlikereal.com/s/images/content/sexlikereal.png")
	addSLRScraper("waapvr", "WAAPVR", "WAAPVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/117/logo_crop_1606868278.png")
	addSLRScraper("wankitnowvr-slr", "WankitnowVR", "WankitnowVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/154/logo_crop_1610456190.png")
	addSLRScraper("wankzvr-slr", "WankzVR", "WankzVR", "")
	addSLRScraper("wastelandvr", "Wasteland VR", "WastelandVR", "")
	addSLRScraper("whorecraftvr", "Whorecraft VR", "WhorecraftVR", "")
	addSLRScraper("wow", "WOW", "WOW!", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/249/logo_crop_1606869271.png")
	addSLRScraper("xjellyfish", "xJellyFish", "xJellyFish", "")
	addSLRScraper("xvirtual", "xVirtual", "xVirtual", "https://mcdn.vrporn.com/files/20181116133947/xvirtuallogo.jpg")
	addSLRScraper("yanksvr", "Yanks VR", "YanksVR", "")
	addSLRScraper("zaawaadivr", "Zaawaadi VR", "ZaawaadiVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/268/logo_crop_1608553140.png")
	addSLRScraper("zexyvr-slr", "ZexyVR", "ZexyVR", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/157/logo_crop_1610456086.png")
}
