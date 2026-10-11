package scrape

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
)

// Custom R18.dev Scrapers: what the "Create Custom Site (JAV)" tab needs. Custom Scrapers live in the
// "custom" -> "r18dev" part of scrapers.json in the app dir and load at startup like every other
// Scraper, so saving one takes a restart.

// r18devLoadedCustom: the custom Scrapers as they were loaded at startup, to tell the form which
// saved ones still need a restart.
var r18devLoadedCustom = map[string]config.R18devScraperConfig{}

var r18devSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// R18devSlug turns a name into a Scraper ID: "V:one VR" -> "v-one-vr".
func R18devSlug(name string) string {
	return strings.Trim(r18devSlugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// R18devSourceFromURL reads an R18.dev list link:
// https://r18.dev/videos/vod/movies/list/?id=46483&type=studio (also label, category and list2/json).
func R18devSourceFromURL(link string) (config.R18devSource, error) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || !strings.HasSuffix(strings.ToLower(u.Hostname()), "r18.dev") {
		return config.R18devSource{}, fmt.Errorf("not an R18.dev link")
	}
	q := u.Query()
	srcType := strings.ToLower(q.Get("type"))
	id, err := strconv.ParseInt(q.Get("id"), 10, 64)
	if !strings.Contains(u.Path, "/list") || err != nil || id <= 0 {
		return config.R18devSource{}, fmt.Errorf("not an R18.dev list link: open a studio, label or category on R18.dev and copy that link")
	}
	if srcType != "label" && srcType != "studio" && srcType != "category" {
		return config.R18devSource{}, fmt.Errorf("list type %q: only studio, label and category lists work", srcType)
	}
	return config.R18devSource{Type: srcType, ID: id}, nil
}

// R18devListPreview is page 1 of an R18.dev list, boiled down for the form.
type R18devListPreview struct {
	Type     string   `json:"type"`
	ID       int64    `json:"id"`
	Total    int      `json:"total"`
	Maker    string   `json:"maker"`    // maker of the newest scene: a name suggestion
	Prefixes []string `json:"prefixes"` // DVD prefixes on page 1, most common first
	Newest   struct {
		ContentID string `json:"content_id"`
		DisplayID string `json:"display_id"`
		Released  string `json:"released"`
		Thumb     string `json:"thumb"`
	} `json:"newest"`
}

// R18devPreview fetches page 1 of a list (one request, through the same pacing as the crawler).
func R18devPreview(src config.R18devSource) (R18devListPreview, error) {
	p := R18devListPreview{Type: src.Type, ID: src.ID, Prefixes: []string{}}
	c := newR18devClient()
	link := fmt.Sprintf("%s/list2/json?id=%d&type=%s&page=1", r18devBaseURL, src.ID, src.Type)
	res, err := c.get(link)
	if err != nil {
		return p, err
	}
	if res.StatusCode() != 200 {
		return p, fmt.Errorf("R18.dev answered HTTP %d", res.StatusCode())
	}
	body := res.String()
	p.Total = int(gjson.Get(body, "total_results").Int())

	counts := map[string]int{}
	for i, r := range gjson.Get(body, "results").Array() {
		cid := strings.ToLower(r.Get("content_id").String())
		display := ""
		if dvd := r.Get("dvd_id").String(); dvd != "" && dvd != "----" {
			display = r18devNormalizeDVDID(dvd)
		} else {
			display = r18devDisplayID(cid)
		}
		if i == 0 {
			p.Maker = r.Get("maker_name_en").String()
			p.Newest.ContentID = cid
			p.Newest.DisplayID = display
			p.Newest.Released = strings.TrimSpace(strings.Split(r.Get("release_date").String(), " ")[0])
			p.Newest.Thumb = r.Get("jacket_thumb_url").String()
		}
		if prefix := r18devDisplayPrefix(display); prefix != "" && !r18devCompilationPrefixes[prefix] {
			counts[prefix]++
		}
	}
	for prefix := range counts {
		p.Prefixes = append(p.Prefixes, prefix)
	}
	sort.Slice(p.Prefixes, func(i, j int) bool {
		a, b := p.Prefixes[i], p.Prefixes[j]
		return counts[a] > counts[b] || (counts[a] == counts[b] && a < b)
	})
	return p, nil
}

// R18devCheckCustom cleans up a custom Scraper from the form and says what's wrong with it, if anything.
// official: the shipped presets (their IDs are taken).
func R18devCheckCustom(cfg *config.R18devScraperConfig, official []config.R18devScraperConfig) error {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Company = strings.TrimSpace(cfg.Company)
	cfg.AvatarUrl = strings.TrimSpace(cfg.AvatarUrl)
	cfg.SitesDisplay = strings.TrimSpace(cfg.SitesDisplay)
	if cfg.ID == "" {
		cfg.ID = R18devSlug(cfg.Name)
	}
	cfg.ID = R18devSlug(cfg.ID)
	if cfg.Name == "" || cfg.ID == "" {
		return fmt.Errorf("a name is needed")
	}
	if cfg.Company == "" {
		cfg.Company = cfg.Name
	}
	if cfg.ID == strings.TrimPrefix(r18devSingleSceneID, "r18dev-") {
		return fmt.Errorf("the ID %q is reserved", cfg.ID)
	}
	for _, o := range official {
		if strings.EqualFold(o.ID, cfg.ID) {
			return fmt.Errorf("the ID %q belongs to the %s preset, pick another name", cfg.ID, o.Name)
		}
	}
	for _, src := range cfg.Sources {
		if (src.Type != "label" && src.Type != "studio" && src.Type != "category") || src.ID <= 0 {
			return fmt.Errorf("source %s %d: only studio, label and category lists work", src.Type, src.ID)
		}
	}
	clean := func(list []string, upper bool) []string {
		out := []string{}
		seen := map[string]bool{}
		for _, s := range list {
			s = strings.TrimSpace(s)
			if upper {
				s = strings.ToUpper(s)
			} else {
				s = strings.ToLower(s)
			}
			if s != "" && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
		return out
	}
	cfg.Sites = clean(cfg.Sites, true)
	cfg.OldSites = clean(cfg.OldSites, true)
	cfg.ContentIDs = clean(cfg.ContentIDs, false)
	if len(cfg.ContentIDs) == 0 {
		cfg.ContentIDs = nil
	}
	if len(cfg.OldSites) == 0 {
		cfg.OldSites = nil
	}
	if len(cfg.Sources) == 0 && len(cfg.ContentIDs) == 0 {
		return fmt.Errorf("add at least one R18.dev list link")
	}
	// Quality modes are for presets only
	cfg.AvatarText, cfg.Exclusive = "", false
	return nil
}

// R18devCustomStatus: "loaded", "restart" (saved but not loaded yet, or changed since) or "removed"
// (deleted, still loaded until a restart).
func R18devCustomStatus(cfg config.R18devScraperConfig) string {
	loaded, ok := r18devLoadedCustom[strings.ToLower(cfg.ID)]
	if !ok {
		return "restart"
	}
	a, _ := json.Marshal(loaded)
	b, _ := json.Marshal(cfg)
	if string(a) != string(b) {
		return "restart"
	}
	return "loaded"
}

// R18devLoadedCustomIDs: IDs of the custom Scrapers loaded at startup.
func R18devLoadedCustomIDs() []string {
	ids := []string{}
	for id := range r18devLoadedCustom {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
