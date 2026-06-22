package tasks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
)

var (
	previewCmdMu sync.Mutex
	previewCmd   *exec.Cmd
)

// StopPreviewGeneration signals the preview loop to stop and kills the current
// FFmpeg process so the UI can interrupt generation without closing the app.
func StopPreviewGeneration() {
	models.SetStopFlag("previews")

	previewCmdMu.Lock()
	defer previewCmdMu.Unlock()
	if previewCmd != nil && previewCmd.Process != nil {
		if err := previewCmd.Process.Kill(); err != nil {
			log.Warnf("failed to kill ffmpeg process: %v", err)
		}
	}
}

func GeneratePreviews(endTime *time.Time) {
	if !models.CheckLock("previews") {
		models.ClearStopFlag("previews")
		models.CreateLock("previews")
		defer func() {
			models.RemoveLock("previews")
			models.ClearStopFlag("previews")
		}()
		log.Infof("Generating previews")
		db, _ := models.GetDB()
		defer db.Close()

		var scenes []models.Scene
		db.Model(&models.Scene{}).Where("is_available = ?", true).Where("has_video_preview = ?", false).Order("release_date desc").Find(&scenes)

		for _, scene := range scenes {
			if models.CheckStopFlag("previews") {
				log.Infof("Preview generation stopped by user")
				return
			}
			files, _ := scene.GetFiles()
			if len(files) > 0 {
				if endTime != nil && time.Now().After(*endTime) {
					return
				}
				i := 0
				for i < len(files) && files[i].Exists() {
					if files[i].Type == "video" {
						log.Infof("Rendering %v", scene.SceneID)
						destFile := filepath.Join(common.VideoPreviewDir, scene.SceneID+".mp4")
						err := RenderPreview(
							files[i].GetPath(),
							destFile,
							files[i].VideoProjection,
							config.Config.Library.Preview.SnippetLength,
							config.Config.Library.Preview.SnippetAmount,
							config.Config.Library.Preview.Resolution,
							config.Config.Library.Preview.ExtraSnippet,
							config.Config.Library.Preview.UseCUDA,
							config.Config.Library.Preview.Pitch,
						)
						if err == nil {
							scene.HasVideoPreview = true
							scene.Save()
							break
						} else {
							log.Warn(err)
						}
					}
					i++
				}
			}
		}
	}
	log.Infof("Previews generated")
}

func RenderPreview(inputFile string, destFile string, videoProjection string, snippetLength float64, snippetAmount int, resolution int, extraSnippet bool, useCUDA bool, pitch int) error {
	tmpPath := filepath.Join(common.VideoPreviewDir, "tmp", filepath.Base(destFile)+"-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	os.MkdirAll(tmpPath, os.ModePerm)
	defer os.RemoveAll(tmpPath)

	ffdata, err := ffprobe.GetProbeData(inputFile, time.Second*10)
	if err != nil {
		return err
	}
	dur := ffdata.Format.DurationSeconds

	filenameLower := strings.ToLower(filepath.Base(inputFile))
	projectionLower := strings.ToLower(videoProjection)
	isHighBitDepth := isHighBitDepth(ffdata.GetFirstVideoStream())
	isFlat := projectionLower == "flat"

	// Detect projection sub-type.
	isTB := strings.Contains(filenameLower, "360_tb") || projectionLower == "360_tb"
	is180SBS := strings.Contains(filenameLower, "180_sbs") || projectionLower == "180_sbs" ||
		strings.Contains(filenameLower, "180f") || strings.Contains(projectionLower, "180f") ||
		strings.Contains(filenameLower, "f180") || strings.Contains(projectionLower, "f180") ||
		strings.Contains(filenameLower, "vr180") || strings.Contains(projectionLower, "vr180")
	isFisheye := strings.Contains(filenameLower, "fisheye") || strings.Contains(projectionLower, "fisheye") ||
		strings.Contains(filenameLower, "mkx") || strings.Contains(projectionLower, "mkx") ||
		strings.Contains(filenameLower, "rf52") || strings.Contains(projectionLower, "rf52") ||
		strings.Contains(filenameLower, "vrca220") || strings.Contains(projectionLower, "vrca220")

	idFov := 190
	if isFisheye {
		switch {
		case strings.Contains(filenameLower, "mkx200") || strings.Contains(filenameLower, "fisheye200") ||
			strings.Contains(projectionLower, "mkx200") || strings.Contains(projectionLower, "fisheye200"):
			idFov = 200
		case strings.Contains(filenameLower, "fisheye190") || strings.Contains(projectionLower, "fisheye190"):
			idFov = 190
		case strings.Contains(filenameLower, "fisheye180") || strings.Contains(projectionLower, "fisheye180"):
			idFov = 180
		case strings.Contains(filenameLower, "vrca220") || strings.Contains(projectionLower, "vrca220"):
			idFov = 220
		}
	}

	// NVENC settings optimized for small preview size.
	nvencArgs := []string{"-c:v", "h264_nvenc", "-preset", "p4", "-profile:v", "high", "-rc", "vbr", "-cq", "32", "-b:v", "0"}

	// Build the filter graph and hardware acceleration flags according to the
	// pipeline matrix.
	var hwaccelArgs []string
	var vfArgs string

	if isFlat {
		// Path 1: flat video.
		vfArgs = fmt.Sprintf("scale=%v:-2", resolution)
		if useCUDA {
			hwaccelArgs = []string{"-hwaccel", "cuda"}
		}
	} else if useCUDA && !isHighBitDepth && isTB {
		// Path 2: 360 top-bottom VR with GPU Vulkan pipeline.
		// v360_vulkan supports equirect input but not hequirect/fisheye, so only TB
		// uses this path. The filter does not support in_stereo/out_stereo, so we crop
		// one eye before uploading to Vulkan. To keep the CPU<->GPU transfer small, the
		// whole frame is first downscaled on the GPU with scale_cuda.
		hwaccelArgs = []string{"-hwaccel", "cuda", "-init_hw_device", "vulkan=vk:0", "-filter_hw_device", "vk", "-hwaccel_output_format", "cuda"}

		vulkanFilter := fmt.Sprintf("v360_vulkan=input=e:output=flat:pitch=%d:ih_fov=360:iv_fov=180:h_fov=100:v_fov=50", pitch)
		vfArgs = fmt.Sprintf("scale_cuda=3840:-2,hwdownload,format=nv12,crop=iw:ih/2:0:0,format=yuv444p,hwupload,%v,scale_vulkan=%v:%v,hwdownload,format=yuv444p",
			vulkanFilter, resolution, resolution/2)
	} else {
		// Path 3: 10-bit+ VR with CUDA decode and CPU filters.
		// Path 4: VR without CUDA (pure CPU).
		if useCUDA {
			// Without -noaccurate_seek, NVDEC can now decode 10-bit streams reliably.
			// GPU decode removes the 20-30 s CPU decode lag on 8K HEVC.
			hwaccelArgs = []string{"-hwaccel", "cuda"}
		}

		var dewarpFilter string
		switch {
		case isTB:
			dewarpFilter = fmt.Sprintf("v360=equirect:flat:in_stereo=tb:out_stereo=2d:pitch=-%d:h_fov=100:v_fov=50", pitch)
		case is180SBS:
			dewarpFilter = fmt.Sprintf("v360=hequirect:flat:in_stereo=sbs:out_stereo=2d:pitch=-%d:h_fov=100:v_fov=60", pitch)
		case isFisheye:
			dewarpFilter = fmt.Sprintf("v360=fisheye:flat:in_stereo=sbs:out_stereo=2d:pitch=-%d:id_fov=%v:h_fov=100:v_fov=60", pitch, idFov)
		default:
			dewarpFilter = fmt.Sprintf("v360=hequirect:flat:in_stereo=sbs:out_stereo=2d:pitch=-%d:h_fov=100:v_fov=60", pitch)
		}
		if isHighBitDepth {
			// Downsample the already scaled/dewarped image to 8-bit before handing it
			// to h264_nvenc, which cannot accept 10-bit input on the high profile.
			vfArgs = fmt.Sprintf("setpts=PTS-STARTPTS,scale=1600:-2,%v,scale=%v:-2,format=yuv420p", dewarpFilter, resolution)
		} else {
			vfArgs = fmt.Sprintf("setpts=PTS-STARTPTS,scale=1600:-2,%v,scale=%v:-2", dewarpFilter, resolution)
		}
	}

	// Helper to render a single snippet.
	// Ищет быстрым поиском, при ошибке пробует точный поиск,
	// затем чистый CPU, а если файл полностью разбит — создает пустую заглушку.
	renderSnippet := func(startSeconds float64, snippetFile string) error {
		// Внутренний хелпер для сборки аргументов
		buildArgs := func(accurateSeek bool, pureCPU bool) []string {
			// -nostdin критически важен, чтобы ffmpeg не ждал ввода в фоне
			args := []string{"-y", "-nostdin", "-threads", "4"}

			if !pureCPU {
				args = append(args, hwaccelArgs...)
			}

			if accurateSeek {
				args = append(args,
					"-i", inputFile,
					"-ss", fmt.Sprintf("%.3f", startSeconds),
				)
			} else {
				args = append(args,
					"-ss", fmt.Sprintf("%.3f", startSeconds),
					"-noaccurate_seek",
					"-i", inputFile,
				)
			}

			args = append(args, "-vf", vfArgs)

			if useCUDA && !pureCPU {
				args = append(args, nvencArgs...)
			} else {
				args = append(args, "-c:v", "libx264", "-preset", "ultrafast")
			}

			args = append(args,
				"-pix_fmt", "yuv420p",
				"-t", fmt.Sprintf("%v", snippetLength),
				"-an", snippetFile,
			)
			return args
		}

		// Хелпер для запуска ffmpeg с жестким таймаутом
		runWithTimeout := func(args []string, timeout time.Duration) error {
			log.Infof("ffmpeg args: %v", args)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// exec.CommandContext kills the process when ctx expires.
			// Copy SysProcAttr from buildCmd to preserve CREATE_NO_WINDOW on Windows.
			tmpl := buildCmd(GetBinPath("ffmpeg"), args...)
			cmd := exec.CommandContext(ctx, GetBinPath("ffmpeg"), args...)
			cmd.SysProcAttr = tmpl.SysProcAttr

			previewCmdMu.Lock()
			previewCmd = cmd
			previewCmdMu.Unlock()
			defer func() {
				previewCmdMu.Lock()
				previewCmd = nil
				previewCmdMu.Unlock()
			}()

			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			err := cmd.Run()
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("ffmpeg hung and was killed by timeout")
			}
			if err != nil {
				log.Errorf("ffmpeg error: %v", err)
				if stderr.Len() > 0 {
					log.Errorf("ffmpeg stderr: %s", stderr.String())
				}
				return fmt.Errorf("ffmpeg failed: %w (stderr: %s)", err, stderr.String())
			}
			return nil
		}

		// Попытка 1: Fast Seek + GPU (Самый быстрый вариант)
		if err := runWithTimeout(buildArgs(false, false), 10*time.Second); err == nil {
			return nil
		}
		log.Warnf("[Preview] Fast seek failed or hung at %.3f, retrying with Accurate Seek + GPU...", startSeconds)

		// Попытка 2: Accurate Seek + GPU (Точный поиск, если уплыли ключевые кадры)
		if err := runWithTimeout(buildArgs(true, false), 10*time.Second); err == nil {
			return nil
		}
		log.Warnf("[Preview] Accurate Seek + GPU failed or hung at %.3f, retrying on pure CPU...", startSeconds)

		// Попытка 3: Accurate Seek + Pure CPU (обычно спасает, если завис драйвер GPU)
		if err := runWithTimeout(buildArgs(true, true), 10*time.Second); err == nil {
			return nil
		}

		// Попытка 4: Финальный щит (генерация заглушки)
		log.Errorf("[Preview] Critical corruption/hang at %.3f. Generating black frame placeholder.", startSeconds)

		dummyArgs := []string{
			"-y", "-nostdin", "-f", "lavfi",
			"-i", fmt.Sprintf("color=c=black:s=600x400:d=%v:r=60", snippetLength),
			"-pix_fmt", "yuv420p",
			snippetFile,
		}

		// Для лавфи ставим короткий таймаут в 3 секунды, там зависать нечему
		if dummyErr := runWithTimeout(dummyArgs, 3*time.Second); dummyErr != nil {
			return fmt.Errorf("failed to render snippet and failed to create fallback placeholder: %v", dummyErr)
		}

		return nil
	}

	// Prepare snippets in parallel with a max concurrency of 4.
	type snippetJob struct {
		index    int
		startSec float64
		filePath string
	}

	// New timestamp grid: safe window from 60.0s to (dur - 20.0 - 1.2).
	startTime := 60.0
	availableDuration := dur - startTime - 20.0 - 1.2
	if availableDuration < 0 {
		availableDuration = 0
	}
	step := availableDuration / float64(snippetAmount)

	jobs := make([]snippetJob, 0, snippetAmount+1)
	for i := 1; i <= snippetAmount; i++ {
		startSeconds := startTime + float64(i-1)*step
		if startSeconds < 0 {
			startSeconds = 0
		}
		if startSeconds > dur-snippetLength {
			startSeconds = dur - snippetLength
		}
		if startSeconds < 0 {
			startSeconds = 0
		}
		jobs = append(jobs, snippetJob{
			index:    i,
			startSec: startSeconds,
			filePath: filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", i)),
		})
	}

	// Ensure ending is always in preview.
	if extraSnippet && dur/float64(snippetAmount) > float64(150) {
		snippetAmount++
		startSeconds := dur - 150.0
		if startSeconds+snippetLength > dur {
			startSeconds = dur - snippetLength
		}
		if startSeconds < 0 {
			startSeconds = 0
		}
		jobs = append(jobs, snippetJob{
			index:    snippetAmount,
			startSec: startSeconds,
			filePath: filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", snippetAmount)),
		})
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 2)
	var errMu sync.Mutex
	jobErrors := make(map[int]error)

	for _, job := range jobs {
		wg.Add(1)
		go func(job snippetJob) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := renderSnippet(job.startSec, job.filePath); err != nil {
				errMu.Lock()
				jobErrors[job.index] = err
				errMu.Unlock()
				log.Errorf("snippet %v render failed: %v", job.index, err)
				return
			}

			snippetSize := int64(0)
			if fi, statErr := os.Stat(job.filePath); statErr == nil {
				snippetSize = fi.Size()
			}
			if snippetSize < 1000 {
				err := fmt.Errorf("snippet %v is missing or too small (path=%v, size=%v)", job.index, job.filePath, snippetSize)
				errMu.Lock()
				jobErrors[job.index] = err
				errMu.Unlock()
				log.Errorf("%v", err)
				return
			}
		}(job)
	}

	wg.Wait()

	for _, job := range jobs {
		if err := jobErrors[job.index]; err != nil {
			return err
		}
	}

	// Prepare concat file.
	concatFile := filepath.Join(tmpPath, "concat.txt")
	f, err := os.Create(concatFile)
	if err != nil {
		return err
	}
	for i := 1; i <= snippetAmount; i++ {
		snippetPath := filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", i))
		if _, err := os.Stat(snippetPath); os.IsNotExist(err) {
			f.Close()
			return fmt.Errorf("snippet %v does not exist: %w", i, err)
		}
		f.WriteString(fmt.Sprintf("file '%v'\n", filepath.ToSlash(snippetPath)))
	}
	f.Close()

	// Log concat input for debugging.
	concatData, _ := os.ReadFile(concatFile)
	log.Infof("concat input: %s", string(concatData))
	for i := 1; i <= snippetAmount; i++ {
		snippetPath := filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", i))
		if fi, err := os.Stat(snippetPath); err == nil {
			log.Infof("snippet %v size: %v bytes", i, fi.Size())
		} else {
			log.Warnf("snippet %v stat error: %v", i, err)
		}
	}

	// Save result.
	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", filepath.ToSlash(concatFile),
		"-c", "copy",
		filepath.ToSlash(destFile),
	}
	return runFFmpeg(args)
}

func isHighBitDepth(vs *ffprobe.Stream) bool {
	if vs == nil {
		return false
	}
	return strings.Contains(vs.PixFmt, "10le") || strings.Contains(vs.PixFmt, "10be") ||
		strings.Contains(vs.PixFmt, "12le") || strings.Contains(vs.PixFmt, "12be") ||
		strings.Contains(vs.PixFmt, "16le") || strings.Contains(vs.PixFmt, "16be")
}

func runFFmpeg(args []string) error {
	log.Infof("ffmpeg args: %v", args)
	cmd := buildCmd(GetBinPath("ffmpeg"), args...)

	previewCmdMu.Lock()
	previewCmd = cmd
	previewCmdMu.Unlock()
	defer func() {
		previewCmdMu.Lock()
		previewCmd = nil
		previewCmdMu.Unlock()
	}()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Errorf("ffmpeg error: %v", err)
		if stderr.Len() > 0 {
			log.Errorf("ffmpeg stderr: %s", stderr.String())
		}
		return fmt.Errorf("ffmpeg failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
