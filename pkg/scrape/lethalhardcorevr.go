package scrape

import (
	"regexp"
	"strings"
	"sync"
	"context"
	"math"
	"strconv"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// the default user-agent header is something like: Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_3) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/89.0.4389.114 Safari/537.36
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36"),
		// Needed due class name change on small size and predict card count per scroll
		chromedp.WindowSize(1920, 2000), 
		// Disables Img loading. Should reduce network traffic
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx,)
	defer cancel()

	sceneURL := []string {}

	if singleSceneURL != "" {
		sceneURL = append(sceneURL, singleSceneURL)
	} else {

		// Used to store the nodes containing the anchor tags for the index page links
		var siteURLNodes []*cdp.Node

		// Use to store total scene count. Used to determine how many cards should exist on the last page
		var sceneCountStr string
		siteURL := []string {URL + `/en/videos/sort/latest`}

		// Fetch All the indexPage nodes when viewed from page 1
		err := chromedp.Run(ctx,
			chromedp.Navigate(siteURL[0]),
			chromedp.Nodes("//a[contains(@class,'Pagination-Page-Link')]", &siteURLNodes),
			chromedp.Text("//span[contains(@class,'SearchListing-ResultCount-Text')]", &sceneCountStr),
		)
		if err != nil {
			log.Errorln(err)
		}

		// The scene counter string contains a-z chars which we need to dump
		re := regexp.MustCompile(`\d+`)
		sceneCountStr = re.FindStringSubmatch(sceneCountStr)[0]
		// We need to convert the string to an integer for math later on
		sceneCountInt, _ := strconv.Atoi(sceneCountStr)

		// Convert the nodes to URL strings using the href attrib and append it to the supplied URL 
		if !limitScraping {
			for _, n := range siteURLNodes {
				url := strings.TrimSpace(URL + n.AttributeValue("href"))
				
				if !funk.ContainsString(siteURL, url) {
					siteURL = append(siteURL, url)
				}
			}
		}

		for i, indexPage := range siteURL {
			
			log.Infoln("Visiting:", indexPage)
			
			// Max Card count per page
			divCount := 60

			// Find the remainder of diving the total scene count by cards on a full page
			if i == len(siteURL) - 1 {
				divCount = int(math.Mod(float64(sceneCountInt), 60))
			}

			if err := chromedp.Run(ctx,

				// Prevents changes of class names depending on resolution
				// chromedp.EmulateViewport(1920, 2000),
				chromedp.Navigate(indexPage),
				chromedp.ActionFunc(func(ctx context.Context) error {
					// Do 5 loops as 12 cards are load in each scroll reliably
					for i:=1; i<6; i++ {
						
						var sceneURLNodes []*cdp.Node

						// Variables used to evenly divide each scroll
						divNumber := divCount*i/5
						precentScrollHeight := float64(i)/5

						if err := chromedp.Run(ctx,
							// Make sure 12 Cards are loaded
							chromedp.WaitReady("//div[@class='ListingGrid-ListingGridItem'][" + strconv.Itoa(divNumber) + "]//h3/a"),
							// Grab the nodes for later use of each card
							chromedp.Nodes("//div[@class='ListingGrid-ListingGridItem']//h3/a", &sceneURLNodes),
							chromedp.ActionFunc(func(ctx context.Context) error {
								//Scroll to the next set of cards
								_, exp, err := runtime.Evaluate(`window.scrollTo(0, document.body.scrollHeight*` + strconv.FormatFloat(precentScrollHeight, 'f', -1, 64) + `);`).Do(ctx)
									if err != nil {
										return err
									}
									if exp != nil {
										return exp
									}
									return nil
							}),
						); err != nil {
							log.Errorln(err)
						}
						// Grab the scene href from each card the append it to the supplied URL
						for _, n := range sceneURLNodes {
							url := strings.TrimSpace(URL + n.AttributeValue("href"))
							// Check to ensure we are duplicating urls due to uneven loading of page and scene we have already visited on previous scrapes
							if !funk.ContainsString(knownScenes, url) && !funk.ContainsString(sceneURL, url) {
								sceneURL = append(sceneURL, url)
							}
						}
					}
					return nil
				}),
				); err != nil {
				log.Errorln(err)
			}

			log.Infoln("Found a total of:", len(sceneURL), "scene urls on https://www.lethalhardcorevr.com")
		}
	}
	
	
	// Iterate over each sceneURL we have obtained that we haven't visited yet
	for _, scene := range sceneURL {	
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Celestial Productions"
		sc.HomepageURL = scene
		log.Infoln("Visiting:", scene)
		
		// Site ID
		sc.Site = siteID

		// Search for the scene info node and the cover node 
		var tagNode []*cdp.Node
		var coverImage, sceneTitle, sceneDate, sceneActor, sceneActorImgUrl string
		err := chromedp.Run(ctx,
			// chromedp.EmulateViewport(1920, 2000),
			chromedp.Navigate(scene),
			//chromedp.WaitReady("//span[contains(@class, 'ScenePlayerHeaderDesktop-PlayIcon-Icon-Svg')]"),
			// chromedp.Nodes("//div[contains(@class,'InfoContainer')]", &sceneNode), // Contains all scene info data
			chromedp.Nodes("//a[contains(@class, 'ScenePlayerHeaderDesktop-Categories-Link')]", &tagNode), // All nodes for tags
			chromedp.AttributeValue("//meta[@property='og:image']", "content", &coverImage, nil), // Contains the background cover image
			chromedp.AttributeValue("//meta[@property='og:title']", "content", &sceneTitle, nil), // Contains the title
			chromedp.Text("//span[contains(@class, 'ScenePlayerHeaderDesktop-Date-Text')]", &sceneDate), //Contains the Date
			chromedp.Text("//a[contains(@class, 'ActorThumb-Name-Link')]", &sceneActor), // Contains the Actor name in upper case
			chromedp.AttributeValue("//div[contains(@class, 'component-ActorThumb-List')]//img", "src", &sceneActorImgUrl, nil),
		)
		if err != nil {
			log.Errorln(err)
		}

		// Title
		sc.Title = strings.TrimSpace(sceneTitle)

		// Release Date
		tmpDate, _ := goment.New(strings.TrimSpace(sceneDate), "YYYY-MM-DD")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		re := regexp.MustCompile(`https://[\s\S]+\.jpg`)
		sc.Covers = append(sc.Covers, re.FindStringSubmatch(coverImage)[0])
		sceneActorImgUrl = re.FindStringSubmatch(sceneActorImgUrl)[0]

		// Tags
		for _, n := range tagNode {
			tag := strings.TrimSpace(strings.ToLower(n.AttributeValue("title")))
			if isGoodTag(tag) {
				sc.Tags = append(sc.Tags, tag)
			}
		}

		// Cast - may causes issues if LH ever does multiple cast members
		sc.Cast = append(sc.Cast, cases.Title(language.English).String(strings.ToLower(strings.TrimSpace(sceneActor))))
		sc.ActorDetails = make(map[string]models.ActorDetails)
		sc.ActorDetails[sc.Cast[0]] = models.ActorDetails{ImageUrl: sceneActorImgUrl}
		
		// Gallery TODO Separate webpage at https://www.lethalhardcorevr.com/en/photo/scene_title/photoshoot_id photoshoot_id not available on same page as scene

		// Synposis No longer posted

		out <- sc
	}
	
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
