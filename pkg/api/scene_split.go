package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

// Splitting a JAV scene into parts: VRKM-139 -> VRKM-139_split1, VRKM-139_split2, ...
// API: /api/scene/javsplit (+ /unmatched-files, /status/{id}, /undo in scene_split_undo.go)
//
// How it stays put through rescans and rescrapes:
//   - Every filename belongs to exactly one scene, because XBVR leaves a file unmatched when two
//     scenes list its name (pkg/tasks/volume.go).
//   - A rescrape overwrites a scene's filename list, title and duration, then ReapplyEdits replays
//     every edit action. So each split's filenames, title, duration and its "split javr" tag are
//     written as edit actions, and so is the original's leftover filename list.

const splitTag = "split javr"

var splitSceneIDRe = regexp.MustCompile(`_split\d+$`)

type RequestSplitScenePart struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	Duration   int      `json:"duration"`
	Filenames  []string `json:"filenames"`
	Favourite  bool     `json:"favourite"`
	Watched    bool     `json:"watched"`
	Hidden     bool     `json:"hidden"`
	StarRating float64  `json:"star_rating"`
}

type RequestSplitScene struct {
	SceneID  uint                    `json:"scene_id"` // database ID of the original
	Parts    []RequestSplitScenePart `json:"parts"`
	Keep     []string                `json:"keep"`     // filenames staying with the original ("none")
	Original string                  `json:"original"` // "hide" or "delete"
}

type ResponseSplitScene struct {
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Scenes  []models.Scene `json:"scenes,omitempty"`
}

// sceneIsJAVR: the split button only exists for R18.dev crawler scenes and scenes from the JAV
// import box (no scraper ID, an r18.dev or dmm.co.jp scene URL).
func sceneIsJAVR(scene *models.Scene) bool {
	if strings.HasPrefix(scene.ScraperId, "r18dev-") {
		return true
	}
	u := strings.ToLower(scene.SceneURL)
	return scene.ScraperId == "" && (strings.Contains(u, "r18.dev") || strings.Contains(u, "dmm.co.jp"))
}

// sceneIsSplit: splits can't be split again.
func sceneIsSplit(scene *models.Scene) bool {
	if splitSceneIDRe.MatchString(scene.SceneID) {
		return true
	}
	for _, t := range scene.Tags {
		if t.Name == splitTag {
			return true
		}
	}
	return false
}

func splitSceneID(original string, n int) string {
	return fmt.Sprintf("%s_split%d", original, n)
}

// validateSplit checks a split request against the original scene, before anything is written.
func validateSplit(scene *models.Scene, r *RequestSplitScene) error {
	if !sceneIsJAVR(scene) {
		return fmt.Errorf("%s is not a JAV scene from R18.dev or the JAV import box", scene.SceneID)
	}
	if sceneIsSplit(scene) {
		return fmt.Errorf("%s is already a split scene", scene.SceneID)
	}
	if len(r.Parts) == 0 {
		return fmt.Errorf("no parts to split into")
	}
	if r.Original != "hide" && r.Original != "delete" {
		return fmt.Errorf("original must be hide or delete, not %q", r.Original)
	}
	numbers := map[int]bool{}
	owner := map[string]string{}
	for _, p := range r.Parts {
		if p.Number < 1 || p.Number > 99 {
			return fmt.Errorf("part number %d out of range", p.Number)
		}
		if numbers[p.Number] {
			return fmt.Errorf("part %d is listed twice", p.Number)
		}
		numbers[p.Number] = true
		for _, f := range p.Filenames {
			if prev, ok := owner[f]; ok {
				return fmt.Errorf("%s is in both %s and _split%d: a file can only match one scene", f, prev, p.Number)
			}
			owner[f] = fmt.Sprintf("_split%d", p.Number)
		}
	}
	for _, f := range r.Keep {
		if prev, ok := owner[f]; ok {
			return fmt.Errorf("%s is in both %s and %s: a file can only match one scene", f, prev, scene.SceneID)
		}
	}
	return nil
}

// splitCopiesAction: which of the original's edits get copied to each split. The split writes its
// own filenames, title and duration; the multipart flag is not copied at all.
func splitCopiesAction(column string) bool {
	switch column {
	case "filenames_arr", "is_multipart", "title", "duration":
		return false
	}
	return true
}

type RequestSplitFiles struct {
	Names []string `json:"names"`
}

// splitUnmatchedFiles: unmatched files with exactly these names, so the split dialog can show the
// length of a part that was only added by name.
func (i SceneResource) splitUnmatchedFiles(req *restful.Request, resp *restful.Response) {
	var r RequestSplitFiles
	if err := req.ReadEntity(&r); err != nil || len(r.Names) == 0 {
		resp.WriteHeaderAndEntity(http.StatusOK, []models.File{})
		return
	}
	db, _ := models.GetDB()
	defer db.Close()
	var files []models.File
	db.Where("scene_id = 0 AND filename IN (?)", r.Names).Find(&files)
	if files == nil {
		files = []models.File{}
	}
	resp.WriteHeaderAndEntity(http.StatusOK, files)
}

func (i SceneResource) splitScene(req *restful.Request, resp *restful.Response) {
	fail := func(code int, msg string) {
		log.Error("split scene: " + msg)
		resp.WriteHeaderAndEntity(code, ResponseSplitScene{Status: "error", Message: msg})
	}

	var r RequestSplitScene
	if err := req.ReadEntity(&r); err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}

	var orig models.Scene
	if err := orig.GetIfExistByPK(r.SceneID); err != nil {
		fail(http.StatusNotFound, fmt.Sprintf("scene %d not found", r.SceneID))
		return
	}
	if err := validateSplit(&orig, &r); err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	if existing := liveSplitsOf(db, orig.SceneID); len(existing) > 0 {
		fail(http.StatusConflict, orig.SceneID+" is already split ("+existing[0].SceneID+", ...), undo that first")
		return
	}
	for _, p := range r.Parts {
		id := splitSceneID(orig.SceneID, p.Number)
		var existing models.Scene
		if db.Where(&models.Scene{SceneID: id}).First(&existing).Error == nil {
			fail(http.StatusConflict, id+" already exists")
			return
		}
	}

	var origActions []models.Action
	db.Where(&models.Action{SceneID: orig.SceneID}).Order("id").Find(&origActions)

	var tag models.Tag
	db.Where(&models.Tag{Name: splitTag}).FirstOrCreate(&tag)

	partOf := map[string]int{} // filename -> index in r.Parts
	for pi, p := range r.Parts {
		for _, f := range p.Filenames {
			partOf[f] = pi
		}
	}

	tx := db.Begin()
	addAction := func(sceneID, column, value string) error {
		return tx.Create(&models.Action{SceneID: sceneID, ActionType: "edit", ChangedColumn: column, NewValue: value}).Error
	}
	abort := func(err error) {
		tx.Rollback()
		fail(http.StatusInternalServerError, err.Error())
	}

	newIDs := make([]uint, len(r.Parts))
	for pi, p := range r.Parts {
		sceneID := splitSceneID(orig.SceneID, p.Number)
		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = orig.Title
		}
		names := p.Filenames
		if names == nil {
			names = []string{}
		}
		namesJSON, _ := json.Marshal(names)

		// A deleted scene of the same ID (an earlier split that was deleted) would hand its old
		// edits to this one: ReapplyEdits matches edits by scene ID.
		if err := tx.Where(&models.Action{SceneID: sceneID}).Delete(&models.Action{}).Error; err != nil {
			abort(err)
			return
		}

		ns := orig
		ns.ID = 0
		ns.CreatedAt, ns.UpdatedAt, ns.DeletedAt = time.Time{}, time.Time{}, nil
		ns.SceneID = sceneID
		ns.Title = title
		ns.Duration = p.Duration
		ns.FilenamesArr = string(namesJSON)
		ns.Favourite = p.Favourite
		ns.IsWatched = p.Watched
		ns.IsHidden = p.Hidden
		ns.StarRating = p.StarRating
		ns.IsMultipart = false
		ns.Wishlist = false // XBVR clears it once a scene has files anyway
		ns.Tags, ns.Cast, ns.Files, ns.Cuepoints, ns.History, ns.AlternateSource = nil, nil, nil, nil, nil, nil
		ns.TotalWatchTime = 0
		ns.LastOpened = time.Time{}
		ns.LegacySceneID = ""
		ns.NeedsUpdate = false
		ns.EditsApplied = true
		ns.AddedDate = time.Now()
		if err := tx.Create(&ns).Error; err != nil {
			abort(err)
			return
		}
		newIDs[pi] = ns.ID

		tags := append(append([]models.Tag{}, orig.Tags...), tag)
		if err := tx.Model(&ns).Association("Tags").Append(tags).Error; err != nil {
			abort(err)
			return
		}
		if len(orig.Cast) > 0 {
			if err := tx.Model(&ns).Association("Cast").Append(orig.Cast).Error; err != nil {
				abort(err)
				return
			}
		}

		// The original's edits first, then the split's own
		for _, a := range origActions {
			if splitCopiesAction(a.ChangedColumn) {
				if err := tx.Create(&models.Action{SceneID: sceneID, ActionType: a.ActionType, ChangedColumn: a.ChangedColumn, NewValue: a.NewValue}).Error; err != nil {
					abort(err)
					return
				}
			}
		}
		for _, a := range [][2]string{
			{"title", title},
			{"duration", fmt.Sprint(p.Duration)},
			{"filenames_arr", string(namesJSON)},
			{"tags", "+" + splitTag},
		} {
			if err := addAction(sceneID, a[0], a[1]); err != nil {
				abort(err)
				return
			}
		}
	}

	// Matched files go with the split their name was assigned to. Unmatched files with exactly that
	// name (a part only added in the dialog, ...uhqe4.mp4) get matched to it right away instead of
	// waiting for the next rescan.
	// UpdateColumn everywhere: gorm v1's Update also re-saves preloaded associations, which would put
	// the original's files straight back on it.
	for _, f := range orig.Files {
		if pi, ok := partOf[path.Base(f.Filename)]; ok {
			if err := tx.Model(&models.File{}).Where("id = ?", f.ID).UpdateColumn("scene_id", newIDs[pi]).Error; err != nil {
				abort(err)
				return
			}
		}
	}
	for pi, p := range r.Parts {
		if len(p.Filenames) == 0 {
			continue
		}
		// plus the .funscript / .hsp / .srt / .cmscript XBVR matches through an .mp4 name
		names := append([]string{}, p.Filenames...)
		for _, f := range p.Filenames {
			if strings.HasSuffix(strings.ToLower(f), ".mp4") {
				base := f[:len(f)-len(".mp4")]
				for _, e := range []string{".funscript", ".hsp", ".srt", ".cmscript"} {
					names = append(names, base+e)
				}
			}
		}
		if err := tx.Model(&models.File{}).Where("scene_id = 0 AND filename IN (?)", names).UpdateColumn("scene_id", newIDs[pi]).Error; err != nil {
			abort(err)
			return
		}
	}

	// The original keeps only the "none" filenames, as an edit so a rescrape doesn't bring the
	// split filenames back (that would leave those files matching two scenes, so neither)
	keep := r.Keep
	if keep == nil {
		keep = []string{}
	}
	keepJSON, _ := json.Marshal(keep)
	if err := tx.Model(&models.Scene{}).Where("id = ?", orig.ID).UpdateColumn("filenames_arr", string(keepJSON)).Error; err != nil {
		abort(err)
		return
	}
	if err := addAction(orig.SceneID, "filenames_arr", string(keepJSON)); err != nil {
		abort(err)
		return
	}

	if r.Original == "hide" {
		if err := tx.Model(&models.Scene{}).Where("id = ?", orig.ID).UpdateColumn("is_hidden", true).Error; err != nil {
			abort(err)
			return
		}
	} else {
		// Same as Delete Scene: unmatch its files, mark the scene deleted
		if err := tx.Model(&models.File{}).Where("scene_id = ?", orig.ID).UpdateColumn("scene_id", 0).Error; err != nil {
			abort(err)
			return
		}
		if err := tx.Where("id = ?", orig.ID).Delete(&models.Scene{}).Error; err != nil {
			abort(err)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		fail(http.StatusInternalServerError, err.Error())
		return
	}

	// Availability, file size, scripts etc. from the files each scene has now
	var out []models.Scene
	for _, id := range newIDs {
		var s models.Scene
		if s.GetIfExistByPK(id) == nil {
			s.UpdateStatus()
			s.GetIfExistByPK(id)
			out = append(out, s)
		}
	}
	if r.Original == "hide" {
		var o models.Scene
		if o.GetIfExistByPK(orig.ID) == nil {
			o.UpdateStatus()
		}
	}

	log.Infof("Split %s into %d scenes, original %s", orig.SceneID, len(out), map[string]string{"hide": "hidden", "delete": "deleted"}[r.Original])

	go func() {
		tasks.CountTags()
		tasks.IndexScenes(&out)
	}()

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseSplitScene{Status: "OK", Scenes: out})
}
