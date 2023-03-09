package models

import (
	"time"

	"github.com/avast/retry-go/v4"
)

type TagGroup struct {
	ID            uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt     time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt     time.Time `json:"-" xbvrbackup:"-"`
	Name          string    `json:"name" xbvrbackup:"name"`
	TagGroupTagId uint      `json:"tag_group_tag_id" xbvrbackup:"-"`

	TagGroupTag Tag   `gorm:"foreignKey:TagGroupTagId; references:ID" json:"tag_group_tag"  xbvrbackup:"tag_group_tag"`
	Tags        []Tag `gorm:"many2many:tag_group_tags;" json:"tags"  xbvrbackup:"tags"`
}

func (i *TagGroup) Save() error {
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

func (o *TagGroup) GetIfExistByPK(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Tags").
		Where(&TagGroup{ID: id}).First(o).Error
}

func (o *TagGroup) GetIfExistByName(name string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Tags").
		Where(&TagGroup{Name: name}).First(o).Error
}

func (o *TagGroup) UpdateSceneTagRecords() {
	db, _ := GetDB()
	defer db.Close()

	// Queries to update the scene_tags table for the tag group are complex but fast.
	//  Significating faster than iterating through the results of multiple simpler queries.
	// 	The Raw Sql used is compatible between mysql & sqlite

	// add missing scene_cast records for tag groups
	db.Exec(`
	insert into scene_tags
	select distinct st.scene_id, tg.tag_group_tag_id 
	from tag_groups tg
	join tag_group_tags tgt on tgt.tag_group_id =tg.id
	join scene_tags st on st.tag_id = tgt.tag_id 
	left join scene_tags st2 on st2.scene_id = st.scene_id  and st2.tag_id  = tg.tag_group_tag_id
	where st2.tag_id is NULL 
	`)

	// delete scene_tags for tag groups that have been removed
	type DeleteList struct {
		TagGroupTagId uint
		SceneId       uint
	}

	db.Exec(`
	with SceneIds as (
		select distinct tg.id, st.scene_id
			from tag_groups tg  
			join tag_group_tags tgt on tgt.tag_group_id =tg.id 
			join scene_tags st on st.tag_id=tgt.tag_id 
			),
		DeleteRows as (
			select distinct tg.tag_group_tag_id, st.scene_id  from tag_groups tg
			join scene_tags st on st.tag_id=tg.tag_group_tag_id  
			left join SceneIds si on si.id=tg.id and st.scene_id= si.scene_id
			where si.scene_id is null
		)
		delete from scene_tags
		where EXISTS (
		Select 1 from DeleteRows
		WHERE DeleteRows.scene_id=scene_tags.scene_id  
		  AND DeleteRows.tag_group_tag_id=scene_tags.tag_id
		)
	`)

	var tag Tag
	tag.CountTags()
}
