package models

import (
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
)

// MatchFileToScene assigns a file to a scene, the same way the match screen's Assign button does:
// the file is linked to the scene, its filename is recorded on the scene so the match survives the
// file being moved, and the scene's availability is recalculated.
func MatchFileToScene(db *gorm.DB, fileID uint, sceneID string) error {
	var scene Scene
	if err := scene.GetIfExist(sceneID); err != nil {
		return fmt.Errorf("scene %q not found: %w", sceneID, err)
	}

	var f File
	if err := db.Preload("Volume").Where(&File{ID: fileID}).First(&f).Error; err != nil {
		return fmt.Errorf("file %d not found: %w", fileID, err)
	}
	f.SceneID = scene.ID
	f.Save()

	var filenames []string
	if err := json.Unmarshal([]byte(scene.FilenamesArr), &filenames); err != nil {
		return fmt.Errorf("scene %q has unreadable filenames: %w", sceneID, err)
	}
	// Assigning the same file twice, or saving a draft that already lists it, must not record the
	// filename twice.
	known := false
	for _, n := range filenames {
		if n == f.Filename {
			known = true
			break
		}
	}
	if !known {
		filenames = append(filenames, f.Filename)
	}
	if tmp, err := json.Marshal(filenames); err == nil {
		scene.FilenamesArr = string(tmp)
	}

	AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)
	scene.UpdateStatus()
	return nil
}
