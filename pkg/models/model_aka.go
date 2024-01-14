package models

import (
	"sort"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
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
	commonDb, _ := GetCommonDB()

	err := retry.Do(
		func() error {
			err := commonDb.Save(&i).Error
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
	commonDb, _ := GetCommonDB()

	return commonDb.
		Preload("Actors").
		Where(&Aka{ID: id}).First(o).Error
}

func (o *Aka) UpdateAkaSceneCastRecords() {
	commonDb, _ := GetCommonDB()

	// Queries to update the scene_cast table for the aka actor are comlex but fast.
	//  Significating faster than iterating through the results of multiple simpler queries.
	// 	The Raw Sql used is compatible between mysql & sqlite

	// add missing scene_cast records for aka actors
	commonDb.Exec(`
	insert into scene_cast 
	select distinct sc.scene_id, a.aka_actor_id 
	from akas a 
	join actor_akas aa on aa.aka_id =a.id 
	join scene_cast sc on sc.actor_id = aa.actor_id
	left join scene_cast sc2 on sc2.scene_id = sc.scene_id  and sc2.actor_id = a.aka_actor_id 
	where sc2.actor_id is NULL 
	`)

	// delete scene_cast records for aka actors that have been removed
	commonDb.Exec(`
		with SceneIds as (
			select distinct a.id, sc.scene_id
			from akas a 
			join actor_akas aa on aa.aka_id =a.id 
			join scene_cast sc on sc.actor_id=aa.actor_id
			),
		DeleteRows as (
			select distinct a.aka_actor_id, sc.scene_id  from akas a
			join scene_cast sc on sc.actor_id=a.aka_actor_id 
			left join SceneIds si on si.id=a.id and sc.scene_id= si.scene_id
			where si.scene_id is null
			)
		delete from scene_cast
		where EXISTS (
		Select 1 from DeleteRows
		WHERE DeleteRows.scene_id=scene_cast.scene_id  
			AND DeleteRows.aka_actor_id=scene_cast.actor_id
			`)

	var actor Actor
	actor.CountActorTags()
	o.RefreshAkaActorNames()
}

func (o *Aka) RefreshAkaActorNames() {
	commonDb, _ := GetCommonDB()

	type SortedList struct {
		AkaActorId uint
		SortedName string
	}
	var sortedList []SortedList

	// this update the aka names by reordering the actor names based on descending count
	commonDb.Raw(`
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
		commonDb.Model(&actor).Where("name != ?", "aka:"+listItem.SortedName).Update("name", "aka:"+listItem.SortedName)
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
