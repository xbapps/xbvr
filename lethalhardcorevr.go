package scrape

import (
	"regexp"
	"strings"
	"sync"
	"context"
	//"time"

	// "github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
)

func isGoodTag(lookup string) bool {
	switch lookup {
	case
		"vr",
		"whorecraft",
		"video",
		"streaming",
		"porn",
		"movie":
		return false
	}
	return true
}

func LethalHardcoreSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	// Create a new chromedp context - boiler plate
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	
	// Create Two nodes to the links to the site and scene URLS
	var siteURLNodes []*cdp.Node
	siteURL := []string {URL + `/en/videos/sort/latest`}

	// Fetch All the indexPage nodes when viewed from page 1
	err := chromedp.Run(ctx,
		chromedp.Navigate(siteURL[0]),
		chromedp.Nodes("a.Pagination-Page-Link", &siteURLNodes, chromedp.ByQueryAll),
	)
	if err != nil {
		log.Errorln(err)
	}

	

	// Convert the nodes to URL strings using the href attrib
	if !limitScraping {
		for _, n := range siteURLNodes {
			url := strings.TrimSpace(URL + n.AttributeValue("href"))
			
			if !funk.ContainsString(siteURL, url) {
				log.Infoln(`site url:`, url)
				siteURL = append(siteURL, url)
			}
		}
	}
	

	
	sceneURL := []string {}
	for _, indexPage := range siteURL {
		log.Infoln(indexPage)
		var sceneURLNodes []*cdp.Node
		// Fetch all the sceneURL nodes on each index page
		err := chromedp.Run(ctx,
			chromedp.Navigate(indexPage),
			chromedp.WaitReady("#reactApplication"),
			// Scrolls to the bottom of the page to ensure all scenes are loaded into the DOM
			chromedp.ActionFunc(func(ctx context.Context) error {
                _, exp, err := runtime.Evaluate(`window.scrollTo(0,document.body.scrollHeight);`).Do(ctx)
                if err != nil {
                    return err
                }
                if exp != nil {
                    return exp
                }
                return nil
            }),
			chromedp.WaitReady("div.ListingGrid-ListingGridItem"),
			chromedp.Nodes("div.ListingGrid-ListingGridItem a.SceneThumb-SceneImageLink-Link", &sceneURLNodes),
		)
		if err != nil {
			log.Errorln(err)
		}

		// Convert the nodes to URL strings using the href attrib
		for _, n := range sceneURLNodes {
			url := strings.TrimSpace(URL + n.AttributeValue("href"))
			if !funk.ContainsString(knownScenes, url) || !funk.ContainsString(sceneURL, url) {
				log.Infoln(`scene url:`, url)
				sceneURL = append(sceneURL, url)
			}
		}
	}
	

	
	// if err := chromedp.Run(ctx,
	// 	chromedp.Navigate(URL),
	// 	chromedp.Nodes(`document`, &nodes,
	// 		chromedp.ByJSPath, chromedp.Populate(-1, true)),
	// 	chromedp.WaitVisible(`#reactApplication`),
	// ); err != nil {
	// 	log.Errorln(err)
	// }

	// log.Infoln("Document tree:")
	// log.Infoln(nodes[0].Dump("  ", "  ", false))

	
	
	
	// allowedDomains := []string{"lethalhardcorevr.com", "www.lethalhardcorevr.com", "whorecraftvr.com"}

	// sceneCollector := createCollector(allowedDomains...)
	// siteCollector := createCollector(allowedDomains...)
	// siteCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
	// 	log.Infoln(e)
	// })
	// sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
	
	// Iterate over each sceneURL we have obtained that we havent visited yet
	for _, scene := range sceneURL {	
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Celestial Productions"
		sc.HomepageURL = scene
		log.Infoln(scene)
		
		// Site ID
		sc.Site = siteID

		// Search for the scene info node and the cover node 
		var sceneNode, coverNode []*cdp.Node
		err := chromedp.Run(ctx,
			chromedp.Navigate(scene),
			chromedp.Nodes("div.InfoContainer", &sceneNode, chromedp.ByQuery), // Contains all scene meta data
			chromedp.WaitReady("div.video-react-poster"),
			chromedp.Nodes("div.video-react-poster", &coverNode, chromedp.ByQuery), // Contains the background cover image
		)
		if err != nil {
			log.Errorln(err)
		}
		log.Infoln("Nodes Complete")
		// Scrape these values
		var sceneDate, sceneTitle, sceneActor string
		var tagNode []*cdp.Node
		for _, n := range sceneNode {
			chromedp.Run(ctx,
				chromedp.Text("span.ScenePlayerHeaderDesktop-Date-Text", &sceneDate, chromedp.ByQuery, chromedp.FromNode(n)),
				chromedp.Text("h1.ScenePlayerHeaderDesktop-PlayerTitle-Title", &sceneTitle, chromedp.ByQuery, chromedp.FromNode(n)),
				chromedp.Text("a.ActorThumb-Name-Link", &sceneActor, chromedp.ByQuery, chromedp.FromNode(n)),
				chromedp.Nodes("a.ScenePlayerHeaderDesktop-Categories-Link", &tagNode, chromedp.ByQueryAll),
			)
		}

		// Title
		sc.Title = strings.TrimSpace(sceneTitle)
		log.Infoln(sc.Title)

		// Release Date
		tmpDate, _ := goment.New(strings.TrimSpace(sceneDate), "YYYY-MM-DD")
		sc.Released = tmpDate.Format("YYYY-MM-DD")
		log.Infoln(sc.Released)

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		log.Infoln(sc.SiteID)
		log.Infoln(sc.SceneID)
		// Cover
		coverImage := coverNode[0].AttributeValue("style")
		re := regexp.MustCompile(`https://[\s\S]+\.jpg`)
		sc.Covers = append(sc.Covers, re.FindStringSubmatch(coverImage)[0])
		log.Infoln(sc.Covers)
		// Tags
		for _, n := range tagNode {
			tag := strings.ToLower(n.AttributeValue("text"))
			log.Infoln(tag)
			if isGoodTag(tag) {
				sc.Tags = append(sc.Tags, tag)
			}
		}

		// Cast
		sc.Cast = append(sc.Cast, strings.TrimSpace(sceneActor))
		// sc.ActorDetails = make(map[string]models.ActorDetails)
		// sc.ActorDetails[strings.TrimSpace(r.Replace(e.Text))] = models.ActorDetails{ImageUrl: sceneActorURL}
		
		// Gallery TODO Seperate webpage at https://www.lethalhardcorevr.com/en/photo/scene_title/photoshoot_id photoshoot_id not avialable on same page as scene

		// Synposis No longer posted

		break

	}
	// 	e.ForEach(`style`, func(id int, e *colly.HTMLElement) {
	// 		if id == 0 {
	// 			html, err := e.DOM.Html()
	// 			if err == nil {
	// 				re := regexp.MustCompile(`background\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
	// 				i := re.FindStringSubmatch(html)[1]
	// 				if len(i) > 0 {
	// 					sc.Covers = append(sc.Covers, re.FindStringSubmatch(html)[1])
	// 				}
	// 			}
	// 		}
	// 	})

	// 	// trailer details
	// 	sc.TrailerType = "url"
	// 	sc.TrailerSrc = e.ChildAttr(`span.link-player-action-inner a.btn`, `href`)

	// 	// Title
	// 	e.ForEach(`div.item-page-details h1`, func(id int, e *colly.HTMLElement) {
	// 		if id == 0 {
	// 			sc.Title = strings.TrimSpace(e.Text)
	// 		}
	// 	})

	// 	// Gallery TODO Seperate webpage at https://www.lethalhardcorevr.com/en/photo/sceneurl
	// 	e.ForEach(`div.screenshots-block img`, func(id int, e *colly.HTMLElement) {
	// 		sc.Gallery = append(sc.Gallery, strings.TrimSpace(e.Attr("src")))
	// 	})

	// 	// Synposis
	// 	e.ForEach(`#synopsis-full p`, func(id int, e *colly.HTMLElement) {
	// 		if id == 0 {
	// 			sc.Synopsis = strings.TrimSpace(e.Text)
	// 		}
	// 	})

	// 	// Cast
	// 	sc.ActorDetails = make(map[string]models.ActorDetails)
	// 	r := strings.NewReplacer("(", "", ")", "")
	// 	e.ForEach(`div.item-page-details a[data-target="#bodyShotModal"]`, func(id int, e *colly.HTMLElement) {
	// 		img := ""
	// 		e.ForEach(`img`, func(id int, e *colly.HTMLElement) {
	// 			style := e.Attr("style")
	// 			regexPattern := `url\((.*?)\)`
	// 			regex, _ := regexp.Compile(regexPattern)
	// 			matches := regex.FindStringSubmatch(style)
	// 			if len(matches) > 1 {
	// 				img = matches[1]
	// 			} else {
	// 				if e.Attr("src") != "https://imgs1cdn.adultempire.com/res/pm/pixel.gif" {
	// 					img = e.Attr("src")
	// 				}
	// 			}

	// 		})
	// 		e.ForEach(`.overlay small`, func(id int, e *colly.HTMLElement) {
	// 			if id <= 1 {
	// 				sc.Cast = append(sc.Cast, strings.TrimSpace(r.Replace(e.Text)))
	// 				sc.ActorDetails[strings.TrimSpace(r.Replace(e.Text))] = models.ActorDetails{ImageUrl: img}
	// 			}
	// 		})
	// 	})

	// 	// Tags
	// 	e.ForEach(`meta[name=Keywords]`, func(id int, e *colly.HTMLElement) {
	// 		k := strings.Split(e.Attr("content"), ",")
	// 		for i, tag := range k {
	// 			if i >= len(k)-2 {
	// 				for _, actor := range sc.Cast {
	// 					if funk.Contains(tag, actor) {
	// 						tag = strings.Replace(tag, actor, "", -1)
	// 					}
	// 				}
	// 			}
	// 			tag = strings.ToLower(strings.TrimSpace(tag))
	// 			if isGoodTag(tag) {
	// 				sc.Tags = append(sc.Tags, tag)
	// 			}
	// 		}
	// 	})

	// 	out <- sc
	// })

	
	// siteCollector.OnHTML(`a.Pagination-Page-Link`, func(e *colly.HTMLElement) {
	// 	if !limitScraping {
	// 		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
	// 		log.Infof(pageURL)
	// 		siteCollector.Visit(pageURL)
	// 	}
	// })

	// siteCollector.OnHTML(`div.scene-list-item`, func(e *colly.HTMLElement) {
	// 	sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))

	// 	ctx := colly.NewContext()
	// 	e.ForEach(`p.scene-update-stats a~span`, func(id int, e *colly.HTMLElement) {
	// 		if id == 0 {
	// 			ctx.Put("date", strings.TrimSpace(e.Text))
	// 		}
	// 	})

	// 	// If scene exist in database, there's no need to scrape
	// 	if !funk.ContainsString(knownScenes, sceneURL) {
	// 		sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
	// 	}
	// })

	// if singleSceneURL != "" {
	// 	ctx := colly.NewContext()
	// 	ctx.Put("date", "")

	// 	sceneCollector.Visit(singleSceneURL)
	// } else {
	// 	chromedp.Run(ctx,
	// 		chromedp.Navigate(URL),
	// 		chromedp.Nodes(".product", &nodes, chromedp.ByQueryAll),
	// 	)
	// 	//siteCollector.Visit(URL)
	// }

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func LethalHardcoreVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return LethalHardcoreSite(wg, updateSite, knownScenes, out, singleSceneURL, "lethalhardcorevr", "LethalHardcoreVR", "https://www.lethalhardcorevr.com", singeScrapeAdditionalInfo, limitScraping)
}

func WhorecraftVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return LethalHardcoreSite(wg, updateSite, knownScenes, out, singleSceneURL, "whorecraftvr", "WhorecraftVR", "https://lethalhardcorevr.com/lethal-hardcore-vr-scenes.html?studio=95347&sort=released", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("whorecraftvr", "WhorecraftVR", "https://imgs1cdn.adultempire.com/bn/Whorecraft-VR-apple-touch-icon.png", "lethalhardcorevr.com", WhorecraftVR)
	registerScraper("lethalhardcorevr", "LethalHardcoreVR", "https://imgs1cdn.adultempire.com/bn/Lethal-Hardcore-apple-touch-icon.png", "lethalhardcorevr.com", LethalHardcoreVR)
}
