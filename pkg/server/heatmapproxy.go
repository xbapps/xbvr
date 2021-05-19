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
	"github.com/xbapps/xbvr/pkg/models"
	"willnorris.com/go/imageproxy"
)

const thumbnailWidth = 700
const thumbnailHeight = 420
const heatmapHeight = 10
const heatmapMargin = 3

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

func getScriptFileId(urlpart string) (uint, error) {
	sceneId, err := strconv.Atoi(urlpart)
	if err != nil {
		return 0, err
	}

	var scene models.Scene
	err = scene.GetIfExistByPK(uint(sceneId))
	if err != nil {
		return 0, err
	}
	scriptfiles, err := scene.GetScriptFiles()
	if err != nil || len(scriptfiles) < 1 {
		return 0, fmt.Errorf("scene %d has no script files", sceneId)
	}
	return scriptfiles[0].ID, nil
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

func createHeatmapThumbnail(out *bytes.Buffer, r io.Reader, heatmapImage image.Image) error {
	thumbnailImage, err := jpeg.Decode(r)

	if err != nil {
		return err
	}

	rect := thumbnailImage.Bounds()
	if rect.Dx() != thumbnailWidth || rect.Dy() != thumbnailHeight-heatmapHeight-heatmapMargin {
		thumbnailImage = imaging.Fill(thumbnailImage, thumbnailWidth, thumbnailHeight-heatmapHeight-heatmapMargin, imaging.Center, imaging.Linear)
	}
	heatmapImage = imaging.Resize(heatmapImage, thumbnailWidth, heatmapHeight, imaging.Linear)

	canvas := image.NewNRGBA(image.Rect(0, 0, thumbnailWidth, thumbnailHeight))

	drawRect := image.Rect(0, 0, thumbnailWidth, thumbnailHeight-heatmapHeight-heatmapMargin)
	draw.Draw(canvas, drawRect, thumbnailImage, image.Point{}, draw.Over)
	drawRect = image.Rect(0, thumbnailHeight-heatmapHeight, thumbnailWidth, thumbnailHeight)
	draw.Draw(canvas, drawRect, heatmapImage, image.Point{}, draw.Over)
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
	fileId, err := getScriptFileId(parts[1])
	if err != nil {
		p.serveImageproxyResponse(w, r, imageURL)
		return
	}

	cacheKey := fmt.Sprintf("%d:%s", fileId, imageURL)
	cachedContent, ok := p.Cache.Get(cacheKey)
	if ok {
		w.Header().Add("Content-Type", "image/jpeg")
		w.Header().Add("Content-Length", fmt.Sprint(len(cachedContent)))
		if _, err := io.Copy(w, bytes.NewReader(cachedContent)); err != nil {
			log.Printf("Failed to send out response: %v", err)
		}
		return
	}

	heatmapImage, err := getHeatmapImageForScene(fileId)
	if err != nil {
		p.serveImageproxyResponse(w, r, imageURL)
		return
	}

	proxyURL := fmt.Sprintf("/%dx%d,jpeg/%s", thumbnailWidth, thumbnailHeight-heatmapHeight-heatmapMargin, imageURL)
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
		err = createHeatmapThumbnail(&output, bytes.NewReader(respbody), heatmapImage)
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
