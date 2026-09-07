package api

import (
	"regexp"
	"strings"

	"github.com/mcuadros/go-version"
)

// versionCoreRe captures the leading numeric version, ignoring snapshot
// suffixes such as goreleaser's "{{.Tag}}-next" and git-describe's
// "-N-gHASH". mcuadros/go-version treats those suffixes as older than the
// tag they were built from, which false-notifies source/snapshot builds.
var versionCoreRe = regexp.MustCompile(`(?i)^v?(\d+(?:\.\d+){1,3})`)

func isDevVersion(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "", "CURRENT", "HEAD", "DEV":
		return true
	default:
		return false
	}
}

func versionCore(v string) string {
	v = strings.TrimSpace(v)
	if isDevVersion(v) {
		return ""
	}
	m := versionCoreRe.FindStringSubmatch(v)
	if m == nil {
		return ""
	}
	return m[1]
}

// shouldNotifyUpdate reports whether current is a released build older than
// latest. Dev sentinels, bare git hashes, and snapshot stamps of the same
// or newer release do not notify.
func shouldNotifyUpdate(current, latest string) bool {
	currentCore := versionCore(current)
	latestCore := versionCore(latest)
	if currentCore == "" || latestCore == "" {
		return false
	}
	return version.Compare(currentCore, latestCore, "<")
}
