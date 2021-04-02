package server

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

const thumbnailWidth = 700
const thumbnailHeight = 420
const heatmapHeight = 10

type MyResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (myrw *MyResponseWriter) Write(p []byte) (int, error) {
	return myrw.buf.Write(p)
}

func getHeatmapImageForScene(urlpart string) (image.Image, error) {
	sceneId, err := strconv.Atoi(urlpart)
	if err != nil {
		return nil, err
	}

	var scene models.Scene
	err = scene.GetIfExistByPK(uint(sceneId))
	if err != nil {
		return nil, err
	}
	scriptfiles := scene.getScriptFiles()
	if len(scriptfiles) < 1 {
		return nil, fmt.Errorf("scene %d has no script files", sceneId)
	}

	heatmapFilename := filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%d.png", scriptfiles[0].ID))
	heatmapFile, err := os.Open(heatmapFilename)
	if err != nil {
		return nil, err
	}

	heatmapImage, err := png.Decode(heatmapFile)
	heatmapFile.Close()
	if err != nil {
		return nil, err
	}

	return heatmapImage, nil
}

func createHeatmapThumbnail(w http.ResponseWriter, r io.Reader, heatmapImage image.Image) error {
	thumbnailImage, err := png.Decode(r)

	if err != nil {
		return err
	}

	rect := thumbnailImage.Bounds()
	if rect.Dx() != thumbnailWidth || rect.Dy() != thumbnailHeight-heatmapHeight {
		thumbnailImage = imaging.Fill(thumbnailImage, thumbnailWidth, thumbnailHeight-heatmapHeight, imaging.Center, imaging.Linear)
	}
	heatmapImage = imaging.Resize(heatmapImage, thumbnailWidth, heatmapHeight, imaging.Linear)
	drawRect := image.Rect(0, thumbnailHeight-heatmapHeight, thumbnailWidth, thumbnailHeight)

	canvas := image.NewNRGBA(image.Rect(0, 0, thumbnailWidth, thumbnailHeight))

	draw.Draw(canvas, thumbnailImage.Bounds(), thumbnailImage, image.Point{}, draw.Over)
	draw.Draw(canvas, drawRect, heatmapImage, image.Point{}, draw.Over)
	png.Encode(w, canvas)
	return nil
}

func ThumbnailHeatmapHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		parts := strings.SplitN(r.URL.Path, "/", 3)
		if len(parts) != 3 {
			http.NotFound(w, r)
			return
		}

		heatmapImage, err := getHeatmapImageForScene(parts[1])
		if err != nil {
			log.Debug(err)
			// passthrough unaltered imageproxy response
			p := "/700x/" + parts[2]
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			next.ServeHTTP(w, r2)
			return
		}

		p := fmt.Sprintf("/%dx%d,png/%s", thumbnailWidth, thumbnailHeight-heatmapHeight, parts[2])
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = p
		myResponseWriter := &MyResponseWriter{
			ResponseWriter: w,
			buf:            &bytes.Buffer{},
		}
		next.ServeHTTP(myResponseWriter, r2)

		respbody, err := ioutil.ReadAll(myResponseWriter.buf)
		if err == nil {
			err = createHeatmapThumbnail(w, bytes.NewReader(respbody), heatmapImage)
		}
		if err != nil {
			log.Printf("%v", err)
			// serve original response
			if _, err := io.Copy(w, bytes.NewReader(respbody)); err != nil {
				log.Printf("Failed to send out response: %v", err)
			}
		}
	})
}
