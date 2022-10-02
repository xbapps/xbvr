package models

import (
	"sort"
	"strings"
	"time"

	"github.com/avast/retry-go/v3"
)

type Aka struct {
	ID         uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt  time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt  time.Time `json:"-" xbvrbackup:"-"`
	Name       string    `json:"name" xbvrbackup:"name"`
	AkaActorId uint      `json:"aka_actor_id" xbvrbackup:"-"`

	AkaActor Actor   `gorm:"foreignKey:AkaActorId; references:ID" json:"aka_actor"  xbvrbackup:"aka_actor"`
	Akas     []Actor `gorm:"many2many:actor_akas;" json:"actors"  xbvrbackup:"actors"`
}

func (i *Aka) Save() error {
	db, _ := GetDB()
	defer db.Close()

	err := retry.Do(
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

func (o *Aka) GetIfExistByPK(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Actors").
		Where(&Aka{ID: id}).First(o).Error
}

func (o *Aka) UpdateAkaSceneCastRecords() {
	db, _ := GetDB()
	defer db.Close()

	// Queries to update the scene_cast table for the aka actor are comlex but fast.
	//  Significating faster than iterating through the results of multiple simpler queries.
	// 	The Raw Sql used is compatible between mysql & sqlite

	// add missing scene_cast records for aka actors
	db.Exec(`
	insert into scene_cast 
	select distinct sc.scene_id, a.aka_actor_id 
	from akas a 
	join actor_akas aa on aa.aka_id =a.id 
	join scene_cast sc on sc.actor_id = aa.actor_id
	left join scene_cast sc2 on sc2.scene_id = sc.scene_id  and sc2.actor_id = a.aka_actor_id 
	where sc2.actor_id is NULL 
	`)

	// make a list of scene_cast records for aka actors that have been removed
	type DeleteList struct {
		AkaActorId uint
		SceneId    uint
	}
	var deleteList []DeleteList

	db.Raw(`
		with SceneIds as (
			select distinct a.id, sc.scene_id
			from akas a 
			join actor_akas aa on aa.aka_id =a.id 
			join scene_cast sc on sc.actor_id=aa.actor_id
			)
			select distinct a.aka_actor_id, sc.scene_id  from akas a
			join scene_cast sc on sc.actor_id=a.aka_actor_id 
			left join SceneIds si on si.id=a.id and sc.scene_id= si.scene_id
			where si.scene_id is null	
			`).Scan(&deleteList)

	for _, scenecast := range deleteList {
		db.Exec(` delete from scene_cast where scene_id = ? and actor_id = ?`, scenecast.SceneId, scenecast.AkaActorId)
	}

	var actor Actor
	actor.CountActorTags()
	o.RefreshAkaActorNames()
}

func (o *Aka) RefreshAkaActorNames() {
	db, _ := GetDB()
	defer db.Close()

	type SortedList struct {
		AkaActorId uint
		SortedName string
	}
	var sortedList []SortedList

	// this update the aka names by reordering the actor names based on descending count
	db.Raw(`
	with sorted as (
		select a.aka_actor_id, a2.name, a2.count  from akas a 
		join actor_akas aa on a.id =aa.aka_id 
		join actors a2 on a2.id =aa.actor_id		
		ORDER BY a.id, a2.count DESC
		)
		select aka_actor_id, GROUP_CONCAT(name) as sorted_name from sorted group by aka_actor_id
		`).Scan(&sortedList)

	for _, listItem := range sortedList {
		var actor Actor
		actor.ID = listItem.AkaActorId
		db.Model(&actor).Where("name != ?", "aka:"+listItem.SortedName).Update("name", "aka:"+listItem.SortedName)
	}
}

func (o *Aka) AkaNameSortedAlphabetcally() string {
	var names []string
	for _, a := range o.Akas {
		names = append(names, a.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
