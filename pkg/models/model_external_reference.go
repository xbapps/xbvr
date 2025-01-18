package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/markphelps/optional"

	"github.com/xbapps/xbvr/pkg/common"
)

type ExternalReference struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"created_at-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"updated_at"`

	XbvrLinks      []ExternalReferenceLink `json:"xbvr_links" xbvrbackup:"xbvr_links"`
	ExternalSource string                  `json:"external_source" xbvrbackup:"external_source"`
	ExternalId     string                  `json:"external_id" gorm:"index" xbvrbackup:"external_id"`
	ExternalURL    string                  `json:"external_url" gorm:"size:1000" xbvrbackup:"external_url"`
	ExternalDate   time.Time               `json:"external_date" xbvrbackup:"external_date"`
	ExternalData   string                  `json:"external_data" sql:"type:longtext;" xbvrbackup:"external_data"`
	UdfBool1       bool                    `json:"udf_bool1" xbvrbackup:"udf_bool1"` // user defined fields, use depends what type of data the extref is for.
	UdfBool2       bool                    `json:"udf_bool2" xbvrbackup:"udf_bool2"`
	UdfDatetime1   time.Time               `json:"udf_datetime1" xbvrbackup:"udf_datetime1"`
}

type ExternalReferenceLink struct {
	ID             uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt      time.Time `json:"-" xbvrbackup:"created_at-"`
	UpdatedAt      time.Time `json:"-" xbvrbackup:"updated_at"`
	InternalTable  string    `json:"internal_table" xbvrbackup:"internal_table"`
	InternalDbId   uint      `json:"internal_db_id" gorm:"index" xbvrbackup:"-"`
	InternalNameId string    `json:"internal_name_id" gorm:"index" xbvrbackup:"internal_name_id"`

	ExternalReferenceID uint      `json:"external_reference_id" gorm:"index" xbvrbackup:"-"`
	ExternalSource      string    `json:"external_source" xbvrbackup:"-"`
	ExternalId          string    `json:"external_id" gorm:"index" xbvrbackup:"-"`
	MatchType           int       `json:"match_type" xbvrbackup:"match_type"`
	UdfDatetime1        time.Time `json:"udf_datetime1" xbvrbackup:"udf_datetime1"`

	ExternalReference ExternalReference `json:"external_reference" gorm:"foreignKey:ExternalReferenceId" xbvrbackup:"-"`
}

type ActorScraperConfig struct {
	StashSceneMatching         map[string][]StashSiteConfig
	GenericActorScrapingConfig map[string]GenericScraperRuleSet
}
type GenericScraperRuleSet struct {
	SiteRules []GenericActorScraperRule `json:"rules"`
	Domain    string                    `json:"domain"`
	IsJson    bool                      `json:"isJson"`
}

type GenericActorScraperRule struct {
	XbvrField string `json:"xbvr_field"`

	// Go implementation of the rule. If specified, other fields below are ignored.
	// This function receives the body of the page as json or html and must return one or multiple values for the field.
	// This cannot be loaded from json.
	Native func(interface{}) []string `json:"-"`

	Selector       string           `json:"selector"`        // css selector to identify data
	PostProcessing []PostProcessing `json:"post_processing"` // call routines for specific handling, eg dates parshing, json extracts, etc, see PostProcessing function
	First          optional.Int     `json:"first"`           // used to limit how many results you want, the start position you want.  First index pos	 is 0
	Last           optional.Int     `json:"last"`            // used to limit how many results you want, the end position you want
	ResultType     string           `json:"result_type"`     // how to treat the result, text, attribute value, json
	Attribute      string           `json:"attribute"`       // name of the atribute you want
}
type PostProcessing struct {
	Function string                  `json:"post_processing"` // call routines for specific handling, eg dates, json extracts
	Params   []string                `json:"params"`          // used to pass params to PostProcessing functions, eg date format
	SubRule  GenericActorScraperRule `json:"sub_rule"`        // sub rules allow for a foreach within a foreach, use Function CollyForEach
}

type StashSiteConfig struct {
	StashId     string
	ParentId    string
	TagIdFilter string
	Rules       []SceneMatchRule
}
type SceneMatchRule struct {
	XbvrField                string
	XbvrMatch                string
	XbvrMatchResultPosition  int
	StashField               string
	StashRule                string
	StashMatchResultPosition int
}

func (o *ExternalReference) GetIfExist(id uint) error {
	commonDb, _ := GetCommonDB()

	return commonDb.Preload("XbvrLinks").Where(&ExternalReference{ID: id}).First(o).Error
}

func (o *ExternalReference) FindExternalUrl(externalSource string, externalUrl string) error {
	commonDb, _ := GetCommonDB()

	return commonDb.Preload("XbvrLinks").Where(&ExternalReference{ExternalSource: externalSource, ExternalURL: externalUrl}).First(o).Error
}

func (o *ExternalReference) FindExternalId(externalSource string, externalId string) error {
	commonDb, _ := GetCommonDB()

	return commonDb.Preload("XbvrLinks").Where(&ExternalReference{ExternalSource: externalSource, ExternalId: externalId}).First(o).Error
}

func (o *ExternalReferenceLink) FindByInternalID(internalTable string, internalId uint) []ExternalReferenceLink {
	commonDb, _ := GetCommonDB()
	var refs []ExternalReferenceLink
	commonDb.Preload("ExternalReference").Where(&ExternalReferenceLink{InternalTable: internalTable, InternalDbId: internalId}).Find(&refs)
	return refs
}
func (o *ExternalReferenceLink) FindByInternalName(internalTable string, internalName string) []ExternalReferenceLink {
	commonDb, _ := GetCommonDB()
	var refs []ExternalReferenceLink
	commonDb.Preload("ExternalReference").Where(&ExternalReferenceLink{InternalTable: internalTable, InternalNameId: internalName}).Find(&refs)
	return refs
}
func (o *ExternalReferenceLink) FindByExternalSource(internalTable string, internalId uint, externalSource string) []ExternalReferenceLink {
	commonDb, _ := GetCommonDB()
	var refs []ExternalReferenceLink
	commonDb.Preload("ExternalReference").Where(&ExternalReferenceLink{InternalTable: internalTable, InternalDbId: internalId, ExternalSource: externalSource}).Find(&refs)
	return refs
}
func (o *ExternalReferenceLink) FindByExternaID(externalSource string, externalId string) {
	commonDb, _ := GetCommonDB()
	commonDb.Preload("ExternalReference").Where(&ExternalReferenceLink{ExternalSource: externalSource, ExternalId: externalId}).Find(&o)
}

func (o *ExternalReference) Save() {
	commonDb, _ := GetCommonDB()

	err := retry.Do(
		func() error {
			err := commonDb.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *ExternalReference) Delete() {
	commonDb, _ := GetCommonDB()
	commonDb.Delete(&o)
}

func (o *ExternalReferenceLink) Delete() {
	commonDb, _ := GetCommonDB()
	commonDb.Delete(&o)
}

func (o *ExternalReference) AddUpdateWithUrl() {
	commonDb, _ := GetCommonDB()

	existingRef := ExternalReference{ExternalSource: o.ExternalSource, ExternalURL: o.ExternalURL}
	existingRef.FindExternalUrl(o.ExternalSource, o.ExternalURL)
	if existingRef.ID > 0 {
		o.ID = existingRef.ID
		for _, oldlink := range existingRef.XbvrLinks {
			for idx, newLink := range o.XbvrLinks {
				if newLink.InternalDbId == oldlink.InternalDbId {
					o.XbvrLinks[idx].ID = oldlink.ID
				}
			}
		}
	}

	err := retry.Do(
		func() error {
			err := commonDb.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *ExternalReference) AddUpdateWithId() {
	commonDb, _ := GetCommonDB()

	existingRef := ExternalReference{ExternalSource: o.ExternalSource, ExternalId: o.ExternalId}
	existingRef.FindExternalId(o.ExternalSource, o.ExternalId)
	if existingRef.ID > 0 {
		o.ID = existingRef.ID
		for _, oldlink := range existingRef.XbvrLinks {
			for idx, newLink := range o.XbvrLinks {
				if newLink.InternalDbId == oldlink.InternalDbId {
					o.XbvrLinks[idx].ID = oldlink.ID
				}
			}
		}
	}

	err := retry.Do(
		func() error {
			err := commonDb.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *ExternalReferenceLink) Save() {
	commonDb, _ := GetCommonDB()

	err := retry.Do(
		func() error {
			err := commonDb.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *ExternalReferenceLink) Find(externalSource string, internalName string) error {
	commonDb, _ := GetCommonDB()

	return commonDb.Where(&ExternalReferenceLink{ExternalSource: externalSource, InternalNameId: internalName}).First(o).Error
}

func FormatInternalDbId(input uint) string {
	if input == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(input), 10)
}

func InternalDbId2Uint(input string) uint {
	if input == "" {
		return 0
	}
	val, _ := strconv.Atoi(input)
	return uint(val)
}

func (o *ExternalReference) DetermineActorScraperByUrl(url string) string {
	url = strings.ToLower(url)
	site := url
	re := regexp.MustCompile(`^(https?:\/\/)?(www\.)?([a-zA-Z0-9\-]+)\.[a-zA-Z]{2,}(\/.*)?`)
	match := re.FindStringSubmatch(url)
	if len(match) >= 3 {
		site = match[3]
	}

	switch site {
	case "stashdb":
		return "stashdb performer"
	case "sexlikereal":
		return "slr scrape"
	case "xsinsvr":
		return "sinsvr scrape"
	case "naughtyamerica":
		return "naughtyamericavr scrape"
	case "virtualporn":
		return "bvr scrape"
	case "fuckpassvr":
		return "fuckpassvr-native scrape"
	default:
		return site + " scrape"
	}
}

func (o *ExternalReference) DetermineActorScraperBySiteId(siteId string) string {
	commonDb, _ := GetCommonDB()

	var site Site
	commonDb.Where("id = ?", siteId).First(&site)
	if site.Name == "" {
		return siteId
	}

	if strings.HasSuffix(site.Name, "POVR)") {
		return "povr scrape"
	}
	if strings.HasSuffix(site.Name, "SLR)") {
		return "slr scrape"
	}
	if strings.HasSuffix(site.Name, "VRP Hub)") {
		return "vrphub scrape"
	}
	if strings.HasSuffix(site.Name, "VRPorn)") {
		return "slr scrape"
	}
	return siteId + " scrape"
}

// Scrape Config Rules
func BuildActorScraperRules() ActorScraperConfig {
	var config ActorScraperConfig
	config.GenericActorScrapingConfig = make(map[string]GenericScraperRuleSet)
	config.StashSceneMatching = map[string][]StashSiteConfig{}
	config.loadActorScraperRules()
	return config
}

func (config ActorScraperConfig) loadActorScraperRules() {
	config.buildGenericActorScraperRules()
	config.getSiteUrlMatchingRules()
	config.getCustomRules()
}

func (scrapeRules ActorScraperConfig) buildGenericActorScraperRules() {
	commonDb, _ := GetCommonDB()
	var sites []Site

	// To understand the regex used, sign up to chat.openai.com and just ask something like Explain (.*, )?(.*)$
	// To test regex I use https://regex101.com/
	siteDetails := GenericScraperRuleSet{}
	siteDetails.Domain = "zexyvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `li:contains("Birth date") > b`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"Jan 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `li:contains("Height") > b:first-of-type`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: "li:contains(\"Nationality\") > b", PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: "li:contains(\"Bra size\") > b:first-of-type", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: "li:contains(\"Bra size\") > b:first-of-type", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`[A-K]{1,2}`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: "li:contains(\"Eye Color\") > b"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: "li:contains(\"Hair Color\") > b"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: "li:contains(\"Weight\") > b:first-of-type", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "images", Selector: `div.col-12.col-lg-5 > img, div.col-12.col-lg-7 img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.col-12.col-lg-5 > img`, ResultType: "attr", Attribute: "src", First: optional.NewInt(0)})
	scrapeRules.GenericActorScrapingConfig["zexyvr scrape"] = siteDetails

	siteDetails.Domain = "wankitnowvr.com"
	scrapeRules.GenericActorScrapingConfig["wankitnowvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.sexlikereal.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"image"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "birth_date", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"birthDate"}},
			{Function: "Parse Date", Params: []string{"January 2, 2006"}},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"height"}},
		{Function: "RegexString", Params: []string{`(\d{3})\s?cm`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"weight"}},
		{Function: "RegexString", Params: []string{`(\d{2,3})\s?kg`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"nationality"}},
			{Function: "RegexString", Params: []string{`^(.*,)?\s?(.*)$`, "2"}},
			{Function: "Lookup Country"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"description"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div[data-qa="model-info-aliases"] div.u-wh`})
	scrapeRules.GenericActorScrapingConfig["slr-originals scrape"] = siteDetails
	scrapeRules.GenericActorScrapingConfig["slr-jav-originals scrape"] = siteDetails
	commonDb.Where("name like ?", "%SLR)").Find(&sites)
	scrapeRules.GenericActorScrapingConfig["slr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "baberoticavr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div[id="model"] div:contains('Birth date:')+div`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"January 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `div[id="model"] div:contains('Eye Color:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div[id="model"] div:contains('Hair color:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div[id="model"] div:contains('Height:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `div[id="model"] div:contains('Weight:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `div[id="model"] div:contains('Ethnicity:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div[id="model"] div:contains('Country:')+div`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div[id="model"] div:contains('Aliases:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.m5 img`, ResultType: "attr", Attribute: "src", First: optional.NewInt(0)})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(B)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(W)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(H)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `div[id="model"] div:contains('Breasts Cup:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`[A-K]{1,2}`, "0"}}}})
	scrapeRules.GenericActorScrapingConfig["baberoticavr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `li:contains('Birthdate:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Birthdate: )(.+)`, "2"}}, {Function: "Parse Date", Params: []string{"02/01/2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `li:contains('Country of origin:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Country of origin: )(.+)`, "2"}}, {Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `li:contains('Height:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Height: )(\d{2,3})`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `li:contains('Weight:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Weight: )(\d{2,3})`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `li:contains('Breast Size:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Breast Size: )(\d{2,3})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `li:contains('Breast Size:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Breast Size: )(\d{2,3})(.+)`, "3"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `li:contains('Hair color:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Hair color: )(.+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `li:contains('Eye color:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Eye color: )(.+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div.list_aliases_pornstar li`})
	scrapeRules.GenericActorScrapingConfig["vrporn scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "virtualrealporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "birth_date", Selector: `script[type="application/ld+json"][class!='yoast-schema-graph']`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"birthDate"}},
			{Function: "Parse Date", Params: []string{"01/02/2006"}},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `script[type="application/ld+json"][class!='yoast-schema-graph']`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"birthPlace"}}, {Function: "Lookup Country"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "image_url", Selector: `script[type="application/ld+json"][class!='yoast-schema-graph']`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"image"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `table[id="table_about"] tr th:contains('Eyes Color')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `table[id="table_about"] tr th:contains('Hair Color')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `table[id="table_about"] tr th:contains('Bust')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `table[id="table_about"] tr th:contains('Waist')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `table[id="table_about"] tr th:contains('Hips')+td`})
	scrapeRules.GenericActorScrapingConfig["virtualrealporn scrape"] = siteDetails

	siteDetails.Domain = "virtualrealtrans.com"
	scrapeRules.GenericActorScrapingConfig["virtualrealtrans scrape"] = siteDetails

	siteDetails.Domain = "virtualrealgay.com"
	scrapeRules.GenericActorScrapingConfig["virtualrealgay scrape"] = siteDetails

	siteDetails.Domain = "virtualrealpassion.com"
	scrapeRules.GenericActorScrapingConfig["virtualrealpassion scrape"] = siteDetails

	siteDetails.Domain = "virtualrealamateurporn.com"
	scrapeRules.GenericActorScrapingConfig["virtualrealamateurporn scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.groobyvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "image_url", Selector: `div.model_photo img`, ResultType: "attr", Attribute: "src",
		PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}},
	})

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div[id="bio"] ul`, First: optional.NewInt(1), Last: optional.NewInt(1)})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "ethnicity", Selector: `div[id="bio"] li:contains('Ethnicity:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Ethnicity: )(.+)`, "2"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `div[id="bio"] li:contains('Nationality:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Nationality: )(.+)`, "2"}}, {Function: "Lookup Country"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "height", Selector: `div[id="bio"] li:contains('Height:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Height: )(.+)`, "2"}}, {Function: "Feet+Inches to cm", Params: []string{`(\d+)\'(\d+)\"`, "1", "2"}}},
	})
	scrapeRules.GenericActorScrapingConfig["groobyvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.hologirlsvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "height", Selector: `.starBio`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d+\s*ft\s*\d+\s*in`, "0"}},
			{Function: "Replace", Params: []string{" ft ", `'`}},
			{Function: "Replace", Params: []string{" in", `"`}},
			{Function: "Feet+Inches to cm", Params: []string{`(\d+)\'(\d+)\"`, "1", "2"}},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}-\d{2,3}-\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})-\d{2,3}-\d{2,3}`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "waist_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-(\d{2,3})-\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hip_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-\d{2,3}-(\d{2,3})`, "1"}},
			{Function: "inch to cm"},
		},
	})
	scrapeRules.GenericActorScrapingConfig["hologirlsvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrbangers.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.single-model-profile__image > img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.single-model-biography__content div.toggle-content__text`, First: optional.NewInt(1), Last: optional.NewInt(1)})
	scrapeRules.GenericActorScrapingConfig["vrbangers scrape"] = siteDetails
	siteDetails.Domain = "vrbtrans.com"
	scrapeRules.GenericActorScrapingConfig["vrbtrans scrape"] = siteDetails
	siteDetails.Domain = "vrbgay.com"
	scrapeRules.GenericActorScrapingConfig["vrbgay scrape"] = siteDetails
	siteDetails.Domain = "vrconk.com"
	scrapeRules.GenericActorScrapingConfig["vrconk scrape"] = siteDetails
	siteDetails.Domain = "blowvr.com"
	scrapeRules.GenericActorScrapingConfig["blowvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "virtualporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `section[data-cy="actorProfilePicture"] img`, ResultType: "attr", Attribute: "src"})
	scrapeRules.GenericActorScrapingConfig["bvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "realitylovers.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "image_url", Selector: `img.girlDetails-posterImage`, ResultType: "attr", Attribute: "srcset",
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(.*) \dx,`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(.{3} \d{2}.{2} \d{4})`, "1"}},
		{Function: "Parse Date", Params: []string{"Jan 02 2006"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`Country:\s*(.*)`, "1"}},
		{Function: "Lookup Country"},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) kg`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `.girlDetails-bio`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Biography:\s*(.*)`, "1"}}}})
	scrapeRules.GenericActorScrapingConfig["realitylovers scrape"] = siteDetails

	siteDetails.Domain = "tsvirtuallovers.com"
	scrapeRules.GenericActorScrapingConfig["tsvirtuallovers scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrphub.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `.model-thumb img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `span.details:contains("Aliases:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `span.details:contains("Ethnicity:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}-\d{2,3}-\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})-\d{2,3}-\d{2,3}`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "waist_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-(\d{2,3})-\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hip_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-\d{2,3}-(\d{2,3})`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `span.details:contains("Bra cup size:") + span.details-value`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `span.details:contains("Bra cup size:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "tattoos", Selector: `span.tattoo-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(No Tattoos)?(.*)`, "2"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "piercings", Selector: `span.details:contains("Piercings:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(No Piercings)?(.*)`, "2"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `span.bio-details`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "birth_date", Selector: `span.details:contains("Date of birth:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"January 2, 2006"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `span.details:contains("Birthplace:") + span.details-value`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(.*, )?(.*)$`, "2"}},
			{Function: "Lookup Country"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `span.details:contains("Hair Color:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `span.details:contains("Eye Color:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `span.details:contains("Height:") + span.details-value`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `span.details:contains("Weight:") + span.details-value`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) kg`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "urls", Selector: `.model-info-block2 a`, ResultType: "attr", Attribute: "href"})
	scrapeRules.GenericActorScrapingConfig["vrphub scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrhush.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"props.pageProps.model.gender"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `.thumbnail img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"props.pageProps.model.Bio"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"props.pageProps.model.eyes"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"props.pageProps.model.hair"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"props.pageProps.model.race"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"props.pageProps.model.height"}},
		{Function: "Feet+Inches to cm", Params: []string{`(\d+)\'(\d+)\"`, "1", "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `script[Id="__NEXT_DATA__"]`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"props.pageProps.model.weight"}},
		{Function: "lbs to kg"}}})
	scrapeRules.GenericActorScrapingConfig["vrhush scrape"] = siteDetails

	siteDetails.Domain = "vrallure.com"
	scrapeRules.GenericActorScrapingConfig["vrallure scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrlatina.com"
	// The data-pagespeed-lazy-src attribute holds the URL of the image that should be loaded lazily, the PageSpeed module dynamically replaces the data-pagespeed-lazy-src attribute with the standard src attribute, triggering the actual loading of the image.
	// In my testing sometime, I got the data-pagespeed-lazy-src with a blank image in the src attribute (with a relative url) and other times I just got src with the correct url
	// The following will first load the data-pagespeed-lazy-src then the src attribute.  The check for thehttp prefix, stops the blank image been processed with the relative url
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model-avatar img`, ResultType: "attr", Attribute: "data-pagespeed-lazy-src", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(http.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model-avatar img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(http.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `ul.model-list>li:contains("Aka:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `ul.model-list>li:contains("Dob:")>span+span`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"2006-01-02"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `ul.model-list>li:contains("Height:")>span+span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `ul.model-list>li:contains("Weight:")>span+span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `ul.model-list>li:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `ul.model-list>li:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `ul.model-list>li:contains("Hair:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `ul.model-list>li:contains("Eyes:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `ul.model-list>li:contains("Biography:")>span+span`})
	scrapeRules.GenericActorScrapingConfig["vrlatina scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "badoinkvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.girl-details-photo`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}-\d{2,3}-\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})-\d{2,3}-\d{2,3}`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "waist_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-(\d{2,3})-\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hip_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-\d{2,3}-(\d{2,3})`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "height", Selector: `.girl-details-stats-item:contains("Height:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "Feet+Inches to cm", Params: []string{`(\d+)\D*(\d{1,2})`, "1", "2"}}},
	})

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "weight", Selector: `.girl-details-stats-item:contains("Weight:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})`, "1"}}, {Function: "lbs to kg"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `.girl-details-stats-item:contains("Aka:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `.girl-details-stats-item:contains("Country:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "Lookup Country"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `.girl-details-stats-item:contains("Hair:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `.girl-details-stats-item:contains("Eyes:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `.girl-details-stats-item:contains("Ethnicity:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.girl-details-bio p`})
	scrapeRules.GenericActorScrapingConfig["badoinkvr scrape"] = siteDetails

	siteDetails.Domain = "babevr.com"
	scrapeRules.GenericActorScrapingConfig["babevr scrape"] = siteDetails
	siteDetails.Domain = "vrcosplayx.com"
	scrapeRules.GenericActorScrapingConfig["vrcosplayx scrape"] = siteDetails
	siteDetails.Domain = "18vr.com"
	scrapeRules.GenericActorScrapingConfig["18vr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "darkroomvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.pornstar-detail__picture`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "urls", Selector: `div.pornstar-detail__social a`, ResultType: "attr", Attribute: "href"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `div.pornstar-detail__info span`, Last: optional.NewInt(1),
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`^(.*?),`, "1"}},
			{Function: "Lookup Country"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "start_year", Selector: `div.pornstar-detail__info span:contains("Career Start")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Career Start .*(\d{4})`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "aliases", Selector: `div.pornstar-detail__info span:contains("aka ")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`aka (.*)`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "birth_date", Selector: `div.pornstar-detail__params:contains("Birthday:")`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`Birthday: (.{3} \d{1,2}, \d{4})`, "1"}},
			{Function: "Parse Date", Params: []string{"Jan 2, 2006"}},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "waist_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)(\d{2,3})(?:\s?-|\s-\s)\d{2,3}`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hip_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)(\d{2,3})`, "1"}},
			{Function: "inch to cm"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "height", Selector: `div.pornstar-detail__params:contains("Height:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Height:\s*(\d{2,3})\s*cm`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "weight", Selector: `div.pornstar-detail__params:contains("Weight:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Weight:\s*(\d{2,3})\s*kg`, "1"}}},
	})
	scrapeRules.GenericActorScrapingConfig["darkroomvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.fuckpassvr.com"
	siteDetails.IsJson = true

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `data.seo.porn_star.national`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `data.seo.porn_star.ethnicity`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `data.seo.porn_star.eye_color`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `data.seo.porn_star.hair_color`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "band_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}}, {Function: "inch to cm"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "cup_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "waist_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)(\d{2,3})(?:\s?-|\s-\s)\d{2,3}`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hip_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)(\d{2,3})`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `data.seo.porn_star.birthday`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"2006-01-02"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `data.seo.porn_star.height`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `data.seo.porn_star.weight`, PostProcessing: []PostProcessing{{Function: "lbs to kg"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "biography", Selector: `data.seo.porn_star.write_up`,
		PostProcessing: []PostProcessing{
			{Function: "Replace", Params: []string{"<p>", ``}},
			{Function: "Replace", Params: []string{"</p>", `
		`}},
			{Function: "Replace", Params: []string{"<br>", `
		`}},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "urls", Selector: `data.seo.porn_star.slug`,
		PostProcessing: []PostProcessing{{Function: "RegexReplaceAll", Params: []string{`^(.*)$`, `https://www.fuckpassvr.com/vr-pornstars/$0`}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `data.seo.porn_star.thumbnail_url`}) // image will expiry, hopefully cache will keep it
	scrapeRules.GenericActorScrapingConfig["fuckpassvr-native scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "realjamvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.actor-view img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `div.details div div:contains("Gender:")`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Gender: (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `div.details div div:contains("City and Country:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`City and Country:\s?(.*,)?(.*)$`, "2"}}, {Function: "Lookup Country"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "birth_date", Selector: `div.details div div:contains("Date of Birth:")`,
		PostProcessing: []PostProcessing{
			{Function: "RegexString", Params: []string{`Date of Birth: (.*)`, "1"}},
			{Function: "Parse Date", Params: []string{"Jan. 2, 2006"}},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "height", Selector: `div.details div div:contains("Height:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})\s?cm`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "weight", Selector: `div.details div div:contains("Weight:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})\s?kg`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "eye_color", Selector: `div.details div div:contains("Eyes color:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Eyes color: (.*)`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hair_color", Selector: `div.details div div:contains("Hair color:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Hair color: (.*)`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "piercings", Selector: `div.details div div:contains("Piercing:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Piercing:\s?([v|V]arious)?([t|T]rue)?(.*)`, "3"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "tattoos", Selector: `div.details div div:contains("Tattoo:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Tattoo:\s?([v|V]arious)?([t|T]rue)?(.*)`, "3"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "biography", Selector: `div.details div div:contains("About:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`About: (.*)`, "1"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "waist_size", Selector: `div.details div div:contains("Waist:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2})`, "1"}}, {Function: "inch to cm"}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "hip_size", Selector: `div.details div div:contains("Hips:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2})`, "1"}}, {Function: "inch to cm"}},
	})
	scrapeRules.GenericActorScrapingConfig["realjamvr scrape"] = siteDetails

	// use the site rules just setup for realjamvr, just need to update the Domain to use
	siteDetails.Domain = "porncornvr.com"
	scrapeRules.GenericActorScrapingConfig["porncornvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "povr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `script[type="application/ld+json"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"image"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `script[type="application/ld+json"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"gender"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "birth_date", Selector: `script[type="application/ld+json"]`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"birthDate"}}},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
		XbvrField: "nationality", Selector: `script[type="application/ld+json"]`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"birthPlace"}},
			{Function: "RegexString", Params: []string{`^(.*,)?\s?(.*)$`, "2"}},
			{Function: "Lookup Country"},
		},
	})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `script[type="application/ld+json"]`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"height"}},
		{Function: "RegexString", Params: []string{`(\d{3})`, "1"}},
	}})
	scrapeRules.GenericActorScrapingConfig["povr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "tmwvrnet.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model-page__image img`, ResultType: "attr", Attribute: "data-src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "start_year", Selector: `div.model-page__information span.title:contains("Debut year:") + span.value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div.model-page__information span.title:contains("Hair:") + span.value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `div.model-page__information span.title:contains("Eyes:") + span.value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `p.about`})
	scrapeRules.GenericActorScrapingConfig["tmwvrnet scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "xsinsvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `.model-header__photo img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `time`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"02/01/2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(\d{2,3})\s?.{1,2}\s?-\s?\d{2,3}\s?-\s?\d{2,3}?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^\d{2,3}\s?(.{1,2})\s?-\s?\d{2,3}\s?-\s?\d{2,3}?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^\d{2,3}\s?.{1,2}\s?-\s?(\d{2,3})\s?-\s?\d{2,3}?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^\d{2,3}\s?.{1,2}\s?-\s?\d{2,3}\s?-\s?(\d{2,3})?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `h2:contains("Country") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(.*)\s?(([\(|-]))`, "1"}}, // stops at - or (
		{Function: "Lookup Country"},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `h2:contains("Weight") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(\d{2,3})\s?/`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `h2:contains("Weight") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`/\s?(\d{2,3})`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `h2:contains("Hair ") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`^(.*)\s?\/\s?(.*)?`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `h2:contains("Hair ") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`^(.*)\s?\/\s?(.*)?`, "2"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", ResultType: "html", Selector: `div.model-header__intro`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(?s)<h2>Bio<\/h2>(.*)`, "1"}}, // get everything after the H2 Bio
		{Function: "RegexReplaceAll", Params: []string{`<[^>]*>`, ``}},            // replace html tags with nothing, ie remove them
		{Function: "RegexReplaceAll", Params: []string{`^\s+|\s+$`, ``}},
	}}) // now remove leading & trailing whitespace
	scrapeRules.GenericActorScrapingConfig["sinsvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.naughtyamerica.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `p.bio_about_text`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.performer-pic`, ResultType: "attr", Attribute: "data-src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	scrapeRules.GenericActorScrapingConfig["naughtyamericavr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "sexbabesvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.cover-picture`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `h3:contains("Country") + span`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `h3:contains("Weight / Height") + span`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`^(\d{2,3}) ?\/`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `h3:contains("Weight / Height") + span`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`\/ ?(\d{2,3})$`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.model-detail__box`, ResultType: "html", PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`<\/div>\s*([^<]+)$`, "1"}}, // get everything after the last div
		{Function: "UnescapeString"},
	}})
	scrapeRules.GenericActorScrapingConfig["sexbabesvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrspy.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `.star-bio .show-more-text-container`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `.avatar img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "RemoveQueryParams"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "images", Selector: `.avatar img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "RemoveQueryParams"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `.star-info-row-title:contains("Height:") + span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `.star-info-row-title:contains("Weight:") + span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `.star-info-row-title:contains("Measurements:") + span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d+)([A-Za-z]*)-(\d+)-(\d+)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `.star-info-row-title:contains("Measurements:") + span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d+)([A-Za-z]*)-(\d+)-(\d+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `.star-info-row-title:contains("Measurements:") + span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d+)([A-Za-z]*)-(\d+)-(\d+)`, "3"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `.star-info-row-title:contains("Measurements:") + span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d+)([A-Za-z]*)-(\d+)-(\d+)`, "4"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `.star-info-row-title:contains("Nationality:") + span`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `.star-info-row-title:contains("Hair Color:") + span`})
	scrapeRules.GenericActorScrapingConfig["vrspy scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.javdatabase.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img[src^="https://www.javdatabase.com/idolimages/full/"]`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{
		{Function: "AbsoluteUrl"},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "images", Selector: `a[href^="https://pics.dmm.co.jp/digital/video/"]:not([href^="https://pics.dmm.co.jp/digital/video/mdj010/"])`, ResultType: "attr", Attribute: "href", PostProcessing: []PostProcessing{
		{Function: "AbsoluteUrl"},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div[id="biography"] > div`, ResultType: "text"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div > b:contains("Hair Color(s):") + a`, ResultType: "text"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div > b:contains("DOB:") + a`, ResultType: "text"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div > b:contains("Height:") + a`, ResultType: "text", PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`\d+`, "0"}},
	}})

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `div > b:contains("Measurements:")`, ResultType: "text", PostProcessing: []PostProcessing{
		{Function: "DOMNextText"},
		{Function: "RegexString", Params: []string{`(\d+)-(\d+)-(\d+)`, "1"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `div > b:contains("Cup:") + a`, ResultType: "text"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `div > b:contains("Measurements:")`, ResultType: "text", PostProcessing: []PostProcessing{
		{Function: "DOMNextText"},
		{Function: "RegexString", Params: []string{`(\d+)-(\d+)-(\d+)`, "2"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `div > b:contains("Measurements:")`, ResultType: "text", PostProcessing: []PostProcessing{
		{Function: "DOMNextText"},
		{Function: "RegexString", Params: []string{`(\d+)-(\d+)-(\d+)`, "3"}},
	}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div > p > b:contains("Alt:")`, ResultType: "text", PostProcessing: []PostProcessing{
		{Function: "DOMNextText"},
	}})

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `div > p:contains("Tags")`, ResultType: "text", PostProcessing: []PostProcessing{
		{Function: "SetWhenValueContains", Params: []string{"Trans", "Transgender Female"}},
		{Function: "SetWhenValueNotContains", Params: []string{"Trans", "Female"}},
	}})

	scrapeRules.GenericActorScrapingConfig["javdatabase scrape"] = siteDetails
}

// Loads custom rules from actor_scrapers_examples.json
// Building custom rules for Actor scrapers is an advanced task, requiring developer or scraping skills
// Most likely to be used to post updated rules by developers, prior to an offical release
func (scrapeRules ActorScraperConfig) getCustomRules() {
	// first see if we have an example file with the builting rules
	//	this is to give examples, it is not loaded
	fName := filepath.Join(common.AppDir, "actor_scraper_config_examples.json")
	out, _ := json.MarshalIndent(scrapeRules, "", "  ")
	os.WriteFile(fName, out, 0644)

	// now check if the user has any custom rules
	fName = filepath.Join(common.AppDir, "actor_scraper_custom_config.json")
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		if _, err := os.Stat(fName); os.IsNotExist(err) {
			// create a dummy template
			exampleConfig := ActorScraperConfig{
				StashSceneMatching:         make(map[string][]StashSiteConfig),
				GenericActorScrapingConfig: make(map[string]GenericScraperRuleSet),
			}

			siteDetails := GenericScraperRuleSet{}
			siteDetails.Domain = ".com"
			siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{
				XbvrField:  "actor field eg nationailty",
				Selector:   `css selector (or json path is IsJson is true) to find the data in the actors web page`,
				ResultType: "blank (text), attr or html",
				Attribute:  "attribute id you want, eg src for an image of href for a link",
				PostProcessing: []PostProcessing{{
					Function: "builtin function to apply to the extarcted text, eg RegexString to extract with regex, Parse Date, lbs to kg, see postProcessing function for options.  You may specify multiple function, eg RegexString to extract a Date followed by Parse Date if not in the right format",
					Params:   []string{`Parameter depends on the functions requirements `},
				}},
			})
			exampleConfig.GenericActorScrapingConfig["example scrape"] = siteDetails

			stashMatch := StashSiteConfig{}
			stashMatch.StashId = "Stash guid of the Staudio, used when names don't match exactly"
			stashMatch.ParentId = "Stash guid of parent, if tag filtering used (used by NAVR)"
			stashMatch.TagIdFilter = "Stash guid of tag, if tag filtering used (used by NAVR)"
			stashMatch.Rules = []SceneMatchRule{{
				XbvrField:                "Enter xbvr field you are matching to, scene_url or scene_id",
				XbvrMatch:                "Enter regex express to extract value from field to match on",
				XbvrMatchResultPosition:  0,
				StashField:               "Enter the stash field to cmpare, default Url",
				StashRule:                "Enter rule name, ie title, title/date, studio_code or regex expression to extract value to match from the stash url",
				StashMatchResultPosition: 0,
			}}
			exampleConfig.StashSceneMatching["siteid"] = []StashSiteConfig{stashMatch}

			out, _ := json.MarshalIndent(exampleConfig, "", "  ")
			os.WriteFile(fName, out, 0644)
		}
	} else {
		// load any custom rules and update the builtin list
		var customScrapeRules ActorScraperConfig
		b, err := os.ReadFile(fName)
		if err != nil {
			log.Infof("Error reading actor_scraper_custom_config %s", err.Error())
			return
		}
		json.Unmarshal(b, &customScrapeRules)
		for key, rule := range customScrapeRules.GenericActorScrapingConfig {
			if key != " scrape" {
				scrapeRules.GenericActorScrapingConfig[key] = rule
			}
		}
		for key, matchrule := range customScrapeRules.StashSceneMatching {
			scrapeRules.StashSceneMatching[key] = matchrule
		}
	}
}

func (scrapeRules ActorScraperConfig) getSiteUrlMatchingRules() {
	commonDb, _ := GetCommonDB()

	var sites []Site

	// if the scene_url in xbvr and stash typically matches, then no special rules required
	scrapeRules.StashSceneMatching["allvrporn-vrporn"] = []StashSiteConfig{StashSiteConfig{StashId: "44fd483b-85eb-4b22-b7f2-c92c1a50923a"}}
	scrapeRules.StashSceneMatching["bvr"] = []StashSiteConfig{StashSiteConfig{StashId: "1ffbd972-7d69-4ccb-b7da-c6342a9c3d70"}}
	scrapeRules.StashSceneMatching["cuties-vr"] = []StashSiteConfig{StashSiteConfig{StashId: "1e5240a8-29b3-41ed-ae28-fc9231eac449"}}
	scrapeRules.StashSceneMatching["czechvrintimacy"] = []StashSiteConfig{StashSiteConfig{StashId: "ddff31bc-e9d0-475e-9c5b-1cc151eda27b"}}
	scrapeRules.StashSceneMatching["darkroomvr"] = []StashSiteConfig{StashSiteConfig{StashId: "e57f0b82-a8d0-4904-a611-71e95f9b9248"}}
	scrapeRules.StashSceneMatching["ellielouisevr"] = []StashSiteConfig{StashSiteConfig{StashId: "47764349-fb49-42b9-8445-7fa4fb13f9e1"}}
	scrapeRules.StashSceneMatching["emilybloom"] = []StashSiteConfig{StashSiteConfig{StashId: "b359a2fe-dcf0-46e2-8ace-a684df52573e"}}
	scrapeRules.StashSceneMatching["herpovr"] = []StashSiteConfig{StashSiteConfig{StashId: "7d94a83d-2b0b-4076-9e4c-fd9dc6222b8a"}}
	scrapeRules.StashSceneMatching["jimmydraws"] = []StashSiteConfig{StashSiteConfig{StashId: "bf7b7b9a-b96a-401d-8412-ec3f52bcfb6c"}}
	scrapeRules.StashSceneMatching["kinkygirlsberlin"] = []StashSiteConfig{StashSiteConfig{StashId: "7d892a03-dfbe-4476-917d-4940be13fb24"}}
	scrapeRules.StashSceneMatching["lethalhardcorevr"] = []StashSiteConfig{StashSiteConfig{StashId: "3a9883f6-9642-4be1-9a65-d8d13eadbdf0"}}
	scrapeRules.StashSceneMatching["lustreality"] = []StashSiteConfig{StashSiteConfig{StashId: "f31021ba-f4c3-46eb-89c5-b114478d88d2"}}
	scrapeRules.StashSceneMatching["mongercash"] = []StashSiteConfig{StashSiteConfig{StashId: "96ee2435-0b0f-4fb4-8b53-8c929aa493bd"}}
	scrapeRules.StashSceneMatching["only3xvr"] = []StashSiteConfig{StashSiteConfig{StashId: "57391302-bac4-4f15-a64d-7cd9a9c152e0"}}
	scrapeRules.StashSceneMatching["povcentralvr"] = []StashSiteConfig{StashSiteConfig{StashId: "57391302-bac4-4f15-a64d-7cd9a9c152e0"}}
	scrapeRules.StashSceneMatching["realhotvr"] = []StashSiteConfig{StashSiteConfig{StashId: "cf3510db-5fe5-4212-b5da-da27b5352d1c"}}
	scrapeRules.StashSceneMatching["realitylovers"] = []StashSiteConfig{StashSiteConfig{StashId: "3463e72d-6af3-497f-b841-9119065d2916"}}
	scrapeRules.StashSceneMatching["sinsvr"] = []StashSiteConfig{StashSiteConfig{StashId: "805820d0-8fb2-4b04-8c0c-6e392842131b"}}
	scrapeRules.StashSceneMatching["squeeze-vr"] = []StashSiteConfig{StashSiteConfig{StashId: "b2d048da-9180-4e43-b41a-bdb4d265c8ec"}}
	scrapeRules.StashSceneMatching["swallowbay"] = []StashSiteConfig{StashSiteConfig{StashId: "17ff0143-3961-4d38-a80a-fe72407a274d"}}
	scrapeRules.StashSceneMatching["tonightsgirlfriend"] = []StashSiteConfig{StashSiteConfig{StashId: "69a66a95-15de-4b0a-9537-7f15b358392f"}}
	scrapeRules.StashSceneMatching["virtualrealamateur"] = []StashSiteConfig{StashSiteConfig{StashId: "cac0470b-7802-4946-b5ef-e101e166cdaf"}}
	scrapeRules.StashSceneMatching["virtualtaboo"] = []StashSiteConfig{StashSiteConfig{StashId: "1e6defb1-d3a4-4f0c-8616-acd5c343ca2b"}}
	scrapeRules.StashSceneMatching["virtualxporn"] = []StashSiteConfig{StashSiteConfig{StashId: "d55815ac-955f-45a0-a0fa-f6ad335e212d"}}
	scrapeRules.StashSceneMatching["vrbangers"] = []StashSiteConfig{StashSiteConfig{StashId: "f8a826f6-89c2-4db0-a899-1229d11865b3"}}
	scrapeRules.StashSceneMatching["vrconk"] = []StashSiteConfig{StashSiteConfig{StashId: "b038d55c-1e94-41ff-938a-e6aafb0b1759"}}
	scrapeRules.StashSceneMatching["vrmansion-slr"] = []StashSiteConfig{StashSiteConfig{StashId: "a01012bc-42e9-4372-9c25-58f0f94e316b"}}
	scrapeRules.StashSceneMatching["vrsexygirlz"] = []StashSiteConfig{StashSiteConfig{StashId: "b346fe21-5d12-407f-9f50-837f067956d7"}}
	scrapeRules.StashSceneMatching["vrsolos"] = []StashSiteConfig{StashSiteConfig{StashId: "b2d048da-9180-4e43-b41a-bdb4d265c8ec"}}
	scrapeRules.StashSceneMatching["vrspy"] = []StashSiteConfig{StashSiteConfig{StashId: "513001ef-dff4-476d-840d-e22ef27e81ed"}}
	scrapeRules.StashSceneMatching["wankitnowvr"] = []StashSiteConfig{StashSiteConfig{StashId: "acb1ed8f-4967-4c5a-b16a-7025bdeb75c5"}}
	scrapeRules.StashSceneMatching["porncornvr"] = []StashSiteConfig{StashSiteConfig{StashId: "9ecb1d29-64e8-4336-9bd2-5dda53341e29"}}

	scrapeRules.StashSceneMatching["wetvr"] = []StashSiteConfig{StashSiteConfig{StashId: "981887d6-da48-4dfc-88d1-7ed13a2754f2"}}

	// setup special rules to match scenes in xbvr and stashdb, rather than assuming scene_urls match
	scrapeRules.StashSceneMatching["wankzvr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "b04bca51-15ea-45ab-80f6-7b002fd4a02d",
		Rules:   []SceneMatchRule{{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}},
	}}
	scrapeRules.StashSceneMatching["naughtyamericavr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "049c167b-0cf3-4965-aae5-f5150122a928", ParentId: "2be8463b-0505-479e-a07d-5abc7a6edd54", TagIdFilter: "6458e5cf-4f65-400b-9067-582141e2a329",
		Rules: []SceneMatchRule{{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(naughtyamerica).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}},
	}}
	scrapeRules.StashSceneMatching["povr-originals"] = []StashSiteConfig{StashSiteConfig{
		StashId: "b95c0ee4-2e95-46cf-aa67-45c82bdcd5fc",
		Rules:   []SceneMatchRule{{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}},
	}}
	scrapeRules.StashSceneMatching["brasilvr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "511e41c8-5063-48b8-a8d9-4e18852da338",
		Rules:   []SceneMatchRule{{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(brasilvr|povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}},
	}}
	scrapeRules.StashSceneMatching["milfvr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "38382977-9f5e-42fb-875b-2f4dd1272b11",
		Rules:   []SceneMatchRule{{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(milfvr|povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}},
	}}

	scrapeRules.StashSceneMatching["czechvr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "a9ed3948-5263-46f6-a3f0-e0dfc059ee73",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}},
	}}
	scrapeRules.StashSceneMatching["czechvrcasting"] = []StashSiteConfig{StashSiteConfig{
		StashId: "2fa76fba-ccd7-457d-bc7c-ebc1b09e580b",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}},
	}}
	scrapeRules.StashSceneMatching["czechvrfetish"] = []StashSiteConfig{StashSiteConfig{
		StashId: "19399096-7b83-4404-b960-f8f8c641a93e",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}},
	}}
	scrapeRules.StashSceneMatching["czechvrintimacy"] = []StashSiteConfig{StashSiteConfig{
		StashId: "ddff31bc-e9d0-475e-9c5b-1cc151eda27b",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}},
	}}
	scrapeRules.StashSceneMatching["tmwvrnet"] = []StashSiteConfig{StashSiteConfig{
		StashId: "fd1a7f1d-9cc3-4d30-be0d-1c05b2a8b9c3",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(teenmegaworld.net|tmwvrnet.com)(\/trailers)?\/([^\/]+)\/?$`, XbvrMatchResultPosition: 3, StashRule: `(teenmegaworld.net|tmwvrnet.com)(\/trailers)?\/([^\/]+)\/?$`, StashMatchResultPosition: 3}},
	}}
	scrapeRules.StashSceneMatching["virtualrealporn"] = []StashSiteConfig{StashSiteConfig{
		StashId: "191ba106-00d3-4f01-8c57-0cf0e88a2a50",
		Rules: []SceneMatchRule{
			{XbvrField: "scene_url", XbvrMatch: `virtualrealporn`, XbvrMatchResultPosition: 3, StashRule: `(\/[^\/]+)\/?$`, StashMatchResultPosition: 1},
			{XbvrField: "scene_url", XbvrMatch: `virtualrealporn`, XbvrMatchResultPosition: 3, StashRule: `(\/[^\/]+)(-\d{3,10}?)\/?$`, StashMatchResultPosition: 1},
		},
	}}
	scrapeRules.StashSceneMatching["realjamvr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "2059fbf9-94fe-4986-8565-2a7cc199636a",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(realjamvr.com)(.*)\/(\d*-?)([^\/]+)\/?$`, XbvrMatchResultPosition: 4, StashRule: `(realjamvr.com)(.*)\/(\d*-?)([^\/]+)\/?$`, StashMatchResultPosition: 4}},
	}}
	scrapeRules.StashSceneMatching["vrhush"] = []StashSiteConfig{StashSiteConfig{
		StashId: "c85a3d13-c1b9-48d0-986e-3bfceaf0afe5",
		// ignores optional /vrh999_ from old urls
		Rules: []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `\/([^\/]+)$`, XbvrMatchResultPosition: 4, StashRule: `\/((vrh\d+)_)?([^\/?]+)(?:\?.*)?$`, StashMatchResultPosition: 3}, // handle trailing query params
			{XbvrField: "scene_url", XbvrMatch: `\/([^\/]+)$`, XbvrMatchResultPosition: 4, StashRule: `\/((vrh\d+)_)?([^\/?]+)(?:_180.*)?$`, StashMatchResultPosition: 3}, // handle _180 suffix now gone from urls
		},
	}}
	scrapeRules.StashSceneMatching["sexbabesvr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "b80d419c-4a81-44c9-ae79-d9614dd30351",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(sexbabesvr.com)(.*)\/([^\/]+)\/?$`, XbvrMatchResultPosition: 3, StashRule: `(sexbabesvr.com)(.*)\/([^\/]+)\/?$`, StashMatchResultPosition: 3}},
	}}
	scrapeRules.StashSceneMatching["lethalhardcorevr"] = []StashSiteConfig{StashSiteConfig{
		StashId: "3a9883f6-9642-4be1-9a65-d8d13eadbdf0",
		Rules:   []SceneMatchRule{{XbvrField: "scene_url", XbvrMatch: `(lethalhardcorevr.com).*\/(\d{6,8})\/.*`, XbvrMatchResultPosition: 2, StashRule: `(lethalhardcorevr.com).*\/(\d{6,8})\/.*`, StashMatchResultPosition: 2}},
	}}

	commonDb.Where(&Site{IsEnabled: true}).Order("id").Find(&sites)
	for _, site := range sites {
		if _, found := scrapeRules.StashSceneMatching[site.ID]; !found {
			if strings.HasSuffix(site.Name, "SLR)") {
				siteConfig := scrapeRules.StashSceneMatching[site.ID]
				extRefLink := ExternalReferenceLink{}
				extRefLink.Find("stashdb studio", site.ID)
				siteConfig = []StashSiteConfig{StashSiteConfig{}}
				siteConfig[0].StashId = extRefLink.ExternalId
				siteConfig[0].Rules = append(siteConfig[0].Rules, SceneMatchRule{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(sexlikereal).com\/[^0-9]*(-\d*)`, StashMatchResultPosition: 2})
				scrapeRules.StashSceneMatching[site.ID] = siteConfig
			}
			if strings.HasSuffix(site.Name, "POVR)") {
				siteConfig := scrapeRules.StashSceneMatching[site.ID]
				extRefLink := ExternalReferenceLink{}
				extRefLink.Find("stashdb studio", site.ID)
				siteConfig = []StashSiteConfig{StashSiteConfig{}}
				siteConfig[0].StashId = extRefLink.ExternalId
				if len(siteConfig[0].Rules) == 0 {
					siteConfig[0].Rules = append(siteConfig[0].Rules, SceneMatchRule{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashRule: `(povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 2})
					scrapeRules.StashSceneMatching[site.ID] = siteConfig
				}
			}
		}
	}
}
