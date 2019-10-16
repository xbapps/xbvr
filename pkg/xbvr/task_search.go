package xbvr

import (
	"fmt"
	"path/filepath"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/index/scorch"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type Index struct {
	bleve bleve.Index
}

func NewIndex(name string) *Index {
	i := new(Index)

	path := filepath.Join(common.IndexDir, name)

	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewUsing(path, mapping, scorch.Name, scorch.Name, nil)
	if err != nil && err == bleve.ErrorIndexPathExists {
		idx, err = bleve.Open(path)
	}

	i.bleve = idx
	return i
}

func (i *Index) GetScene(id string) (models.Scene, error) {
	if _, err := i.bleve.Document(id); err != nil {
		return models.Scene{}, err
	}

	data, err := i.bleve.GetInternal(i.formatInternalKey(id))
	if err != nil {
		return models.Scene{}, err
	}

	s := models.Scene{}
	err = s.FromJSON(data)
	return s, err
}

func (i *Index) PutScene(scene models.Scene) error {
	scene.Fulltext = fmt.Sprintf("%v %v %v", scene.Title, scene.Site, scene.Synopsis)

	databytes, err := scene.ToJSON()
	if err != nil {
		return err
	}

	if err = i.bleve.Index(scene.SceneID, scene); err != nil {
		return err
	}

	if err = i.bleve.SetInternal(i.formatInternalKey(scene.SceneID), databytes); err != nil {
		i.bleve.Delete(scene.SceneID)
		return err
	}

	return nil
}

func (i *Index) formatInternalKey(id string) []byte {
	return []byte(fmt.Sprintf("raw:document:%s", id))
}

func SearchIndex() {
	if !models.CheckLock("index") {
		models.CreateLock("index")

		tlog := log.WithFields(logrus.Fields{"task": "scrape"})

		idx := NewIndex("scenes")

		db, _ := models.GetDB()
		defer db.Close()

		total := 0
		offset := 0
		current := 0
		var scenes []models.Scene
		tx := db.Model(models.Scene{}).Preload("Cast").Preload("Tags")
		tx.Count(&total)

		for {
			tx.Offset(offset).Limit(100).Find(&scenes)
			if len(scenes) == 0 {
				break
			}

			for i := range scenes {
				if _, err := idx.GetScene(scenes[i].SceneID); err != nil {
					err := idx.PutScene(scenes[i])
					if err != nil {
						log.Error(err)
					}
				}
				current = current + 1

				tlog.Infof("Indexing scene %v of %v", current, total)
			}

			offset = offset + 100
		}

		idx.bleve.Close()

		tlog.Infof("Search index built!")

		models.RemoveLock("index")
	}
}
