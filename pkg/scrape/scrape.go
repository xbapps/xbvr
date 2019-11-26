package scrape

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"golang.org/x/net/html"
)

var log = &common.Log

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"

func createPersistentCacheCollector(cacheID string, domains ...string) *colly.Collector {
	c := createCollector(domains...)
	c.CacheDir = getPersistentScrapeCacheDir(cacheID)
	return c
}

func createCollector(domains ...string) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains(domains...),
		colly.CacheDir(getScrapeCacheDir()),
		colly.UserAgent(userAgent),
	)

	c = createCallbacks(c)
	return c
}

func cloneCollector(c *colly.Collector) *colly.Collector {
	x := c.Clone()
	x = createCallbacks(x)
	return x
}

func createCallbacks(c *colly.Collector) *colly.Collector {
	const maxRetries = 15

	c.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")

		if attempt == nil {
			r.Ctx.Put("attempt", 1)
		}

		log.Infoln("visiting", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		attempt := r.Ctx.GetAny("attempt").(int)

		if r.StatusCode == 429 {
			log.Errorln("Error:", r.StatusCode, err)

			if attempt <= maxRetries {
				unCache(r.Request.URL.String(), c.CacheDir)
				log.Errorln("Waiting 2 seconds before next request...")
				r.Ctx.Put("attempt", attempt+1)
				time.Sleep(2 * time.Second)
				r.Request.Retry()
			}
		}
	})

	return c
}

func DeletePersistentScrapeCache(cacheID string) error {
	return os.RemoveAll(getPersistentScrapeCacheDir(cacheID))
}

func DeleteScrapeCache() error {
	return os.RemoveAll(getScrapeCacheDir())
}

func getPersistentScrapeCacheDir(cacheID string) string {
	return filepath.Join(common.PersistentScrapeCacheDir, cacheID)
}

func getScrapeCacheDir() string {
	return common.ScrapeCacheDir
}

func registerScraper(id string, name string, avatarURL string, f models.ScraperFunc) {
	models.RegisterScraper(id, name, avatarURL, f)
}

func logScrapeStart(id string, name string) {
	log.WithFields(logrus.Fields{
		"task":      "scraperProgress",
		"scraperID": id,
		"progress":  0,
		"started":   true,
		"completed": false,
	}).Infof("Starting %v scraper", name)
}

func logScrapeFinished(id string, name string) {
	log.WithFields(logrus.Fields{
		"task":      "scraperProgress",
		"scraperID": id,
		"progress":  0,
		"started":   false,
		"completed": true,
	}).Infof("Finished %v scraper", name)
}

func unCache(URL string, cacheDir string) {
	sum := sha1.Sum([]byte(URL))
	hash := hex.EncodeToString(sum[:])
	dir := path.Join(cacheDir, hash[:2])
	filename := path.Join(dir, hash)
	if err := os.Remove(filename); err != nil {
		log.Fatal(err)
	}
}

func updateSiteLastUpdate(id string) {
	var site models.Site
	err := site.GetIfExist(id)
	if err != nil {
		log.Error(err)
		return
	}
	site.LastUpdate = time.Now()
	site.Save()
}

func traverseNodes(node *html.Node, fn func(*html.Node)) {
	if node == nil {
		return
	}

	fn(node)

	for cur := node.FirstChild; cur != nil; cur = cur.NextSibling {
		traverseNodes(cur, fn)
	}
}

func findComments(sel *goquery.Selection) []string {
	comments := []string{}
	for _, node := range sel.Nodes {
		traverseNodes(node, func(node *html.Node) {
			if node.Type == html.CommentNode {
				comments = append(comments, node.Data)
			}
		})
	}
	return comments
}

func getFilenameFromURL(u string) string {
	p, _ := url.Parse(u)
	return path.Base(p.Path)
}

func getTextFromHTMLWithSelector(data string, sel string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(doc.Find(sel).Text())
}
