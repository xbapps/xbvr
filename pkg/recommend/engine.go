// Package recommend computes "For You" (watch) and "Cleanup" (delete) scores for
// scenes and persists them to scenes.rec_watch_score / scenes.rec_delete_score so
// the seeded deo-enabled playlists can surface them, ranked, in HereSphere/DeoVR.
//
// See DESIGN.md for the algorithm rationale.
package recommend

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// Generate recomputes recommendation scores for the whole library.
func Generate() {
	cfg := loadConfig()
	if !cfg.Enabled {
		common.Log.Info("recommend: disabled in config, skipping")
		return
	}

	db, err := models.GetDB()
	if err != nil {
		common.Log.Errorf("recommend: cannot open db: %v", err)
		return
	}
	defer db.Close()

	now := time.Now()
	grace := time.Duration(cfg.GraceDays) * 24 * time.Hour

	// Watch-session counts per scene (implicit engagement signal).
	type histAgg struct {
		SceneID  uint
		Sessions int
	}
	var hist []histAgg
	db.Table("histories").Select("scene_id, count(*) as sessions").Group("scene_id").Scan(&hist)
	sessions := make(map[uint]int, len(hist))
	for _, h := range hist {
		sessions[h.SceneID] = h.Sessions
	}

	// Load every scene that is either a candidate (available) or carries taste
	// signal (watched / favourite / rated), with cast, tags and files.
	var scenes []models.Scene
	db.Preload("Cast").Preload("Tags").Preload("Files").
		Where("is_available = ? OR is_watched = ? OR favourite = ? OR star_rating > 0", true, true, true).
		Find(&scenes)

	// Global tag frequency (for IDF damping of ubiquitous tags).
	tagGlobal := make(map[string]int)
	for i := range scenes {
		for _, t := range scenes[i].Tags {
			tagGlobal[t.Name]++
		}
	}

	// Build the taste profile from per-scene affinity.
	actorW := make(map[uint]float64)
	tagW := make(map[string]float64)
	siteW := make(map[string]float64)
	for i := range scenes {
		s := &scenes[i]
		a := affinity(s, sessions[s.ID])
		if a == 0 {
			continue
		}
		for _, c := range s.Cast {
			actorW[c.ID] += a
		}
		for _, t := range s.Tags {
			tagW[t.Name] += a
		}
		if s.Site != "" {
			siteW[s.Site] += a
		}
	}
	// IDF damping: a generic tag present on thousands of scenes is uninformative.
	for name, w := range tagW {
		tagW[name] = w / math.Log(1+float64(tagGlobal[name]))
	}
	normalizeUint(actorW)
	normalizeStr(tagW)
	normalizeStr(siteW)

	// Largest file group, for size normalisation in the delete score.
	var maxSize int64 = 1
	for i := range scenes {
		if scenes[i].TotalFileSize > maxSize {
			maxSize = scenes[i].TotalFileSize
		}
	}

	// Optionally learn the taste function from feedback instead of the hand-tuned
	// profile weights. Falls back to the heuristic profile if there isn't enough
	// labelled signal yet (cold start).
	var model tasteScorer
	if cfg.UseLearnedModel {
		model = trainModel(scenes, sessions, now, cfg.ModelType, cfg.UseVisualEmbeddings)
	}
	tasteOf := func(s *models.Scene) float64 {
		if model != nil {
			return model.taste(s, now)
		}
		return tasteMatch(s, actorW, tagW, siteW, cfg)
	}

	// Score candidates.
	watchScores := make(map[uint]float64)
	deleteScores := make(map[uint]float64)
	sceneCast := make(map[uint][]uint) // for diversity-aware watch selection
	for i := range scenes {
		s := &scenes[i]
		if !s.IsAvailable {
			continue
		}
		taste := tasteOf(s)
		if watchEligible(s, cfg, now) {
			if sc := watchScore(s, taste, cfg, now); sc > 0 {
				watchScores[s.ID] = sc * dailyJitter(s.ID, "w", now, cfg.NoiseWeight)
				ids := make([]uint, 0, len(s.Cast))
				for _, c := range s.Cast {
					ids = append(ids, c.ID)
				}
				sceneCast[s.ID] = ids
			}
		}
		if deleteEligible(s, cfg, now, grace) {
			if sc := deleteScore(s, taste, sessions[s.ID], cfg, now, maxSize); sc > 0 {
				deleteScores[s.ID] = sc * dailyJitter(s.ID, "d", now, cfg.NoiseWeight)
			}
		}
	}

	// Pick the watch list with a per-actor diversity penalty so it isn't dominated
	// by a handful of performers; a scene recommended to watch is never also a
	// delete candidate (watch wins).
	topWatch := selectDiverse(watchScores, sceneCast, cfg.WatchListSize, cfg.DiversityDecay)
	for id := range topWatch {
		delete(deleteScores, id)
	}
	topDelete := topN(deleteScores, cfg.DeleteListSize)

	// Persist the base scores immediately so the lists populate right away.
	db.Model(&models.Scene{}).
		Where("rec_watch_score > 0 OR rec_delete_score > 0").
		Updates(map[string]interface{}{"rec_watch_score": 0, "rec_delete_score": 0})
	for id, sc := range topWatch {
		db.Model(&models.Scene{}).Where("id = ?", id).
			Updates(map[string]interface{}{"rec_watch_score": sc, "rec_scored_at": now})
	}
	for id, sc := range topDelete {
		db.Model(&models.Scene{}).Where("id = ?", id).
			Updates(map[string]interface{}{"rec_delete_score": sc, "rec_scored_at": now})
	}
	common.Log.Infof("recommend: scored %d scenes -> %d to watch, %d to clean up",
		len(scenes), len(topWatch), len(topDelete))

	// Second pass: measure no-reference visual quality of the selected scenes' files
	// and re-rank as results land (crisp scenes rise in "For You", soft ones rise in
	// "Cleanup"). Lists are already live; this refines their order. Quality is measured
	// once per file and cached.
	if cfg.WVisualQuality > 0 && cfg.VQMaxSamples > 0 {
		applyVisualQuality(db, scenes, topWatch, topDelete, cfg)
	}

	// Compute visual embeddings for any available files that lack them (one-time per
	// file, cached). These feed the learned model as features on the NEXT recompute.
	if cfg.UseVisualEmbeddings {
		embedScenes(db, scenes)
	}
}

// affinity expresses how much the user liked a scene they have engaged with,
// in [-1, 1]. Scenes with zero affinity do not shape the taste profile.
func affinity(s *models.Scene, sessions int) float64 {
	a := 0.0
	if s.Favourite {
		a += 1.0
	}
	if s.StarRating > 0 {
		a += (s.StarRating - 3) / 2 // 1..5 -> -1..+1
	}
	a += engagement(s, sessions)
	return clamp(a, -1, 1)
}

func engagement(s *models.Scene, sessions int) float64 {
	eff := sessions
	if eff == 0 && s.IsWatched {
		eff = 1
	}
	if eff == 0 {
		return 0
	}
	completion := 0.0
	if s.Duration > 0 {
		completion = math.Min(1, float64(s.TotalWatchTime)/float64(s.Duration))
	}
	if completion < 0.10 && eff <= 1 {
		return -0.3 // sampled once and abandoned
	}
	base := 0.4*completion + 0.3*math.Min(1, float64(eff-1)) // rewatching is a strong like
	return math.Min(0.8, base)
}

// tasteMatch combines the actor/tag/site profile weights for a scene.
func tasteMatch(s *models.Scene, actorW map[uint]float64, tagW, siteW map[string]float64, cfg recConfig) float64 {
	actorScore := 0.0
	if len(s.Cast) > 0 {
		sum := 0.0
		for _, c := range s.Cast {
			sum += actorW[c.ID]
		}
		actorScore = sum / float64(len(s.Cast))
	}
	tagScore := 0.0
	if len(s.Tags) > 0 {
		sum := 0.0
		for _, t := range s.Tags {
			sum += tagW[t.Name]
		}
		tagScore = sum / math.Sqrt(float64(len(s.Tags)))
	}
	siteScore := siteW[s.Site]
	return cfg.WActor*actorScore + cfg.WTag*tagScore + cfg.WSite*siteScore
}

// recentWatchWindow is how long a scene stays "recently watched" and is kept out of
// the For You list when ExcludeRecentlyWatched is enabled.
const recentWatchWindow = 30 * 24 * time.Hour

func watchEligible(s *models.Scene, cfg recConfig, now time.Time) bool {
	if s.Favourite {
		return false // already known-liked, no need to recommend
	}
	if cfg.ExcludeRecentlyWatched && s.IsWatched && !s.LastOpened.IsZero() &&
		now.Sub(s.LastOpened) < recentWatchWindow {
		return false // just watched it; don't resurface yet
	}
	return true
}

func watchScore(s *models.Scene, taste float64, cfg recConfig, now time.Time) float64 {
	return taste + cfg.WQuality*qualityBoost(s) + cfg.WFreshness*freshness(s, now)
}

func deleteEligible(s *models.Scene, cfg recConfig, now time.Time, grace time.Duration) bool {
	if s.Favourite || s.Watchlist || s.Wishlist {
		return false
	}
	if cfg.ProtectRating > 0 && s.StarRating >= cfg.ProtectRating {
		return false
	}
	// Recently rated (any value) in the last 3 months -> you're actively engaging
	// with it, so don't suggest deleting it yet.
	if s.StarRating > 0 && !s.StarRatingUpdatedAt.IsZero() && now.Sub(s.StarRatingUpdatedAt) < 90*24*time.Hour {
		return false
	}
	if !s.AddedDate.IsZero() && now.Sub(s.AddedDate) < grace {
		return false // too recently added to judge
	}
	return true
}

func deleteScore(s *models.Scene, taste float64, sessions int, cfg recConfig, now time.Time, maxSize int64) float64 {
	unloved := 0.0
	if s.IsWatched || sessions > 0 {
		completion := 0.0
		if s.Duration > 0 {
			completion = math.Min(1, float64(s.TotalWatchTime)/float64(s.Duration))
		}
		if completion < 0.5 {
			unloved += 0.5 * (1 - completion) // watched but never finished
		}
		if !s.LastOpened.IsZero() {
			unloved += 0.3 * clamp(now.Sub(s.LastOpened).Hours()/24/365, 0, 1) // not opened in ages
		}
	} else {
		if !s.AddedDate.IsZero() {
			unloved += 0.6 * clamp(now.Sub(s.AddedDate).Hours()/24/365, 0, 1) // had it for ages, never watched
		} else {
			unloved += 0.3
		}
	}
	if taste < 0 {
		unloved += math.Min(0.5, -taste) // cast/tags/site you don't favour
	}
	if s.StarRating > 0 && s.StarRating <= 2 {
		unloved += 0.8 // you rated it poorly
	}
	if unloved <= 0 {
		return 0
	}
	sizeNorm := float64(s.TotalFileSize) / float64(maxSize)
	return unloved * (1 + cfg.WSize*sizeNorm)
}

// qualityBoost rewards higher resolution (max across the scene's files).
func qualityBoost(s *models.Scene) float64 {
	maxH := 0
	for _, f := range s.Files {
		if f.VideoHeight > maxH {
			maxH = f.VideoHeight
		}
	}
	if maxH <= 0 {
		return 0
	}
	return clamp((float64(maxH)-1920)/(4096-1920), 0, 1)
}

// freshness decays with age (newer acquisitions/releases score higher).
func freshness(s *models.Scene, now time.Time) float64 {
	ref := s.AddedDate
	if ref.IsZero() {
		ref = s.ReleaseDate
	}
	if ref.IsZero() {
		return 0
	}
	ageDays := now.Sub(ref).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Exp(-ageDays / 365) // ~0.37 at one year
}

// --- helpers ---

type recConfig struct {
	Enabled                bool
	UseLearnedModel        bool
	ModelType              string
	UseVisualEmbeddings    bool
	WatchListSize          int
	DeleteListSize         int
	ProtectRating          float64
	GraceDays              int
	ExcludeRecentlyWatched bool
	DiversityDecay         float64
	WActor                 float64
	WTag                   float64
	WSite                  float64
	WQuality               float64
	WFreshness             float64
	WSize                  float64
	WVisualQuality         float64
	VQMaxSamples           int
	NoiseWeight            float64
}

func loadConfig() recConfig {
	c := config.Config.Recommendation
	return recConfig{
		Enabled:                c.Enabled,
		UseLearnedModel:        c.UseLearnedModel,
		ModelType:              c.ModelType,
		UseVisualEmbeddings:    c.UseVisualEmbeddings,
		WatchListSize:          c.WatchListSize,
		DeleteListSize:         c.DeleteListSize,
		ProtectRating:          c.ProtectRating,
		GraceDays:              c.GraceDays,
		ExcludeRecentlyWatched: c.ExcludeRecentlyWatched,
		DiversityDecay:         c.DiversityDecay,
		WActor:                 c.WActor,
		WTag:                   c.WTag,
		WSite:                  c.WSite,
		WQuality:               c.WQuality,
		WFreshness:             c.WFreshness,
		WSize:                  c.WSize,
		WVisualQuality:         c.WVisualQuality,
		VQMaxSamples:           c.VQMaxSamples,
		NoiseWeight:            c.NoiseWeight,
	}
}

// dailyJitter returns a multiplier in [1-weight, 1+weight] that is stable for a given
// scene within a calendar day but changes each day, so the lists rotate daily. A
// separate salt keeps the watch and cleanup jitter independent.
func dailyJitter(id uint, salt string, now time.Time, weight float64) float64 {
	if weight <= 0 {
		return 1
	}
	h := fnv.New64a()
	h.Write([]byte(now.Format("2006-01-02") + salt + strconv.FormatUint(uint64(id), 10)))
	frac := float64(h.Sum64()%100000) / 100000 // [0,1)
	f := 1 + weight*(frac*2-1)
	if f < 0.05 {
		f = 0.05
	}
	return f
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func normalizeUint(m map[uint]float64) {
	max := 0.0
	for _, v := range m {
		if a := math.Abs(v); a > max {
			max = a
		}
	}
	if max == 0 {
		return
	}
	for k := range m {
		m[k] /= max
	}
}

func normalizeStr(m map[string]float64) {
	max := 0.0
	for _, v := range m {
		if a := math.Abs(v); a > max {
			max = a
		}
	}
	if max == 0 {
		return
	}
	for k := range m {
		m[k] /= max
	}
}

// topN returns the n highest-scoring entries.
func topN(scores map[uint]float64, n int) map[uint]float64 {
	if n <= 0 || len(scores) <= n {
		return scores
	}
	type kv struct {
		id uint
		sc float64
	}
	arr := make([]kv, 0, len(scores))
	for id, sc := range scores {
		arr = append(arr, kv{id, sc})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].sc != arr[j].sc {
			return arr[i].sc > arr[j].sc
		}
		return arr[i].id < arr[j].id
	})
	out := make(map[uint]float64, n)
	for i := 0; i < n; i++ {
		out[arr[i].id] = arr[i].sc
	}
	return out
}

// selectDiverse greedily picks up to n scenes, penalising those whose cast is
// already over-represented (penalty = decay^maxActorCount). It returns the chosen
// scenes mapped to their *adjusted* score, so the surfaced playlist order reflects
// the diversified ranking. A decay of 1 (or <= 0) disables the penalty.
func selectDiverse(scores map[uint]float64, cast map[uint][]uint, n int, decay float64) map[uint]float64 {
	if decay <= 0 || decay >= 1 {
		return topN(scores, n)
	}
	type cand struct {
		id  uint
		raw float64
	}
	cands := make([]cand, 0, len(scores))
	for id, sc := range scores {
		cands = append(cands, cand{id, sc})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].raw != cands[j].raw {
			return cands[i].raw > cands[j].raw
		}
		return cands[i].id < cands[j].id
	})
	if n <= 0 || n > len(cands) {
		n = len(cands)
	}

	actorCount := make(map[uint]int)
	used := make([]bool, len(cands))
	chosen := make(map[uint]float64, n)
	for len(chosen) < n {
		bestIdx := -1
		bestAdj := -1.0
		for i := range cands {
			if used[i] {
				continue
			}
			maxCount := 0
			for _, a := range cast[cands[i].id] {
				if actorCount[a] > maxCount {
					maxCount = actorCount[a]
				}
			}
			adj := cands[i].raw * math.Pow(decay, float64(maxCount))
			if adj > bestAdj {
				bestAdj = adj
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		used[bestIdx] = true
		chosen[cands[bestIdx].id] = bestAdj
		for _, a := range cast[cands[bestIdx].id] {
			actorCount[a]++
		}
	}
	return chosen
}
