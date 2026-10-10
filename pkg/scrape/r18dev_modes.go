package scrape

import (
	"strings"

	"github.com/xbapps/xbvr/pkg/models"
)

// Quality modes: 8KVR and High-Quality VR (4K60+) crawl a whole R18.dev category, every studio.
// They overlap every studio Scraper, so only one mode or the studio Scrapers can be on:
//   - turning a mode on turns every other R18.dev Scraper off, and remembers the studio picks
//   - turning a studio on while a mode is on turns the mode off
//   - leaving the modes (either way) turns the remembered studio picks back on

const r18devStudioPicksKey = "r18dev-studio-picks"

func r18devInfoByID(id string) (R18devScraperInfo, bool) {
	for _, info := range r18devRegistered {
		if info.ID == id {
			return info, true
		}
	}
	return R18devScraperInfo{}, false
}

// R18devSiteToggled is called after the Enabled switch of site id was flipped; enabled is the new state.
func R18devSiteToggled(id string, enabled bool) {
	info, ok := r18devInfoByID(id)
	if !ok {
		return
	}
	db, _ := models.GetCommonDB()
	var sites []models.Site
	db.Where("id LIKE ?", "r18dev-%").Find(&sites)

	var modesOn, studiosOn []string
	for _, st := range sites {
		other, ok := r18devInfoByID(st.ID)
		if !ok || st.ID == id || !st.IsEnabled {
			continue
		}
		if other.Exclusive {
			modesOn = append(modesOn, st.ID)
		} else {
			studiosOn = append(studiosOn, st.ID)
		}
	}
	setEnabled := func(ids []string, on bool) {
		if len(ids) > 0 {
			db.Model(&models.Site{}).Where("id IN (?)", ids).UpdateColumn("is_enabled", on)
		}
	}

	switch {
	case enabled && info.Exclusive:
		if len(modesOn) == 0 {
			// coming from studio mode: remember the picks (switching 8KVR -> 4K60+ keeps the old ones)
			kv := models.KV{Key: r18devStudioPicksKey, Value: strings.Join(studiosOn, ",")}
			kv.Save()
		}
		setEnabled(append(modesOn, studiosOn...), false)
		log.Infof("R18.dev quality mode %s on: turned off %s", info.Name, strings.Join(append(modesOn, studiosOn...), ", "))
	case enabled && len(modesOn) > 0:
		setEnabled(modesOn, false)
		r18devRestoreStudioPicks(id)
		log.Infof("R18.dev studio %s on: quality mode %s turned off", info.Name, strings.Join(modesOn, ", "))
	case !enabled && info.Exclusive:
		r18devRestoreStudioPicks("")
		log.Infof("R18.dev quality mode %s off", info.Name)
	}
}

// r18devRestoreStudioPicks turns the studio Scrapers back on that were on before a mode was.
func r18devRestoreStudioPicks(except string) {
	db, _ := models.GetCommonDB()
	var kv models.KV
	if db.Where(&models.KV{Key: r18devStudioPicksKey}).First(&kv).Error != nil {
		return
	}
	var ids []string
	for _, id := range strings.Split(kv.Value, ",") {
		if info, ok := r18devInfoByID(id); ok && !info.Exclusive && id != except {
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		db.Model(&models.Site{}).Where("id IN (?)", ids).UpdateColumn("is_enabled", true)
		log.Infof("R18.dev studio picks back on: %s", strings.Join(ids, ", "))
	}
	kv.Delete()
}
