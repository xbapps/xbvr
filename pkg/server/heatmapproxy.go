package server

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
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
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"willnorris.com/go/imageproxy"
)

const thumbnailWidth = 700
const thumbnailHeight = 420
const heatmapHeight = 10
const heatmapMargin = 3
const maximumHeatmaps = 20 // maximumHeatmaps*(heatmapHeight+heatmapMargin) needs to be lower than thumbnailHeight

type BufferResponseWriter struct {
	header     http.Header
	statusCode int
	buf        *bytes.Buffer
}

func (myrw *BufferResponseWriter) Write(p []byte) (int, error) {
	return myrw.buf.Write(p)
}

func (w *BufferResponseWriter) Header() http.Header {
	return w.header
}

func (w *BufferResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

type HeatmapThumbnailProxy struct {
	ImageProxy *imageproxy.Proxy
	Cache      imageproxy.Cache
}

func NewHeatmapThumbnailProxy(imageproxy *imageproxy.Proxy, cache imageproxy.Cache) *HeatmapThumbnailProxy {
	proxy := &HeatmapThumbnailProxy{
		ImageProxy: imageproxy,
		Cache:      cache,
	}
	return proxy
}

func getScriptFileIds(urlpart string) ([]uint, error) {
	sceneId, err := strconv.Atoi(urlpart)
	ids := make([]uint, 0)
	if err != nil {
		return ids, err
	}

	var scene models.Scene
	err = scene.GetIfExistByPK(uint(sceneId))
	if err != nil {
		return ids, err
	}

	scriptfiles, err := scene.GetScriptFilesSorted(config.Config.Interfaces.Players.ScriptSortSeq)
	if err != nil || len(scriptfiles) < 1 {
		return ids, fmt.Errorf("scene %d has no script files", sceneId)
	}

	for i := range scriptfiles {
		ids = append(ids, scriptfiles[i].ID)
	}
	return ids, nil
}

func getHeatmapImageForScene(fileId uint) (image.Image, error) {

	heatmapFilename := filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%d.png", fileId))
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

func createHeatmapThumbnail(out *bytes.Buffer, r io.Reader, heatmapImages []image.Image) error {
	thumbnailImage, err := jpeg.Decode(r)

	if err != nil {
		return err
	}

	heatmapsHeight := len(heatmapImages) * (heatmapHeight + heatmapMargin)
	rect := thumbnailImage.Bounds()
	if rect.Dx() != thumbnailWidth || rect.Dy() != thumbnailHeight-heatmapsHeight {
		thumbnailImage = imaging.Fill(thumbnailImage, thumbnailWidth, thumbnailHeight-heatmapsHeight, imaging.Center, imaging.Linear)
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, thumbnailWidth, thumbnailHeight))

	drawRect := image.Rect(0, 0, thumbnailWidth, thumbnailHeight-heatmapsHeight)
	draw.Draw(canvas, drawRect, thumbnailImage, image.Point{}, draw.Over)

	for i := range heatmapImages {
		heatmapImage := imaging.Resize(heatmapImages[i], thumbnailWidth, heatmapHeight, imaging.Linear)

		drawRect = image.Rect(0, thumbnailHeight-heatmapsHeight+heatmapMargin+i*(heatmapHeight+heatmapMargin), thumbnailWidth, thumbnailHeight)
		draw.Draw(canvas, drawRect, heatmapImage, image.Point{}, draw.Over)
	}

	jpeg.Encode(out, canvas, &jpeg.Options{Quality: 90})
	return nil
}

func (p *HeatmapThumbnailProxy) serveImageproxyResponse(w http.ResponseWriter, r *http.Request, imageURL string) {
	proxyURL := "/700x/" + imageURL
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = proxyURL
	p.ImageProxy.ServeHTTP(w, r2)
}

func (p *HeatmapThumbnailProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	parts := strings.SplitN(r.URL.Path, "/", 3)
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}

	imageURL := parts[2]
	fileIds, err := getScriptFileIds(parts[1])
	if err != nil {
		p.serveImageproxyResponse(w, r, imageURL)
		return
	}

	cacheKey := fmt.Sprintf("%d:%s", fileIds[0], imageURL)
	cachedContent, ok := p.Cache.Get(cacheKey)
	if ok {
		w.Header().Add("Content-Type", "image/jpeg")
		w.Header().Add("Content-Length", fmt.Sprint(len(cachedContent)))
		if _, err := io.Copy(w, bytes.NewReader(cachedContent)); err != nil {
			log.Printf("Failed to send out response: %v", err)
		}
		return
	}

	heatmapImages := make([]image.Image, 0)

	for i := range fileIds {
		heatmapImage, err := getHeatmapImageForScene(fileIds[i])
		if err == nil {
			heatmapImages = append(heatmapImages, heatmapImage)
			if len(heatmapImages) == maximumHeatmaps {
				break
			}
		}
	}

	if len(heatmapImages) == 0 {
		p.serveImageproxyResponse(w, r, imageURL)
		return
	}

	heatmapsHeight := len(heatmapImages) * (heatmapHeight + heatmapMargin)
	proxyURL := fmt.Sprintf("/%dx%d,jpeg/%s", thumbnailWidth, thumbnailHeight-heatmapsHeight, imageURL)
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = proxyURL
	imageproxyResponseWriter := &BufferResponseWriter{
		header: http.Header{},
		buf:    &bytes.Buffer{},
	}
	p.ImageProxy.ServeHTTP(imageproxyResponseWriter, r2)

	respbody, err := ioutil.ReadAll(imageproxyResponseWriter.buf)
	if err == nil {
		var output bytes.Buffer
		err = createHeatmapThumbnail(&output, bytes.NewReader(respbody), heatmapImages)
		if err == nil {
			p.Cache.Set(cacheKey, output.Bytes())
			w.Header().Add("Content-Type", "image/jpeg")
			w.Header().Add("Content-Length", fmt.Sprint(len(output.Bytes())))
			if _, err := io.Copy(w, bytes.NewReader(output.Bytes())); err != nil {
				log.Printf("Failed to send out response: %v", err)
			}
			return
		}
	}
	if err != nil {
		log.Printf("%v", err)
		// serve original response
		if _, err := io.Copy(w, bytes.NewReader(respbody)); err != nil {
			log.Printf("Failed to send out response: %v", err)
		}
	}
}
