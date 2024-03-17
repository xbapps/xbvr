package api

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"strconv"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type RequestGenerateThumbnail struct {
	FileID uint   `json:"file_id"`
	crop   string `json:"crop"`
}

type ThumbnailResource struct{}

func (i ThumbnailResource) WebService() *restful.WebService {
	tags := []string{"Thumbs"}

	ws := new(restful.WebService)

	ws.Path("/api/thumbs").
		Consumes("*/*").
		Produces("image/jpeg")

	ws.Route(ws.POST("/generate").To(i.generateThumbnail).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]byte{}))
	
	return ws
}

func (i ThumbnailResource) generateThumbnail(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestGenerateThumbnail
	var file models.File

	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	err = db.Preload("Volume").Where(&models.File{ID: r.FileID}).First(&file).Error
	if err == nil {
		// 画像ファイルからサムネイル連結画像を生成
		resultImage, err := RenderPreview(
			filepath.Join(file.Path, file.Filename),
			"",
			file.VideoProjection,
			config.Config.Library.Preview.StartTime,
			config.Config.Library.Preview.SnippetLength,
			config.Config.Library.Preview.SnippetAmount,
			config.Config.Library.Preview.Resolution,
			config.Config.Library.Preview.ExtraSnippet,
		)
		if err != nil {
			resp.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}

		// 生成した結合した画像をクライアントに送信
		buffer := new(bytes.Buffer)
		if err := jpeg.Encode(buffer, resultImage, nil); err != nil {
			resp.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}

		resp.AddHeader("Content-Type", "image/jpeg")
		resp.AddHeader("Content-Length", strconv.Itoa(len(buffer.Bytes())))
		if _, err := resp.ResponseWriter.Write(buffer.Bytes()); err != nil {
			resp.WriteErrorString(http.StatusInternalServerError, err.Error())
			return
		}
	}
}

func RenderPreview(inputFile string, destFile string, videoProjection string, startTime int, snippetLength float64, snippetAmount int, resolution int, extraSnippet bool)  (*image.RGBA, error) {
	tmpPath := filepath.Join(common.VideoPreviewDir, "tmp")
	os.MkdirAll(tmpPath, os.ModePerm)
	defer os.RemoveAll(tmpPath)

	// Get video duration
	ffdata, err := ffprobe.GetProbeData(inputFile, time.Second*10)
	if err != nil {
		return nil, err
	}
	vs := ffdata.GetFirstVideoStream()
	//dur := ffdata.Format.DurationSeconds

	crop := "iw/2:ih:iw/2:ih" // LR videos
	if vs.Height == vs.Width {
		crop = "iw/2:ih/2:iw/4:ih/2" // TB videos
	}
	if videoProjection == "flat" {
		crop = "iw:ih:iw:ih" // LR videos
	}
	// Mono 360 crop args: (no way of accurately determining)
	// "iw/2:ih:iw/4:ih"
	vfArgs := fmt.Sprintf("select='between(t\\,5\\,60)',fps=1/60,crop=%v,scale=%v:%v", crop, resolution, resolution)

	args := []string{
		"-i", inputFile,
		"-vf", vfArgs,
		"-f", "image2pipe",
		"-c:v", "mjpeg",
		"-",
	}

	cmd := buildCmd(tasks.GetBinPath("ffmpeg"), args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail images: %v", err)
	}
	
	err = convertAndSaveToJPEG(output, "thumbnail.jpg")
	if err != nil {
		fmt.Println("Error:", err)
		return nil,err
	}

	var images []*image.RGBA
	for _, chunk := range bytes.Split(output, []byte("\n")) {
		if len(chunk) == 0 {
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(chunk))
		if err != nil {
			return nil, fmt.Errorf("failed to decode thumbnail image: %v", err)
		}

		rgba := image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
		images = append(images, rgba)
	}

	resultImage := mergeImages(images)

	return resultImage, nil
}

func mergeImages(images []*image.RGBA) *image.RGBA {
	firstImage := images[0]
	totalWidth := firstImage.Bounds().Max.X
	totalHeight := firstImage.Bounds().Max.Y * len(images)

	resultImage := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))
	for i, img := range images {
		draw.Draw(resultImage, image.Rect(0, i*firstImage.Bounds().Max.Y, totalWidth, (i+1)*firstImage.Bounds().Max.Y), img, image.Point{}, draw.Src)
	}

	return resultImage
}

func convertAndSaveToJPEG(imageData []byte, filename string) error {
	// バイト配列を画像にデコード
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return err
	}

	// JPEG形式にエンコードしてファイルに保存
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// JPEG形式でエンコードしてファイルに書き込む
	err = jpeg.Encode(file, img, nil)
	if err != nil {
		return err
	}

	return nil
}