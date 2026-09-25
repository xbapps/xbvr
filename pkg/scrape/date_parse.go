package scrape

import (
	"regexp"
	"strings"

	"github.com/nleeper/goment"
)

// Warm goment's lazily-initialized global parse/format tables once,
// single-threaded, before any scraper goroutine can run. goment fills those
// tables on first use with no mutex or sync.Once, so the first burst of
// concurrent scrapers races reader against writer and can read back a
// half-written entry — a nil func value that kills the whole server with
// SIGSEGV (see #2279). After this init the tables are read-only and
// concurrent use is safe.
func init() {
	d, _ := goment.New("January 1, 2000", "MMMM D, YYYY")
	_ = d.Format("YYYY-MM-DD")
}

// groobyDateRe matches the "Month D, YYYY" shape Grooby-network pages put in
// div.set_meta (e.g. "September 11, 2025"). Anything else is page drift.
var groobyDateRe = regexp.MustCompile(`^[A-Za-z]+ \d{1,2}, \d{4}$`)

// parseGroobyDate converts a Grooby-network release-date string to
// YYYY-MM-DD, returning "" for missing or malformed input. The shape check
// keeps unexpected page content away from the parser; a dateless scene
// beats a poisoned one.
func parseGroobyDate(raw string) string {
	s := strings.TrimSpace(strings.Replace(raw, "Added ", "", -1))
	if !groobyDateRe.MatchString(s) {
		return ""
	}
	if d, err := goment.New(s, "MMMM D, YYYY"); err == nil && d != nil {
		return d.Format("YYYY-MM-DD")
	}
	return ""
}
