package server

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/imgbadge"
	"github.com/xbapps/xbvr/pkg/models"
	"willnorris.com/go/imageproxy"
)

const thumbnailWidth = 700
const thumbnailHeight = 420
const heatmapHeight = 10
const heatmapMargin = 3
const maximumHeatmaps = 20 // maximumHeatmaps*(heatmapHeight+heatmapMargin) needs to be lower than thumbnailHeight

type BufferResponseWriter struct {
	header http.Header
	buf    *bytes.Buffer
}

func (myrw *BufferResponseWriter) Write(p []byte) (int, error) {
	return myrw.buf.Write(p)
}

func (w *BufferResponseWriter) Header() http.Header {
	return w.header
}

func (w *BufferResponseWriter) WriteHeader(statusCode int) {}

type HereSphereThumbnailProxy struct {
	ImageProxy *imageproxy.Proxy
	Cache      imageproxy.Cache
}

func NewHereSphereThumbnailProxy(imageproxy *imageproxy.Proxy, cache imageproxy.Cache) *HereSphereThumbnailProxy {
	proxy := &HereSphereThumbnailProxy{
		ImageProxy: imageproxy,
		Cache:      cache,
	}
	return proxy
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

func promotedTagHash(tagNames []string) uint32 {
	h := fnv.New32a()
	for _, name := range tagNames {
		h.Write([]byte(name))
		h.Write([]byte{0})
	}
	return h.Sum32()
}

func fileIDHash(files []models.File) uint32 {
	h := fnv.New32a()
	for _, f := range files {
		h.Write([]byte(strconv.FormatUint(uint64(f.ID), 10)))
		h.Write([]byte{0})
	}
	return h.Sum32()
}

func createHereSphereThumbnail(out *bytes.Buffer, r io.Reader, heatmapImages []image.Image, promotedTagNames []string) error {
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

	if len(promotedTagNames) > 0 {
		if err := imgbadge.DrawPromotedBadges(canvas, promotedTagNames); err != nil {
			return err
		}
	}

	jpeg.Encode(out, canvas, &jpeg.Options{Quality: 90})
	return nil
}

// requestWithPath returns a shallow copy of r whose URL.Path is replaced with
// path, so the imageproxy can be handed a rewritten request without mutating
// the original.
func requestWithPath(r *http.Request, path string) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = path
	return r2
}

func (p *HereSphereThumbnailProxy) serveImageproxyResponse(w http.ResponseWriter, r *http.Request, imageURL string) {
	p.ImageProxy.ServeHTTP(w, requestWithPath(r, "/700x/"+imageURL))
}

func (p *HereSphereThumbnailProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	parts := strings.SplitN(r.URL.Path, "/", 3)
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}

	imageURL := parts[2]
	sceneID, err := strconv.Atoi(parts[1])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var scene models.Scene
	if err := scene.GetIfExistByPK(uint(sceneID)); err != nil {
		p.serveImageproxyResponse(w, r, imageURL)
		return
	}

	promotedTagNames := make([]string, 0)
	for i := range scene.Tags {
		if scene.Tags[i].IsPromoted {
			promotedTagNames = append(promotedTagNames, scene.Tags[i].Name)
		}
	}

	scriptfiles, err := scene.GetScriptFilesSorted(config.Config.Interfaces.Players.ScriptSortSeq)
	if err != nil {
		scriptfiles = nil
	}

	heatmapImages := make([]image.Image, 0)

	for i := range scriptfiles {
		heatmapImage, err := getHeatmapImageForScene(scriptfiles[i].ID)
		if err == nil {
			heatmapImages = append(heatmapImages, heatmapImage)
			if len(heatmapImages) == maximumHeatmaps {
				break
			}
		}
	}

	if len(heatmapImages) == 0 && len(promotedTagNames) == 0 {
		p.serveImageproxyResponse(w, r, imageURL)
		return
	}

	cacheKey := fmt.Sprintf("%d:%x:%x:%s", scene.ID, promotedTagHash(promotedTagNames), fileIDHash(scriptfiles), imageURL)

	loadFromCache := true

	for i := range scriptfiles {
		if scriptfiles[i].RefreshHeatmapCache {
			loadFromCache = false
			break
		}
	}

	if loadFromCache {
		cachedContent, ok := p.Cache.Get(cacheKey)
		if ok {
			w.Header().Add("Content-Type", "image/jpeg")
			w.Header().Add("Content-Length", fmt.Sprint(len(cachedContent)))
			if _, err := io.Copy(w, bytes.NewReader(cachedContent)); err != nil {
				log.Printf("Failed to send out response: %v", err)
			}
			return
		}
	}

	if len(heatmapImages) > 0 {
		for i := range scriptfiles {
			file := scriptfiles[i]
			file.RefreshHeatmapCache = false
			file.Save()
		}
	}

	heatmapsHeight := len(heatmapImages) * (heatmapHeight + heatmapMargin)
	proxyURL := fmt.Sprintf("/%dx%d,jpeg/%s", thumbnailWidth, thumbnailHeight-heatmapsHeight, imageURL)
	imageproxyResponseWriter := &BufferResponseWriter{
		header: http.Header{},
		buf:    &bytes.Buffer{},
	}
	p.ImageProxy.ServeHTTP(imageproxyResponseWriter, requestWithPath(r, proxyURL))

	respbody := imageproxyResponseWriter.buf.Bytes()

	var output bytes.Buffer
	err = createHereSphereThumbnail(&output, bytes.NewReader(respbody), heatmapImages, promotedTagNames)
	if err != nil {
		log.Printf("%v", err)
		if _, err := io.Copy(w, bytes.NewReader(respbody)); err != nil {
			log.Printf("Failed to send out response: %v", err)
		}
		return
	}
	p.Cache.Set(cacheKey, output.Bytes())
	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Add("Content-Length", fmt.Sprint(len(output.Bytes())))
	if _, err := io.Copy(w, bytes.NewReader(output.Bytes())); err != nil {
		log.Printf("Failed to send out response: %v", err)
	}
}
