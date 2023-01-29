package config

import (
	_ "embed"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

//go:embed scrapers.json
var officalList []byte

type ScraperList struct {
	Warnings       []string       `json:"warning"`
	CusomtScrapers CusomtScrapers `json:"custom"`
	XbvrScrapers   XbvrScrapers   `json:"xbvr"`
}
type XbvrScrapers struct {
	PovrScrapers   []ScraperConfig `json:"povr"`
	SlrScrapers    []ScraperConfig `json:"slr"`
	VrpornScrapers []ScraperConfig `json:"vrporn"`
	VrphubScrapers []ScraperConfig `json:"vrphub"`
}
type CusomtScrapers struct {
	PovrScrapers   []ScraperConfig `json:"povr"`
	SlrScrapers    []ScraperConfig `json:"slr"`
	VrpornScrapers []ScraperConfig `json:"vrporn"`
	VrphubScrapers []ScraperConfig `json:"vrphub"`
}
type ScraperConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	AvatarUrl string `json:"avatar_url"`
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

	// remove custom sites that are now offical for the same aggregation site
	o.CusomtScrapers.PovrScrapers = RemoveCustomListNowOffical(o.CusomtScrapers.PovrScrapers, o.XbvrScrapers.PovrScrapers)
	o.CusomtScrapers.SlrScrapers = RemoveCustomListNowOffical(o.CusomtScrapers.SlrScrapers, o.XbvrScrapers.SlrScrapers)
	o.CusomtScrapers.VrphubScrapers = RemoveCustomListNowOffical(o.CusomtScrapers.VrphubScrapers, o.XbvrScrapers.VrphubScrapers)
	o.CusomtScrapers.VrpornScrapers = RemoveCustomListNowOffical(o.CusomtScrapers.VrpornScrapers, o.XbvrScrapers.VrpornScrapers)

	// remove custom sites with no scenes and now offical for a different aggregation site
	o.CusomtScrapers.PovrScrapers = RemoveCustomListNowOfficalIfEmpty(o.CusomtScrapers.PovrScrapers, o.XbvrScrapers)
	o.CusomtScrapers.SlrScrapers = RemoveCustomListNowOfficalIfEmpty(o.CusomtScrapers.SlrScrapers, o.XbvrScrapers)
	o.CusomtScrapers.VrphubScrapers = RemoveCustomListNowOfficalIfEmpty(o.CusomtScrapers.VrphubScrapers, o.XbvrScrapers)
	o.CusomtScrapers.VrpornScrapers = RemoveCustomListNowOfficalIfEmpty(o.CusomtScrapers.VrpornScrapers, o.XbvrScrapers)

	list, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		ioutil.WriteFile(fName, list, 0644)
	}

	// check for offical/custom duplicates across aggregation sites (leaves them in the local file already written)
	o.XbvrScrapers.PovrScrapers = RemoveDuplicateOfficalSite(o.XbvrScrapers.PovrScrapers, o.CusomtScrapers)
	o.XbvrScrapers.SlrScrapers = RemoveDuplicateOfficalSite(o.XbvrScrapers.SlrScrapers, o.CusomtScrapers)
	o.XbvrScrapers.VrphubScrapers = RemoveDuplicateOfficalSite(o.XbvrScrapers.VrphubScrapers, o.CusomtScrapers)
	o.XbvrScrapers.VrpornScrapers = RemoveDuplicateOfficalSite(o.XbvrScrapers.VrpornScrapers, o.CusomtScrapers)

	return nil
}

func RemoveCustomListNowOffical(customSiteList []ScraperConfig, officalSiteList []ScraperConfig) []ScraperConfig {
	newList := []ScraperConfig{}
	for _, customSite := range customSiteList {
		if !CheckMatchingSite(customSite, officalSiteList) {
			newList = append(newList, customSite)
		} else {
			offileSite := GetMatchingSite(customSite, officalSiteList)
			if offileSite.Name != customSite.Name || offileSite.Company != customSite.Company {
				db, _ := models.GetDB()
				defer db.Close()

				db.Model(&models.Scene{}).Where("site = ?", customSite.Name).Update("needs_update", true)
			}
		}
	}
	return newList
}

func RemoveCustomListNowOfficalIfEmpty(customSiteList []ScraperConfig, officalScrapers XbvrScrapers) []ScraperConfig {
	db, _ := models.GetDB()
	defer db.Close()

	newList := []ScraperConfig{}
	for _, customSite := range customSiteList {
		if CheckMatchingSite(customSite, officalScrapers.PovrScrapers) || CheckMatchingSite(customSite, officalScrapers.SlrScrapers) || CheckMatchingSite(customSite, officalScrapers.VrphubScrapers) || CheckMatchingSite(customSite, officalScrapers.VrpornScrapers) {
			cnt := 0
			db.Model(models.Scene{}).
				Where("site = ?", customSite.Name).Count(&cnt)
			if cnt > 0 {
				// has scenes, keep it, can't get rid of the custom site until all scenes are deleted
				newList = append(newList, customSite)
			}
		} else {
			newList = append(newList, customSite)
		}
	}
	return newList
}

func RemoveDuplicateOfficalSite(officalSiteList []ScraperConfig, customScrapers CusomtScrapers) []ScraperConfig {
	newList := []ScraperConfig{}
	for _, officalSite := range officalSiteList {
		if !CheckMatchingSite(officalSite, customScrapers.PovrScrapers) && !CheckMatchingSite(officalSite, customScrapers.SlrScrapers) && !CheckMatchingSite(officalSite, customScrapers.VrphubScrapers) && !CheckMatchingSite(officalSite, customScrapers.VrpornScrapers) {
			newList = append(newList, officalSite)
		}
	}
	return newList
}
func CheckMatchingSite(findSite ScraperConfig, searchList []ScraperConfig) bool {
	for _, customSite := range searchList {
		if findSite.ID == customSite.ID {
			return true
		}
	}
	return false
}
func GetMatchingSite(findSite ScraperConfig, searchList []ScraperConfig) ScraperConfig {
	for _, site := range searchList {
		if findSite.ID == site.ID {
			return site
		}
	}
	return ScraperConfig{}
}
