package organize

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

func applyPlan(db *gorm.DB, p *scenePlan, opts Options, res *Result, removedPaths, dryRemovedDirs map[string]bool) {
	if !opts.DryRun {
		os.MkdirAll(p.targetDir, 0o755)
	}

	deletedRows := false

	// Identical-content duplicates first (both files still at source; md5 confirms).
	for _, d := range p.contentDup {
		rel, _ := filepath.Rel(p.targetDir, d.identicalTo)
		if d.maybe {
			// Dry-run candidate: md5 not verified, so don't count it as a real delete.
			res.Actions = append(res.Actions, Action{SceneID: p.sceneID, Kind: "dedup-maybe", From: d.src, Note: "possible == " + rel + " (md5 unconfirmed)"})
			continue
		}
		if !opts.DryRun {
			m1, ok1 := fileMD5(d.src)
			m2, ok2 := fileMD5(d.identicalTo)
			if !ok1 || !ok2 || m1 != m2 {
				continue
			}
			if err := os.Remove(d.src); err != nil {
				continue
			}
			db.Delete(&models.File{}, d.file.ID)
			deletedRows = true
		}
		res.Actions = append(res.Actions, Action{SceneID: p.sceneID, Kind: "dedup-delete", From: d.src, Note: "== " + rel})
		res.Dedups++
		res.BytesReclaimed += d.size
	}

	// Moves / renames of kept files.
	for _, m := range p.moves {
		if m.isMove {
			kind := "move"
			if filepath.Dir(m.src) == p.targetDir {
				kind = "rename"
			}
			if !opts.DryRun {
				if err := moveFile(m.src, m.dst); err != nil {
					common.Log.Warnf("organize: move %s failed: %v", m.src, err)
					continue
				}
			}
			res.Actions = append(res.Actions, Action{SceneID: p.sceneID, Kind: kind, From: m.src, To: m.dst})
			if kind == "rename" {
				res.FilesRenamed++
			} else {
				res.FilesMoved++
			}
		}
		if !opts.DryRun && (m.file.Path != p.targetDir || m.file.Filename != m.name) {
			db.Model(&models.File{}).Where("id = ?", m.file.ID).
				Updates(map[string]interface{}{"path": p.targetDir, "filename": m.name})
		}
	}

	// Hard-link duplicates: unlink, drop the stale DB row.
	for _, f := range p.hlUnlink {
		src := f.GetPath()
		if !opts.DryRun {
			os.Remove(src)
			db.Delete(&models.File{}, f.ID)
			deletedRows = true
		}
		res.Actions = append(res.Actions, Action{SceneID: p.sceneID, Kind: "hardlink-unlink", From: src})
		res.Hardlinks++
	}

	// Sidecars.
	for _, s := range p.sidecars {
		if !opts.DryRun {
			if err := moveFile(s[0], s[1]); err != nil {
				continue
			}
			db.Model(&models.File{}).Where("path = ? AND filename = ?", filepath.Dir(s[0]), filepath.Base(s[0])).
				Updates(map[string]interface{}{"path": p.targetDir, "filename": filepath.Base(s[1])})
		}
		res.Actions = append(res.Actions, Action{SceneID: p.sceneID, Kind: "sidecar", From: s[0], To: s[1]})
		res.Sidecars++
	}

	// Per-actor symlinks into CamelCase folders.
	for _, s := range p.symlinks {
		link, target := s[0], s[1]
		if !opts.DryRun {
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				common.Log.Warnf("organize: mkdir %s failed: %v", filepath.Dir(link), err)
				continue
			}
			if err := os.Symlink(target, link); err != nil {
				common.Log.Warnf("organize: symlink %s failed: %v", link, err)
				continue
			}
		}
		res.Actions = append(res.Actions, Action{SceneID: p.sceneID, Kind: "symlink", From: link, To: target})
		res.Symlinks++
	}

	// Record which source files left, for empty-dir detection.
	for _, m := range p.moves {
		if m.isMove {
			removedPaths[m.src] = true
		}
	}
	for _, f := range p.hlUnlink {
		removedPaths[f.GetPath()] = true
	}
	for _, d := range p.contentDup {
		if d.maybe {
			continue // candidate only; not actually removed
		}
		removedPaths[d.src] = true
	}
	for _, s := range p.sidecars {
		removedPaths[s[0]] = true
	}

	// Remove emptied source directories, deepest first.
	dirs := make([]string, 0, len(p.rmdirs))
	for d := range p.rmdirs {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if d == p.targetDir {
			continue
		}
		removeEmptyDirs(d, p.targetDir, p.volumeRoot, opts, res, removedPaths, dryRemovedDirs)
	}

	// Deleting a scene's own files can change its availability; refresh it.
	if deletedRows {
		var s models.Scene
		if db.First(&s, p.sceneID).Error == nil {
			s.UpdateStatus()
		}
	}
}

func removeEmptyDirs(start, target, prefix string, opts Options, res *Result, removedPaths, dryRemovedDirs map[string]bool) {
	if prefix == "" {
		return
	}
	d := start
	// Stop at the storage-folder root, and never walk into a directory that contains the
	// target (the target keeps it non-empty once files land there).
	for d != "" && d != prefix && strings.HasPrefix(d, prefix+"/") && !isUnder(target, d) {
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		remaining := 0
		for _, e := range entries {
			p := filepath.Join(d, e.Name())
			if opts.DryRun && (removedPaths[p] || dryRemovedDirs[p]) {
				continue
			}
			remaining++
		}
		if remaining > 0 {
			return
		}
		if opts.DryRun {
			dryRemovedDirs[d] = true
		} else if err := os.Remove(d); err != nil {
			return
		}
		res.Actions = append(res.Actions, Action{Kind: "rmdir", From: d})
		res.DirsRemoved++
		d = filepath.Dir(d)
	}
}
