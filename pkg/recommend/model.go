package recommend

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// A self-contained logistic-regression ranker trained on the user's own feedback.
// It replaces the hand-tuned actor/tag/site weights with weights LEARNED from which
// scenes were favourited / rated / actually watched. Pure Go, sparse features, trained
// by SGD in well under a second — no external ML dependency.
//
// The model predicts P(like) for a scene from its content features; the engine turns
// that into a taste value in [-1,1] that feeds the existing watch/delete formulas, so
// the heuristic and learned scorers are interchangeable (config toggle).

const (
	modelEpochs      = 20
	modelLearnRate   = 0.05
	modelL2          = 0.002
	modelMinPositive = 10 // need at least this many liked scenes to train
)

type featureIndex struct {
	idx  map[string]int
	keys []string
}

func newFeatureIndex() *featureIndex {
	return &featureIndex{idx: make(map[string]int)}
}

// get returns the index for a feature key, allocating one when training.
func (fi *featureIndex) get(key string, train bool) (int, bool) {
	if i, ok := fi.idx[key]; ok {
		return i, true
	}
	if !train {
		return 0, false
	}
	i := len(fi.keys)
	fi.idx[key] = i
	fi.keys = append(fi.keys, key)
	return i, true
}

type learnedModel struct {
	fi       *featureIndex
	w        []float64
	useEmbed bool
}

func sigmoid(z float64) float64 {
	if z >= 0 {
		return 1 / (1 + math.Exp(-z))
	}
	e := math.Exp(z)
	return e / (1 + e)
}

// resBand buckets the scene's best vertical resolution.
func resBand(s *models.Scene) string {
	maxH := 0
	for i := range s.Files {
		if s.Files[i].Type == "video" && s.Files[i].VideoHeight > maxH {
			maxH = s.Files[i].VideoHeight
		}
	}
	switch {
	case maxH >= 3800:
		return "8k"
	case maxH >= 2900:
		return "6k"
	case maxH >= 2100:
		return "5k"
	case maxH > 0:
		return "sd"
	default:
		return "unknown"
	}
}

// sceneFeatures builds the sparse feature vector for a scene. When useEmbed is set
// and the scene has a cached visual embedding, its dimensions are added as dense
// features so the learned ranker can pick up visual taste.
func sceneFeatures(s *models.Scene, fi *featureIndex, train bool, now time.Time, useEmbed bool) map[int]float64 {
	f := make(map[int]float64, len(s.Cast)+len(s.Tags)+6)
	add := func(key string, val float64) {
		if i, ok := fi.get(key, train); ok {
			f[i] = val
		}
	}
	add("bias", 1)
	for _, c := range s.Cast {
		add("actor:"+strconv.Itoa(int(c.ID)), 1)
	}
	for _, t := range s.Tags {
		add("tag:"+t.Name, 1)
	}
	if s.Site != "" {
		add("site:"+s.Site, 1)
	}
	add("res:"+resBand(s), 1)
	if s.IsScripted {
		add("scripted", 1)
	}
	add("freshness", freshness(s, now))
	if useEmbed {
		if emb := sceneEmbedding(s); emb != nil {
			for i, v := range emb {
				add("emb:"+strconv.Itoa(i), float64(v))
			}
		}
	}
	return f
}

// sceneEmbedding returns the visual embedding of the scene's first embedded video file.
func sceneEmbedding(s *models.Scene) []float32 {
	for i := range s.Files {
		if s.Files[i].Type == "video" && len(s.Files[i].VisualEmbedding) > 0 {
			return decodeFloats(s.Files[i].VisualEmbedding)
		}
	}
	return nil
}

// labelOf derives a training label from feedback: 1 = liked, 0 = disliked, ok=false
// when the scene carries no clear signal (excluded from training).
func labelOf(s *models.Scene, sessions int, now time.Time) (float64, bool) {
	completion := 0.0
	if s.Duration > 0 {
		completion = float64(s.TotalWatchTime) / float64(s.Duration)
	}
	switch {
	case s.Favourite, s.StarRating >= 4, sessions >= 2, s.IsWatched && completion >= 0.6:
		return 1, true
	case s.StarRating > 0 && s.StarRating <= 2:
		return 0, true
	case s.IsWatched && completion < 0.10 && !s.Favourite:
		return 0, true // sampled then abandoned
	case s.IsAvailable && !s.IsWatched && !s.AddedDate.IsZero() && now.Sub(s.AddedDate) > 60*24*time.Hour:
		return 0, true // owned for ages, never watched
	default:
		return 0, false
	}
}

type trainExample struct {
	feat   map[int]float64
	label  float64
	weight float64
}

// tasteScorer maps a scene to a taste value in [-1, 1] for the scoring formulas.
// Both the linear and factorization-machine models implement it.
type tasteScorer interface {
	taste(s *models.Scene, now time.Time) float64
}

// buildExamples labels the loaded scenes and builds class-balanced training examples.
func buildExamples(scenes []models.Scene, sessions map[uint]int, now time.Time, useEmbed bool) ([]trainExample, *featureIndex, int, int) {
	fi := newFeatureIndex()
	var examples []trainExample
	pos, neg := 0, 0
	for i := range scenes {
		s := &scenes[i]
		label, ok := labelOf(s, sessions[s.ID], now)
		if !ok {
			continue
		}
		examples = append(examples, trainExample{feat: sceneFeatures(s, fi, true, now, useEmbed), label: label})
		if label == 1 {
			pos++
		} else {
			neg++
		}
	}
	if pos == 0 || neg == 0 {
		return examples, fi, pos, neg
	}
	// Balance the classes so the rarer label isn't drowned out.
	total := float64(pos + neg)
	wPos := total / (2 * float64(pos))
	wNeg := total / (2 * float64(neg))
	for i := range examples {
		if examples[i].label == 1 {
			examples[i].weight = wPos
		} else {
			examples[i].weight = wNeg
		}
	}
	return examples, fi, pos, neg
}

// trainModel fits the chosen learner on the user's feedback. Returns nil (falling
// back to the heuristic) when there is not enough labelled signal yet.
func trainModel(scenes []models.Scene, sessions map[uint]int, now time.Time, modelType string, useEmbed bool) tasteScorer {
	examples, fi, pos, neg := buildExamples(scenes, sessions, now, useEmbed)
	if pos < modelMinPositive || neg == 0 {
		common.Log.Infof("recommend: not enough labelled signal to train (pos=%d neg=%d), using heuristic", pos, neg)
		return nil
	}
	if modelType == "fm" {
		return trainFM(examples, fi, pos, neg, useEmbed)
	}
	return trainLinear(examples, fi, pos, neg, useEmbed)
}

// trainLinear fits a logistic regression by SGD.
func trainLinear(examples []trainExample, fi *featureIndex, pos, neg int, useEmbed bool) *learnedModel {
	w := make([]float64, len(fi.keys))
	for epoch := 0; epoch < modelEpochs; epoch++ {
		// Deterministic shuffle (index rotation) keeps it dependency-free and stable.
		for k := range examples {
			ex := examples[(k*7+epoch)%len(examples)]
			z := 0.0
			for i, v := range ex.feat {
				z += w[i] * v
			}
			g := (sigmoid(z) - ex.label) * ex.weight
			for i, v := range ex.feat {
				w[i] -= modelLearnRate * (g*v + modelL2*w[i])
			}
		}
	}
	m := &learnedModel{fi: fi, w: w, useEmbed: useEmbed}
	m.logTopFeatures(pos, neg, "linear")
	return m
}

// predictLike returns P(like) for a scene.
func (m *learnedModel) predictLike(s *models.Scene, now time.Time) float64 {
	feat := sceneFeatures(s, m.fi, false, now, m.useEmbed)
	z := 0.0
	for i, v := range feat {
		z += m.w[i] * v
	}
	return sigmoid(z)
}

// taste maps the like probability to [-1, 1] for the existing scoring formulas.
func (m *learnedModel) taste(s *models.Scene, now time.Time) float64 {
	return (m.predictLike(s, now) - 0.5) * 2
}

// logTopFeatures surfaces what the model learned (most liked actors/tags/sites).
func (m *learnedModel) logTopFeatures(pos, neg int, kind string) {
	type fw struct {
		key string
		w   float64
	}
	arr := make([]fw, 0, len(m.fi.keys))
	for i, k := range m.fi.keys {
		arr = append(arr, fw{k, m.w[i]})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].w > arr[j].w })
	top := []string{}
	for i := 0; i < len(arr) && len(top) < 8; i++ {
		if arr[i].key == "bias" {
			continue
		}
		top = append(top, arr[i].key+" "+strconv.FormatFloat(arr[i].w, 'f', 2, 64))
	}
	common.Log.Infof("recommend: trained %s model on %d liked / %d disliked scenes; top signals: %v", kind, pos, neg, top)
}
