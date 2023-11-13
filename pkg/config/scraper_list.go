package config

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

//go:embed scrapers.json
var officalList []byte

type ScraperList struct {
	Warnings       []string       `json:"warning"`
	CustomScrapers CustomScrapers `json:"custom"`
	XbvrScrapers   XbvrScrapers   `json:"xbvr"`
}
type XbvrScrapers struct {
	PovrScrapers   []ScraperConfig `json:"povr"`
	SlrScrapers    []ScraperConfig `json:"slr"`
	VrpornScrapers []ScraperConfig `json:"vrporn"`
	VrphubScrapers []ScraperConfig `json:"vrphub"`
}
type CustomScrapers struct {
	PovrScrapers   []ScraperConfig `json:"povr"`
	SlrScrapers    []ScraperConfig `json:"slr"`
	VrpornScrapers []ScraperConfig `json:"vrporn"`
	VrphubScrapers []ScraperConfig `json:"vrphub"`
}
type ScraperConfig struct {
	ID        string `json:"-"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	AvatarUrl string `json:"avatar_url"`
	FileID    string `json:"id,omitempty"`
}

var loadLock sync.Mutex

func (o *ScraperList) Load() error {
	loadLock.Lock()
	defer loadLock.Unlock()

	// load standard scraper config embeded in distribution
	var officalScrapers ScraperList
	json.Unmarshal(officalList, &officalScrapers)

	fName := filepath.Join(common.AppDir, "scrapers.json")
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		list, _ := json.MarshalIndent(officalScrapers, "", "  ")
		os.WriteFile(fName, list, 0644)
	} else {
		b, err := os.ReadFile(fName)
		if err != nil {
			o.XbvrScrapers = officalScrapers.XbvrScrapers
			return err
		}
		json.Unmarshal(b, &o)
	}

	//	overwrite the local files offical list
	o.XbvrScrapers = officalScrapers.XbvrScrapers
	o.Warnings = officalScrapers.Warnings

	SetSiteId(&o.XbvrScrapers.PovrScrapers, "")
	SetSiteId(&o.XbvrScrapers.SlrScrapers, "")
	SetSiteId(&o.XbvrScrapers.VrphubScrapers, "")
	SetSiteId(&o.XbvrScrapers.VrpornScrapers, "")
	SetSiteId(&o.CustomScrapers.PovrScrapers, "povr")
	SetSiteId(&o.CustomScrapers.SlrScrapers, "slr")
	SetSiteId(&o.CustomScrapers.VrphubScrapers, "vrphub")
	SetSiteId(&o.CustomScrapers.VrpornScrapers, "vrporn")

	// remove custom sites that are now offical for the same aggregation site
	o.CustomScrapers.PovrScrapers = RemoveCustomListNowOffical(o.CustomScrapers.PovrScrapers, o.XbvrScrapers.PovrScrapers)
	o.CustomScrapers.SlrScrapers = RemoveCustomListNowOffical(o.CustomScrapers.SlrScrapers, o.XbvrScrapers.SlrScrapers)
	o.CustomScrapers.VrphubScrapers = RemoveCustomListNowOffical(o.CustomScrapers.VrphubScrapers, o.XbvrScrapers.VrphubScrapers)
	o.CustomScrapers.VrpornScrapers = RemoveCustomListNowOffical(o.CustomScrapers.VrpornScrapers, o.XbvrScrapers.VrpornScrapers)

	list, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		os.WriteFile(fName, list, 0644)
	}

	return nil
}

func RemoveCustomListNowOffical(customSiteList []ScraperConfig, officalSiteList []ScraperConfig) []ScraperConfig {
	newList := []ScraperConfig{}
	for _, customSite := range customSiteList {
		if !CheckMatchingSite(customSite, officalSiteList) {
			newList = append(newList, customSite)
		} else {
			db, _ := models.GetDB()
			defer db.Close()
			db.Model(&models.Scene{}).Where("scraper_id = ?", customSite.ID).Update("needs_update", true)
			db.Delete(&models.Site{ID: customSite.ID})
			common.Log.Infof("Studio %s is now an offical Studio and has been shifted from your custom list.  Enable the offical scraper and run it to update existing scenes", customSite.Name)
		}
	}
	return newList
}

func CheckMatchingSite(findSite ScraperConfig, searchList []ScraperConfig) bool {
	for _, customSite := range searchList {
		s1 := strings.ToLower(customSite.URL)
		s2 := strings.ToLower(findSite.URL)
		if !strings.HasSuffix(s1, "/") {
			s1 += "/"
		}
		if !strings.HasSuffix(s2, "/") {
			s2 += "/"
		}
		if s1 == s2 {
			return true
		}
	}
	return false
}
func GetMatchingSite(findSite ScraperConfig, searchList []ScraperConfig) ScraperConfig {
	for _, site := range searchList {
		if findSite.URL == site.URL {
			return site
		}
	}
	return ScraperConfig{}
}
func CheckMatchingSiteID(findSite ScraperConfig, searchList []ScraperConfig) bool {
	for _, customSite := range searchList {
		if findSite.ID == customSite.ID {
			return true
		}
	}
	return false
}

func SetSiteId(configList *[]ScraperConfig, customId string) {
	for idx, siteconfig := range *configList {
		if siteconfig.FileID == "" || customId != "" {
			id := strings.TrimRight(siteconfig.URL, "/")
			siteconfig.ID = strings.ToLower(id[strings.LastIndex(id, "/")+1:])
		} else {
			siteconfig.ID = strings.ToLower(siteconfig.FileID)
		}
		if customId != "" {
			siteconfig.ID = strings.ToLower(siteconfig.ID + "-" + customId)
		}
		(*configList)[idx] = siteconfig
	}

}
