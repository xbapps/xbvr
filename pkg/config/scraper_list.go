package config

import (
	_ "embed"
	"encoding/json"
	"io/ioutil"
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
		ioutil.WriteFile(fName, list, 0644)
		return nil
	} else {
		b, err := ioutil.ReadFile(fName)
		if err != nil {
			o.XbvrScrapers = officalScrapers.XbvrScrapers
			return err
		}
		json.Unmarshal(b, &o)
	}

	//	overwrite the local files offical list
	o.XbvrScrapers = officalScrapers.XbvrScrapers
	o.Warnings = officalScrapers.Warnings

	SetSiteId(&o.XbvrScrapers.PovrScrapers)
	SetSiteId(&o.XbvrScrapers.SlrScrapers)
	SetSiteId(&o.XbvrScrapers.VrphubScrapers)
	SetSiteId(&o.XbvrScrapers.VrpornScrapers)
	SetSiteId(&o.CustomScrapers.PovrScrapers)
	SetSiteId(&o.CustomScrapers.SlrScrapers)
	SetSiteId(&o.CustomScrapers.VrphubScrapers)
	SetSiteId(&o.CustomScrapers.VrpornScrapers)

	// remove custom sites that are now offical for the same aggregation site
	o.CustomScrapers.PovrScrapers = RemoveCustomListNowOffical(o.CustomScrapers.PovrScrapers, o.XbvrScrapers.PovrScrapers)
	o.CustomScrapers.SlrScrapers = RemoveCustomListNowOffical(o.CustomScrapers.SlrScrapers, o.XbvrScrapers.SlrScrapers)
	o.CustomScrapers.VrphubScrapers = RemoveCustomListNowOffical(o.CustomScrapers.VrphubScrapers, o.XbvrScrapers.VrphubScrapers)
	o.CustomScrapers.VrpornScrapers = RemoveCustomListNowOffical(o.CustomScrapers.VrpornScrapers, o.XbvrScrapers.VrpornScrapers)

	// if a custom site has the same Site Id as an offical one, we need to change it
	o.CustomScrapers.PovrScrapers = RenameDuplicateIds(o.CustomScrapers.PovrScrapers, o.XbvrScrapers, "-povr")
	o.CustomScrapers.SlrScrapers = RenameDuplicateIds(o.CustomScrapers.SlrScrapers, o.XbvrScrapers, "-slr")
	o.CustomScrapers.VrphubScrapers = RenameDuplicateIds(o.CustomScrapers.VrphubScrapers, o.XbvrScrapers, "-vrphub")
	o.CustomScrapers.VrpornScrapers = RenameDuplicateIds(o.CustomScrapers.VrpornScrapers, o.XbvrScrapers, "-vrporn")

	list, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		ioutil.WriteFile(fName, list, 0644)
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
			officialSite := GetMatchingSite(customSite, officalSiteList)
			if officialSite.Name != customSite.Name || officialSite.Company != customSite.Company {
				db.Model(&models.Scene{}).Where("site = ?", customSite.Name).Update("needs_update", true)
			}
			if customSite.FileID != "" {
				// the user specied a different  site id, delete the old site record
				db.Delete(&models.Site{}, customSite.ID)
			}
		}
	}
	return newList
}

func RenameDuplicateIds(customSiteList []ScraperConfig, officalScrapers XbvrScrapers, newSuffix string) []ScraperConfig {
	db, _ := models.GetDB()
	defer db.Close()
	newList := []ScraperConfig{}
	for _, customSite := range customSiteList {
		if CheckMatchingSiteID(customSite, officalScrapers.PovrScrapers) || CheckMatchingSiteID(customSite, officalScrapers.SlrScrapers) || CheckMatchingSiteID(customSite, officalScrapers.VrphubScrapers) || CheckMatchingSiteID(customSite, officalScrapers.VrpornScrapers) {
			oldId := customSite.ID
			customSite.ID = customSite.ID + newSuffix
			customSite.FileID = customSite.ID
			db.Model(models.Site{}).Where("id = ?", oldId).Update("id", customSite.ID)
		}
		newList = append(newList, customSite)
	}
	return newList
}

func CheckMatchingSite(findSite ScraperConfig, searchList []ScraperConfig) bool {
	for _, customSite := range searchList {
		if findSite.URL == customSite.URL {
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

func SetSiteId(configList *[]ScraperConfig) {
	for idx, siteconfig := range *configList {
		if siteconfig.FileID == "" {
			id := strings.TrimRight(siteconfig.URL, "/")
			siteconfig.ID = id[strings.LastIndex(id, "/")+1:]
		} else {
			siteconfig.ID = siteconfig.FileID
		}
		(*configList)[idx] = siteconfig
	}

}
