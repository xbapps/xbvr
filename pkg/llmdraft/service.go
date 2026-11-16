// Package llmdraft proposes XBVR scenes for unmatched files: it finds candidate pages with a web
// search, extracts them with llmscrape, and holds the results as draft scenes for review.
package llmdraft

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mozillazg/go-slugify"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/llmscrape"
	"github.com/xbapps/xbvr/pkg/models"
)

// Settings controls how search results are turned into drafts.
type Settings struct {
	MaxPageChars      int
	ResultsPerFile    int
	MinConfidence     float64
	Concurrency       int
	MaxFollowRounds   int // links to follow from overview pages before giving up
	SkipDownloadSites bool
	BlockedDomains    []string
}

// rejectBelow is the confidence under which a search result is judged to be a different scene
// and not proposed at all. Pages the user chooses are always kept.
const rejectBelow = 0.15

// Service is the LLM scraper wired to XBVR's database.
type Service struct {
	llm      *llmscrape.Client
	fetcher  *llmscrape.Fetcher
	searcher llmscrape.Searcher // nil when no search provider is configured
	settings Settings
}

// NewService builds a service from XBVR's configuration. Pages are fetched with userAgent.
func NewService(userAgent string) (*Service, error) {
	c := config.Config.LLMScraper
	llm, err := llmscrape.NewClient(llmscrape.LLMConfig{
		BaseURL:          c.BaseURL,
		Model:            c.Model,
		APIKey:           c.APIKey,
		DisableThinking:  c.DisableThinking,
		StructuredOutput: c.StructuredOutput,
		Timeout:          time.Duration(c.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	s := &Service{
		llm:     llm,
		fetcher: llmscrape.NewFetcher(userAgent, c.AllowPrivateNetworks, 30*time.Second),
		settings: Settings{
			MaxPageChars:      c.MaxPageChars,
			ResultsPerFile:    clampInt(c.ResultsPerFile, 1, 20, 5),
			MinConfidence:     c.MinConfidence,
			Concurrency:       clampInt(c.Concurrency, 1, 8, 2),
			MaxFollowRounds:   clampRounds(c.MaxFollowRounds),
			SkipDownloadSites: c.SkipDownloadSites,
			BlockedDomains:    ParseDomains(c.BlockedDomains),
		},
	}
	if c.BraveAPIKey != "" {
		s.searcher = llmscrape.NewBraveSearch(c.BraveAPIKey)
	}
	return s, nil
}

func clampInt(v, lo, hi, def int) int {
	if v <= 0 {
		return def
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// clampRounds bounds the follow-up hops. Zero is a valid setting (never follow a link).
func clampRounds(v int) int {
	if v < 0 {
		return 0
	}
	if v > 5 {
		return 5
	}
	return v
}

// ParseDomains splits a comma or newline separated domain list.
func ParseDomains(s string) []string {
	var out []string
	for _, d := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' || r == ';' }) {
		d = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(d), "www."))
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

// DomainBlocked reports whether rawURL's host is one of domains or a subdomain of one.
func DomainBlocked(rawURL string, domains []string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	for _, d := range domains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// ScrapeResult is the outcome for one page.
type ScrapeResult struct {
	URL     string             `json:"url"`
	Draft   *models.DraftScene `json:"draft,omitempty"`
	Skipped string             `json:"skipped,omitempty"` // why no draft was kept
	Error   string             `json:"error,omitempty"`
}

// ScrapeURL turns one page into a saved draft scene. keepAny is set for pages the user chose, which
// become drafts even when the LLM doubts they are scene pages; search results are filtered.
func (s *Service) ScrapeURL(ctx context.Context, rawURL string, fileID uint, file *llmscrape.FileInfo, query string, keepAny bool) ScrapeResult {
	res := s.scrapePage(ctx, rawURL, fileID, file, query, keepAny)
	if res.Draft != nil {
		if err := res.Draft.Save(); err != nil {
			res.Draft = nil
			res.Error = "saving draft: " + err.Error()
		}
	}
	return res
}

// scrapePage builds a draft from a page without saving it. When the page is an overview or
// listing rather than the scene itself, its extraction can point at a link to the scene's own
// page; that page is fetched and read instead, up to Settings.MaxFollowRounds times.
func (s *Service) scrapePage(ctx context.Context, rawURL string, fileID uint, file *llmscrape.FileInfo, query string, keepAny bool) ScrapeResult {
	res := ScrapeResult{URL: rawURL}
	if DomainBlocked(rawURL, s.settings.BlockedDomains) {
		res.Skipped = "domain is blocked in the LLM scraper settings"
		return res
	}

	current := rawURL
	visited := map[string]bool{}
	var page *llmscrape.Page
	var ext *llmscrape.Extraction
	for round := 0; ; round++ {
		visited[current] = true
		p, err := s.fetcher.Fetch(ctx, current, s.settings.MaxPageChars)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		raw, err := s.llm.Complete(ctx, llmscrape.BuildMessages(p, file), "xbvr_scene",
			llmscrape.ExtractionSchema(len(p.Images), len(p.Videos), len(p.Links)), 4000)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		e, err := llmscrape.ParseExtraction(raw, p)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		llmscrape.ApplyDurationCheck(e, file)
		llmscrape.ApplyNameCheck(e, file)
		if e.PageKind == llmscrape.KindDownload {
			RecordDownloadSite(p.URL)
		}
		page, ext = p, e

		if round >= s.settings.MaxFollowRounds || e.FollowLink < 0 || e.FollowLink >= len(p.Links) {
			break
		}
		next := p.Links[e.FollowLink].URL
		if next == "" || visited[next] || DomainBlocked(next, s.settings.BlockedDomains) {
			break
		}
		current = next
	}
	res.URL = page.URL

	if !keepAny {
		switch {
		case !ext.IsScenePage && !ext.IsParentPage:
			res.Skipped = "not a page about a single scene: " + ext.Reason
			return res
		case s.settings.SkipDownloadSites && ext.PageKind == llmscrape.KindDownload:
			res.Skipped = "download or piracy site"
			return res
		case ext.MatchConfidence < rejectBelow:
			res.Skipped = fmt.Sprintf("a different scene (%.0f%%): %s", ext.MatchConfidence*100, ext.Reason)
			return res
		}
	}

	draft := &models.DraftScene{
		FileID:          fileID,
		SourceURL:       page.URL,
		Query:           query,
		Status:          models.DraftStatusDraft,
		MatchConfidence: ext.MatchConfidence,
		PageKind:        ext.PageKind,
		Reason:          ext.Reason,
		Model:           s.llm.Model(),
	}
	if err := draft.SetScene(BuildScrapedScene(page, ext, file)); err != nil {
		res.Error = err.Error()
		return res
	}
	res.Draft = draft
	return res
}

// BuildScrapedScene converts an extraction into the structure every XBVR site scraper produces.
func BuildScrapedScene(page *llmscrape.Page, ext *llmscrape.Extraction, file *llmscrape.FileInfo) models.ScrapedScene {
	home := page.CanonicalURL
	if home == "" {
		home = page.URL
	}
	// A store or aggregator page names itself as the site; the scene belongs to the studio's
	// brand, which is how XBVR's own scrapers for such hosts file their scenes.
	site := ext.Site
	if ext.PageKind != llmscrape.KindOfficial {
		site = firstNonEmpty(ext.Studio, ext.Site)
	}
	site = brandName(firstNonEmpty(site, ext.Studio, hostOf(home)))
	studio := brandName(firstNonEmpty(ext.Studio, site))

	// Stable across runs, so scraping the same page again updates the scene rather than
	// duplicating it. The prefix keeps these apart from IDs owned by XBVR's site scrapers.
	idPart := ext.SiteSceneID
	if idPart == "" {
		sum := sha1.Sum([]byte(home))
		idPart = hex.EncodeToString(sum[:])[:10]
	}
	title := ext.Title
	// A page about a whole game or series serves every part of it, so the part names the scene,
	// and each part needs its own ID or saving a second ending would overwrite the first.
	if ext.IsParentPage && ext.PartTitle != "" {
		idPart += "-" + ext.PartTitle
		if !strings.Contains(strings.ToLower(title), strings.ToLower(ext.PartTitle)) {
			title = strings.TrimSpace(title + " - " + ext.PartTitle)
		}
	}

	scene := models.ScrapedScene{
		SceneID:     slugify.Slugify("llm-" + site + "-" + idPart),
		SiteID:      idPart,
		SceneType:   "VR",
		Title:       title,
		Studio:      studio,
		Site:        site,
		Cast:        ext.Cast,
		Tags:        ext.Tags,
		Synopsis:    ext.Synopsis,
		Released:    ext.Released,
		Duration:    ext.DurationMinutes,
		HomepageURL: home,
	}
	if scene.Title == "" {
		scene.Title = page.Title
	}
	for _, i := range ext.GalleryImages {
		scene.Gallery = append(scene.Gallery, page.Images[i].URL)
	}
	// Covers must stay nil rather than empty: the scene import reads Covers[0] whenever the
	// slice is non-nil. Without a cover the best still stands in, rather than a blank scene.
	switch {
	case ext.CoverImage >= 0:
		scene.Covers = []string{page.Images[ext.CoverImage].URL}
	case len(scene.Gallery) > 0:
		scene.Covers = []string{scene.Gallery[0]}
		scene.Gallery = scene.Gallery[1:]
	}
	if ext.Trailer >= 0 {
		scene.TrailerType = "url"
		scene.TrailerSrc = page.Videos[ext.Trailer].URL
	}
	if file != nil && file.Filename != "" {
		scene.Filenames = []string{file.Filename}
	}
	return scene
}

var domainLikeRe = regexp.MustCompile(`(?i)^([a-z0-9][a-z0-9-]*)\.(com|net|org|xxx|tv|io|co|eu|vr|porn|app)$`)

// brandName turns a bare domain such as "Dezyred.com" into the brand "Dezyred".
func brandName(s string) string {
	s = strings.TrimSpace(s)
	if m := domainLikeRe.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

// SuggestResult is the outcome of searching for drafts for one file.
type SuggestResult struct {
	FileID  uint           `json:"file_id"`
	Query   string         `json:"query"`
	Results []ScrapeResult `json:"results"`
	// KnownDownloadSites counts search results skipped without being read because their domain
	// has already been recognised as a download or piracy site.
	KnownDownloadSites int `json:"known_download_sites"`
}

// SuggestForFile searches the web for the file's scene and turns the promising pages into drafts.
// Pages already drafted for this file, including rejected ones, are not scraped again. With
// refresh, the file's unreviewed drafts from earlier searches are replaced rather than kept, so
// a better query or better checks get to re-judge those pages; rejected drafts and drafts from a
// pasted URL are kept.
func (s *Service) SuggestForFile(ctx context.Context, fileID uint, refresh bool) (*SuggestResult, error) {
	if s.searcher == nil {
		return nil, errors.New("no web search provider is configured (add a Brave Search API key)")
	}
	file, err := LoadFileInfo(fileID)
	if err != nil {
		return nil, err
	}
	if refresh {
		if err := clearSearchDrafts(fileID); err != nil {
			return nil, err
		}
	}
	out := &SuggestResult{FileID: fileID, Query: llmscrape.BuildQuery(file)}

	// Ask for the most results one query allows: skipped results cost nothing, and without spare
	// ones a page of piracy links leaves nothing to read.
	hits, err := s.searcher.Search(ctx, out.Query, 20)
	if err != nil {
		return nil, err
	}
	models.MarkFileSearched(fileID)
	known := draftedURLs(fileID)
	var learned []string
	if s.settings.SkipDownloadSites {
		learned = LearnedBlocked()
	}
	var urls []string
	for _, h := range hits {
		if known[h.URL] || DomainBlocked(h.URL, s.settings.BlockedDomains) {
			continue
		}
		if DomainBlocked(h.URL, learned) {
			out.KnownDownloadSites++
			continue
		}
		known[h.URL] = true
		urls = append(urls, h.URL)
		if len(urls) >= s.settings.ResultsPerFile {
			break
		}
	}

	out.Results = make([]ScrapeResult, len(urls))
	sem := make(chan struct{}, s.settings.Concurrency)
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out.Results[i] = s.scrapePage(ctx, u, fileID, file, out.Query, false)
		}(i, u)
	}
	wg.Wait()

	collapseDuplicates(out.Results)
	for i := range out.Results {
		d := out.Results[i].Draft
		if d == nil {
			continue
		}
		if err := d.Save(); err != nil {
			out.Results[i].Draft = nil
			out.Results[i].Error = "saving draft: " + err.Error()
			continue
		}
		if d.Status == models.DraftStatusDismissed {
			out.Results[i].Draft = nil // recorded so the page is not read again, but not proposed
		}
	}

	sort.SliceStable(out.Results, func(a, b int) bool {
		return confidence(out.Results[a]) > confidence(out.Results[b])
	})
	return out, nil
}

// collapseDuplicates keeps one draft per scene when several pages describe it, the best by
// confidence, then an official page, then having a cover, then the most stills. The others are
// marked dismissed; they are still saved so the same pages are not read again.
func collapseDuplicates(results []ScrapeResult) {
	best := map[string]int{}
	for i, r := range results {
		key := sceneKey(r.Draft)
		if key == "" {
			continue
		}
		j, seen := best[key]
		if !seen {
			best[key] = i
			continue
		}
		loser := i
		if betterDraft(r.Draft, results[j].Draft) {
			best[key], loser = i, j
		}
		results[loser].Draft.Status = models.DraftStatusDismissed
		results[loser].Skipped = "same scene as " + results[best[key]].URL
	}
}

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

func sceneKey(d *models.DraftScene) string {
	if d == nil {
		return ""
	}
	sc, err := d.Scene()
	if err != nil || strings.TrimSpace(sc.Title) == "" {
		return ""
	}
	norm := func(s string) string { return nonAlnumRe.ReplaceAllString(strings.ToLower(s), "") }
	return norm(sc.Site) + "|" + norm(sc.Title)
}

func betterDraft(a, b *models.DraftScene) bool {
	if a.MatchConfidence != b.MatchConfidence {
		return a.MatchConfidence > b.MatchConfidence
	}
	if (a.PageKind == llmscrape.KindOfficial) != (b.PageKind == llmscrape.KindOfficial) {
		return a.PageKind == llmscrape.KindOfficial
	}
	sa, _ := a.Scene()
	sb, _ := b.Scene()
	if (len(sa.Covers) > 0) != (len(sb.Covers) > 0) {
		return len(sa.Covers) > 0
	}
	return len(sa.Gallery) > len(sb.Gallery)
}

func confidence(r ScrapeResult) float64 {
	if r.Draft == nil {
		return -1
	}
	return r.Draft.MatchConfidence
}

// LoadFileInfo reads what the LLM needs to know about a file.
func LoadFileInfo(fileID uint) (*llmscrape.FileInfo, error) {
	db, _ := models.GetDB()
	defer db.Close()
	var f models.File
	if err := db.Preload("Volume").Where(&models.File{ID: fileID}).First(&f).Error; err != nil {
		return nil, fmt.Errorf("file %d not found", fileID)
	}

	// A folder holding just this one video is the scene's own and its name usually describes the
	// scene; a folder shared by several, or the library root, is only a place.
	var videosInFolder int
	db.Model(&models.File{}).Where("path = ? AND type = ?", f.Path, "video").Count(&videosInFolder)
	isRoot := strings.TrimRight(f.Path, "/") == strings.TrimRight(f.Volume.Path, "/")

	return &llmscrape.FileInfo{
		Filename:        f.Filename,
		DurationMinutes: int(f.VideoDuration / 60),
		Context:         models.GetFileMatchContext(fileID),
		Folder:          filepath.Base(f.Path),
		FolderIsScene:   videosInFolder == 1 && !isRoot,
	}, nil
}

// clearSearchDrafts deletes a file's unreviewed drafts that came from a web search. Drafts from a
// pasted URL have no query and are kept, as are rejected ones, which stop a page being proposed.
func clearSearchDrafts(fileID uint) error {
	db, _ := models.GetDB()
	defer db.Close()
	return models.RetryIfBusy(func() error {
		return db.Where("file_id = ? AND status = ? AND query <> ''", fileID, models.DraftStatusDraft).
			Delete(&models.DraftScene{}).Error
	})
}

func draftedURLs(fileID uint) map[string]bool {
	db, _ := models.GetDB()
	defer db.Close()
	var drafts []models.DraftScene
	db.Select("source_url").Where("file_id = ?", fileID).Find(&drafts)
	m := make(map[string]bool, len(drafts))
	for _, d := range drafts {
		m[d.SourceURL] = true
	}
	return m
}
