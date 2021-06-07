package tasks

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/index/scorch"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type Index struct {
	Bleve bleve.Index
}

type SceneIndexed struct {
	Fulltext string `json:"fulltext"`
}

func NewIndex(name string) (*Index, error) {
	i := new(Index)

	path := filepath.Join(common.IndexDirV2, name)

	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewUsing(path, mapping, scorch.Name, scorch.Name, nil)
	if err != nil && err == bleve.ErrorIndexPathExists {
		idx, err = bleve.Open(path)
	}
	if err != nil {
		return nil, err
	}

	i.Bleve = idx
	return i, nil
}

func (i *Index) Exist(id string) bool {
	d, err := i.Bleve.Document(id)
	if err != nil || d == nil {
		return false
	}
	return true
}

func (i *Index) PutScene(scene models.Scene) error {
	cast := ""
	castConcat := ""
	for _, c := range scene.Cast {
		cast = cast + " " + c.Name
		castConcat = castConcat + " " + strings.Replace(c.Name, " ", "", -1)
	}

	si := SceneIndexed{
		Fulltext: fmt.Sprintf("%v %v %v %v %v %v", scene.SceneID, scene.Title, scene.Site, scene.Synopsis, cast, castConcat),
	}
	if err := i.Bleve.Index(scene.SceneID, si); err != nil {
		return err
	}

	return nil
}

func SearchIndex() {
	if !models.CheckLock("index") {
		models.CreateLock("index")

		tlog := log.WithFields(logrus.Fields{"task": "scrape"})

		idx, err := NewIndex("scenes")
		if err != nil {
			log.Error(err)
			models.RemoveLock("index")
			return
		}

		db, _ := models.GetDB()
		defer db.Close()

		total := 0
		offset := 0
		current := 0
		var scenes []models.Scene
		tx := db.Model(models.Scene{}).Preload("Cast").Preload("Tags")
		tx.Count(&total)

		tlog.Infof("Building search index...")

		for {
			tx.Offset(offset).Limit(100).Find(&scenes)
			if len(scenes) == 0 {
				break
			}

			for i := range scenes {
				if !idx.Exist(scenes[i].SceneID) {
					err := idx.PutScene(scenes[i])
					if err != nil {
						log.Error(err)
					}
				}
				current = current + 1
			}
			tlog.Infof("Indexed %v/%v scenes", current, total)

			offset = offset + 100
		}

		idx.Bleve.Close()

		tlog.Infof("Search index built!")

		models.RemoveLock("index")
	}
}
