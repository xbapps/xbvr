package scrape

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
	nethtml "golang.org/x/net/html"
)

type outputList struct {
	Id       uint
	Url      string
	Linktype string
}

var mutex sync.Mutex
var semaphore chan struct{}

func GenericActorScrapers() {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Scraping Actor Details from Sites")

	commonDb, _ := models.GetCommonDB()

	scraperConfig := models.BuildActorScraperRules()

	var actors []models.Actor
	commonDb.Preload("Scenes").
		Where("id =1").
		Find(&actors)

	sqlcmd := ""

	var output []outputList
	switch commonDb.Dialect().GetName() {
	// gets the list of an actors Urls for scraper and join to external_reference_links to see if the haven't been linked
	case "mysql":
		sqlcmd = `
		WITH actorlist AS (
			SELECT actors.id, trim('"' from JSON_EXTRACT(json_each.value, '$.url')) AS url, trim('"' from JSON_EXTRACT(json_each.value, '$.type')) AS linktype
			FROM actors
			CROSS JOIN JSON_TABLE(actors.urls, '$[*]' COLUMNS(value JSON PATH '$')) AS json_each
			WHERE urls like '% scrape%' and JSON_TYPE(actors.urls) = 'ARRAY'
		)
		SELECT actorlist.id, url, linktype
		FROM actorlist
		LEFT JOIN external_reference_links erl ON erl.internal_db_id = actorlist.id AND external_source = linktype
		where erl.id is null and linktype like '% scrape'
			`
	case "sqlite3":
		sqlcmd = `
		with actorlist as (
			SELECT actors.id, json_extract(json_each.value, '$.url') as url, json_extract(json_each.value, '$.type') as linktype
			FROM actors, json_each(urls)
			WHERE urls like '% scrape%' and  json_type(urls) = 'array'
			and json_extract(json_each.value, '$.type') like '% scrape'
		)
		select actorlist.id, url, linktype from actorlist
		left join external_reference_links erl on erl.internal_db_id = actorlist.id and external_source = linktype
		where erl.id is null and linktype like '% scrape'
			`
	}

	processed := 0
	lastMessage := time.Now()
	commonDb.Raw(sqlcmd).Scan(&output)

	var wg sync.WaitGroup
	concurrentLimit := 10 // Maximum number of concurrent tasks

	semaphore = make(chan struct{}, concurrentLimit)
	actorSemMap := make(map[uint]chan struct{})

	for _, row := range output {
		wg.Add(1)
		go func(row outputList) {

			// Check if a semaphore exists for the actor (don't want to process 2 links for an actor at the same time)
			mutex.Lock()
			actorSem, exists := actorSemMap[row.Id]
			if !exists {
				// Create a new semaphore for the customer
				actorSem = make(chan struct{}, 1)
				actorSemMap[row.Id] = actorSem
			}
			mutex.Unlock()
			actorSem <- struct{}{} // Acquire the semaphore
			semaphore <- struct{}{}

			processAuthorLink(row, scraperConfig.GenericActorScrapingConfig, &wg)
			processed += 1
			<-actorSem

			if time.Since(lastMessage) > time.Duration(config.Config.Advanced.ProgressTimeInterval)*time.Second {
				tlog.Infof("Scanned %v of %v Actors Links to Sites", processed, len(output))
				lastMessage = time.Now()
			}

		}(row)
	}
	wg.Wait()
	tlog.Infof("Scraping Actors Completed")
}

func processAuthorLink(row outputList, siteRules map[string]models.GenericScraperRuleSet, wg *sync.WaitGroup) {
	defer wg.Done()
	var actor models.Actor
	actor.GetIfExistByPK(row.Id)
	for source, rule := range siteRules {
		// this handles examples like 'vrphub-vrhush scrape' needing to match 'vrphub scrape'
		if strings.HasPrefix(row.Linktype, strings.TrimSuffix(source, " scrape")) {
			applyRules(row.Url, row.Linktype, rule, &actor, false)
		}
	}
	// Release the semaphore
	<-semaphore
}

func GenericSingleActorScraper(actorId uint, actorPage string) {
	log.Infof("Scraping Actor Details from %s", actorPage)
	commonDb, _ := models.GetCommonDB()

	var actor models.Actor
	actor.ID = actorId
	commonDb.Find(&actor)
	scraperConfig := models.BuildActorScraperRules()

	var extRefLink models.ExternalReferenceLink
	commonDb.Preload("ExternalReference").
		Where(&models.ExternalReferenceLink{ExternalId: actorPage, InternalDbId: actor.ID}).
		First(&extRefLink)

	for source, rule := range scraperConfig.GenericActorScrapingConfig {
		if extRefLink.ExternalSource == source {
			applyRules(actorPage, source, rule, &actor, true)
		}
	}

	log.Infof("Scraping Actor Details from %s Completed", actorPage)
}

// scrapes all the actors for the site, will overwrite existing actor details
func GenericActorScrapersBySite(site string) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Scraping Actor Details from %s", site)

	commonDb, _ := models.GetCommonDB()
	scraperConfig := models.BuildActorScraperRules()

	er := models.ExternalReference{}
	scrapeId := er.DetermineActorScraperBySiteId(site)

	var actors []models.Actor
	commonDb.Debug().Select("DISTINCT actors.*").
		Joins("JOIN scene_cast ON scene_cast.actor_id = actors.id").
		Joins("JOIN scenes ON scenes.id = scene_cast.scene_id").
		Where("scenes.scraper_id = ?", site).
		Find(&actors)

	lastMessage := time.Now()
	for idx, actor := range actors {
		if time.Since(lastMessage) > time.Duration(config.Config.Advanced.ProgressTimeInterval)*time.Second {
			tlog.Infof("Scanned %v of %v Actors", idx, len(actors))
			lastMessage = time.Now()
		}

		// find the url for the actor for this site
		var extreflink models.ExternalReferenceLink
		commonDb.Where(`internal_table = 'actors' and internal_db_id = ? and external_source = ?`, actor.ID, scrapeId).First(&extreflink)
		for source, rule := range scraperConfig.GenericActorScrapingConfig {
			if source == scrapeId {
				applyRules(extreflink.ExternalId, scrapeId, rule, &actor, true)
			}
		}
	}
	tlog.Infof("Scraping Actor Details Completed")
}

func applyRules(actorPage string, source string, rules models.GenericScraperRuleSet, actor *models.Actor, overwrite bool) {
	actorCollector := CreateCollector(rules.Domain)

	data := make(map[string]string)
	actorChanged := false
	if rules.IsJson {
		actorCollector.OnResponse(func(r *colly.Response) {
			log.Info("Colly OnResponse")

			if r.StatusCode != 200 {
				return
			}
			
			resp := gjson.ParseBytes(r.Body)
			log.Info("Colly Body:" + fmt.Sprintf("%s", r.Body ))
			for _, rule := range rules.SiteRules {
				var results []string
				if rule.Native != nil {
					results = rule.Native(&resp)
				} else {
					log.Info ("Selector:" + rule.Selector)
					results = []string{resp.Get(rule.Selector).String()}
					if len(rule.PostProcessing) > 0 {

						results[0] = postProcessing(rule, results[0], nil)
						log.Info ("postProcessing:" + results[0])
					}
				}

				for _, result := range results {
					if assignField(rule.XbvrField, result, actor, overwrite) {
						actorChanged = true
					}
					if data[rule.XbvrField] == "" {
						data[rule.XbvrField] = result
					} else {
						data[rule.XbvrField] = data[rule.XbvrField] + ", " + result
					}
				}
			}
		})
	} else {
		actorCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
			for _, rule := range rules.SiteRules {
				var results []string
				if rule.Native != nil {
					results = rule.Native(e)
				} else {
					recordCnt := 1
					e.ForEach(rule.Selector, func(id int, e *colly.HTMLElement) {
						if !(rule.First.Present() && rule.First.OrElse(0) > recordCnt) || (rule.Last.Present() && recordCnt > rule.Last.OrElse(0)) {
							var result string
							switch rule.ResultType {
							case "text", "":
								result = strings.TrimSpace(e.Text)
							case "attr":
								result = strings.TrimSpace(e.Attr(rule.Attribute))
							case "html":
								result, _ = e.DOM.Html()
							}
							if len(rule.PostProcessing) > 0 {
								result = postProcessing(rule, result, e)
							}
							results = append(results, result)
						}
						recordCnt += 1
					})
				}

				for _, result := range results {
					if assignField(rule.XbvrField, result, actor, overwrite) {
						actorChanged = true
					}
					//log.Infof("set %s to %s", rule.XbvrField, result)
					if data[rule.XbvrField] == "" {
						data[rule.XbvrField] = result
					} else {
						data[rule.XbvrField] = data[rule.XbvrField] + ", " + result
					}
				}
			}
		})
	}
	if rules.IsJson {
		actorCollector.Request("GET", actorPage, nil, nil, nil)
	} else {
		actorCollector.Visit(actorPage)
	}
	var extref models.ExternalReference
	var extreflink models.ExternalReferenceLink

	commonDb, _ := models.GetCommonDB()
	commonDb.Preload("ExternalReference").
		Where(&models.ExternalReferenceLink{ExternalSource: source, InternalDbId: actor.ID}).
		First(&extreflink)
	extref = extreflink.ExternalReference

	if actorChanged || extref.ID == 0 {
		actor.Save()
		dataJson, _ := json.Marshal(data)

		extrefLink := []models.ExternalReferenceLink{{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, ExternalSource: source, ExternalId: actorPage}}
		extref = models.ExternalReference{ID: extref.ID, XbvrLinks: extrefLink, ExternalSource: source, ExternalId: actorPage, ExternalURL: actorPage, ExternalDate: time.Now(), ExternalData: string(dataJson)}
		extref.AddUpdateWithId()
	} else {
		extref.ExternalDate = time.Now()
		extref.AddUpdateWithId()
	}
}
func getSubRuleResult(rule models.GenericActorScraperRule, e *colly.HTMLElement) string {
	recordCnt := 1
	var result string
	e.ForEach(rule.Selector, func(id int, e *colly.HTMLElement) {
		if (rule.First.Present() && rule.First.OrElse(0) > recordCnt) || (rule.Last.Present() && recordCnt > rule.Last.OrElse(0)) {
		} else {
			switch rule.ResultType {
			case "text", "":
				result = strings.TrimSpace(e.Text)
			case "attr":
				result = strings.TrimSpace(e.Attr(rule.Attribute))
			}
			if len(rule.PostProcessing) > 0 {
				result = postProcessing(rule, result, e)
			}
		}
		recordCnt += 1
	})
	return result
}

func assignField(field string, value string, actor *models.Actor, overwrite bool) bool {
	changed := false
	switch field {
	case "birth_date":
		// check Birth date is not in the last 15 years, some sites just set the BirthDay to the current date when created
		// also don't trust existing birth_dates on Jan 1, probably a site been lazy
		t, err := time.Parse("2006-01-02", value)
		if err == nil && (overwrite || actor.BirthDate.IsZero() || (actor.BirthDate.Month() == 1 && actor.BirthDate.Day() == 1)) && t.Before(time.Now().AddDate(-15, 0, 0)) && externalreference.CheckAndSetDateActorField(&actor.BirthDate, field, value, actor.ID) {
			actor.BirthDate = t
			changed = true
		}
	case "height":
		num, _ := strconv.Atoi(value)
		if (overwrite || actor.Height == 0) && num > 0 && externalreference.CheckAndSetIntActorField(&actor.Height, field, num, actor.ID) {
			actor.Height = num
			changed = true
		}
	case "weight":
		num, _ := strconv.Atoi(value)
		if (overwrite || actor.Weight == 0) && num > 0 && externalreference.CheckAndSetIntActorField(&actor.Weight, field, num, actor.ID) {
			actor.Weight = num
			changed = true
		}
	case "nationality":
		if (overwrite || actor.Nationality == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.Nationality, field, value, actor.ID) {
			actor.Nationality = value
			changed = true
		}
	case "ethnicity":
		if (overwrite || actor.Ethnicity == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.Ethnicity, field, value, actor.ID) {
			switch strings.ToLower(value) {
			case "white":
				value = "Caucasian"
			}
			actor.Ethnicity = value
			changed = true
		}
	case "gender":
		if (overwrite || actor.Gender == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.Gender, field, value, actor.ID) {
			actor.Gender = value
			changed = true
		}
	case "band_size":
		num, _ := strconv.Atoi(value)
		if (overwrite || actor.BandSize == 0) && num > 0 && externalreference.CheckAndSetIntActorField(&actor.BandSize, field, num, actor.ID) {
			actor.BandSize = num
			changed = true
		}
	case "waist_size":
		num, _ := strconv.Atoi(value)
		if (overwrite || actor.WaistSize == 0) && num > 0 && externalreference.CheckAndSetIntActorField(&actor.WaistSize, field, num, actor.ID) {
			actor.WaistSize = num
			changed = true
		}
	case "hip_size":
		num, _ := strconv.Atoi(value)
		if (overwrite || actor.HipSize == 0) && num > 0 && externalreference.CheckAndSetIntActorField(&actor.HipSize, field, num, actor.ID) {
			actor.HipSize = num
			changed = true
		}
	case "cup_size":
		if (overwrite || actor.CupSize == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.CupSize, field, value, actor.ID) {
			actor.CupSize = value
			changed = true
		}
	case "eye_color":
		if (overwrite || actor.EyeColor == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.EyeColor, field, value, actor.ID) {
			actor.EyeColor = value
			changed = true
		}
	case "hair_color":
		if (overwrite || actor.HairColor == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.HairColor, field, value, actor.ID) {
			actor.HairColor = value
			changed = true
		}
	case "biography":
		if (overwrite || actor.Biography == "") && value > "" && externalreference.CheckAndSetStringActorField(&actor.Biography, field, value, actor.ID) {
			actor.Biography = value
			changed = true
		}
	case "image_url":
		if actor.AddToImageArray(value) {
			changed = true
		}
		if value != "" && (actor.ImageUrl == "" || (overwrite && !actor.CheckForSetImage())) {
			//if (overwrite || actor.ImageUrl == "" ) && value != ""  && !actor.CheckForSetImage() {
			actor.ImageUrl = value
			changed = true
		}
	case "images":
		if !actor.CheckForUserDeletes("images", value) && actor.AddToImageArray(value) {
			changed = true
		}
	case "aliases":
		value = strings.Replace(value, ";", ",", -1)
		array := strings.Split(value, ",")
		for _, item := range array {
			if !actor.CheckForUserDeletes("aliases", strings.TrimSpace(item)) && actor.AddToAliases(strings.TrimSpace(item)) {
				changed = true
			}
		}
	case "urls":
		value = strings.Replace(value, ";", ",", -1)
		array := strings.Split(value, ",")
		for _, item := range array {
			if actor.AddToActorUrlArray(models.ActorLink{Url: strings.TrimSpace(item)}) {
				changed = true
			}
		}
	case "piercings":
		value = strings.Replace(value, ";", ",", -1)
		array := strings.Split(value, ",")
		for _, item := range array {
			if !actor.CheckForUserDeletes("piercings", strings.TrimSpace(item)) && actor.AddToPiercings(strings.TrimSpace(item)) {
				changed = true
			}
		}
	case "tattoos":
		value = strings.Replace(value, ";", ",", -1)
		array := strings.Split(value, ",")
		for _, item := range array {
			if !actor.CheckForUserDeletes("tattoos", strings.TrimSpace(item)) && actor.AddToTattoos(strings.TrimSpace(item)) {
				changed = true
			}
		}
	case "start_year":
		num, _ := strconv.Atoi(value)
		// alow the start year to move back, as sites may only list when the actor started with them
		if (overwrite || actor.StartYear == 0 || actor.StartYear > num) && num > 0 {
			actor.StartYear = num
			changed = true
		}
	case "end_year":
		num, _ := strconv.Atoi(value)
		if (overwrite || actor.StartYear == 0) && num > 0 {
			actor.StartYear = num
			changed = true
		}
	}
	return changed
}
func getRegexResult(value string, pattern string, pos int) string {
	re := regexp.MustCompile(pattern)
	if pos == 0 {
		return re.FindString(value)
	} else {
		groups := re.FindStringSubmatch(value)
		if len(groups) < pos+1 {
			return ""
		}
		return re.FindStringSubmatch(value)[pos]
	}

}

func postProcessing(rule models.GenericActorScraperRule, value string, htmlElement *colly.HTMLElement) string {
	for _, postprocessing := range rule.PostProcessing {
		switch postprocessing.Function {
		case "Lookup Country":
			value = getCountryCode(value)
		case "Parse Date":
			t, err := time.Parse(postprocessing.Params[0], strings.Replace(strings.Replace(strings.Replace(strings.Replace(value, "1st", "1", -1), "nd", "", -1), "rd", "", -1), "th", "", -1))
			if err != nil {
				return ""
			}
			value = t.Format("2006-01-02")
		case "inch to cm":
			num, _ := strconv.ParseFloat(value, 64)
			num = num * 2.54
			value = strconv.Itoa(int(math.Round(num)))
		case "Feet+Inches to cm":
			// param 1 regex, param 2 feet pos, param 3 inches pos
			re := regexp.MustCompile(postprocessing.Params[0])
			matches := re.FindStringSubmatch(value)
			if len(matches) >= 3 {
				feetpos, _ := strconv.Atoi(postprocessing.Params[1])
				inchpos, _ := strconv.Atoi(postprocessing.Params[2])
				feet, _ := strconv.Atoi(matches[feetpos])
				inches, _ := strconv.Atoi(matches[inchpos])
				num := float64(feet*12+inches) * 2.54
				value = strconv.Itoa(int(math.Round(num)))
			}
		case "lbs to kg":
			num, _ := strconv.ParseFloat(value, 64)
			const conversionFactor float64 = 0.453592
			value = strconv.Itoa(int(math.Round(float64(num) * conversionFactor)))
		case "jsonString":
			value = strings.TrimSpace(html.UnescapeString(gjson.Get(value, postprocessing.Params[0]).String()))
		case "RegexString":
			pos, _ := strconv.Atoi(postprocessing.Params[1])
			value = getRegexResult(value, postprocessing.Params[0], pos)
		case "RegexReplaceAll":
			// tip to add a Prefix or Suffix, use `Prefix$0Suffix`
			regex := regexp.MustCompile(postprocessing.Params[0])
			value = regex.ReplaceAllString(value, postprocessing.Params[1])
		case "Replace":
			value = strings.Replace(value, postprocessing.Params[0], postprocessing.Params[1], 1)
		case "AbsoluteUrl":
			value = htmlElement.Request.AbsoluteURL(value)
		case "RemoveQueryParams":
			if urlValue, err := url.Parse(value); err == nil {
				urlValue.RawQuery = ""
				value = urlValue.String()
			}
		case "CollyForEach":
			value = getSubRuleResult(postprocessing.SubRule, htmlElement)
		case "DOMNext":
			value = strings.TrimSpace(htmlElement.DOM.Next().Text())
		case "DOMNextText":
			node := htmlElement.DOM.Get(0)
			textNodeType := nethtml.TextNode
			nextSibling := node.NextSibling

			if nextSibling != nil && nextSibling.Type == textNodeType {
				value = strings.TrimSpace(nextSibling.Data)
			}
		case "SetWhenValueContains":
			searchValue := postprocessing.Params[0]
			newValue := postprocessing.Params[1]

			if strings.Contains(value, searchValue) {
				value = newValue
			}
		case "SetWhenValueNotContains":
			searchValue := postprocessing.Params[0]
			newValue := postprocessing.Params[1]

			if !strings.Contains(value, searchValue) {
				value = newValue
			}
		case "UnescapeString":
			value = html.UnescapeString(value)
		}
	}
	return value
}

func substr(s string, start, end int) string {
	counter, startIdx := 0, 0
	for i := range s {
		if counter == start {
			startIdx = i
		}
		if counter == end {
			return s[startIdx:i]
		}
		counter++
	}
	return s[startIdx:]
}

func getCountryCode(countryName string) string {
	switch strings.ToLower(countryName) {
	case "united states", "american":
		return "US"
	case "english", "scottish":
		return "GB"
	default:
		code, err := lookupCountryCode(countryName)
		if err != nil {
			return countryName
		} else {
			return code
		}
	}
}

func lookupCountryCode(countryName string) (string, error) {
	// Construct the API URL with the country name as a query parameter
	url := fmt.Sprintf("https://restcountries.com/v2/name/%s", countryName)

	// Send a GET request to the API and decode the JSON response
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var countries []struct {
		Alpha2Code string `json:"alpha2Code"`
	}
	err = json.NewDecoder(resp.Body).Decode(&countries)
	if err != nil {
		return "", err
	}

	// Check if a country code was found
	if len(countries) == 0 {
		return "", fmt.Errorf("no country code found for %s", countryName)
	}

	return countries[0].Alpha2Code, nil
}
