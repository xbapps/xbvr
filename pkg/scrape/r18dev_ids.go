package scrape

import (
	"regexp"
	"strings"
)

// Scene IDs for the R18.dev crawler.
//
// Scene ID rule: a normalized dvd_id when R18.dev has one, otherwise the raw FANZA content ID
// (lowercase, minus a reissue suffix). Content IDs are unique, so post-2022 scenes (which never
// have a dvd_id) can't collide. This also matches what the comma-box R18.dev scraper has always
// done for scenes without a dvd_id.
//
// The "pretty" DVD-ID style ID built from a content ID (r18devDisplayID) is display only: it goes
// in the title and the Site field, never in the scene ID. A mistake there is cosmetic.

var (
	r18devReissueRe   = regexp.MustCompile(`(\d)re\d+$`)               // hnvr00144re01 -> hnvr00144
	r18devMakerRe     = regexp.MustCompile(`^h_\d+`)                   // h_1337wvr100001 -> wvr100001
	r18devLeadDigitRe = regexp.MustCompile(`^\d+`)                     // 84vrkm00139 -> vrkm00139
	r18devVRSokenCid  = regexp.MustCompile(`^h_(1337|1390)wvr`)        // VR Research Institute maker codes
	r18devVRSokenRe   = regexp.MustCompile(`^wvr0*(\d[a-z]?)(\d+)$`)   // wvr + section digit (+ letter) + number
	r18devSplitRe     = regexp.MustCompile(`^(.*[a-z])(\d+)([a-z]*)$`) // prefix + number + part letters
	r18devDashLetter  = regexp.MustCompile(`-([A-Z]+\d+[A-Z]*)$`)      // WVR6-D071, YP-Y003, BOOYAH-VR0002
	r18devDvdTailRe   = regexp.MustCompile(`^(\d+)([A-Z]*)$`)
	r18devDvdNoDashRe = regexp.MustCompile(`^(.*[A-Z])(\d+)([A-Z]*)$`)
)

// r18devSceneID returns the XBVR scene ID for an R18.dev scene.
func r18devSceneID(contentID string, dvdID string) string {
	if strings.TrimSpace(dvdID) != "" && dvdID != "----" {
		return r18devNormalizeDVDID(dvdID)
	}
	return r18devStripReissue(strings.ToLower(strings.TrimSpace(contentID)))
}

// r18devStripReissue drops a FANZA reissue suffix (re01, re02, ...) that directly follows the
// number. A bare "re" elsewhere is part of the prefix (5725ore00001, kore00004).
func r18devStripReissue(contentID string) string {
	return r18devReissueRe.ReplaceAllString(contentID, "$1")
}

// r18devNormalizeDVDID cleans up a dvd_id as R18.dev returns it, touching as little as possible:
//   - uppercase (ftvr-007, 3dsvr-0140 exist)
//   - never letters right after the dash: WVR6-D071 -> WVR6D-071, YP-Y003 -> YPY-003
//   - add the dash if missing: CAFR009 -> CAFR-009
//   - 4 digit numbers stay as printed (SOD: 3DSVR-0137), anything else becomes at least 3 digits
//     (MIVR-00077 -> MIVR-077, ZVR-01 -> ZVR-001)
//   - trailing part letters stay: OYCVR-025A
//
// Anything that doesn't fit those shapes is returned uppercased and otherwise untouched.
func r18devNormalizeDVDID(dvdID string) string {
	d := strings.ToUpper(strings.TrimSpace(dvdID))
	d = r18devDashLetter.ReplaceAllString(d, "$1")

	var head, num, suffix string
	if i := strings.LastIndex(d, "-"); i >= 0 {
		m := r18devDvdTailRe.FindStringSubmatch(d[i+1:])
		if m == nil {
			return d
		}
		head, num, suffix = d[:i], m[1], m[2]
	} else {
		m := r18devDvdNoDashRe.FindStringSubmatch(d)
		if m == nil {
			return d
		}
		head, num, suffix = m[1], m[2], m[3]
	}

	width := 3
	if len(num) == 4 {
		width = 4
	}
	return head + "-" + r18devPadNumber(num, width) + suffix
}

// r18devDisplayID builds a DVD-ID style ID from a FANZA content ID, for titles and the Site field.
// Returns "" if the content ID doesn't look like prefix + number.
//
//	84vrkm00139       -> VRKM-139
//	h_1337wvr100001   -> WVR1-001
//	13dsvr02015       -> 3DSVR-2015
//	hnvr00144re01     -> HNVR-144
//	urvrsp00494a      -> URVRSP-494A
func r18devDisplayID(contentID string) string {
	cid := strings.ToLower(strings.TrimSpace(contentID))
	c := r18devStripReissue(cid)
	c = r18devMakerRe.ReplaceAllString(c, "")

	// SOD: maker code 1 + a "3D..." series (13dsvr, 13dsivr). Everyone else: drop the maker digits.
	if strings.HasPrefix(c, "13d") {
		c = c[1:]
	} else {
		c = r18devLeadDigitRe.ReplaceAllString(c, "")
	}

	var prefix, num, suffix string
	if r18devVRSokenCid.MatchString(cid) {
		// VR Research Institute numbers its sections inside the content ID:
		// wvr90012 -> WVR9-012, wvr08002 -> WVR8-002, wvr6d00071 -> WVR6D-071
		if m := r18devVRSokenRe.FindStringSubmatch(c); m != nil {
			prefix, num = "wvr"+m[1], m[2]
		}
	}
	if prefix == "" {
		m := r18devSplitRe.FindStringSubmatch(c)
		if m == nil {
			return ""
		}
		prefix, num, suffix = m[1], m[2], m[3]
		if len(num) > 5 {
			// More than 5 digits: the extra leading digits belong to the prefix (wvr100001 -> WVR1),
			// unless they're all zeros, then they're junk (rvr000001 -> RVR-001).
			extra := num[:len(num)-5]
			num = num[len(num)-5:]
			if strings.Trim(extra, "0") != "" {
				prefix += extra
			}
		}
	}
	return strings.ToUpper(prefix) + "-" + r18devPadNumber(num, 3) + strings.ToUpper(suffix)
}

// r18devDisplayPrefix returns the part of a DVD-ID style ID before the last dash (VRKM-139 -> VRKM).
func r18devDisplayPrefix(id string) string {
	if i := strings.LastIndex(id, "-"); i > 0 {
		return id[:i]
	}
	return id
}

func r18devPadNumber(num string, width int) string {
	n := strings.TrimLeft(num, "0")
	for len(n) < width {
		n = "0" + n
	}
	return n
}
