package recommend

import (
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// Factorization Machine: extends the linear model with learned pairwise feature
// interactions via low-rank latent vectors, so it can capture "I like this actor
// especially in this studio" or particular tag combinations that a linear model
// can't. Still pure Go, sparse, CPU-cheap. Prediction:
//
//	y = w0 + Σ wi·xi + 0.5·Σ_f [ (Σ_i vi,f·xi)² − Σ_i (vi,f·xi)² ]
const (
	fmFactors   = 8
	fmEpochs    = 15
	fmLearnRate = 0.02
	fmL2        = 0.002
)

type fmModel struct {
	fi       *featureIndex
	w0       float64
	w        []float64
	v        [][]float64 // len(features) x fmFactors
	useEmbed bool
}

func trainFM(examples []trainExample, fi *featureIndex, pos, neg int, useEmbed bool) *fmModel {
	n := len(fi.keys)
	rng := rand.New(rand.NewSource(1)) // deterministic
	v := make([][]float64, n)
	for i := range v {
		v[i] = make([]float64, fmFactors)
		for f := range v[i] {
			v[i][f] = rng.NormFloat64() * 0.01
		}
	}
	m := &fmModel{fi: fi, w: make([]float64, n), v: v, useEmbed: useEmbed}

	sums := make([]float64, fmFactors)
	for epoch := 0; epoch < fmEpochs; epoch++ {
		for k := range examples {
			ex := examples[(k*7+epoch)%len(examples)]
			p := sigmoid(m.raw(ex.feat, sums))
			g := (p - ex.label) * ex.weight

			m.w0 -= fmLearnRate * (g + fmL2*m.w0)
			for i, x := range ex.feat {
				m.w[i] -= fmLearnRate * (g*x + fmL2*m.w[i])
				vi := m.v[i]
				for f := 0; f < fmFactors; f++ {
					// dY/dv_if = x_i * (sum_f - v_if * x_i)
					grad := x * (sums[f] - vi[f]*x)
					vi[f] -= fmLearnRate * (g*grad + fmL2*vi[f])
				}
			}
		}
	}
	m.logTop(pos, neg)
	return m
}

// raw computes the FM score and leaves the per-factor sums in `sums` (reused buffer),
// which the training step needs for the interaction gradient.
func (m *fmModel) raw(feat map[int]float64, sums []float64) float64 {
	z := m.w0
	for i, x := range feat {
		z += m.w[i] * x
	}
	for f := 0; f < fmFactors; f++ {
		var s, sq float64
		for i, x := range feat {
			vv := m.v[i][f] * x
			s += vv
			sq += vv * vv
		}
		sums[f] = s
		z += 0.5 * (s*s - sq)
	}
	return z
}

func (m *fmModel) taste(s *models.Scene, now time.Time) float64 {
	feat := sceneFeatures(s, m.fi, false, now, m.useEmbed)
	sums := make([]float64, fmFactors)
	return (sigmoid(m.raw(feat, sums)) - 0.5) * 2
}

func (m *fmModel) logTop(pos, neg int) {
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
	common.Log.Infof("recommend: trained fm model (k=%d) on %d liked / %d disliked scenes; top linear signals: %v",
		fmFactors, pos, neg, top)
}
