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

	"github.com/ProtonMail/go-appdir"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"golang.org/x/net/html"
)

var log = &common.Log
var appDir string
var cacheDir string

var siteCacheDir string
var sceneCacheDir string

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"

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

func trimSpaceFromSlice(s []string) []string {
	for i := range s {
      s[i] = strings.TrimSpace(s[i])
	}
	return s
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

func init() {
	appDir = appdir.New("xbvr").UserConfig()

	cacheDir = filepath.Join(appDir, "cache")

	siteCacheDir = filepath.Join(cacheDir, "site_cache")
	sceneCacheDir = filepath.Join(cacheDir, "scene_cache")

	_ = os.MkdirAll(appDir, os.ModePerm)
	_ = os.MkdirAll(cacheDir, os.ModePerm)

	_ = os.MkdirAll(siteCacheDir, os.ModePerm)
	_ = os.MkdirAll(sceneCacheDir, os.ModePerm)
}
