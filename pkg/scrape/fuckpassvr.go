package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

var fpvrSiteIDRegex = regexp.MustCompile(`FPVR(\d+)`)

func FuckPassVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "fuckpassvr-native"
	siteID := "FuckPassVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.fuckpassvr.com")
	siteCollector := createCollector("www.fuckpassvr.com")

	client := resty.New()
	client.SetHeader("User-Agent", UserAgent)
	client.SetTimeout(5 * time.Second)
	warmBase := imgProxyBase()

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "FuckPassVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		if m := fpvrSiteIDRegex.FindStringSubmatch(e.ChildAttr(`meta[property="og:image"]`, "content")); len(m) > 1 {
			sc.SiteID = m[1]
			sc.SceneID = "fpvr-" + m[1]
		}

		sc.Title = strings.TrimSpace(e.ChildAttr(`meta[property="og:title"]`, "content"))
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.readMoreWrapper2`))
		sc.Duration = parseDurationMinutes(e.ChildText(`em.scene-joinPanel__duration`))

		if poster := strings.Trim(e.ChildAttr(`pornhall-player`, "poster"), " '"); poster != "" {
			sc.Covers = append(sc.Covers, poster)
		}
		if len(sc.Covers) == 0 { // fallback for scenes without a player poster
			if src := strings.Trim(e.ChildAttr(`#sfwVideo img`, "src"), " '"); src != "" {
				sc.Covers = append(sc.Covers, src)
			}
		}
		// temp: FPVR0292's cover is broken on the site; fall through main -> SLR -> vrporn, keeping the first that loads
		if sc.SiteID == "0292" {
			candidates := append(sc.Covers,
				"https://cdn-vr.sexlikereal.com/images/60766/vr-porn-Strippers-Need-Love-Too-cover-desktop.jpg",
				"https://mcdn.vrporn.com/files/20250609161112/Strippers-Need-Love-Too-Gia-DiBella-FuckPassVR-vr-porn-video.jpg",
			)
			sc.Covers = nil
			for _, u := range candidates {
				if u != "" && imageReachable(client, u) {
					sc.Covers = []string{u}
					break
				}
			}
		}

		if t, err := goment.New(strings.TrimSpace(e.ChildText(`span.scene-cast__date`)), "MMM D, YYYY"); err == nil {
			sc.Released = t.Format("YYYY-MM-DD")
		}

		// thumbnail src, not the full-size href — UI caps gallery at 700px, so the 8K original is wasted bloat
		e.ForEach(`div.profile__gallery a.profile__galleryElement img`, func(id int, e *colly.HTMLElement) {
			src := strings.TrimSpace(e.Attr("src"))
			if base := strings.ToLower(strings.SplitN(src, "?", 2)[0]); strings.HasSuffix(base, ".jpg") || strings.HasSuffix(base, ".jpeg") {
				sc.Gallery = append(sc.Gallery, src)
			}
		})

		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a.scene-cast__hostessPill`, func(id int, e *colly.HTMLElement) {
			name := strings.TrimSpace(e.Attr("title"))
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		e.ForEach(`div.scene-cats__pills a.scene-cats__pill`, func(id int, e *colly.HTMLElement) {
			if tag := strings.TrimSpace(e.Text); !strings.EqualFold(tag, "8K") {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Filenames "<cast>-180[-POV]-FPVR_<q>.mp4": quality from the trailer URL (exact casing), or card title when the URL is broken.
		var pre string
		var qparts []string
		e.ForEach(`div.scene-downloads__card`, func(_ int, c *colly.HTMLElement) {
			fn := getFilenameFromURL(c.ChildAttr(`a.scene-downloads__cardBtn`, "href"))
			low := strings.ToLower(fn)
			if i := strings.Index(low, "-fpvr-2min"); i >= 0 && strings.HasSuffix(low, ".mp4") {
				if pre == "" {
					pre = fn[:i]
				}
				if !strings.Contains(low, "_1k") {
					qparts = append(qparts, fn[i+len("-fpvr-2min"):])
				}
				return
			}
			if q := fpvrQuality(c.ChildText(`.scene-downloads__cardTitle`)); q != "" {
				qparts = append(qparts, "_"+q+".mp4")
			}
		})
		if pre != "" {
			for _, qpart := range qparts {
				sc.Filenames = append(sc.Filenames, pre+"-FPVR"+qpart)
			}
		}

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "pornhall-player source", ContentPath: "src", QualityPath: "data-quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// emit first so the scene reaches the DB pipeline before the warm fetches
		warmURLs := append(append([]string{}, sc.Covers...), sc.Gallery...)
		out <- sc
		warmImageCache(client, warmBase, warmURLs)
	})

	siteCollector.OnHTML(`section.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			siteCollector.Visit(e.Attr("href"))
		}
	})

	siteCollector.OnHTML(`div.videos__element a.videos__videoTitle`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.fuckpassvr.com/destination")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

// parseDurationMinutes converts "MM:SS" or "HH:MM:SS" to whole minutes; 0 if unparseable.
func parseDurationMinutes(s string) int {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		if n, err := strconv.Atoi(p); err == nil && n >= 0 {
			nums[i] = n
		} else {
			return 0
		}
	}
	if nums[len(nums)-1] > 59 || nums[len(nums)-2] > 59 { // minutes and seconds fields
		return 0
	}
	mins := nums[len(nums)-2]
	if len(parts) == 3 {
		mins += nums[0] * 60
	}
	return mins
}

var fpvrQualityRe = regexp.MustCompile(`(?i)(\d+)K(?:\s+(UHD|HD))?`)

// fpvrQuality maps a download-card label ("8K UHD", "4K") to a filename suffix ("8k_UHD", "4k"); "" skips 1K / unrecognized.
func fpvrQuality(title string) string {
	m := fpvrQualityRe.FindStringSubmatch(title)
	if m == nil || m[1] == "1" {
		return ""
	}
	q := m[1] + "k"
	if m[2] != "" {
		q += "_" + strings.ToUpper(m[2])
	}
	return q
}

// imgProxyBase returns the local imageproxy prefix; size is arbitrary since the cache key is the bare source URL.
func imgProxyBase() string {
	host := config.Config.Server.BindAddress
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return "http://" + host + ":" + strconv.Itoa(config.Config.Server.Port) + "/img/700x/"
}

// imageReachable reports whether u serves an image (2xx with an image content-type).
func imageReachable(client *resty.Client, u string) bool {
	resp, err := client.R().Get(u)
	return err == nil && resp.IsSuccess() && strings.HasPrefix(resp.Header().Get("Content-Type"), "image/")
}

// warmImageCache caches image URLs via the local imageproxy before their tokens expire; concurrent, best-effort, failures only logged.
func warmImageCache(client *resty.Client, base string, urls []string) {
	var wg sync.WaitGroup
	for _, u := range urls {
		if u == "" {
			continue
		}
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			resp, err := client.R().Get(base + strings.Replace(u, "://", ":/", 1))
			if err != nil {
				log.Debugf("FuckPassVR image cache warm failed for %s: %v", u, err)
			} else if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
				log.Debugf("FuckPassVR image cache warm got HTTP %d for %s", resp.StatusCode(), u)
			}
		}(u)
	}
	wg.Wait()
}

func init() {
	registerScraper("fuckpassvr-native", "FuckPassVR", "https://www.fuckpassvr.com/favicon.png", "fuckpassvr.com", FuckPassVR)
}
