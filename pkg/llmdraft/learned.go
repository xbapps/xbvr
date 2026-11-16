package llmdraft

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/xbapps/xbvr/pkg/models"
)

// Download and piracy sites are only recognised after a page has been fetched and read, so each
// one costs a slot among the pages read per file. Domains the model classifies as such are
// remembered and, once seen often enough, skipped straight from the search results.

const (
	learnedKey = "llmscrape_download_domains"
	// learnAfter is how many pages of a domain must be classified as download or piracy before
	// the domain is skipped, so a single misjudgement cannot block a legitimate store.
	learnAfter = 2
)

// LearnedDomain is a domain recognised as a download or piracy site.
type LearnedDomain struct {
	Domain     string    `json:"domain"`
	Count      int       `json:"count"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	ExampleURL string    `json:"example_url"`
	// Allowed is set by the user for a domain that was misjudged; it is then never skipped.
	Allowed bool `json:"allowed"`
	Blocked bool `json:"blocked"` // derived, for display
}

var learnedMu sync.Mutex

// registrableDomain returns the domain someone registered, so cdn1.example.co.uk and
// www.example.co.uk both count as example.co.uk.
func registrableDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ""
	}
	if d, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return d
	}
	return strings.TrimPrefix(host, "www.")
}

func loadLearned() map[string]*LearnedDomain {
	db, _ := models.GetDB()
	defer db.Close()
	m := map[string]*LearnedDomain{}
	var kv models.KV
	if db.Where(&models.KV{Key: learnedKey}).First(&kv).Error == nil {
		json.Unmarshal([]byte(kv.Value), &m)
	}
	return m
}

func saveLearned(m map[string]*LearnedDomain) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	db, _ := models.GetDB()
	defer db.Close()
	kv := models.KV{Key: learnedKey, Value: string(b)}
	return models.RetryIfBusy(func() error { return db.Save(&kv).Error })
}

// RecordDownloadSite notes that a page was classified as a download or piracy site.
func RecordDownloadSite(pageURL string) {
	domain := registrableDomain(pageURL)
	if domain == "" {
		return
	}
	learnedMu.Lock()
	defer learnedMu.Unlock()
	m := loadLearned()
	now := time.Now()
	d, ok := m[domain]
	if !ok {
		d = &LearnedDomain{Domain: domain, FirstSeen: now, ExampleURL: pageURL}
		m[domain] = d
	}
	d.Count++
	d.LastSeen = now
	saveLearned(m)
}

// blockedDomains returns the domains to skip from a learned set.
func blockedDomains(m map[string]*LearnedDomain) []string {
	var out []string
	for _, d := range m {
		if !d.Allowed && d.Count >= learnAfter {
			out = append(out, d.Domain)
		}
	}
	sort.Strings(out)
	return out
}

// LearnedBlocked returns the learned download and piracy domains that are being skipped.
func LearnedBlocked() []string {
	learnedMu.Lock()
	defer learnedMu.Unlock()
	return blockedDomains(loadLearned())
}

// ListLearned returns every learned domain, most often seen first.
func ListLearned() []LearnedDomain {
	learnedMu.Lock()
	defer learnedMu.Unlock()
	m := loadLearned()
	out := make([]LearnedDomain, 0, len(m))
	for _, d := range m {
		v := *d
		v.Blocked = !d.Allowed && d.Count >= learnAfter
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}

// SetLearnedAllowed marks a learned domain as allowed (never skipped) or not.
func SetLearnedAllowed(domain string, allowed bool) bool {
	learnedMu.Lock()
	defer learnedMu.Unlock()
	m := loadLearned()
	d, ok := m[strings.ToLower(domain)]
	if !ok {
		return false
	}
	d.Allowed = allowed
	saveLearned(m)
	return true
}
