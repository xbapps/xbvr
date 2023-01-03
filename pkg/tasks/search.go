package tasks

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/simple"
	"github.com/blevesearch/bleve/index/scorch"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type Index struct {
	Bleve bleve.Index
}

type SceneIndexed struct {
	Description string `json:"description"`
	Title       string `json:"title"`
	Cast        string `json:"cast"`
	Site        string `json:"site"`
	Id          string `json:"id"`
}

func NewIndex(name string) (*Index, error) {
	i := new(Index)

	path := filepath.Join(common.IndexDirV2, name)

	// the simple analyzer is more approriate for the title and cast
	// note this does not effect search unless the query includes cast: or title:
	titleFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping.Analyzer = simple.Name
	castFieldMapping := bleve.NewTextFieldMapping()
	castFieldMapping.Analyzer = simple.Name
	sceneMapping := bleve.NewDocumentMapping()
	sceneMapping.AddFieldMappingsAt("title", titleFieldMapping)
	sceneMapping.AddFieldMappingsAt("cast", castFieldMapping)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("_default", sceneMapping)

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
		Title:       fmt.Sprintf("%v", scene.Title),
		Description: fmt.Sprintf("%v", scene.Synopsis),
		Cast:        fmt.Sprintf("%v %v", cast, castConcat),
		Site:        fmt.Sprintf("%v", scene.Site),
		Id:          fmt.Sprintf("%v", scene.SceneID),
	}
	if err := i.Bleve.Index(scene.SceneID, si); err != nil {
		return err
	}

	return nil
}

func SearchIndex() {
	if !models.CheckLock("index") {
		models.CreateLock("index")
		defer models.RemoveLock("index")

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
	}
}
