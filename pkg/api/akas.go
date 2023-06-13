package api

import (
	"net/http"
	"strconv"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
)

type RequestDeleteAka struct {
	AkaID uint   `json:"aka_id"`
	Name  string `json:"name"`
}

type RequestEditAkaMembers struct {
	Actors []string `json:"actorList"`
}

type ResponseGetAkas struct {
	Results int          `json:"results"`
	Scenes  []models.Aka `json:"akas"`
}

type ResponseAka struct {
	Status string     `json:"status"`
	Aka    models.Aka `json:"akas"`
}
type ResponseAkas struct {
	Error error        `json:"error"`
	Akas  []models.Aka `json:"akas"`
}

type AkaResource struct{}

func (i AkaResource) WebService() *restful.WebService {
	tags := []string{"Aka"}

	ws := new(restful.WebService)

	ws.Path("/api/aka").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/list").To(i.getAkas).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.POST("/create").To(i.createAkaGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/delete").To(i.deleteAka).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/{aka-id}").To(i.getAka).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/add").To(i.addToAkaGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/remove").To(i.removeFromAkaGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))
	return ws
}

func (i AkaResource) createAkaGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditAkaMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	//Construct aka record
	var aka models.Aka
	var akaActorList []models.Actor
	var tmpActor models.Actor
	var akaActor models.Actor
	var names []string
	var errors []string
	ret := http.StatusOK

	actorCnt := 0
	for _, a := range r.Actors {
		tmpActor.ID = 0
		db.Where("name = ?", a).First(&tmpActor)
		if tmpActor.ID != 0 {
			// check if the actor is already in a group
			cnt := 0
			db.Model(&aka).
				Joins("join actor_akas on actor_akas.aka_id =akas.id").
				Where("actor_akas.actor_id  = ?", tmpActor.ID).Count(&cnt)
			if cnt > 0 {
				errors = append(errors, a+" already in aka group")
			}
			akaActorList = append(akaActorList, tmpActor)
			names = append(names, a)
			actorCnt++
		} else {
			errors = append(errors, a+" not found")
			ret = http.StatusNotFound
		}
	}

	if ret != http.StatusNotFound {
		if actorCnt > 0 {
			// create an actor to represent the aka group
			akaActor.ID = 0
			akaActor.Name = "aka:" + strings.Join(names, ",")
			akaActor.Save()

			aka.ID = 0
			aka.AkaActor = akaActor
			aka.Akas = akaActorList
			aka.Name = aka.AkaNameSortedAlphabetcally()
			aka.Save()
		}
	} else {
		createResp := &ResponseAka{
			Status: strings.Join(errors, ","),
			Aka:    aka,
		}
		RefreshAka(&aka)
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	RefreshAka(&aka)
	createResp := &ResponseAka{
		Status: strings.Join(errors, ","),
		Aka:    aka,
	}

	externalreference.LinkOnXbvrAkaGroups()
	resp.WriteHeaderAndEntity(http.StatusOK, createResp)
}

func (i AkaResource) deleteAka(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestDeleteAka
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var aka models.Aka
	if r.AkaID != 0 {
		err = db.First(&aka, r.AkaID).Error
	} else {
		// find on the aka name
		err = db.Where(&models.Aka{Name: r.Name}).First(&aka).Error
		if err != nil {
			// find on the aka actor name
			var actor models.Actor
			db.Where("name = ?", r.Name).First(&actor)
			err = db.Where("aka_actor_id = ?", actor.ID).First(&aka).Error
		}
	}

	if err != nil {
		log.Error(err)
		resp.WriteHeaderAndEntity(http.StatusNotFound, nil)
		return
	}

	db.Model(&aka).Association("Akas").Clear() // delete aka links with actors
	db.Delete(&aka)
	defer resp.WriteHeaderAndEntity(http.StatusOK, aka)

	var akaActor models.Actor
	akaActor.ID = aka.AkaActorId
	db.Model(&akaActor).Association("Scenes").Clear() // delete scene links with aka
	db.Delete(&akaActor)                              // delete aka actor
	aka.UpdateAkaSceneCastRecords()
}

func (i AkaResource) getAkas(req *restful.Request, resp *restful.Response) {

	var akas []models.Aka
	db, _ := models.GetDB()
	defer db.Close()

	db.Preload("Actors").Find(&akas)
	resp.WriteHeaderAndEntity(http.StatusOK, akas)
}

func (i AkaResource) getAka(req *restful.Request, resp *restful.Response) {
	sceneId, err := strconv.Atoi(req.PathParameter("aka-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var aka models.Aka
	db, _ := models.GetDB()
	err = aka.GetIfExistByPK(uint(sceneId))
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, aka)
}

func (i AkaResource) removeFromAkaGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditAkaMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var aka models.Aka
	var errors []string
	ret := http.StatusOK

	akaActorName := ""
	for _, a := range r.Actors {
		if strings.HasPrefix(a, "aka:") {
			akaActorName = a
		}
	}
	var actor models.Actor
	db.Where("name = ?", akaActorName).First(&actor)
	db.Where("aka_actor_id = ?", actor.ID).Preload("AkaActor").Preload("Akas").First(&aka)

	// check we aren't going to remove everyone
	remainActorCount := len(aka.Akas)
	for _, actor := range r.Actors {
		if !strings.HasPrefix(actor, "aka:") {
			for _, rec := range aka.Akas {
				if rec.Name == actor {
					remainActorCount--
				}
			}
		}
	}
	if remainActorCount < 2 {
		RefreshAka(&aka)
		createResp := &ResponseAka{
			Status: "A Group needs at least 2 actors. Delete the group instead",
			Aka:    aka,
		}
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	for _, actor := range r.Actors {
		if !strings.HasPrefix(actor, "aka:") {
			found := false
			for idx, rec := range aka.Akas {
				if rec.Name == actor {
					db.Model(&aka).Association("Akas").Delete(&aka.Akas[idx])
					found = true
				}
			}
			if !found {
				errors = append(errors, actor+" not found in group")
			}
		}
	}

	aka.Name = aka.AkaNameSortedAlphabetcally()
	aka.Save()

	RefreshAka(&aka)
	if len(errors) > 0 {
		createResp := &ResponseAka{
			Status: strings.Join(errors, ","),
			Aka:    aka,
		}
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	createResp := &ResponseAka{
		Aka: aka,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, createResp)

}

func (i AkaResource) addToAkaGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditAkaMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	//Construct aka record
	var aka models.Aka
	var tmpActor models.Actor
	var errors []string
	ret := http.StatusOK

	akaActorName := ""
	actorCnt := 0
	for _, a := range r.Actors {
		if strings.HasPrefix(a, "aka:") {
			akaActorName = a
		}
	}
	var actor models.Actor
	db.Where("name = ?", akaActorName).First(&actor)
	db.Where("aka_actor_id = ?", actor.ID).Preload("AkaActor").Preload("Akas").First(&aka)

	for _, a := range r.Actors {
		if !strings.HasPrefix(a, "aka:") {

			tmpActor.ID = 0
			db.Where("name = ?", a).First(&tmpActor)
			if tmpActor.ID != 0 {
				// check if the actor is already this group
				cnt := 0
				db.Model(&aka).
					Joins("join actor_akas on actor_akas.aka_id = akas.id").
					Where("actor_akas.actor_id  = ?", tmpActor.ID).Count(&cnt)
				if cnt > 0 {
					errors = append(errors, a+" already in this group")
				} else {
					// check if the actor is already in another group
					cnt = 0
					db.Model(models.Aka{}).
						Joins("join actor_akas on akas.id=actor_akas.aka_id").
						Where("actor_akas.actor_id  = ?", tmpActor.ID).Count(&cnt)
					if cnt > 0 {
						errors = append(errors, a+" already in another group")
					}
					aka.Akas = append(aka.Akas, tmpActor)
					actorCnt++

				}
			} else {
				errors = append(errors, a+" not found")
			}
		}
	}

	if actorCnt > 0 {
		aka.Name = aka.AkaNameSortedAlphabetcally()
		aka.Save()
	}

	RefreshAka(&aka)
	if len(errors) > 0 {
		createResp := &ResponseAka{
			Status: strings.Join(errors, ","),
			Aka:    aka,
		}
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	createResp := &ResponseAka{
		Aka: aka,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, createResp)

}
func RefreshAka(aka *models.Aka) {
	db, _ := models.GetDB()
	defer db.Close()

	aka.UpdateAkaSceneCastRecords()
	db.Preload("AkaActor").Preload("Akas").Find(&aka)
}
