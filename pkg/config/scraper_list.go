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
	PovrScrapers    []ScraperConfig `json:"povr"`
	SlrScrapers     []ScraperConfig `json:"slr"`
	StashDbScrapers []ScraperConfig `json:"stashdb"`
	VrpornScrapers  []ScraperConfig `json:"vrporn"`
	VrphubScrapers  []ScraperConfig `json:"vrphub"`
}
type CustomScrapers struct {
	PovrScrapers    []ScraperConfig `json:"povr"`
	SlrScrapers     []ScraperConfig `json:"slr"`
	StashDbScrapers []ScraperConfig `json:"stashdb"`
	VrpornScrapers  []ScraperConfig `json:"vrporn"`
	VrphubScrapers  []ScraperConfig `json:"vrphub"`
}
type ScraperConfig struct {
	ID           string `json:"-"`
	URL          string `json:"url"`
	Name         string `json:"name"`
	Company      string `json:"company"`
	AvatarUrl    string `json:"avatar_url"`
	FileID       string `json:"id,omitempty"`
	MasterSiteId string `json:"master_site_id,omitempty"`
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
	SetSiteId(&o.XbvrScrapers.StashDbScrapers, "")
	SetSiteId(&o.XbvrScrapers.VrphubScrapers, "")
	SetSiteId(&o.XbvrScrapers.VrpornScrapers, "")
	SetSiteId(&o.CustomScrapers.PovrScrapers, "povr")
	SetSiteId(&o.CustomScrapers.SlrScrapers, "slr")
	SetSiteId(&o.CustomScrapers.StashDbScrapers, "stashdb")
	SetSiteId(&o.CustomScrapers.VrphubScrapers, "vrphub")
	SetSiteId(&o.CustomScrapers.VrpornScrapers, "vrporn")

	// remove custom sites that are now offical for the same aggregation site
	o.CustomScrapers.PovrScrapers = RemoveCustomListNowOffical(o.CustomScrapers.PovrScrapers, o.XbvrScrapers.PovrScrapers)
	o.CustomScrapers.SlrScrapers = RemoveCustomListNowOffical(o.CustomScrapers.SlrScrapers, o.XbvrScrapers.SlrScrapers)
	o.CustomScrapers.StashDbScrapers = RemoveCustomListNowOffical(o.CustomScrapers.StashDbScrapers, o.XbvrScrapers.StashDbScrapers)
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
		if s1 == s2 && customSite.MasterSiteId == findSite.MasterSiteId {
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

func MigrateFromOfficalToCustom(id string, url string, name string, company string, avatarUrl string, customId string, suffix string) error {

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Where("scraper_id = ?", id).Find(&scenes)

	if len(scenes) != 0 {
		common.Log.Infoln(name + ` Scenes found migration needed`)

		// Update scene data to reflect change
		db.Model(&models.Scene{}).Where("scraper_id = ?", id).Update("needs_update", true)

		// Determine the new id from the URL using the same template as the scraper list code
		tmp := strings.TrimRight(url, "/")
		newId := strings.ToLower(tmp[strings.LastIndex(tmp, "/")+1:]) + `-` + customId

		var scraperConfig ScraperList
		scraperConfig.Load()

		// Data taken from offical scraper list
		scraper := ScraperConfig{URL: url, Name: name, Company: company, AvatarUrl: avatarUrl}

		// Update any alt sites that is using the old id to the new id
		updateMasterSite := func(sites []ScraperConfig) {
			for idx, site := range sites {
				if site.MasterSiteId == id {
					sites[idx].MasterSiteId = newId
				}
			}
		}

		updateMasterSite(scraperConfig.CustomScrapers.SlrScrapers)
		updateMasterSite(scraperConfig.CustomScrapers.PovrScrapers)
		updateMasterSite(scraperConfig.CustomScrapers.VrpornScrapers)
		updateMasterSite(scraperConfig.CustomScrapers.VrphubScrapers)

		// Append our scraper to the the Custom Scraper list unless its new id already exists
		switch customId {
		case "slr":
			if CheckMatchingSite(scraper, scraperConfig.CustomScrapers.SlrScrapers) == false {
				scraperConfig.CustomScrapers.SlrScrapers = append(scraperConfig.CustomScrapers.SlrScrapers, scraper)
			}
		case "povr":
			if CheckMatchingSite(scraper, scraperConfig.CustomScrapers.PovrScrapers) == false {
				scraperConfig.CustomScrapers.PovrScrapers = append(scraperConfig.CustomScrapers.PovrScrapers, scraper)
			}
		case "vrporn":
			if CheckMatchingSite(scraper, scraperConfig.CustomScrapers.VrpornScrapers) == false {
				scraperConfig.CustomScrapers.VrpornScrapers = append(scraperConfig.CustomScrapers.VrpornScrapers, scraper)
			}
		case "vrphub":
			if CheckMatchingSite(scraper, scraperConfig.CustomScrapers.VrphubScrapers) == false {
				scraperConfig.CustomScrapers.VrphubScrapers = append(scraperConfig.CustomScrapers.VrphubScrapers, scraper)
			}
		}

		// Save the new list file
		fName := filepath.Join(common.AppDir, "scrapers.json")
		list, _ := json.MarshalIndent(scraperConfig, "", "  ")
		os.WriteFile(fName, list, 0644)

		common.Log.Infoln(name + ` migration complete. Please restart XBVR and run ` + name + ` scraper to complete migration`)

	} else {

		common.Log.Infoln(`No ` + name + ` Scenes found no migration needed. Removing DB entry`)

	}

	return db.Delete(&models.Site{ID: id}).Error
}
