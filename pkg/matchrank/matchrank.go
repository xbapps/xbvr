// Package matchrank re-ranks candidate scenes for an unmatched file using a small
// linear model whose weights were learned offline from confirmed matches. It replaces
// the previous fixed duration-boost heuristic: the model weighs the text-search score,
// how much of the (cleaned) filename overlaps the scene's title/cast/site/studio, a
// fuzzy title match, and how close the file's runtime is to the scene's — letting a
// close runtime promote a candidate as a tie-breaker rather than dominate the ranking.
package matchrank

import (
	"regexp"
	"strings"
)

// Learned coefficients in raw-feature space; the ranking score is
// bias + Σ weights[i]·feature[i]. Higher score = better match. Retrained offline; see
// featureNames for the order.
var weights = [...]float64{
	+0.758649, // bleve            (text-search score)
	+1.317023, // title_ov_frac    (query tokens in title / query length)
	+4.694124, // any_ov_frac      (query tokens in title+cast+site+studio / query length)
	-0.294407, // site_ov          (query tokens in site)
	-1.611637, // cast_ov_frac     (query tokens in cast / query length)
	+0.885691, // title_fuzzy      (near-miss title tokens, e.g. favorite~favourite)
	+0.004129, // qlen             (number of query tokens)
	-2.910082, // dur_known        (1 if both durations known)
	-1.457414, // dur_diff_mid     (scaled distance to the scene's mid-minute runtime)
	+2.846316, // within60         (1 if within ~a minute of the scene runtime)
}

const bias = -4.593663

// filename noise tokens (resolution/format/source), matching the match-box cleaner.
var common = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`180 180x180 2880x1440 3d 3dh 3dv 30fps 30m 360
		3840x1920 4k 5k 5400x2700 60fps 6k 7k 7680x3840 8k fb360 fisheye190 funscript
		cmscript h264 h265 hevc hq hsp lq lr mkv mkx200 mkx220 mono mp4 oculus oculus5k
		oculusrift original rf52 smartphone srt ssa tb uhq vrca220 vp9`) {
		common[w] = true
	}
}

var reSep = regexp.MustCompile(`[._+'’` + "`" + `\- ]+`)
var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
var reResolution = regexp.MustCompile(`^[0-9]+p$`)

// QueryTokens cleans a filename-derived query the same way the match box does:
// split on separators, lower-case, and drop resolution/format noise words.
func QueryTokens(q string) []string {
	out := []string{}
	for _, w := range reSep.Split(strings.ToLower(q), -1) {
		if w == "" || common[w] || reResolution.MatchString(w) {
			continue
		}
		out = append(out, w)
	}
	return out
}

// fieldTokens tokenises a scene field (title/cast/site/studio): split on non-alphanumeric,
// lower-case, keep tokens of length >= 2.
func fieldTokens(text string) map[string]bool {
	m := map[string]bool{}
	for _, w := range reNonAlnum.Split(strings.ToLower(text), -1) {
		if len(w) >= 2 {
			m[w] = true
		}
	}
	return m
}

func addTokens(m map[string]bool, text string) {
	for _, w := range reNonAlnum.Split(strings.ToLower(text), -1) {
		if len(w) >= 2 {
			m[w] = true
		}
	}
}

// within1 reports whether a and b (both length >= 5) differ by a single edit.
func within1(a, b string) bool {
	if len(a) < 5 || len(b) < 5 || a == b {
		return false
	}
	la, lb := len(a), len(b)
	if la-lb > 1 || lb-la > 1 {
		return false
	}
	if la == lb {
		diff := 0
		for i := 0; i < la; i++ {
			if a[i] != b[i] {
				diff++
			}
		}
		return diff == 1
	}
	lo, hi := a, b
	if lb < la {
		lo, hi = b, a
	}
	i, j, d := 0, 0, 0
	for i < len(lo) && j < len(hi) {
		if lo[i] == hi[j] {
			i++
			j++
		} else {
			d++
			if d > 1 {
				return false
			}
			j++
		}
	}
	return true
}

// Score returns the re-rank score for one candidate scene. qTokens is the cleaned query
// (from QueryTokens); bleve is the text-search score; cast is the scene's actor names;
// sceneDurationMin is the scene's runtime in minutes; fileDurationSec is the file's
// runtime in seconds (0 if unknown).
func Score(qTokens []string, bleve float64, title, site, studio string, cast []string, sceneDurationMin int, fileDurationSec float64) float64 {
	q := map[string]bool{}
	for _, w := range qTokens {
		q[w] = true
	}
	nq := float64(len(q))
	if nq == 0 {
		nq = 1
	}

	titleTok := fieldTokens(title)
	all := map[string]bool{}
	addTokens(all, title)
	for _, c := range cast {
		addTokens(all, c)
	}
	addTokens(all, site)
	addTokens(all, studio)
	castTok := map[string]bool{}
	for _, c := range cast {
		addTokens(castTok, c)
	}
	siteTok := fieldTokens(site)

	var titleOv, anyOv, siteOv, castOv, fuzzy float64
	for w := range q {
		if titleTok[w] {
			titleOv++
		}
		if all[w] {
			anyOv++
		}
		if siteTok[w] {
			siteOv++
		}
		if castTok[w] {
			castOv++
		}
		if !titleTok[w] {
			for t := range titleTok {
				if within1(w, t) {
					fuzzy++
					break
				}
			}
		}
	}

	durKnown, durDiffMid, within60 := 0.0, 0.5, 0.0
	if fileDurationSec > 0 && sceneDurationMin > 0 {
		durKnown = 1
		dc := fileDurationSec - float64(sceneDurationMin*60+30)
		if dc < 0 {
			dc = -dc
		}
		if dc > 600 {
			durDiffMid = 1
		} else {
			durDiffMid = dc / 600
		}
		if dc <= 60 {
			within60 = 1
		}
	}

	f := [...]float64{
		bleve, titleOv / nq, anyOv / nq, siteOv, castOv / nq, fuzzy, float64(len(q)),
		durKnown, durDiffMid, within60,
	}
	s := bias
	for i := range f {
		s += weights[i] * f[i]
	}
	return s
}
