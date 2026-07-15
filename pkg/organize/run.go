package organize

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type moveOp struct {
	file   *models.File
	src    string
	dst    string
	name   string
	isMove bool
}
type dupOp struct {
	file        *models.File
	src         string
	identicalTo string
	size        int64
	maybe       bool // dry-run only: OsHash match, md5 not yet confirmed
}
type scenePlan struct {
	sceneID    uint
	targetDir  string
	volumeRoot string
	videoCount int
	actionable bool
	moves      []moveOp
	hlUnlink   []*models.File
	contentDup []dupOp
	sidecars   [][2]string // {src, dst}
	symlinks   [][2]string // {linkPath, target} per-actor symlinks to targetDir
	rmdirs     map[string]bool
}

// Run computes (and, unless DryRun, performs) the reorganisation and returns a Result.
func Run(db *gorm.DB, opts Options) *Result {
	res := &Result{DryRun: opts.DryRun}
	now := time.Now()
	cutoff := now.Add(-time.Duration(opts.IncomingMinAge) * 24 * time.Hour).Unix()

	// Storage folders (local volumes) are the move boundaries: a scene is only ever
	// reorganised within the folder its files already live in, never across folders.
	volRoots := localVolumeRoots(db)
	if len(volRoots) == 0 {
		common.Log.Warn("organize: no local storage folders configured; nothing to do")
		return res
	}

	// dir (db path) -> set of scene ids with a video file there (immutable snapshot)
	dirScenes := map[string]map[uint]bool{}
	var frows []models.File
	db.Model(&models.File{}).Select("path, scene_id").Where("type = ? AND scene_id > 0", "video").Find(&frows)
	for _, f := range frows {
		if dirScenes[f.Path] == nil {
			dirScenes[f.Path] = map[uint]bool{}
		}
		dirScenes[f.Path][f.SceneID] = true
	}

	var sceneIDs []uint
	db.Model(&models.File{}).Where("type = ? AND scene_id > 0", "video").Order("scene_id").Pluck("DISTINCT scene_id", &sceneIDs)

	targetOwner := map[string]uint{}
	removedPaths := map[string]bool{}
	dryRemovedDirs := map[string]bool{}
	considered := 0

	for _, sid := range sceneIDs {
		if opts.Limit > 0 && considered >= opts.Limit {
			break
		}
		var scene models.Scene
		if err := db.Preload("Cast").Preload("Files").First(&scene, sid).Error; err != nil {
			continue
		}
		if scene.DeletedAt != nil {
			continue
		}
		plan, deferred := processScene(db, &scene, opts, volRoots, cutoff, dirScenes, res)
		if deferred {
			res.Deferred++
			continue
		}
		if plan == nil || !plan.actionable {
			continue
		}
		if _, ok := targetOwner[plan.targetDir]; ok {
			res.Merged++
		} else {
			targetOwner[plan.targetDir] = sid
		}
		applyPlan(db, plan, opts, res, removedPaths, dryRemovedDirs)
		res.ScenesActed++
		considered += plan.videoCount
	}
	if opts.SymlinkByActor {
		pruneDanglingActorSymlinks(volRoots, opts, res)
	}
	common.Log.Infof("organize [dry=%v]: %d scenes acted, %d moved, %d renamed, %d dedup(-%0.1fGB), %d hardlinks, %d sidecars, %d symlinks (%d pruned), %d dirs removed, %d deferred, %d held",
		opts.DryRun, res.ScenesActed, res.FilesMoved, res.FilesRenamed, res.Dedups,
		float64(res.BytesReclaimed)/1e9, res.Hardlinks, res.Sidecars, res.Symlinks, res.SymlinksPruned, res.DirsRemoved, res.Deferred, res.Held)
	return res
}

// pruneDanglingActorSymlinks removes per-actor symlinks whose scene directory no longer
// exists (e.g. after a scene was renamed) and drops emptied actor folders.
func pruneDanglingActorSymlinks(volRoots []string, opts Options, res *Result) {
	for _, vroot := range volRoots {
		actorRoot := filepath.Join(vroot, opts.TopFolder, opts.ActorFolder)
		actors, err := os.ReadDir(actorRoot)
		if err != nil {
			continue
		}
		for _, a := range actors {
			if !a.IsDir() {
				continue
			}
			adir := filepath.Join(actorRoot, a.Name())
			links, err := os.ReadDir(adir)
			if err != nil {
				continue
			}
			live := 0
			for _, l := range links {
				link := filepath.Join(adir, l.Name())
				if l.Type()&os.ModeSymlink == 0 {
					live++
					continue
				}
				if _, err := os.Stat(link); os.IsNotExist(err) {
					if !opts.DryRun {
						if err := os.Remove(link); err != nil {
							live++
							continue
						}
					}
					res.Actions = append(res.Actions, Action{Kind: "symlink-prune", From: link})
					res.SymlinksPruned++
					continue
				}
				live++
			}
			if live == 0 && !opts.DryRun {
				os.Remove(adir) // emptied actor folder
			}
		}
	}
}

// localVolumeRoots returns the enabled local storage-folder roots, longest first so a
// path is matched to the deepest containing volume.
func localVolumeRoots(db *gorm.DB) []string {
	var vols []models.Volume
	db.Where("type = ? AND is_enabled = ?", "local", true).Find(&vols)
	roots := make([]string, 0, len(vols))
	for _, v := range vols {
		if r := strings.TrimRight(v.Path, "/"); r != "" {
			roots = append(roots, r)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return len(roots[i]) > len(roots[j]) })
	return roots
}

// volumeOf returns the storage-folder root that contains path, or "" if none does.
func volumeOf(path string, roots []string) string {
	for _, r := range roots {
		if path == r || strings.HasPrefix(path, r+"/") {
			return r
		}
	}
	return ""
}

func lookupOsHash(db *gorm.DB, path string) string {
	var f models.File
	if db.Select("os_hash").Where("path = ? AND filename = ?", filepath.Dir(path), filepath.Base(path)).First(&f).Error == nil {
		return f.OsHash
	}
	return ""
}

// processScene builds the plan for one scene. Returns (nil, true) when the whole
// scene is deferred (possible duplicates + DeferDups).
func processScene(db *gorm.DB, scene *models.Scene, opts Options, volRoots []string, cutoff int64, dirScenes map[string]map[uint]bool, res *Result) (*scenePlan, bool) {
	dt, ok := parseReleaseDate(*scene)
	title := sanitizeToken(scene.Title)
	if scene.Site == "" || !ok || title == "" {
		return nil, false
	}

	type exFile struct {
		f      *models.File
		disk   string
		height int
		ino    [2]uint64
		size   int64
	}
	var existing []exFile
	for i := range scene.Files {
		f := &scene.Files[i]
		if f.Type != "video" {
			continue
		}
		disk := f.GetPath()
		fi, err := os.Stat(disk)
		if err != nil {
			continue
		}
		// Staging: hold files still sitting in a storage folder's incoming area that are
		// too recently modified to judge. IncomingDir is relative to each storage folder.
		if opts.IncomingDir != "" {
			if fv := volumeOf(disk, volRoots); fv != "" &&
				isUnder(disk, filepath.Join(fv, opts.IncomingDir)) && fi.ModTime().Unix() > cutoff {
				res.Held++
				continue
			}
		}
		existing = append(existing, exFile{f, disk, f.VideoHeight, inodeKey(fi), fi.Size()})
	}
	if len(existing) == 0 {
		return nil, false
	}

	// representative = max(height, size, -id)
	pickRep := func(xs []exFile) int {
		rep := 0
		for i := range xs {
			a, b := xs[i], xs[rep]
			if a.height > b.height || (a.height == b.height && (a.size > b.size || (a.size == b.size && a.f.ID < b.f.ID))) {
				rep = i
			}
		}
		return rep
	}

	// Organise within the storage folder holding the best copy; only files already in
	// that folder are touched, so nothing is ever moved across storage folders.
	vroot := volumeOf(existing[pickRep(existing)].disk, volRoots)
	if vroot == "" {
		return nil, false
	}
	kept := existing[:0:0]
	for _, e := range existing {
		if volumeOf(e.disk, volRoots) == vroot {
			kept = append(kept, e)
		}
	}
	existing = kept
	if len(existing) == 0 {
		return nil, false
	}

	rep := pickRep(existing)
	repHeight := existing[rep].height
	for _, e := range existing {
		if e.height > repHeight {
			repHeight = e.height
		}
	}
	fov := fovToken(existing[rep].f.VideoProjection)
	cast := castNames(scene.Cast, opts.CastGender)
	studio, dirName := buildDirName(scene.Site, dt, cast, title, fov, repHeight)
	if studio == "" {
		return nil, false
	}

	plan := &scenePlan{sceneID: scene.ID, videoCount: len(existing), volumeRoot: vroot, rmdirs: map[string]bool{}}
	// TopFolder is an optional wrapper under the storage-folder root ("" -> studio
	// folders go directly in the root). filepath.Join drops the empty element.
	plan.targetDir = filepath.Join(vroot, opts.TopFolder, studio, dirName)

	// group by inode (hard links within scene)
	byInode := map[[2]uint64][]int{}
	for i := range existing {
		byInode[existing[i].ino] = append(byInode[existing[i].ino], i)
	}
	var keepers []int
	for _, grp := range byInode {
		keep := grp[0]
		for _, idx := range grp {
			if idx == rep {
				keep = rep
			}
		}
		if keep != rep {
			// pick max within group
			for _, idx := range grp {
				a, b := existing[idx], existing[keep]
				if a.height > b.height || (a.height == b.height && a.size > b.size) {
					keep = idx
				}
			}
		}
		keepers = append(keepers, keep)
		for _, idx := range grp {
			if idx != keep {
				plan.hlUnlink = append(plan.hlUnlink, existing[idx].f)
			}
		}
	}
	// order keepers: representative first, then by descending resolution/size
	sort.Slice(keepers, func(i, j int) bool {
		if (keepers[i] == rep) != (keepers[j] == rep) {
			return keepers[i] == rep
		}
		a, b := existing[keepers[i]], existing[keepers[j]]
		if a.height != b.height {
			return a.height > b.height
		}
		return a.size > b.size
	})

	// content dedup
	type cand struct {
		osHash string
		path   string
		ino    [2]uint64
	}
	sizeIdx := map[int64][]cand{}
	keeperDisks := map[string]bool{}
	for _, k := range keepers {
		keeperDisks[existing[k].disk] = true
	}
	if opts.Dedup {
		if entries, err := os.ReadDir(plan.targetDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				p := filepath.Join(plan.targetDir, e.Name())
				if keeperDisks[p] {
					continue
				}
				if fi, err := e.Info(); err == nil {
					sizeIdx[fi.Size()] = append(sizeIdx[fi.Size()], cand{lookupOsHash(db, p), p, inodeKey(fi)})
				}
			}
		}
	}
	var survivors []int
	for _, k := range keepers {
		e := existing[k]
		var hardlinkOf, dupOf string
		var dupMaybe bool
		cands := sizeIdx[e.size]
		if opts.Dedup {
			for _, c := range cands {
				if c.ino == e.ino {
					hardlinkOf = c.path
					break
				}
			}
			if hardlinkOf == "" {
				for _, c := range cands {
					if e.f.OsHash != "" && c.osHash != "" && e.f.OsHash == c.osHash {
						if opts.DeferDups {
							return nil, true
						}
						if opts.DryRun {
							// Preview only: OsHash matched but md5 isn't verified here, so
							// this is a candidate, not a confirmed delete.
							dupOf = c.path
							dupMaybe = true
							break
						}
						if m1, ok1 := fileMD5(e.disk); ok1 {
							if m2, ok2 := fileMD5(c.path); ok2 && m1 == m2 {
								dupOf = c.path
								break
							}
						}
					}
				}
			}
		}
		switch {
		case hardlinkOf != "":
			plan.hlUnlink = append(plan.hlUnlink, e.f)
			plan.actionable = true
		case dupOf != "":
			plan.contentDup = append(plan.contentDup, dupOp{e.f, e.disk, dupOf, e.size, dupMaybe})
			plan.actionable = true
		default:
			survivors = append(survivors, k)
			sizeIdx[e.size] = append(sizeIdx[e.size], cand{e.f.OsHash, e.disk, e.ino})
		}
	}
	keepers = survivors
	for _, d := range plan.contentDup {
		if !d.maybe {
			plan.rmdirs[filepath.Dir(d.src)] = true
		}
	}

	// name assignment + sidecars
	sceneDiskPaths := map[string]bool{}
	for _, k := range keepers {
		sceneDiskPaths[existing[k].disk] = true
	}
	for _, f := range plan.hlUnlink {
		sceneDiskPaths[f.GetPath()] = true
	}
	// gather sidecars per source dir
	type scDir struct {
		foreign bool
		entries []string
	}
	sidecarSrc := map[string]*scDir{}
	for _, k := range keepers {
		dir := filepath.Dir(existing[k].disk)
		if _, done := sidecarSrc[dir]; done {
			continue
		}
		sd := &scDir{}
		for s := range dirScenes[dir] {
			if s != scene.ID {
				sd.foreign = true
			}
		}
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				p := filepath.Join(dir, e.Name())
				if sceneDiskPaths[p] {
					continue
				}
				ext := lowerExt(e.Name())
				if videoExts[ext] {
					sd.foreign = true
					continue
				}
				if !sidecarExts[ext] {
					continue
				}
				sd.entries = append(sd.entries, p)
			}
		}
		sidecarSrc[dir] = sd
	}

	// reserve foreign names in target
	placedOrCand := map[string]bool{}
	for _, k := range keepers {
		placedOrCand[existing[k].disk] = true
	}
	for _, sd := range sidecarSrc {
		for _, p := range sd.entries {
			placedOrCand[p] = true
		}
	}
	used := map[string]bool{}
	if entries, err := os.ReadDir(plan.targetDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			p := filepath.Join(plan.targetDir, e.Name())
			if !placedOrCand[p] {
				used[e.Name()] = true
			}
		}
	}
	assign := func(src, desired string, height int) (string, bool) {
		if filepath.Dir(src) == plan.targetDir {
			name := filepath.Base(src)
			used[name] = true
			return name, false
		}
		name := resolveName(used, desired, height)
		used[name] = true
		return name, filepath.Join(plan.targetDir, name) != src
	}

	stemmap := map[string]map[string]string{} // srcDir -> origStem -> finalStem
	for _, k := range keepers {
		e := existing[k]
		h := e.height
		if k == rep {
			h = repHeight
		}
		name, isMove := assign(e.disk, e.f.Filename, h)
		if isMove {
			plan.actionable = true
		}
		plan.moves = append(plan.moves, moveOp{e.f, e.disk, filepath.Join(plan.targetDir, name), name, isMove})
		dir := filepath.Dir(e.disk)
		if stemmap[dir] == nil {
			stemmap[dir] = map[string]string{}
		}
		origStem, _ := splitExt(e.f.Filename)
		finalStem, _ := splitExt(name)
		stemmap[dir][origStem] = finalStem
		plan.rmdirs[dir] = true
	}
	if len(plan.hlUnlink) > 0 {
		plan.actionable = true
	}
	for _, f := range plan.hlUnlink {
		plan.rmdirs[filepath.Dir(f.GetPath())] = true
	}

	for dir, sd := range sidecarSrc {
		for _, p := range sd.entries {
			name := filepath.Base(p)
			var pairedFinal string
			for orig, final := range stemmap[dir] {
				if name == orig || (len(name) > len(orig) && name[:len(orig)+1] == orig+".") {
					pairedFinal = final + name[len(orig):]
					break
				}
			}
			var desired string
			if pairedFinal != "" {
				desired = pairedFinal
			} else if !sd.foreign {
				desired = name
			} else {
				continue
			}
			dstName, isMove := assign(p, desired, repHeight)
			if !isMove {
				continue
			}
			plan.sidecars = append(plan.sidecars, [2]string{p, filepath.Join(plan.targetDir, dstName)})
			plan.actionable = true
		}
	}

	// Per-actor symlinks: link the scene dir into a CamelCase folder per performer.
	if opts.SymlinkByActor {
		actorRoot := filepath.Join(vroot, opts.TopFolder, opts.ActorFolder)
		seen := map[string]bool{}
		for _, name := range selectCast(scene.Cast, opts.CastGender) {
			camel := camelActor(name)
			if camel == "" || seen[camel] {
				continue
			}
			seen[camel] = true
			actorDir := filepath.Join(actorRoot, camel)
			linkPath := filepath.Join(actorDir, dirName)
			if _, err := os.Lstat(linkPath); err == nil {
				continue // already linked
			}
			target, err := filepath.Rel(actorDir, plan.targetDir)
			if err != nil {
				target = plan.targetDir
			}
			plan.symlinks = append(plan.symlinks, [2]string{linkPath, target})
			plan.actionable = true
		}
	}

	return plan, false
}

func lowerExt(name string) string {
	return strings.ToLower(filepath.Ext(name))
}
