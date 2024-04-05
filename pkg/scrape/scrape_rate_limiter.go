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
var Limiters []*ScraperRateLimiter

type ScraperRateLimiter struct {
	id          string
	mutex       sync.Mutex
	lastRequest time.Time
	minDelay    time.Duration
	maxDelay    time.Duration
}

func ScraperRateLimiterWait(rateLimiter string) {
	limiter := GetRateLimiter(rateLimiter)
	if limiter == nil {
		return
	}
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	if limiter.lastRequest.IsZero() {
		// no previous time, don't wait
		limiter.lastRequest = time.Now()
		return
	}
	timeSinceLast := time.Since(limiter.lastRequest)

	delay := limiter.minDelay
	if limiter.maxDelay > limiter.minDelay {
		// Introduce a random delay between minDelay and maxDelay
		delay += time.Duration(rand.Int63n(int64(limiter.maxDelay - limiter.minDelay)))
	}
	if timeSinceLast < delay {
		time.Sleep(delay - timeSinceLast)
	}
	limiter.lastRequest = time.Now()
}

func WaitBeforeVisit(rateLimiter string, visitFunc func(string) error, pageURL string) {
	ScraperRateLimiterWait(rateLimiter)
	err := visitFunc(pageURL)
	if err != nil {
		// if an err is returned, then a html a call was not made by colly.  These are errors colly checks before calling the URL
		//		ie the url has not been called.  No need to wait before the next call, as the site was never visited
		limiter := GetRateLimiter(rateLimiter)
		if limiter != nil {
			limiter.lastRequest = time.Time{}
		}
	}
}
func ScraperRateLimiterCheckErrors(domain string, err error) {
	if err != nil {
		limiter := GetRateLimiter(domain)
		if limiter != nil {
			limiter.lastRequest = time.Time{}
		}
	}
}

func LoadScraperRateLimits() {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	var limiters []*ScraperRateLimiter
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
			limiters = append(limiters, &ScraperRateLimiter{id: name, minDelay: time.Duration(minDelay) * time.Millisecond, maxDelay: time.Duration(maxDelay) * time.Millisecond})
		}
		Limiters = limiters
	}
}

func GetRateLimiter(id string) *ScraperRateLimiter {
	for _, limiter := range Limiters {
		if limiter.id == id {
			return limiter
			break
		}
	}
	return nil
}
