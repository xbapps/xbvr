package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/models"
)

type ResponseGetActors struct {
	Results int            `json:"results"`
	Scenes  []models.Actor `json:"actors"`
}

type ActorResource struct{}

func (i ActorResource) WebService() *restful.WebService {
	tags := []string{"Actor"}

	ws := new(restful.WebService)

	ws.Path("/api/actor").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/filters").To(i.getFilters).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetFilters{}))

	ws.Route(ws.POST("/list").To(i.getActors).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetActors{}))

	ws.Route(ws.POST("/rate/{actor-id}").To(i.rateActor).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))

	ws.Route(ws.POST("/toggle").To(i.toggleList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetActors{}))

	ws.Route(ws.GET("/{actor-id}").To(i.getActor).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))

	ws.Route(ws.POST("/edit/{id}").To(i.editActor).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))

	ws.Route(ws.DELETE("/delete/{id}").To(i.deleteActor).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))

	ws.Route(ws.POST("/setimage").To(i.setActorImage).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))
	ws.Route(ws.DELETE("/delimage").To(i.deleteActorImage).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))

	ws.Route(ws.GET("/countrylist").To(i.getCountryList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.CountryDetails{}))

	ws.Route(ws.GET("/akas/{actor-id}").To(i.getActorAkas).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Actor{}))

	ws.Route(ws.GET("/colleagues/{actor-id}").To(i.getActorColleagues).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]models.Actor{}))

	ws.Route(ws.GET("/extrefs/{actor-id}").To(i.getActorExtRefs).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.ExternalReferenceLink{}))

	ws.Route(ws.POST("/edit_extrefs/{id}").To(i.editActorExtRefs).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.ExternalReferenceLink{}))
	return ws
}

func (i ActorResource) getFilters(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var actors []models.Actor
	db.Model(&actors).Order("name").Find(&actors)

	var outCast []string
	for _, actor := range actors {
		outCast = append(outCast, actor.Name)
	}

	var sites []models.Site
	db.Model(&sites).Order("name").Find(&sites)
	var outSites []string
	for _, site := range sites {
		outSites = append(outSites, site.Name)
	}
	// supported attributes
	var outAttributes []string
	outAttributes = append(outAttributes, "Is Favourite")
	outAttributes = append(outAttributes, "In Watchlist")
	outAttributes = append(outAttributes, "Has Rating")
	outAttributes = append(outAttributes, "Rating 0")
	outAttributes = append(outAttributes, "Rating .5")
	outAttributes = append(outAttributes, "Rating 1")
	outAttributes = append(outAttributes, "Rating 1.5")
	outAttributes = append(outAttributes, "Rating 2")
	outAttributes = append(outAttributes, "Rating 2.5")
	outAttributes = append(outAttributes, "Rating 3")
	outAttributes = append(outAttributes, "Rating 3.5")
	outAttributes = append(outAttributes, "Rating 4")
	outAttributes = append(outAttributes, "Rating 4.5")
	outAttributes = append(outAttributes, "Rating 5")
	outAttributes = append(outAttributes, "Has Stashdb Link")
	outAttributes = append(outAttributes, "SLR Scraper")
	outAttributes = append(outAttributes, "VRPorn Scraper")
	outAttributes = append(outAttributes, "Has Tattoo")
	outAttributes = append(outAttributes, "Has Piercing")
	outAttributes = append(outAttributes, "Aka Group")
	outAttributes = append(outAttributes, "Possible Aka")
	outAttributes = append(outAttributes, "In An Aka Group")
	outAttributes = append(outAttributes, "Multiple Stashdb Links")
	outAttributes = append(outAttributes, "Has Image")

	type Results struct {
		Result string
	}
	var results []Results
	db.Table("actors").
		Where("cup_size <> ''").
		Select("distinct cup_size as result").
		Order("cup_size").
		Find(&results)
	for _, r := range results {
		outAttributes = append(outAttributes, "Cup Size "+r.Result)
	}

	db.Table("actors").
		Where("IFNULL(hair_color,'') <> ''").
		Select("distinct hair_color as result").
		Order("hair_color").
		Find(&results)
	for _, r := range results {
		outAttributes = append(outAttributes, "Hair Color "+r.Result)
	}
	db.Table("actors").
		Where("IFNULL(eye_color,'') <> ''").
		Select("distinct eye_color as result").
		Order("eye_color").
		Find(&results)
	for _, r := range results {
		outAttributes = append(outAttributes, "Eye Color "+r.Result)
	}
	db.Table("actors").
		Where("IFNULL(nationality,'') <> ''").
		Select("distinct nationality as result").
		Order("nationality").
		Find(&results)
	countries := models.GetCountryList()
	for _, r := range results {
		countryName := r.Result
		for _, c := range countries {
			if c.Code == r.Result {
				countryName = c.Name
			}
		}
		outAttributes = append(outAttributes, "Nationality "+countryName)
	}

	db.Table("actors").
		Where("IFNULL(ethnicity,'') <> ''").
		Select("distinct ethnicity as result").
		Order("ethnicity").
		Find(&results)
	for _, r := range results {
		outAttributes = append(outAttributes, "Ethnicity "+r.Result)
	}

	db.Table("actors").
		Where("IFNULL(breast_type,'') <> ''").
		Select("distinct breast_type as result").
		Order("breast_type").
		Find(&results)
	for _, r := range results {
		outAttributes = append(outAttributes, "Breast Type "+r.Result)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetActorFilters{
		Attributes: outAttributes,
		Cast:       outCast,
		Sites:      outSites,
	})
}

func (i ActorResource) getActor(req *restful.Request, resp *restful.Response) {
	var actor models.Actor
	db, _ := models.GetDB()

	if strings.Contains(req.PathParameter("actor-id"), "-") {
		actor.GetIfExist(req.PathParameter("actor-id"))
	} else {
		id, err := strconv.Atoi(req.PathParameter("actor-id"))
		if err != nil {
			log.Error(err)
			return
		}
		_ = actor.GetIfExistByPKWithSceneAvg(uint(id))
	}
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

func (i ActorResource) getActors(req *restful.Request, resp *restful.Response) {
	var r models.RequestActorList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	out := models.QueryActors(r, true)
	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ActorResource) rateActor(req *restful.Request, resp *restful.Response) {
	actorId, err := strconv.Atoi(req.PathParameter("actor-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var r RequestSetActorRating
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var actor models.Actor
	db, _ := models.GetDB()
	err = db.Where(models.Actor{ID: uint(actorId)}).First(&actor).Error
	if err == nil {
		actor.StarRating = r.Rating
		actor.Save()
	}
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

type RequestToggleActorList struct {
	ActorID uint   `json:"actor_id"`
	List    string `json:"list"`
}
type RequestSetActorImage struct {
	ActorID uint   `json:"actor_id"`
	Url     string `json:"url"`
}

type RequestSetActorRating struct {
	Rating float64 `json:"rating"`
}

type RequestEditActorDetails struct {
	Name         string    `json:"name"`
	ImageArr     string    `json:"image_arr"`
	BirthDate    time.Time `json:"birth_date"`
	Nationality  string    `json:"nationality"`
	Ethnicity    string    `json:"ethnicity"`
	EyeColor     string    `json:"eye_color"`
	HairColor    string    `json:"hair_color"`
	Height       int       `json:"height"`
	Weight       int       `json:"weight"`
	CupSize      string    `json:"cup_size"`
	Measurements string    `json:"measurements"`
	BandSize     int       `json:"band_size"`
	WaistSize    int       `json:"waist_size"`
	HipSize      int       `json:"hip_size"`
	BreastType   string    `json:"breast_type"`
	StartYear    int       `json:"start_year"`
	EndYear      int       `json:"end_year"`
	Tattoos      string    `json:"tattoos"`
	Piercings    string    `json:"piercings"`

	Biography string `json:"biography"`
	Aliases   string `json:"aliases"`
	Gender    string `json:"gender"`
	URLs      string `json:"urls"`
}

type RequestEditActorExtRefs struct {
	URLs []string
}

type ResponseGetActorFilters struct {
	Cast       []string `json:"cast"`
	Sites      []string `json:"sites"`
	Attributes []string `json:"attributes"`
}

func (i ActorResource) toggleList(req *restful.Request, resp *restful.Response) {
	var r RequestToggleActorList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.ActorID == 0 && r.List == "" {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var actor models.Actor
	err = actor.GetIfExistByPK(r.ActorID)
	if err != nil {
		log.Error(err)
		return
	}

	switch r.List {
	case "watchlist":
		actor.Watchlist = !actor.Watchlist
	case "favourite":
		actor.Favourite = !actor.Favourite
		// case "needs_update":
		// 	actor.NeedsUpdate = !actor.NeedsUpdate
		// case "is_hidden":
		// 	actor.IsHidden = !actor.IsHidden
	}
	actor.Save()
}

func (i ActorResource) editActor(req *restful.Request, resp *restful.Response) {
	name, err := strconv.Atoi(req.PathParameter("id"))
	if err != nil {
		log.Error(err)
		return
	}

	var r RequestEditActorDetails
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var actor models.Actor
	db, _ := models.GetDB()
	defer db.Close()
	err = actor.GetIfExistByPK(uint(name))
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusOK, nil)
	}

	if len(r.Nationality) > 2 {
		countryList := models.GetCountryList()
		for _, country := range countryList {
			if country.Name == r.Nationality {
				r.Nationality = country.Code
				break
			}
		}
	}
	checkDateFieldChanged("birth_date", &r.BirthDate, &actor.BirthDate, actor.ID)
	checkStringFieldChanged("nationality", &r.Nationality, &actor.Nationality, actor.ID)
	checkStringFieldChanged("ethnicity", &r.Ethnicity, &actor.Ethnicity, actor.ID)
	checkStringArrayChanged("image_arr", &r.ImageArr, &actor.ImageArr, actor.ID)
	checkStringFieldChanged("eye_color", &r.EyeColor, &actor.EyeColor, actor.ID)
	checkStringFieldChanged("hair_color", &r.HairColor, &actor.HairColor, actor.ID)
	checkIntFieldChanged("height", &r.Height, &actor.Height, actor.ID)
	checkIntFieldChanged("weight", &r.Weight, &actor.Weight, actor.ID)

	re := regexp.MustCompile(`(?m)(^(\d{2})?([A-Za-z]{0,2})-(\d{2})?-(\d{2}$)?)|^[A-Z]{0,2}$`)
	match := re.FindStringSubmatch(r.Measurements)
	switch len(match) {
	case 1:
		r.CupSize = match[0]
	case 6:
		r.BandSize, _ = strconv.Atoi(match[2])
		r.CupSize = match[3]
		r.WaistSize, _ = strconv.Atoi(match[4])
		r.HipSize, _ = strconv.Atoi(match[5])
	}
	r.BandSize = r.BandSize * 254 / 100
	r.WaistSize = r.WaistSize * 254 / 100
	r.HipSize = r.HipSize * 254 / 100

	checkIntFieldChanged("band_size", &r.BandSize, &actor.BandSize, actor.ID)
	checkStringFieldChanged("cup_size", &r.CupSize, &actor.CupSize, actor.ID)
	checkIntFieldChanged("waist_size", &r.WaistSize, &actor.WaistSize, actor.ID)
	checkIntFieldChanged("hip_size", &r.HipSize, &actor.HipSize, actor.ID)
	checkStringFieldChanged("breast_type", &r.BreastType, &actor.BreastType, actor.ID)
	checkIntFieldChanged("start_year", &r.StartYear, &actor.StartYear, actor.ID)
	checkIntFieldChanged("end_year", &r.EndYear, &actor.EndYear, actor.ID)
	checkStringArrayChanged("tattoos", &r.Tattoos, &actor.Tattoos, actor.ID)
	checkStringArrayChanged("piercings", &r.Piercings, &actor.Piercings, actor.ID)
	checkStringFieldChanged("biography", &r.Biography, &actor.Biography, actor.ID)
	checkStringArrayChanged("aliases", &r.Aliases, &actor.Aliases, actor.ID)
	checkStringArrayChanged("urls", &r.URLs, &actor.URLs, actor.ID)

	actor.Save()

	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

func (i ActorResource) deleteActor(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("id"))
	if err != nil {
		log.Error(err)
		return
	}

	var actor models.Actor
	db, _ := models.GetDB()
	defer db.Close()

	db.Exec(`delete from actor_akas where actor_id=?`, id)
	db.Where("actor_id = ?", uint(id)).Delete(&models.ActionActor{})
	db.Where("internal_table = 'actors' and internal_db_id = ?", uint(id)).Delete(&models.ExternalReferenceLink{})
	db.Where("id = ?", uint(id)).Delete(&models.Actor{})

	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

func checkStringFieldChanged(field_name string, newValue *string, actorField *string, actorId uint) {
	if *actorField != *newValue {
		*actorField = *newValue
		models.AddActionActor(actorId, "edit_actor", "edit", field_name, *newValue)
	}
}
func checkIntFieldChanged(field_name string, newValue *int, actorField *int, actorId uint) {
	if *actorField != *newValue {
		*actorField = *newValue
		models.AddActionActor(actorId, "edit_actor", "edit", field_name, strconv.Itoa(*newValue))
	}
}
func checkDateFieldChanged(field_name string, newValue *time.Time, actorField *time.Time, actorId uint) {
	if *actorField != *newValue {
		*actorField = *newValue
		dt := *newValue
		models.AddActionActor(actorId, "edit_actor", "edit", field_name, dt.Format("2006-01-02"))
	}
}
func checkStringArrayChanged(field_name string, newValue *string, actorField *string, actorId uint) {
	if *actorField != *newValue {
		var actorArray []string
		var newArray []string
		json.Unmarshal([]byte(*newValue), &newArray)
		json.Unmarshal([]byte(*actorField), &actorArray)
		for _, actorField := range actorArray {
			exists := false
			for _, newField := range newArray {
				if newField == actorField {
					exists = true
				}
			}
			if !exists {
				models.AddActionActor(actorId, "edit_actor", "delete", field_name, actorField)
			}
		}
		for _, newField := range newArray {
			exists := false
			for _, actorField := range actorArray {
				if newField == actorField {
					exists = true
				}
			}
			if !exists {
				models.AddActionActor(actorId, "edit_actor", "add", field_name, newField)
			}
		}

		*actorField = *newValue
	}
}

func (i ActorResource) setActorImage(req *restful.Request, resp *restful.Response) {
	var r RequestSetActorImage
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.ActorID == 0 || r.Url == "" {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var actor models.Actor
	err = actor.GetIfExistByPKWithSceneAvg(r.ActorID)
	if err != nil {
		log.Error(err)
		return
	}
	actor.ImageUrl = r.Url
	actor.AddToImageArray(r.Url)
	actor.Save()

	aa := models.ActionActor{ActorID: actor.ID, ActionType: "setimage", Source: "edit_actor", ChangedColumn: "image_url", NewValue: actor.ImageUrl}
	aa.Save()
	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

func (i ActorResource) deleteActorImage(req *restful.Request, resp *restful.Response) {
	var r RequestSetActorImage
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.ActorID == 0 || r.Url == "" {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var actor models.Actor
	err = actor.GetIfExistByPKWithSceneAvg(r.ActorID)
	if err != nil {
		log.Error(err)
		return
	}
	var currentImages []string
	var newImages []string
	if actor.ImageArr == "" {
		return
	}
	json.Unmarshal([]byte(actor.ImageArr), &currentImages)
	for _, img := range currentImages {
		if img != r.Url {
			newImages = append(newImages, img)
		}
	}

	// check if we are deleting our main image
	if actor.ImageUrl == r.Url {
		if len(newImages) == 0 {
			actor.ImageUrl = ""
		} else {
			actor.ImageUrl = newImages[0]
		}
	}

	jsonarray, _ := json.Marshal(newImages)
	actor.ImageArr = string(jsonarray)
	if actor.ImageArr == "null" {
		actor.ImageArr = "[]"
	}
	actor.Save()

	aa := models.ActionActor{ActorID: actor.ID, ActionType: "delete", Source: "edit_actor", ChangedColumn: "image_arr", NewValue: r.Url}
	aa.Save()
	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

func (i ActorResource) getCountryList(req *restful.Request, resp *restful.Response) {
	resp.WriteHeaderAndEntity(http.StatusOK, models.GetCountryList())
}

type AkaResponse struct {
	AkaGroups    []models.Actor `json:"aka_groups"`
	Actors       []models.Actor `json:"actors"`
	PossibleAkas []models.Actor `json:"possible_akas"`
}

func (i ActorResource) getActorAkas(req *restful.Request, resp *restful.Response) {
	var actor models.Actor
	db, _ := models.GetDB()
	defer db.Close()

	var akaresp AkaResponse
	actor_id, _ := strconv.ParseUint(req.PathParameter("actor-id"), 10, 32)
	db.Preload("AkaGroups").Where("id = ?", actor_id).Find(&actor)
	if strings.HasPrefix(actor.Name, "aka:") {
		var akagrp models.Aka
		db.Preload("Akas").Preload("AkaActor").Where("aka_actor_id = ? ", actor.ID).Find(&akagrp)
		// reread actor to get full Preloads
		akagrp.AkaActor.GetIfExistByPKWithSceneAvg(akagrp.AkaActor.ID)
		for _, actor := range akagrp.Akas {
			if actor.ID != uint(actor_id) {
				// reread actor to get full Preloads
				actor.GetIfExistByPKWithSceneAvg(actor.ID)
				akaresp.Actors = append(akaresp.Actors, actor)
			}
		}

	} else {
		for _, grp := range actor.AkaGroups {
			var akagrp models.Aka
			db.Preload("Akas").Preload("AkaActor").Where("id = ?", grp.ID).Find(&akagrp)
			// reread actor to get full Preloads
			akagrp.AkaActor.GetIfExistByPKWithSceneAvg(akagrp.AkaActor.ID)
			akaresp.AkaGroups = append(akaresp.AkaGroups, akagrp.AkaActor)
			for _, actor := range akagrp.Akas {
				if actor.ID != uint(actor_id) {
					// reread actor to get full Preloads
					actor.GetIfExistByPKWithSceneAvg(actor.ID)
					akaresp.Actors = append(akaresp.Actors, actor)
				}
			}
		}
	}

	var possibleAkas []models.Actor
	db.Model(&models.Actor{}).
		Table("external_reference_links erl").
		Joins("JOIN external_reference_links erl2 on erl.external_id = erl2 .external_id and erl2.internal_table ='actors'").
		Joins("JOIN actors on actors.id = erl2.internal_db_id").
		Where("erl.internal_table = 'actors' and erl.internal_db_id = ? and erl2.internal_db_id <> erl.internal_db_id ", actor_id).
		Select("distinct actors.*").
		Find(&possibleAkas)

		// check each possible aka isn't already listed in the aka actors list or another aka group
	for _, possible := range possibleAkas {
		found := false
		if strings.HasPrefix(possible.Name, "aka:") {
			found = true
		}
		for _, existing := range akaresp.Actors {
			if existing.ID == possible.ID {
				found = true
			}
		}
		for _, aka := range akaresp.AkaGroups {
			if aka.ID == possible.ID {
				found = true
			}
		}
		if !found {
			possible.GetIfExistByPKWithSceneAvg(possible.ID)
			akaresp.PossibleAkas = append(akaresp.PossibleAkas, possible)
		}
	}
	resp.WriteHeaderAndEntity(http.StatusOK, akaresp)
}
func (i ActorResource) getActorColleagues(req *restful.Request, resp *restful.Response) {
	var colleagues []models.Actor
	db, _ := models.GetDB()
	defer db.Close()

	actor_id, _ := strconv.ParseUint(req.PathParameter("actor-id"), 10, 32)

	db.Model(&models.Actor{}).
		Table("actors a").
		Joins("JOIN scene_cast sc on sc.actor_id = a.id").                                 // find the scenes the actor is in
		Joins("JOIN scene_cast sc2 on sc2.scene_id =sc.scene_id and sc2.actor_id <>a.id"). // find the OTHER actors
		Joins("join actors a2 on a2.id=sc2.actor_id").                                     // get their actor record
		Where("a.id = ? ", actor_id).
		Group("a2.id").
		Order("count(*) desc"). //sequence by most worked with desc
		Select("distinct a2.*").
		Find(&colleagues)

	for idx, actor := range colleagues {
		actor.GetIfExistByPKWithSceneAvg(actor.ID)
		colleagues[idx] = actor
	}
	resp.WriteHeaderAndEntity(http.StatusOK, colleagues)
}
func (i ActorResource) getActorExtRefs(req *restful.Request, resp *restful.Response) {
	u64, _ := strconv.ParseUint(req.PathParameter("actor-id"), 10, 32)
	actor_id := uint(u64)
	resp.WriteHeaderAndEntity(http.StatusOK, readExtRefs(actor_id))
}

func readExtRefs(actor_id uint) []models.ExternalReferenceLink {
	var extrefs []models.ExternalReferenceLink
	db, _ := models.GetDB()
	defer db.Close()

	db.Preload("ExternalReference").Where("internal_table = 'actors' and internal_db_id = ?", actor_id).Find(&extrefs)
	return extrefs
}

func (i ActorResource) editActorExtRefs(req *restful.Request, resp *restful.Response) {
	u64, err := strconv.ParseUint(req.PathParameter("id"), 10, 32)
	id := uint(u64)
	if err != nil {
		log.Error(err)
		return
	}

	var actor models.Actor
	actor.GetIfExistByPK(id)

	var urls []string
	err = req.ReadEntity(&urls)
	if err != nil {
		log.Error(err)
		return
	}

	var links []models.ExternalReferenceLink

	commonDb, _ := models.GetCommonDB()

	// find any links that were removed
	commonDb.Preload("ExternalReference").Where("internal_table = 'actors' and internal_db_id = ?", id).Find(&links)
	for _, link := range links {
		found := false
		for _, url := range urls {
			if url == link.ExternalReference.ExternalURL {
				found = true
				continue
			}
		}
		if !found {
			commonDb.Delete(&link)
			models.AddActionActor(actor.ID, "edit_actor", "delete", "external_reference_link", link.ExternalReference.ExternalURL)
		}
	}

	// add new links
	for _, url := range urls {
		var extref models.ExternalReference
		extref.FindExternalUrl(extref.DetermineActorScraperByUrl(url), url)
		if extref.ID == 0 {
			// create new extref + link
			extref.ExternalSource = extref.DetermineActorScraperByUrl(url)
			if extref.ExternalSource == "stashdb performer" {
				extref.ExternalId = strings.ReplaceAll(url, "https://stashdb.org/performers/", "")
			} else {
				extref.ExternalId = url
			}
			extref.ExternalURL = url

			extref.XbvrLinks = append(extref.XbvrLinks, models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: id, InternalNameId: actor.Name,
				ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId, MatchType: 0})
			extref.Save()
			models.AddActionActor(actor.ID, "edit_actor", "add", "external_reference_link", url)
		} else {
			// external reference exists, but check it is linked to this actor
			found := false
			for _, link := range extref.XbvrLinks {
				if link.InternalDbId == id {
					found = true
					continue
				}
			}
			if !found {
				//add a link to the actor
				newLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: id, InternalNameId: actor.Name,
					ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId, MatchType: 0}
				newLink.Save()
				models.AddActionActor(actor.ID, "edit_actor", "add", "external_reference_link", url)
			}
		}
	}
	resp.WriteHeaderAndEntity(http.StatusOK, readExtRefs(id))
}
