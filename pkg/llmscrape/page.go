package llmscrape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	maxPageBytes      = 5 << 20
	maxImageCands     = 60
	maxVideoCands     = 12
	maxLinkCands      = 40
	maxStructuredData = 6000
)

// Candidate is an image or video URL found on a page, offered to the LLM by index so it can only
// choose URLs that really exist on the page.
type Candidate struct {
	URL  string `json:"url"`
	Hint string `json:"hint,omitempty"` // where it came from and any alt text
}

// Page is what the LLM is shown: a condensed, deterministic extraction of a fetched page.
type Page struct {
	URL            string // final URL after redirects
	CanonicalURL   string
	Title          string
	Meta           map[string]string
	StructuredData string // JSON-LD, compacted
	Text           string // visible text
	Images         []Candidate
	Videos         []Candidate
	Links          []Candidate // other pages this page links to, offered so the LLM can follow one
}

// Fetcher downloads pages. It refuses private network addresses unless allowed, checked at
// connect time so a hostname cannot be re-pointed at an internal address after validation.
type Fetcher struct {
	http      *http.Client
	userAgent string
}

// NewFetcher returns a fetcher with a browser user agent.
func NewFetcher(userAgent string, allowPrivate bool, timeout time.Duration) *Fetcher {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	if !allowPrivate {
		dialer.Control = refusePrivateAddresses
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext
	transport.Proxy = nil // a proxy would make the address check meaningless
	return &Fetcher{
		userAgent: userAgent,
		http: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 8 {
					return errors.New("too many redirects")
				}
				return checkScheme(req.URL)
			},
		},
	}
}

func checkScheme(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	return nil
}

func refusePrivateAddresses(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("refusing unresolved address %s", host)
	}
	if isPrivateIP(ip) {
		return fmt.Errorf("refusing to fetch from private address %s", ip)
	}
	return nil
}

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() || cgnat.Contains(ip)
}

// Fetch downloads a page and extracts what the LLM needs from it.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string, maxTextChars int) (*Page, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if err := checkScheme(u); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Many adult sites gate content behind an age check stored in a cookie; these are the
	// common names. Sites that gate with JavaScript will still show the gate.
	req.Header.Set("Cookie", "age_verified=1; ageVerified=1; age_confirmed=1; agreedToTerms=1; over18=1; is_adult=1")

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", u, resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "html") {
		return nil, fmt.Errorf("fetching %s: not an HTML page (%s)", u, ct)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", u, err)
	}
	return ExtractPage(resp.Request.URL, string(body), maxTextChars)
}

// ExtractPage condenses HTML into the parts useful for identifying a scene. It is deterministic
// and does no network access, so it can be tested against saved pages.
func ExtractPage(pageURL *url.URL, html string, maxTextChars int) (*Page, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	if maxTextChars <= 0 {
		maxTextChars = 12000
	}

	base := pageURL
	if href, ok := doc.Find("base[href]").First().Attr("href"); ok {
		if b, err := pageURL.Parse(href); err == nil {
			base = b
		}
	}

	p := &Page{URL: pageURL.String(), Meta: map[string]string{}}
	p.Title = collapse(doc.Find("title").First().Text())

	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		key := strings.ToLower(firstAttr(s, "property", "name", "itemprop"))
		val := collapse(s.AttrOr("content", ""))
		if key == "" || val == "" {
			return
		}
		if strings.HasPrefix(key, "og:") || strings.HasPrefix(key, "twitter:") || strings.HasPrefix(key, "video:") ||
			key == "description" || key == "keywords" || key == "author" || key == "duration" || key == "uploaddate" {
			if _, seen := p.Meta[key]; !seen {
				p.Meta[key] = truncate(val, 500)
			}
		}
	})
	if canon, ok := doc.Find(`link[rel="canonical"]`).First().Attr("href"); ok {
		p.CanonicalURL = resolve(base, canon)
	}
	if p.CanonicalURL == "" && p.Meta["og:url"] != "" {
		p.CanonicalURL = resolve(base, p.Meta["og:url"])
	}

	images := newCandidateSet(maxImageCands)
	videos := newCandidateSet(maxVideoCands)
	links := newCandidateSet(maxLinkCands)

	// Social preview images are usually the cover, so they go first.
	for _, k := range []string{"og:image", "og:image:secure_url", "twitter:image", "twitter:image:src"} {
		images.add(resolve(base, p.Meta[k]), k)
	}
	for _, k := range []string{"og:video", "og:video:url", "og:video:secure_url", "twitter:player:stream"} {
		videos.add(resolve(base, p.Meta[k]), k)
	}

	var ld []string
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		raw := strings.TrimSpace(s.Text())
		var v any
		if json.Unmarshal([]byte(raw), &v) != nil {
			return
		}
		walkJSONLD(v, base, images, videos)
		if b, err := json.Marshal(v); err == nil {
			ld = append(ld, string(b))
		}
	})
	p.StructuredData = truncate(strings.Join(ld, "\n"), maxStructuredData)

	doc.Find(`link[rel="image_src"]`).Each(func(_ int, s *goquery.Selection) {
		images.add(resolve(base, s.AttrOr("href", "")), "image_src")
	})
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		hint := "img"
		if alt := collapse(s.AttrOr("alt", "")); alt != "" {
			hint += ` alt="` + truncate(alt, 80) + `"`
		}
		for _, attr := range []string{"data-src", "data-lazy-src", "data-original", "src"} {
			if v := s.AttrOr(attr, ""); v != "" && !strings.HasPrefix(v, "data:") {
				images.add(resolve(base, v), hint)
				break
			}
		}
		if best := largestFromSrcset(s.AttrOr("srcset", s.AttrOr("data-srcset", ""))); best != "" {
			images.add(resolve(base, best), hint+" srcset")
		}
	})
	doc.Find("picture source[srcset]").Each(func(_ int, s *goquery.Selection) {
		if best := largestFromSrcset(s.AttrOr("srcset", "")); best != "" {
			images.add(resolve(base, best), "picture")
		}
	})
	doc.Find("[style*='background-image']").Each(func(_ int, s *goquery.Selection) {
		if m := bgImageRe.FindStringSubmatch(s.AttrOr("style", "")); m != nil {
			images.add(resolve(base, m[1]), "background")
		}
	})
	doc.Find("video[poster]").Each(func(_ int, s *goquery.Selection) {
		images.add(resolve(base, s.AttrOr("poster", "")), "video poster")
	})
	doc.Find("video[src], video source[src]").Each(func(_ int, s *goquery.Selection) {
		videos.add(resolve(base, s.AttrOr("src", "")), "video")
	})
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href := resolve(base, s.AttrOr("href", ""))
		if videoURLRe.MatchString(href) {
			videos.add(href, "link")
			return
		}
		// Links in page furniture are navigation, not pages about this scene, and they crowd
		// out the scene links on listing pages.
		if href == "" || imageExtRe.MatchString(href) || s.Closest("nav, header, footer, aside").Length() > 0 {
			return
		}
		hint := collapse(s.Text())
		if hint == "" {
			hint = collapse(s.Find("img[alt]").First().AttrOr("alt", ""))
		}
		if hint == "" {
			hint = "link"
		}
		links.add(href, truncate(hint, 120))
	})

	p.Images = images.list(func(u string) bool { return isLikelyContentImage(u) && !IsExpiringURL(u) })
	p.Videos = videos.list(func(u string) bool { return !IsExpiringURL(u) })
	p.Links = links.list(nil)

	doc.Find("script, style, noscript, svg, iframe, template").Remove()
	p.Text = truncate(collapse(doc.Find("body").Text()), maxTextChars)
	return p, nil
}

var (
	bgImageRe  = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)
	videoURLRe = regexp.MustCompile(`(?i)\.(mp4|m3u8|webm|mov)(\?|$)`)
	imageExtRe = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|avif)(\?|$)`)
	junkPathRe = regexp.MustCompile(`(?i)(logo|icon|favicon|sprite|avatar|badge|flag|pixel|spacer|placeholder|loading|banner-ad|emoji)`)
	ws         = regexp.MustCompile(`\s+`)
)

// expiringParams are query parameters that sign a URL for a limited time. A library stores URLs
// for years, so a signed URL would stop working soon after it was saved.
var expiringParams = map[string]bool{
	"ttl": true, "token": true, "expires": true, "expiry": true, "exp": true, "signature": true, "sig": true,
	"x-amz-signature": true, "x-amz-expires": true, "x-goog-signature": true, "x-goog-expires": true,
	"hdnts": true, "hdnea": true, "policy": true, "key-pair-id": true, "validfrom": true, "validto": true,
}

// IsExpiringURL reports whether u carries a time-limited signature.
func IsExpiringURL(u string) bool {
	pu, err := url.Parse(u)
	if err != nil {
		return false
	}
	for k := range pu.Query() {
		if expiringParams[strings.ToLower(k)] {
			return true
		}
	}
	return false
}

// isLikelyContentImage drops logos, icons and tracking pixels, which crowd out the cover and
// stills in the list the LLM chooses from.
func isLikelyContentImage(u string) bool {
	if strings.HasSuffix(strings.ToLower(strings.SplitN(u, "?", 2)[0]), ".svg") {
		return false
	}
	path := u
	if pu, err := url.Parse(u); err == nil {
		path = pu.Path
	}
	return !junkPathRe.MatchString(path)
}

func walkJSONLD(v any, base *url.URL, images, videos *candidateSet) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			switch strings.ToLower(k) {
			case "image", "thumbnailurl", "thumbnail":
				for _, s := range jsonStrings(val) {
					images.add(resolve(base, s), "structured data "+k)
				}
			case "contenturl", "embedurl":
				for _, s := range jsonStrings(val) {
					abs := resolve(base, s)
					switch {
					case videoURLRe.MatchString(abs):
						videos.add(abs, "structured data "+k)
					case imageExtRe.MatchString(abs):
						images.add(abs, "structured data "+k)
					}
				}
			}
			walkJSONLD(val, base, images, videos)
		}
	case []any:
		for _, item := range t {
			walkJSONLD(item, base, images, videos)
		}
	}
}

// jsonStrings collects URL strings from a JSON-LD value, which may be a string, an ImageObject
// with a url, or a list of either.
func jsonStrings(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case map[string]any:
		if s, ok := t["url"].(string); ok {
			return []string{s}
		}
		if s, ok := t["contentUrl"].(string); ok {
			return []string{s}
		}
	case []any:
		var out []string
		for _, item := range t {
			out = append(out, jsonStrings(item)...)
		}
		return out
	}
	return nil
}

func largestFromSrcset(srcset string) string {
	best, bestW := "", -1.0
	for _, part := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		w := 0.0
		if len(fields) > 1 {
			d := strings.TrimRight(strings.TrimRight(fields[1], "w"), "x")
			w, _ = strconv.ParseFloat(d, 64)
		}
		if w > bestW {
			best, bestW = fields[0], w
		}
	}
	return best
}

func resolve(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "javascript:") || strings.HasPrefix(ref, "#") {
		return ""
	}
	if strings.HasPrefix(ref, "//") {
		ref = base.Scheme + ":" + ref
	}
	u, err := base.Parse(ref)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func firstAttr(s *goquery.Selection, names ...string) string {
	for _, n := range names {
		if v, ok := s.Attr(n); ok && v != "" {
			return v
		}
	}
	return ""
}

func collapse(s string) string { return strings.TrimSpace(ws.ReplaceAllString(s, " ")) }

type candidateSet struct {
	max   int
	order []string
	hints map[string]string
}

func newCandidateSet(max int) *candidateSet {
	return &candidateSet{max: max, hints: map[string]string{}}
}

func (c *candidateSet) add(u, hint string) {
	if u == "" {
		return
	}
	if _, seen := c.hints[u]; seen {
		return
	}
	c.hints[u] = hint
	c.order = append(c.order, u)
}

func (c *candidateSet) list(keep func(string) bool) []Candidate {
	var out []Candidate
	for _, u := range c.order {
		if keep != nil && !keep(u) {
			continue
		}
		out = append(out, Candidate{URL: u, Hint: c.hints[u]})
		if len(out) >= c.max {
			break
		}
	}
	return out
}

// SortedMetaKeys returns meta keys in a stable order for prompts and tests.
func (p *Page) SortedMetaKeys() []string {
	keys := make([]string, 0, len(p.Meta))
	for k := range p.Meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
