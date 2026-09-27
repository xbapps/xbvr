package recommend

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	ort "github.com/yalue/onnxruntime_go"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// Visual embeddings: a small pretrained CNN (MobileNetV2, ONNX) turns one frame of
// each scene into a 1000-d descriptor of its visual content/style. Embeddings are
// computed once per file (cached on the files table) and fed to the learned ranker as
// dense features, so it can learn your visual taste, not just metadata. Inference is
// ~30ms on CPU; the cost is the one-time frame decode per file.

const (
	embedDim         = 1000
	embedSize        = 224
	embedConcurrency = 3
)

var (
	imgMean = [3]float32{0.485, 0.456, 0.406}
	imgStd  = [3]float32{0.229, 0.224, 0.225}

	embedOnce    sync.Once
	embedReady   bool
	embedSession *ort.DynamicAdvancedSession
	embedInName  string
	embedOutName string
	embedMu      sync.Mutex // ORT session Run is not concurrency-safe
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func embedLib() string   { return envOr("XBVR_ONNX_LIB", "/usr/lib/libonnxruntime.so") }
func embedModel() string { return envOr("XBVR_ONNX_MODEL", "/usr/share/xbvr/mobilenet.onnx") }

// embedInit loads the ONNX runtime and model once. Visual embeddings silently
// disable themselves if the shared library or model file is not present.
func embedInit() bool {
	embedOnce.Do(func() {
		if _, err := os.Stat(embedLib()); err != nil {
			common.Log.Warnf("recommend: onnx runtime %s missing, visual embeddings disabled", embedLib())
			return
		}
		if _, err := os.Stat(embedModel()); err != nil {
			common.Log.Warnf("recommend: onnx model %s missing, visual embeddings disabled", embedModel())
			return
		}
		ort.SetSharedLibraryPath(embedLib())
		if err := ort.InitializeEnvironment(); err != nil {
			common.Log.Errorf("recommend: onnx init failed: %v", err)
			return
		}
		ins, outs, err := ort.GetInputOutputInfo(embedModel())
		if err != nil || len(ins) == 0 || len(outs) == 0 {
			common.Log.Errorf("recommend: onnx model inspect failed: %v", err)
			return
		}
		embedInName, embedOutName = ins[0].Name, outs[0].Name
		sess, err := ort.NewDynamicAdvancedSession(embedModel(),
			[]string{embedInName}, []string{embedOutName}, nil)
		if err != nil {
			common.Log.Errorf("recommend: onnx session failed: %v", err)
			return
		}
		embedSession = sess
		embedReady = true
		common.Log.Infof("recommend: visual embeddings enabled (%s)", embedModel())
	})
	return embedReady
}

// embedFile extracts one representative frame and returns its L2-normalised embedding.
func embedFile(path string, duration float64) ([]float32, bool) {
	if _, err := os.Stat(path); err != nil {
		return nil, false
	}
	ts := 60
	if duration > 30 {
		ts = int(duration / 3) // a representative frame, a third of the way in
	}
	cmd := exec.Command(ffmpegBin(), "-hide_banner", "-loglevel", "error",
		"-ss", strconv.Itoa(ts), "-i", path, "-frames:v", "1",
		"-vf", "scale=224:224", "-pix_fmt", "rgb24", "-f", "rawvideo", "-")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, false
	}
	raw := buf.Bytes()
	if len(raw) < embedSize*embedSize*3 {
		return nil, false
	}
	// HWC uint8 -> normalised NCHW float32
	data := make([]float32, 3*embedSize*embedSize)
	for y := 0; y < embedSize; y++ {
		for x := 0; x < embedSize; x++ {
			for c := 0; c < 3; c++ {
				v := float32(raw[(y*embedSize+x)*3+c]) / 255.0
				data[c*embedSize*embedSize+y*embedSize+x] = (v - imgMean[c]) / imgStd[c]
			}
		}
	}
	inT, err := ort.NewTensor(ort.NewShape(1, 3, embedSize, embedSize), data)
	if err != nil {
		return nil, false
	}
	defer inT.Destroy()
	outT, err := ort.NewEmptyTensor[float32](ort.NewShape(1, embedDim))
	if err != nil {
		return nil, false
	}
	defer outT.Destroy()

	embedMu.Lock()
	err = embedSession.Run([]ort.Value{inT}, []ort.Value{outT})
	embedMu.Unlock()
	if err != nil {
		return nil, false
	}

	emb := make([]float32, embedDim)
	copy(emb, outT.GetData())
	l2normalize(emb)
	return emb, true
}

// embedScenes embeds every available scene's video files that lack an embedding,
// persisting incrementally. The first run over a fresh library is slow (~1 frame
// decode per file); afterwards it is a no-op except for newly added files.
func embedScenes(db *gorm.DB, scenes []models.Scene) {
	if !embedInit() {
		return
	}
	var jobs []*models.File
	for i := range scenes {
		if !scenes[i].IsAvailable {
			continue
		}
		for fi := range scenes[i].Files {
			f := &scenes[i].Files[fi]
			if f.Type == "video" && len(f.VisualEmbedding) == 0 {
				jobs = append(jobs, f)
			}
		}
	}
	if len(jobs) == 0 {
		return
	}
	common.Log.Infof("recommend: embedding %d files (one-time per file)…", len(jobs))

	in := make(chan *models.File)
	type res struct {
		id  uint
		vec []float32
	}
	out := make(chan res)
	var wg sync.WaitGroup
	for w := 0; w < embedConcurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range in {
				if vec, ok := embedFile(f.GetPath(), f.VideoDuration); ok {
					out <- res{f.ID, vec}
				} else {
					out <- res{f.ID, nil}
				}
			}
		}()
	}
	go func() {
		for _, f := range jobs {
			in <- f
		}
		close(in)
		wg.Wait()
		close(out)
	}()

	done := 0
	for r := range out {
		now := time.Now()
		blob := encodeFloats(r.vec) // empty blob for failures so we don't retry forever
		db.Model(&models.File{}).Where("id = ?", r.id).Updates(map[string]interface{}{
			"visual_embedding":    blob,
			"visual_embedding_at": now,
		})
		done++
	}
	common.Log.Infof("recommend: embedded %d files", done)
}

func l2normalize(v []float32) {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	if s == 0 {
		return
	}
	n := float32(1 / math.Sqrt(s))
	for i := range v {
		v[i] *= n
	}
}

func encodeFloats(v []float32) []byte {
	if len(v) == 0 {
		return []byte{}
	}
	buf := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(x))
	}
	return buf
}

func decodeFloats(b []byte) []float32 {
	n := len(b) / 4
	if n == 0 {
		return nil
	}
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}
