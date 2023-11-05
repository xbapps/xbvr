package models

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	Name   string  `gorm:"unique_index" json:"name" xbvrbackup:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"scenes" xbvrbackup:"-"`
	Count  int     `json:"count" xbvrbackup:"-"`

	AvailCount int `json:"avail_count" xbvrbackup:"-"`

	ImageUrl   string  `json:"image_url" xbvrbackup:"image_url"`
	ImageArr   string  `json:"image_arr" sql:"type:text;" xbvrbackup:"image_arr"`
	StarRating float64 `json:"star_rating" xbvrbackup:"star_rating"`
	Favourite  bool    `json:"favourite" gorm:"default:false" xbvrbackup:"favourite"`
	Watchlist  bool    `json:"watchlist" gorm:"default:false" xbvrbackup:"watchlist"`

	BirthDate   time.Time `json:"birth_date" xbvrbackup:"birth_date"`
	Nationality string    `json:"nationality" xbvrbackup:"nationality"`
	Ethnicity   string    `json:"ethnicity" xbvrbackup:"ethnicity"`
	EyeColor    string    `json:"eye_color" xbvrbackup:"eyeColor"`
	HairColor   string    `json:"hair_color" xbvrbackup:"hairColor"`
	Height      int       `json:"height" xbvrbackup:"height"`
	Weight      int       `json:"weight" xbvrbackup:"weight"`
	CupSize     string    `json:"cup_size" xbvrbackup:"cup_size"`
	BandSize    int       `json:"band_size" xbvrbackup:"band_size"`
	WaistSize   int       `json:"waist_size" xbvrbackup:"waist_size"`
	HipSize     int       `json:"hip_size" xbvrbackup:"hip_size"`
	BreastType  string    `json:"breast_type" xbvrbackup:"breast_type"`
	StartYear   int       `json:"start_year" xbvrbackup:"start_year"`
	EndYear     int       `json:"end_year" xbvrbackup:"end_year"`
	Tattoos     string    `json:"tattoos" sql:"type:text;"  xbvrbackup:"tattoos"`
	Piercings   string    `json:"piercings" sql:"type:text;" xbvrbackup:"biercings"`

	Biography string `json:"biography" sql:"type:text;" xbvrbackup:"biography"`
	Aliases   string `json:"aliases" gorm:"size:1000"  xbvrbackup:"aliases"`
	Gender    string `json:"gender" xbvrbackup:"gender"`
	URLs      string `json:"urls" sql:"type:text;" xbvrbackup:"urls"`

	SceneRatingAverage string `json:"scene_rating_average" gorm:"-" `
	AkaGroups          []Aka  `gorm:"many2many:actor_akas;" json:"aka_groups" xbvrbackup:"-"`
}

type RequestActorList struct {
	DlState        optional.String   `json:"dlState"`
	Limit          optional.Int      `json:"limit"`
	Offset         optional.Int      `json:"offset"`
	Lists          []optional.String `json:"lists"`
	Cast           []optional.String `json:"cast"`
	Sites          []optional.String `json:"sites"`
	Tags           []optional.String `json:"tags"`
	Attributes     []optional.String `json:"attributes"`
	JumpTo         optional.String   `json:"jumpTo"`
	MinAge         optional.Int      `json:"min_age"`
	MaxAge         optional.Int      `json:"max_age"`
	MinHeight      optional.Int      `json:"min_height"`
	MaxHeight      optional.Int      `json:"max_height"`
	MinWeight      optional.Int      `json:"min_weight"`
	MaxWeight      optional.Int      `json:"max_weight"`
	MinCount       optional.Int      `json:"min_count"`
	MaxCount       optional.Int      `json:"max_count"`
	MinAvail       optional.Int      `json:"min_avail"`
	MaxAvail       optional.Int      `json:"max_avail"`
	MinRating      optional.Float64  `json:"min_rating"`
	MaxRating      optional.Float64  `json:"max_rating"`
	MinSceneRating optional.Float64  `json:"min_scene_rating"`
	MaxSceneRating optional.Float64  `json:"max_scene_rating"`
	Sort           optional.String   `json:"sort"`
}
type ResponseActorList struct {
	Results            int     `json:"results"`
	Actors             []Actor `json:"actors"`
	CountAny           int     `json:"count_any"`
	CountAvailable     int     `json:"count_available"`
	CountDownloaded    int     `json:"count_downloaded"`
	CountNotDownloaded int     `json:"count_not_downloaded"`
	CountHidden        int     `json:"count_hidden"`
	Offset             int     `json:"offset"`
}

type ActorLink struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

func (i *Actor) Save() error {
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&i).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

func (i *Actor) CountActorTags() {
	db, _ := GetDB()
	defer db.Close()

	type CountResults struct {
		ID            int
		Cnt           int
		Existingcnt   int
		IsAvailable   int
		Existingavail int
	}

	var results []CountResults

	db.Model(&Actor{}).
		Select("actors.id, count as existingcnt, count(*) cnt, sum(scenes.is_available ) is_available, avail_count as existingavail").
		Group("actors.id").
		Joins("join scene_cast on scene_cast.actor_id = actors.id").
		Joins("join scenes on scenes.id=scene_cast.scene_id and scenes.deleted_at is null").
		Scan(&results)

	for i := range results {
		var actor Actor
		if results[i].Cnt != results[i].Existingcnt || results[i].IsAvailable != results[i].Existingavail {
			db.First(&actor, results[i].ID)
			actor.Count = results[i].Cnt
			actor.AvailCount = results[i].IsAvailable
			actor.Save()
		}
	}
}
func QueryActorFull(r RequestActorList) ResponseActorList {
	var actors []Actor
	r.Limit = optional.NewInt(100)
	r.Offset = optional.NewInt(0)

	q := QueryActors(r, true)
	actors = q.Actors

	for len(actors) < q.Results {
		r.Offset = optional.NewInt(len(actors))
		q := QueryActors(r, true)
		actors = append(actors, q.Actors...)
	}

	q.Actors = actors
	return q
}

func QueryActors(r RequestActorList, enablePreload bool) ResponseActorList {
	limit := r.Limit.OrElse(100)
	offset := r.Offset.OrElse(0)

	db, _ := GetDB()
	defer db.Close()

	var actors []Actor
	tx := db.Model(&actors)

	var out ResponseActorList

	for _, i := range r.Lists {
		if i.OrElse("") == "watchlist" {
			tx = tx.Where("actors.watchlist = ?", true)
		}
		if i.OrElse("") == "favourite" {
			tx = tx.Where("actors.favourite = ?", true)
		}
	}

	// handle Attribute selections
	var orAttribute []string
	var andAttribute []string
	combinedWhere := ""
	for idx, attribute := range r.Attributes {
		truefalse := true
		fieldName := attribute.OrElse("")
		erlAlias := "external_reference_links_f" + strconv.Itoa(idx)

		if strings.HasPrefix(fieldName, "!") { // ! prefix indicate NOT filtering
			truefalse = false
			fieldName = fieldName[1:]
		}
		if strings.HasPrefix(fieldName, "&") { // & prefix indicate must have filtering
			fieldName = fieldName[1:]
		}

		value := ""
		where := ""

		countries := GetCountryList()
		attributes := [][2]string{{"Cup Size ", "cup_size"}, {"Hair Color ", "hair_color"}, {"Eye Color ", "eye_color"}, {"Nationality ", "nationality"}, {"Ethnicity ", "ethnicity"}, {"Breast Type ", "breast_type"}}
		for _, attribute := range attributes {
			if strings.HasPrefix(fieldName, attribute[0]) {
				value = fieldName[len(attribute[0]):]
				if attribute[0] == "Nationality " {
					for _, c := range countries {
						if c.Name == value {
							value = c.Code
						}
					}
				}
				if truefalse {
					where = attribute[1] + " = '" + value + "'"
				} else {
					where = attribute[1] + " <> '" + value + "'"
				}
			}
		}

		switch fieldName {
		case "In Watchlist":
			if truefalse {
				where = "actors.watchlist = 1"
			} else {
				where = "actors.watchlist = 0"
			}
		case "Is Scripted":
			if truefalse {
				where = "is_scripted = 1"
			} else {
				where = "is_scripted = 0"
			}
		case "Is Favourite":
			if truefalse {
				where = "actors.favourite = 1"
			} else {
				where = "actors.favourite = 0"
			}
		case "Has Rating":
			if truefalse {
				where = "actors.star_rating > 0"
			} else {
				where = "actors.star_rating = 0"
			}
		case "Aka Group":
			if truefalse {
				where = "name like 'aka:%'"
			} else {
				where = "name not like 'aka:%'"
			}
		case "In An Aka Group":
			if truefalse {
				where = "(select count(*) from actor_akas " + " where actor_akas.actor_id = actors.id) > 0"
			} else {
				where = "(select count(*) from actor_akas " + " where actor_akas.actor_id = actors.id) = 0"
			}
		case "Has Image":
			if truefalse {
				where = "image_url is not null and image_url <> ''"
			} else {
				where = "image_url is null or image_url = ''"
			}
		case "Possible Aka":
			// find where the stashdb actor is linked to more than 1 xbv actor
			if truefalse {
				where = "(select count(*) from external_reference_links " + erlAlias + " join external_reference_links " + erlAlias + "_2" + " on " + erlAlias + ".external_id = " + erlAlias + "_2" + ".external_id and " + erlAlias + "_2" + " .internal_db_id <> " + erlAlias + ".internal_db_id  where " + erlAlias + ".internal_db_id = actors.id and " + erlAlias + ".`external_source` = 'stashdb performer') > 0"
			} else {
				where = "(select count(*) from external_reference_links " + erlAlias + " join external_reference_links " + erlAlias + "_2" + " on " + erlAlias + ".external_id = " + erlAlias + "_2" + ".external_id and " + erlAlias + "_2" + " .internal_db_id <> " + erlAlias + ".internal_db_id  where " + erlAlias + ".internal_db_id = actors.id and " + erlAlias + ".`external_source` = 'stashdb performer') = 0"
			}
		case "Has Stashdb Link":
			if truefalse {
				where = "(select count(*) from external_reference_links " + erlAlias + " where " + erlAlias + ".internal_db_id = actors.id and " + erlAlias + ".`external_source` = 'stashdb performer') > 0"
			} else {
				where = "(select count(*) from external_reference_links " + erlAlias + " where " + erlAlias + ".internal_db_id = actors.id and " + erlAlias + ".`external_source` = 'stashdb performer') = 0"
			}
		case "Multiple Stashdb Links":
			// find where actor is link to more than 1 stashdb actor, indicates dups in stashdb
			if truefalse {
				where = "(select count(*) from external_reference_links " + erlAlias + " where " + erlAlias + ".internal_db_id = actors.id and " + erlAlias + ".`external_source` = 'stashdb performer') > 1"
			} else {
				where = "(select count(*) from external_reference_links " + erlAlias + " where " + erlAlias + ".internal_db_id = actors.id and " + erlAlias + ".`external_source` = 'stashdb performer') < 1"
			}
		case "Rating 0", "Rating .5", "Rating 1", "Rating 1.5", "Rating 2", "Rating 2.5", "Rating 3", "Rating 3.5", "Rating 4", "Rating 4.5", "Rating 5":
			if truefalse {
				where = "actors.star_rating = " + fieldName[7:]
			} else {
				where = "actors.star_rating <> " + fieldName[7:]
			}
		case "Has Tattoo":
			if truefalse {
				where = "tattoos not in ('','[]')"
			} else {
				where = "tattoos in ('','[]')"
			}
		case "Has Piercing":
			if truefalse {
				where = "piercings not in ('','[]')"
			} else {
				where = "piercings in ('','[]')"
			}
		}
		switch firstchar := string(attribute.OrElse(" ")[0]); firstchar {
		case "&", "!":
			andAttribute = append(andAttribute, where)
		default:
			orAttribute = append(orAttribute, where)
		}
	}
	if len(orAttribute) > 0 {
		combinedWhere = "(" + strings.Join(orAttribute, " or ") + ")"
	}
	if len(andAttribute) > 0 {
		if combinedWhere == "" {
			combinedWhere = strings.Join(andAttribute, " and ")
		} else {
			combinedWhere = combinedWhere + " and " + strings.Join(andAttribute, " and ")
		}
	}
	tx = tx.Where(combinedWhere)

	var cast []string
	var excludedCast []string
	for _, i := range r.Cast {
		switch firstchar := string(i.OrElse(" ")[0]); firstchar {
		case "!":
			exCast, _ := i.Get()
			excludedCast = append(excludedCast, exCast[1:])
		default:
			cast = append(cast, i.OrElse(""))
		}
	}
	if len(cast) > 0 {
		tx = tx.Where("actors.name IN (?)", cast)
	}
	if len(excludedCast) > 0 {
		tx = tx.Where("actors.name NOT IN (?)", excludedCast)
	}

	if r.MinAge.OrElse(0) > 18 || r.MaxAge.OrElse(100) < 100 {
		startRange := time.Now().AddDate(r.MinAge.OrElse(0)*-1, 0, 0)
		endRange := time.Now().AddDate(r.MaxAge.OrElse(0)*-1, 0, 0)
		tx = tx.Where("actors.birth_date <= ? and actors.birth_date >= ?", startRange, endRange)
	}
	if r.MinHeight.OrElse(120) > 120 {
		tx = tx.Where("actors.height >= ?", r.MinHeight.OrElse(120))
	}
	if r.MaxHeight.OrElse(220) < 220 {
		tx = tx.Where("actors.height <= ?", r.MaxHeight.OrElse(220))
	}
	if r.MinWeight.OrElse(25) > 25 {
		tx = tx.Where("actors.weight >= ?", r.MinWeight.OrElse(25))
	}
	if r.MaxWeight.OrElse(150) < 150 {
		tx = tx.Where("actors.weight <= ?", r.MaxWeight.OrElse(150))
	}
	if r.MinCount.OrElse(0) > 0 {
		tx = tx.Where("actors.`count` >= ?", r.MinCount.OrElse(0))
	}
	if r.MaxCount.OrElse(150) < 150 {
		tx = tx.Where("actors.`count` <= ?", r.MaxCount.OrElse(150))
	}
	if r.MinAvail.OrElse(0) > 0 {
		tx = tx.Where("actors.avail_count >= ?", r.MinAvail.OrElse(0))
	}
	if r.MaxAvail.OrElse(150) < 150 {
		tx = tx.Where("actors.avail_count <= ?", r.MinAvail.OrElse(150))
	}
	if r.MinRating.OrElse(0) > 0 {
		tx = tx.Where("actors.star_rating >= ?", r.MinRating.OrElse(0))
	}
	if r.MaxRating.OrElse(5) < 5 {
		tx = tx.Where("actors.star_rating <= ?", r.MaxRating.OrElse(5))
	}
	if r.MinSceneRating.OrElse(0) > 0 {
		tx = tx.Where("(select AVG(s.star_rating) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id and s.star_rating > 0 ) >= ?", r.MinSceneRating.OrElse(0))
	}
	if r.MaxSceneRating.OrElse(5) < 5 {
		tx = tx.Where("(select AVG(s.star_rating) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id and s.star_rating > 0 ) <= ?", r.MaxSceneRating.OrElse(50))
	}
	var sites []string
	var mustHaveSites []string
	var excludedSites []string
	for _, i := range r.Sites {
		switch firstchar := string(i.OrElse(" ")[0]); firstchar {
		case "&":
			inclSite, _ := i.Get()
			mustHaveSites = append(mustHaveSites, inclSite[1:])
		case "!":
			exSite, _ := i.Get()
			excludedSites = append(excludedSites, exSite[1:])
		default:
			sites = append(sites, i.OrElse(""))
		}
	}
	if len(sites) > 0 {
		tx = tx.Where("(select count(*) from scene_cast sc join scenes s on s.id=sc.scene_id join sites on sites.id = s.scraper_id where sc.actor_id=actors.id and sites.name IN (?)) > 0", sites)
	}
	for idx, musthave := range mustHaveSites {
		scAlias := "sc_i" + strconv.Itoa(idx)
		sceneAlias := "s_i" + strconv.Itoa(idx)
		siteAlias := "st_i" + strconv.Itoa(idx)
		tx = tx.
			Joins("join scene_cast "+scAlias+" on "+scAlias+".actor_id=actors.id").
			Joins("join scenes "+sceneAlias+" on "+sceneAlias+".id="+scAlias+".scene_id").
			Joins("join sites "+siteAlias+" on "+siteAlias+".id="+sceneAlias+".scraper_id and "+siteAlias+".name = ?", musthave)
	}
	if len(excludedSites) > 0 {
		tx = tx.Where("(select count(*) from scene_cast sc join scenes s on s.id=sc.scene_id join sites on sites.id = s.scraper_id where sc.actor_id=actors.id and sites.name IN (?)) = 0", excludedSites)
	}

	switch r.Sort.OrElse("") {
	case "name_asc":
		tx = tx.Order("name asc")
	case "name_desc":
		tx = tx.Order("name desc")
	case "rating_desc":
		tx = tx.
			Where("actors.star_rating > ?", 0).
			Order("actors.star_rating desc")
	case "rating_asc":
		tx = tx.
			Where("actors.star_rating > ?", 0).
			Order("actors.star_rating asc")
	case "scene_rating_desc":
		tx = tx.
			Order("(select AVG(s.star_rating) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id and s.star_rating > 0 ) desc, (select count(*) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id and s.star_rating > 0 ) desc, (select count(*) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id) desc")
	case "scene_release_desc":
		tx = tx.
			Order("IFNULL((select max(s.release_date) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id),'1970-01-01') DESC, actors.avail_count desc, actors.`count` desc")
	case "scene_added_desc":
		tx = tx.
			Order("IFNULL((select max(s.created_at) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id),'1970-01-01') DESC, actors.avail_count desc, actors.`count` desc")
	case "file_added_desc":
		tx = tx.
			Order("IFNULL((select max(s.added_date) from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id),'1970-01-01') DESC, actors.avail_count desc, actors.`count` desc")
	case "birthday_desc":
		tx = tx.Order("birth_date desc")
	case "birthday_asc":
		tx = tx.Order("birth_date asc")
	case "added_desc":
		tx = tx.Order("created_at desc")
	case "added_asc":
		tx = tx.Order("created_at asc")
	case "modified_desc":
		tx = tx.Order("updated_at desc")
	case "modified_asc":
		tx = tx.Order("updated_at asc")
	case "random":
		if dbConn.Driver == "mysql" {
			tx = tx.Order("rand()")
		} else {
			tx = tx.Order("random()")
		}
	case "scene_count_desc":
		tx = tx.
			Order("actors.`count` desc, actors.name")
	case "scene_available_desc":
		tx = tx.
			Order("actors.`avail_count` desc, actors.name")
	default:
		tx = tx.Order("name asc")
	}

	tx.Group("actors.id").
		Count(&out.Results)

	tx = tx.Preload("Scenes", func(db *gorm.DB) *gorm.DB {
		return db.Order("release_date DESC").Where("is_hidden = 0")
	})

	if r.JumpTo.OrElse("") != "" {
		// if we want to jump to actors starting with a specific letter, then we need to work out the offset to them
		cnt := 0
		txList := tx.Select(`distinct actors.name`)
		txList.Find(&out.Actors)
		for idx, actor := range out.Actors {
			if strings.ToLower(actor.Name) >= strings.ToLower(r.JumpTo.OrElse("")) {
				break
			}
			cnt = idx
		}
		offset = (cnt / limit) * limit
	}
	out.Offset = offset

	tx = tx.Select(`distinct actors.*, 
	(select AVG(s.star_rating) scene_avg from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id and s.star_rating > 0 ) as scene_rating_average	
	`)

	tx.Limit(limit).
		Offset(offset).
		Find(&out.Actors)

	return out
}

func (o *Actor) GetIfExist(id string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_hidden = 0")
		}).
		Where(&Actor{Name: id}).First(o).Error
}

func (o *Actor) GetIfExistByPK(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_hidden = 0")
		}).
		Where(&Actor{ID: id}).First(o).Error
}

func (o *Actor) GetIfExistByPKWithSceneAvg(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	tx := db.Model(&Actor{})
	tx = tx.Select(`actors.*, 
	(select AVG(s.star_rating) scene_avg from scene_cast sc join scenes s on s.id=sc.scene_id where sc.actor_id =actors.id and s.star_rating > 0 ) as scene_rating_average`)

	return tx.
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Order("release_date DESC").Where("is_hidden = 0")
		}).
		Where(&Actor{ID: id}).First(o).Error
}

func (i *Actor) AddToImageArray(newValue string) bool {
	var array []string
	if newValue == "" {
		return false
	}
	newValue = strings.TrimSuffix(newValue, "/")
	if i.ImageArr == "" {
		i.ImageArr = "[]"
	}

	json.Unmarshal([]byte(i.ImageArr), &array)
	for idx, item := range array {
		if strings.EqualFold(item, newValue) {
			// if we are adding an image that is the main actor image, put it at the begining
			if strings.EqualFold(newValue, i.ImageUrl) && idx > 0 {
				array = append(array[:idx], array[idx+1:]...)
				array = append([]string{item}, array...)
				jsonString, _ := json.Marshal(array)
				i.ImageArr = string(jsonString)
			}
			return false
		}
	}
	if newValue == i.ImageUrl {
		array = append([]string{newValue}, array...)
	} else {
		array = append(array, newValue)
	}
	jsonString, _ := json.Marshal(array)
	i.ImageArr = string(jsonString)
	return true
}
func (i *Actor) AddToActorUrlArray(newValue ActorLink) bool {
	if newValue.Url == "" {
		return false
	}
	newValue.Url = strings.TrimSuffix(newValue.Url, "/")
	var array []ActorLink
	if i.URLs == "" {
		i.URLs = "[]"
	}
	json.Unmarshal([]byte(i.URLs), &array)
	for _, item := range array {
		if strings.EqualFold(item.Url, newValue.Url) {
			return false
		}
	}
	array = append(array, newValue)
	jsonString, _ := json.Marshal(array)
	i.URLs = string(jsonString)
	return true
}
func (i *Actor) AddToTattoos(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Tattoos, updated = addToStringArray(i.Tattoos, newValue)
	}
	return updated
}
func (i *Actor) AddToPiercings(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Piercings, updated = addToStringArray(i.Piercings, newValue)
	}
	return updated
}
func (i *Actor) AddToAliases(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Aliases, updated = addToStringArray(i.Aliases, newValue)
	}
	return updated
}
func addToStringArray(inputArray string, newValue string) (string, bool) {
	var array []string
	if inputArray == "" {
		inputArray = "[]"
	}
	json.Unmarshal([]byte(inputArray), &array)
	for _, item := range array {
		if strings.EqualFold(item, newValue) {
			return inputArray, false
		}
	}
	array = append(array, newValue)
	jsonString, _ := json.Marshal(array)
	return string(jsonString), true
}

func (a *Actor) CheckForSetImage() bool {
	// check if the field was deleted by the user,
	db, _ := GetDB()
	defer db.Close()
	var action ActionActor
	db.Where("source = 'edit_actor' and actor_id = ? and changed_column = 'image_url' and action_type = 'setimage'", a.ID).Order("ID desc").First(&action)
	return action.ID != 0
}

func (a *Actor) CheckForUserDeletes(fieldName string, newValue string) bool {
	// check if the field was deleted by the user,
	db, _ := GetDB()
	defer db.Close()
	var action ActionActor
	db.Where("source = 'edit_actor' and actor_id = ? and changed_column = ? and new_value = ?", a.ID, fieldName, newValue).Order("ID desc").First(&action)
	if action.ID != 0 && action.ActionType == "delete" {
		return true
	}
	return false
}
