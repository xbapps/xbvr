package scrape

import (
	"math/rand"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

// Colly provides Rate Limiting on collectors and this works in most scrapers.
// For scrapers that handle multiple sites, eg SLR, VRPorn, this does not work, as each
// each site creates it's own instance of the scraper with it's own colly collector, and it's own independent limits

// The ScraperRateLimiter is provides a way to limit visits across multiple instances of the same scraper.
// The calls to the colly collector Visit function must first be passed to the ScraperRateLimiter which will then coridinate
// between all instances and then call the colly Visit function
var Limiters map[string]*ScraperRateLimiter

type ScraperRateLimiter struct {
	mutex       sync.Mutex
	lastRequest time.Time
	minDelay    time.Duration
	maxDelay    time.Duration
}

func CreateRateLimiter(minDelay time.Duration, maxDelay time.Duration) *ScraperRateLimiter {
	return &ScraperRateLimiter{
		minDelay: minDelay,
		maxDelay: maxDelay,
		//		response: true,
	}
}

func ScraperRateLimiterWait(rateLimiter string) {
	if Limiters[rateLimiter] == nil {
		return
	}
	Limiters[rateLimiter].mutex.Lock()
	defer Limiters[rateLimiter].mutex.Unlock()

	if Limiters[rateLimiter].lastRequest.IsZero() {
		// no previous time, don't wait
		Limiters[rateLimiter].lastRequest = time.Now()
		return
	}
	timeSinceLast := time.Since(Limiters[rateLimiter].lastRequest)

	delay := Limiters[rateLimiter].minDelay
	if Limiters[rateLimiter].maxDelay > Limiters[rateLimiter].minDelay {
		// Introduce a random delay between minDelay and maxDelay
		delay += time.Duration(rand.Int63n(int64(Limiters[rateLimiter].maxDelay - Limiters[rateLimiter].minDelay)))
	}
	if timeSinceLast < delay {
		time.Sleep(delay - timeSinceLast)
	}
	Limiters[rateLimiter].lastRequest = time.Now()
}

func WaitBeforeVisit(rateLimiter string, visitFunc func(string) error, pageURL string) {
	ScraperRateLimiterWait(rateLimiter)
	err := visitFunc(pageURL)
	if err != nil {
		// if an err is returned, then a html a call was not made by colly.  These are errors colly checks before calling the URL
		//		ie the url has not been called.  No need to wait before the next call, as the site was never visited
		if Limiters[rateLimiter] != nil {
			Limiters[rateLimiter].lastRequest = time.Time{}
		}
	}
}
func ScraperRateLimiterCheckErrors(domain string, err error) {
	if err != nil {
		Limiters[domain].lastRequest = time.Time{}
	}

}
func AddScraperRateLimiter(key string, l *ScraperRateLimiter) {
	if Limiters == nil {
		LoadScraperRateLimits()
	}
	Limiters[key] = l
}

func LoadScraperRateLimits() {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	limiters := make(map[string]*ScraperRateLimiter)
	commonDb, _ := models.GetCommonDB()
	var kv models.KV
	commonDb.Where(models.KV{Key: "scraper_rate_limits"}).Find(&kv)
	if kv.Key == "scraper_rate_limits" {
		sites := gjson.Get(kv.Value, "sites")
		for _, site := range sites.Array() {
			name := site.Get("name").String()
			minDelay := int(site.Get("mindelay").Int())
			maxDelay := int(site.Get("maxdelay").Int())
			if maxDelay < minDelay {
				maxDelay = minDelay
			}
			limiters[name] = &ScraperRateLimiter{minDelay: time.Duration(minDelay) * time.Millisecond, maxDelay: time.Duration(maxDelay) * time.Millisecond}
		}

		Limiters = limiters
	}
}
