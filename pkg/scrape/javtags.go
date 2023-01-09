package scrape

import (
	"regexp"
	"strings"
)

/* Returns en empty string if this tag is to be skipped, or the
 * mapped tag if it should be included.
 */
func ProcessJavrTag(tag string) string {
	taglower := strings.ToLower(tag)

	// Skipping some very generic and useless tags
	skiptags := map[string]bool{
		"featured actress":			true,
		"vr exclusive":				true,
		"high quality vr":			true,
		"high-quality vr":			true,
		"vr":						true,
		"vr only":					true,
		"hi-def":					true,
		"exclusive distribution":	true,
		"single work":				true,
		"solo work":				true,
		"solowork":					true,
	}
	if skiptags[taglower] {
		return ""
	}

	// Map some tags to normalize so different sources match
	// TODO: this mapping is totally incomplete and needs help from community to fill
	maptags := map[string]string{
		"blow":						"blowjob",
		"blow job":					"blowjob",
		"kiss":						"kiss kiss",
		"kiss / kiss":				"kiss kiss",
		"prostitute":				"club hostess & sex worker",
		"prostitutes":				"club hostess & sex worker",
		"suntan":					"sun tan",
	}
	if maptags[taglower] != "" {
		return maptags[taglower]
	}

	// Leave out some japanese text tags
	matched, err := regexp.Match("[^a-z0-9_\\- /&()\\+]", []byte(taglower))
	if matched == true || err != nil {
		return ""
	}

	// keep tag as-is (but lowercase)
	return taglower
}
