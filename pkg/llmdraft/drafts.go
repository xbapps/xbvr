package llmdraft

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xbapps/xbvr/pkg/models"
)

// DraftView is a draft with its proposed scene decoded, as the UI shows it.
type DraftView struct {
	models.DraftScene
	Scene models.ScrapedScene `json:"scene"`
}

func toView(d models.DraftScene) DraftView {
	v := DraftView{DraftScene: d}
	v.Scene, _ = d.Scene()
	return v
}

// ListDrafts returns drafts, most confident first. fileID 0 lists drafts for every file; an empty
// status lists every status.
func ListDrafts(fileID uint, status string) []DraftView {
	db, _ := models.GetDB()
	defer db.Close()
	q := db.Model(&models.DraftScene{})
	if fileID != 0 {
		q = q.Where("file_id = ?", fileID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var drafts []models.DraftScene
	q.Order("match_confidence desc, id desc").Find(&drafts)
	out := make([]DraftView, 0, len(drafts))
	for _, d := range drafts {
		out = append(out, toView(d))
	}
	return out
}

// GetDraft returns one draft.
func GetDraft(id uint) (*DraftView, error) {
	db, _ := models.GetDB()
	defer db.Close()
	var d models.DraftScene
	if err := db.Where(&models.DraftScene{ID: id}).First(&d).Error; err != nil {
		return nil, fmt.Errorf("draft %d not found", id)
	}
	v := toView(d)
	return &v, nil
}

// SaveDraft adds a draft's scene to the library through the normal scene import, and matches
// fileID to it when given. The file's other open drafts are dismissed. The new scene still has to
// be added to the search index by the caller.
func SaveDraft(id, fileID uint) (string, error) {
	db, _ := models.GetDB()
	defer db.Close()

	var d models.DraftScene
	if err := db.Where(&models.DraftScene{ID: id}).First(&d).Error; err != nil {
		return "", fmt.Errorf("draft %d not found", id)
	}
	if d.Status == models.DraftStatusSaved {
		return d.SceneID, errors.New("draft is already saved")
	}
	scene, err := d.Scene()
	if err != nil {
		return "", fmt.Errorf("draft %d is unreadable: %w", id, err)
	}
	if fileID == 0 {
		fileID = d.FileID
	}

	if fileID != 0 {
		var f models.File
		if err := db.Where(&models.File{ID: fileID}).First(&f).Error; err != nil {
			return "", fmt.Errorf("file %d not found", fileID)
		}
		if !containsString(scene.Filenames, f.Filename) {
			scene.Filenames = append(scene.Filenames, f.Filename)
		}
	}

	// The scene import replaces a scene's filename list. When the scene already exists, for example
	// because another part of it was saved earlier, keep the filenames it already has: XBVR uses
	// them to re-match files after they move.
	var existing models.Scene
	if existing.GetIfExist(scene.SceneID) == nil {
		var names []string
		if json.Unmarshal([]byte(existing.FilenamesArr), &names) == nil {
			for _, n := range names {
				if !containsString(scene.Filenames, n) {
					scene.Filenames = append(scene.Filenames, n)
				}
			}
		}
	}

	if err := models.SceneCreateUpdateFromExternal(db, scene); err != nil {
		return "", fmt.Errorf("creating scene: %w", err)
	}
	if fileID != 0 {
		if err := models.MatchFileToScene(db, fileID, scene.SceneID); err != nil {
			return scene.SceneID, fmt.Errorf("scene created but matching the file failed: %w", err)
		}
	}

	d.Status = models.DraftStatusSaved
	d.SceneID = scene.SceneID
	d.FileID = fileID
	if err := models.RetryIfBusy(func() error { return db.Save(&d).Error }); err != nil {
		return scene.SceneID, fmt.Errorf("scene created but updating the draft failed: %w", err)
	}
	if fileID != 0 {
		models.RetryIfBusy(func() error {
			return db.Model(&models.DraftScene{}).
				Where("file_id = ? AND status = ? AND id <> ?", fileID, models.DraftStatusDraft, d.ID).
				Update("status", models.DraftStatusDismissed).Error
		})
	}
	return scene.SceneID, nil
}

// RejectDraft marks a draft rejected. It is kept, not deleted, so the same page is not proposed
// for the same file again.
func RejectDraft(id uint) error {
	db, _ := models.GetDB()
	defer db.Close()
	var affected int64
	err := models.RetryIfBusy(func() error {
		res := db.Model(&models.DraftScene{}).Where("id = ? AND status <> ?", id, models.DraftStatusSaved).
			Update("status", models.DraftStatusRejected)
		affected = res.RowsAffected
		return res.Error
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("draft %d not found or already saved", id)
	}
	return nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
