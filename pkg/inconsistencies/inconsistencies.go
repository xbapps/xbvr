package inconsistencies

import (
	"encoding/json"
	"sync"

	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// Inconsistency describes a file/scene data problem found by the "Fix inconsistencies"
// tool, along with a suggested remedy.
type Inconsistency struct {
	Kind          string `json:"kind"` // file-on-deleted-scene | file-on-missing-scene
	FileID        uint   `json:"fileId"`
	Path          string `json:"path"`
	Filename      string `json:"filename"`
	SceneID       uint   `json:"sceneId"` // the broken scene the file points at
	SceneTitle    string `json:"sceneTitle"`
	SceneSite     string `json:"sceneSite"`
	Detail        string `json:"detail"`
	Suggestion    string `json:"suggestion"`    // rematch | unmatch | refresh | recount-tag | recount-actor
	TargetSceneID uint   `json:"targetSceneId"` // suggested live scene for a rematch (0 = none)
	TargetTitle   string `json:"targetTitle"`
	EntityID      uint   `json:"entityId"`    // tag/actor id for count fixes
	EntityLabel   string `json:"entityLabel"` // tag/actor name
}

type incRow struct {
	FileID   uint
	Path     string
	Filename string
	SceneID  uint
	Title    string
	Site     string
}

// ScanInconsistencies finds files whose scene reference is broken: pointing at a
// soft-deleted scene, or at a scene id that no longer exists. Raw SQL is used so gorm's
// soft-delete scope doesn't hide the deleted scenes we're specifically looking for.
func ScanInconsistencies(db *gorm.DB) []Inconsistency {
	out := []Inconsistency{}

	// Files matched to a soft-deleted scene.
	var deleted []incRow
	db.Raw(`SELECT f.id AS file_id, f.path, f.filename, f.scene_id AS scene_id, s.title, s.site
	        FROM files f JOIN scenes s ON s.id = f.scene_id
	        WHERE f.scene_id > 0 AND s.deleted_at IS NOT NULL
	        ORDER BY f.id`).Scan(&deleted)
	for _, r := range deleted {
		inc := Inconsistency{
			Kind: "file-on-deleted-scene", FileID: r.FileID, Path: r.Path, Filename: r.Filename,
			SceneID: r.SceneID, SceneTitle: r.Title, SceneSite: r.Site,
		}
		// A live scene with the same title on the same site is almost always a
		// re-published version (e.g. VirtualTaboo appends a suffix to the URL); prefer
		// one that has no file yet.
		var twin struct {
			ID    uint
			Title string
		}
		db.Raw(`SELECT id, title FROM scenes
		        WHERE deleted_at IS NULL AND site = ? AND title = ?
		        ORDER BY (SELECT count(*) FROM files WHERE scene_id = scenes.id) ASC, id ASC
		        LIMIT 1`, r.Site, r.Title).Scan(&twin)
		if twin.ID != 0 {
			inc.Suggestion = "rematch"
			inc.TargetSceneID = twin.ID
			inc.TargetTitle = twin.Title
			inc.Detail = "Matched to a deleted scene; a live scene with the same title exists (likely a re-published version)."
		} else {
			inc.Suggestion = "unmatch"
			inc.Detail = "Matched to a deleted scene with no live replacement found."
		}
		out = append(out, inc)
	}

	// Files matched to a scene id that has no row at all.
	var missing []incRow
	db.Raw(`SELECT f.id AS file_id, f.path, f.filename, f.scene_id AS scene_id
	        FROM files f
	        WHERE f.scene_id > 0 AND NOT EXISTS (SELECT 1 FROM scenes s WHERE s.id = f.scene_id)
	        ORDER BY f.id`).Scan(&missing)
	for _, r := range missing {
		out = append(out, Inconsistency{
			Kind: "file-on-missing-scene", FileID: r.FileID, Path: r.Path, Filename: r.Filename,
			SceneID: r.SceneID, Suggestion: "unmatch",
			Detail: "Matched to a scene that no longer exists.",
		})
	}

	// Live scenes whose stored status no longer matches their files: marked available or
	// accessible with no video file, or a total_file_size that doesn't equal the sum of
	// the scene's file sizes (UpdateStatus's no-files path never clears these).
	var stale []struct {
		ID    uint
		Title string
		Site  string
	}
	db.Raw(`SELECT id, title, site FROM scenes s
	        WHERE deleted_at IS NULL
	          AND (
	            ((is_available = 1 OR is_accessible = 1)
	              AND NOT EXISTS (SELECT 1 FROM files f WHERE f.scene_id = s.id AND f.type = 'video'))
	            OR total_file_size <> COALESCE((SELECT SUM(f.size) FROM files f WHERE f.scene_id = s.id), 0)
	          )
	        ORDER BY id`).Scan(&stale)
	for _, r := range stale {
		out = append(out, Inconsistency{
			Kind: "scene-stale-status", SceneID: r.ID, SceneTitle: r.Title, SceneSite: r.Site,
			Suggestion: "refresh",
			Detail:     "Scene's size, availability or accessibility doesn't match its files.",
		})
	}

	// Tags whose cached scene count is wrong (the recompute uses an inner join, so tags
	// that drop to zero scenes are never reset).
	var staleTags []struct {
		ID   uint
		Name string
	}
	db.Raw(`SELECT t.id, t.name FROM tags t
	        LEFT JOIN (SELECT st.tag_id AS id, count(*) AS cnt FROM scene_tags st
	                   JOIN scenes s ON s.id = st.scene_id AND s.deleted_at IS NULL
	                   GROUP BY st.tag_id) tc ON tc.id = t.id
	        WHERE t.count <> COALESCE(tc.cnt, 0)
	        ORDER BY t.id`).Scan(&staleTags)
	for _, r := range staleTags {
		out = append(out, Inconsistency{
			Kind: "tag-count-stale", EntityID: r.ID, EntityLabel: r.Name,
			Suggestion: "recount-tag",
			Detail:     "Tag's cached scene count is out of date.",
		})
	}

	// Actors whose cached scene count or available count is wrong (same inner-join defect).
	var staleActors []struct {
		ID   uint
		Name string
	}
	db.Raw(`SELECT a.id, a.name FROM actors a
	        LEFT JOIN (SELECT sc.actor_id AS id, count(*) AS cnt, COALESCE(SUM(s.is_available), 0) AS av
	                   FROM scene_cast sc
	                   JOIN scenes s ON s.id = sc.scene_id AND s.deleted_at IS NULL
	                   GROUP BY sc.actor_id) ac ON ac.id = a.id
	        WHERE a.count <> COALESCE(ac.cnt, 0) OR a.avail_count <> COALESCE(ac.av, 0)
	        ORDER BY a.id`).Scan(&staleActors)
	for _, r := range staleActors {
		out = append(out, Inconsistency{
			Kind: "actor-count-stale", EntityID: r.ID, EntityLabel: r.Name,
			Suggestion: "recount-actor",
			Detail:     "Actor's cached scene / available count is out of date.",
		})
	}

	return out
}

// ---- background "fix all" runner ----

var (
	fixMu      sync.Mutex
	fixRunning bool
	fixPhase   string // scanning | fixing | idle
	fixTotal   int
	fixDone    int
	fixResult  *FixResult
)

// FixResult summarises a completed fix-all run.
type FixResult struct {
	Scanned int            `json:"scanned"`
	Fixed   int            `json:"fixed"`
	Failed  int            `json:"failed"`
	ByKind  map[string]int `json:"byKind"`
}

// StartFixInconsistencies scans for inconsistencies and applies every suggested fix in
// the background. Returns false if a run is already in progress. Progress is reported via
// FixStatus (the scan is a single opaque step; per-item progress is tracked while fixing).
func StartFixInconsistencies() bool {
	fixMu.Lock()
	if fixRunning {
		fixMu.Unlock()
		return false
	}
	fixRunning = true
	fixPhase = "scanning"
	fixTotal, fixDone = 0, 0
	fixMu.Unlock()

	go func() {
		res := &FixResult{ByKind: map[string]int{}}
		db, err := models.GetDB()
		if err == nil {
			items := ScanInconsistencies(db)
			fixMu.Lock()
			fixPhase = "fixing"
			fixTotal = len(items)
			fixMu.Unlock()
			res.Scanned = len(items)
			for _, it := range items {
				var e error
				switch it.Suggestion {
				case "rematch":
					e = RematchFile(db, it.FileID, it.TargetSceneID)
				case "unmatch":
					e = UnmatchFile(db, it.FileID)
				case "refresh":
					e = RefreshScene(db, it.SceneID)
				case "recount-tag":
					e = RecountTag(db, it.EntityID)
				case "recount-actor":
					e = RecountActor(db, it.EntityID)
				}
				if e != nil {
					res.Failed++
				} else {
					res.Fixed++
					res.ByKind[it.Kind]++
				}
				fixMu.Lock()
				fixDone++
				fixMu.Unlock()
			}
			// Scene fixes change is_available, which feeds actor avail_count — so a single
			// pass can leave counts stale for entities that weren't in the initial scan.
			// Recompute all tag/actor counts globally to converge in one run.
			RecountAllTags(db)
			RecountAllActors(db)
			db.Close()
			common.Log.Infof("inconsistencies: fixed %d/%d (%d failed)", res.Fixed, res.Scanned, res.Failed)
		}
		fixMu.Lock()
		fixResult = res
		fixRunning = false
		fixPhase = "idle"
		fixMu.Unlock()
	}()
	return true
}

// FixStatus reports fix-all progress and the most recent result.
func FixStatus() (running bool, phase string, done, total int, result *FixResult) {
	fixMu.Lock()
	defer fixMu.Unlock()
	return fixRunning, fixPhase, fixDone, fixTotal, fixResult
}

// RematchFile moves a file to a live scene, mirroring the manual match: reassign the
// file, add its name to the target scene's filename list, log the action and refresh
// the target scene's availability.
func RematchFile(db *gorm.DB, fileID, targetSceneID uint) error {
	var f models.File
	if err := db.Where(&models.File{ID: fileID}).First(&f).Error; err != nil {
		return err
	}
	var scene models.Scene
	if err := scene.GetIfExistByPK(targetSceneID); err != nil {
		return err
	}
	f.SceneID = scene.ID
	f.Save()

	var names []string
	if json.Unmarshal([]byte(scene.FilenamesArr), &names) == nil {
		seen := false
		for _, n := range names {
			if n == f.Filename {
				seen = true
				break
			}
		}
		if !seen {
			names = append(names, f.Filename)
			if b, err := json.Marshal(names); err == nil {
				scene.FilenamesArr = string(b)
			}
		}
	}
	models.AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)
	scene.UpdateStatus()
	common.Log.Infof("inconsistencies: rematched file %d to scene %d", fileID, targetSceneID)
	return nil
}

// UnmatchFile detaches a file from its (broken) scene reference.
func UnmatchFile(db *gorm.DB, fileID uint) error {
	var f models.File
	if err := db.Where(&models.File{ID: fileID}).First(&f).Error; err != nil {
		return err
	}
	f.SceneID = 0
	f.Save()
	common.Log.Infof("inconsistencies: unmatched file %d", fileID)
	return nil
}

// RefreshScene recomputes a scene's status from its files. For a scene that has lost all
// its files this also clears the stale size that UpdateStatus leaves behind.
func RefreshScene(db *gorm.DB, sceneID uint) error {
	var count int
	db.Model(&models.File{}).Where("scene_id = ?", sceneID).Count(&count)
	if count == 0 {
		return db.Model(&models.Scene{}).Where("id = ?", sceneID).Updates(map[string]interface{}{
			"total_file_size": 0,
			"is_available":    false,
			"is_accessible":   false,
			"is_scripted":     false,
		}).Error
	}
	var scene models.Scene
	if err := scene.GetIfExistByPK(sceneID); err != nil {
		return err
	}
	scene.UpdateStatus()
	return nil
}

// RecountTag recomputes a tag's cached scene count from its non-deleted linked scenes.
func RecountTag(db *gorm.DB, tagID uint) error {
	return db.Exec(`UPDATE tags SET count = COALESCE(
		(SELECT count(*) FROM scene_tags st JOIN scenes s ON s.id = st.scene_id AND s.deleted_at IS NULL
		 WHERE st.tag_id = ?), 0) WHERE id = ?`, tagID, tagID).Error
}

// RecountActor recomputes an actor's cached scene count and available count from its
// non-deleted linked scenes.
func RecountActor(db *gorm.DB, actorID uint) error {
	return db.Exec(`UPDATE actors SET
		count = COALESCE((SELECT count(*) FROM scene_cast sc JOIN scenes s ON s.id = sc.scene_id AND s.deleted_at IS NULL
		                  WHERE sc.actor_id = ?), 0),
		avail_count = COALESCE((SELECT SUM(s.is_available) FROM scene_cast sc JOIN scenes s ON s.id = sc.scene_id AND s.deleted_at IS NULL
		                        WHERE sc.actor_id = ?), 0)
		WHERE id = ?`, actorID, actorID, actorID).Error
}

// RecountAllTags recomputes every tag's cached scene count in one statement.
func RecountAllTags(db *gorm.DB) error {
	return db.Exec(`UPDATE tags SET count = COALESCE(
		(SELECT count(*) FROM scene_tags st JOIN scenes s ON s.id = st.scene_id AND s.deleted_at IS NULL
		 WHERE st.tag_id = tags.id), 0)`).Error
}

// RecountAllActors recomputes every actor's cached scene count and available count.
func RecountAllActors(db *gorm.DB) error {
	return db.Exec(`UPDATE actors SET
		count = COALESCE((SELECT count(*) FROM scene_cast sc JOIN scenes s ON s.id = sc.scene_id AND s.deleted_at IS NULL
		                  WHERE sc.actor_id = actors.id), 0),
		avail_count = COALESCE((SELECT SUM(s.is_available) FROM scene_cast sc JOIN scenes s ON s.id = sc.scene_id AND s.deleted_at IS NULL
		                        WHERE sc.actor_id = actors.id), 0)`).Error
}
