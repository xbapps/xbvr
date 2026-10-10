package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

// Undoing a JAV split: only while the original still exists (it was hidden, not deleted). Deleting
// the original makes the split permanent.

type ResponseJavSplitStatus struct {
	JAVR            bool     `json:"javr"`
	IsSplit         bool     `json:"is_split"`
	OriginalID      uint     `json:"original_id"`
	OriginalSceneID string   `json:"original_scene_id"`
	OriginalExists  bool     `json:"original_exists"`
	Splits          []string `json:"splits"`
	CanSplit        bool     `json:"can_split"`
	CanUndo         bool     `json:"can_undo"`
}

type RequestJavSplitUndo struct {
	SceneID uint `json:"scene_id"` // database ID of the original or of any of its splits
}

// splitBaseID: VRKM-139_split2 -> VRKM-139. "" when the scene ID has no _split<n> suffix.
func splitBaseID(sceneID string) string {
	if loc := splitSceneIDRe.FindStringIndex(sceneID); loc != nil {
		return sceneID[:loc[0]]
	}
	return ""
}

// liveSplitsOf returns the scenes split off base (not deleted), ordered by scene ID.
func liveSplitsOf(db *gorm.DB, base string) []models.Scene {
	var candidates []models.Scene
	// "_" is a wildcard in LIKE, so filter the exact pattern afterwards
	db.Where("scene_id LIKE ?", base+"%").Order("scene_id").Find(&candidates)
	var out []models.Scene
	for _, s := range candidates {
		if splitBaseID(s.SceneID) == base {
			out = append(out, s)
		}
	}
	return out
}

func javSplitStatus(db *gorm.DB, scene *models.Scene) ResponseJavSplitStatus {
	st := ResponseJavSplitStatus{JAVR: sceneIsJAVR(scene), IsSplit: sceneIsSplit(scene), Splits: []string{}}
	base := scene.SceneID
	if st.IsSplit {
		base = splitBaseID(scene.SceneID) // "" for a scene that only has the tag
	}
	if base == "" {
		return st
	}
	st.OriginalSceneID = base
	var orig models.Scene
	if db.Where(&models.Scene{SceneID: base}).First(&orig).Error == nil {
		st.OriginalExists = true
		st.OriginalID = orig.ID
	}
	for _, s := range liveSplitsOf(db, base) {
		st.Splits = append(st.Splits, s.SceneID)
	}
	st.CanSplit = st.JAVR && !st.IsSplit && len(st.Splits) == 0
	st.CanUndo = len(st.Splits) > 0 && st.OriginalExists
	return st
}

func (i SceneResource) javSplitStatus(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("scene-id"))
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, ResponseJavSplitStatus{Splits: []string{}})
		return
	}
	db, _ := models.GetDB()
	defer db.Close()
	var scene models.Scene
	if err := scene.GetIfExistByPK(uint(id)); err != nil {
		resp.WriteHeaderAndEntity(http.StatusNotFound, ResponseJavSplitStatus{Splits: []string{}})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, javSplitStatus(db, &scene))
}

// javSplitUndo puts a split back together: the splits' files and filenames go back to the
// original, the original is shown again, the splits and their edits are deleted.
func (i SceneResource) javSplitUndo(req *restful.Request, resp *restful.Response) {
	fail := func(code int, msg string) {
		log.Error("undo split: " + msg)
		resp.WriteHeaderAndEntity(code, ResponseSplitScene{Status: "error", Message: msg})
	}

	var r RequestJavSplitUndo
	if err := req.ReadEntity(&r); err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	var scene models.Scene
	if err := scene.GetIfExistByPK(r.SceneID); err != nil {
		fail(http.StatusNotFound, fmt.Sprintf("scene %d not found", r.SceneID))
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	st := javSplitStatus(db, &scene)
	if len(st.Splits) == 0 {
		fail(http.StatusBadRequest, scene.SceneID+" has no split scenes to undo")
		return
	}
	if !st.OriginalExists {
		fail(http.StatusBadRequest, st.OriginalSceneID+" was deleted, the split is permanent")
		return
	}

	var orig models.Scene
	if err := orig.GetIfExistByPK(st.OriginalID); err != nil {
		fail(http.StatusNotFound, st.OriginalSceneID+" not found")
		return
	}
	splits := liveSplitsOf(db, orig.SceneID)

	// Filenames: the original's leftovers first, then each split's
	seen := map[string]bool{}
	var names []string
	addNames := func(arr string) {
		var list []string
		json.Unmarshal([]byte(arr), &list)
		for _, n := range list {
			if n != "" && !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}
	addNames(orig.FilenamesArr)
	var splitIDs []uint
	var splitSceneIDs []string
	for _, s := range splits {
		addNames(s.FilenamesArr)
		splitIDs = append(splitIDs, s.ID)
		splitSceneIDs = append(splitSceneIDs, s.SceneID)
	}
	if names == nil {
		names = []string{}
	}
	namesJSON, _ := json.Marshal(names)

	// UpdateColumn everywhere: gorm v1's Update re-saves preloaded associations
	tx := db.Begin()
	steps := []func() error{
		func() error {
			return tx.Model(&models.File{}).Where("scene_id IN (?)", splitIDs).UpdateColumn("scene_id", orig.ID).Error
		},
		func() error {
			return tx.Model(&models.Scene{}).Where("id = ?", orig.ID).UpdateColumns(map[string]interface{}{"filenames_arr": string(namesJSON), "is_hidden": false}).Error
		},
		func() error {
			return tx.Create(&models.Action{SceneID: orig.SceneID, ActionType: "edit", ChangedColumn: "filenames_arr", NewValue: string(namesJSON)}).Error
		},
		func() error {
			return tx.Where("scene_id IN (?)", splitSceneIDs).Delete(&models.Action{}).Error
		},
		func() error {
			return tx.Where("id IN (?)", splitIDs).Delete(&models.Scene{}).Error
		},
	}
	for _, step := range steps {
		if err := step(); err != nil {
			tx.Rollback()
			fail(http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		fail(http.StatusInternalServerError, err.Error())
		return
	}

	var out models.Scene
	out.GetIfExistByPK(orig.ID)
	out.UpdateStatus()
	out.GetIfExistByPK(orig.ID)

	log.Infof("Undid the split of %s: %s deleted, files back on %s", orig.SceneID, strings.Join(splitSceneIDs, ", "), orig.SceneID)

	go func() {
		tasks.CountTags()
		tasks.IndexScenes(&[]models.Scene{out})
	}()

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseSplitScene{Status: "OK", Scenes: []models.Scene{out}})
}
