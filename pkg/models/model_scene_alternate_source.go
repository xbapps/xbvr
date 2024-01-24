package models

import (
	"encoding/json"
	"strings"
	"time"
)

// SceneCuepoint data model
type SceneAlternateSource struct {
	MasterSiteId  string `json:"master_site_id"`
	MatchAchieved int    `json:"MatchAchieved"`
	Query         string `json:"query"` // ******* remove before go live, just to help testing query used for the scene
	Scene         Scene  `json:"scene"`
}
type AltSrcMatchParams struct {
	IgnoreReleasedBefore time.Time `json:"ignore_released_before"` // optionally don't check the released date, if before a certain date, this may be useful to only search on release dates after an inital load of scene on a site
	DelayLinking         int       `json:"delay_linking"`          // allows delaying linking from the release date.  Useful for LethalHardcore which is often released on their site after SLR, etc, which would result in linking to another scene
	ReprocessLinks       int       `json:"reprocess_links"`        // will reprocess linking for the specified number of days.  Another approach for LethalHardcore.
	ReleasedMatchType    string    `json:"released_match_type"`    // must, should, do not
	ReleasedPrior        int       `json:"released_prior"`         // match on where the scene release date within x days prior to the release date of the alternate scene, eg a studio may only release on other sites after 90 days
	ReleasedAfter        int       `json:"released_after"`         // do not match scenes befoew a certain date
	DurationMatchType    string    `json:"duration_match_type"`    // must, should, do not
	DurationMin          int       `json:"duration_min"`           // ignore if the scene duration is below a specific value, may be useful for trailers
	DurationRangeLess    int       `json:"duration_range_less"`    // match on where the scene duration is more than x seconds less than the duration of the alternate scene
	DurationRangeMore    int       `json:"duration_range_more"`    // match on where the scene duration is less than x seconds more than the duration of the alternate scene
	CastMatchType        string    `json:"cast_match_type"`        // should or do not. Note "must" match is not an option for cast members due to alises and sites using different names.
	DescriptionMatchType string    `json:"desc_match_type"`        // should, do not. "Must" is not an option
	BoostTitleExact      float32   `json:"boost_title"`            // boosting allows you control the revalant significant between search fields, default is 1.  Boostig has no effect on "must" matches
	BoostTitleAnyWords   float32   `json:"boost_title_any_words"`  // boosting allows you control the revalant significant between search fields, default is 1.  Boostig has no effect on "must" matches
	BoostReleased        float32   `json:"boost_released"`
	BoostCast            float32   `json:"boost_cast"`
	BoostDescription     float32   `json:"boost_description"`
}

const defaultMatchType = "should"
const releasedPrior = 14
const releasedAfter = 7
const durationMin = 3
const boostTitleExact = 4.0
const boostTitleAnyWords = 1.5
const boostReleased = 2
const boostCast = 2
const boostDescription = 0.25

func (p *AltSrcMatchParams) Default() {
	p.ReleasedMatchType = defaultMatchType
	p.ReleasedPrior = releasedPrior
	p.ReleasedAfter = releasedAfter
	p.DurationMatchType = defaultMatchType
	p.DurationMin = durationMin
	p.CastMatchType = defaultMatchType
	p.DescriptionMatchType = defaultMatchType
	p.BoostReleased = boostReleased
	p.BoostTitleExact = boostTitleExact
	p.BoostTitleAnyWords = boostTitleAnyWords
	p.BoostCast = boostCast
	p.BoostDescription = boostDescription
	p.BoostReleased = boostReleased
}

func (p *AltSrcMatchParams) UnmarshalParams(jsonStr string) error {
	if jsonStr == "" {
		p.Default()
		return nil
	} else {
		err := json.Unmarshal([]byte(jsonStr), &p)
		if err != nil {
			return err
		}
		if !strings.Contains(jsonStr, "released_match_type") {
			p.ReleasedMatchType = defaultMatchType
		}
		if !strings.Contains(jsonStr, "released_prior") {
			p.ReleasedPrior = releasedPrior
		}
		if !strings.Contains(jsonStr, "released_after") {
			p.ReleasedAfter = releasedAfter
		}
		if !strings.Contains(jsonStr, "duration_match_type") {
			p.DurationMatchType = defaultMatchType
		}
		if !strings.Contains(jsonStr, "duration_min") {
			p.DurationMin = durationMin
		}
		if !strings.Contains(jsonStr, "cast_match_type") {
			p.CastMatchType = defaultMatchType
		}
		if !strings.Contains(jsonStr, "desc_match_type") {
			p.DescriptionMatchType = defaultMatchType
		}
		if !strings.Contains(jsonStr, "boost_title") {
			p.BoostTitleExact = boostTitleExact
		}
		if !strings.Contains(jsonStr, "boost_title_any_words") {
			p.BoostTitleAnyWords = boostTitleAnyWords
		}
		if !strings.Contains(jsonStr, "boost_released") {
			p.BoostCast = boostReleased
		}
		if !strings.Contains(jsonStr, "boost_cast") {
			p.BoostCast = boostCast
		}
		if !strings.Contains(jsonStr, "boost_description") {
			p.BoostDescription = boostDescription
		}
	}
	return nil
}
