package tasks

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// Script is the Funscript container type holding Launch data.
type Script struct {
	// Version of Launchscript
	Version string `json:"version"`
	// Inverted causes up and down movement to be flipped.
	Inverted bool `json:"inverted,omitempty"`
	// Range is the percentage of a full stroke to use.
	Range int `json:"range,omitempty"`
	// Actions are the timed moves.
	Actions []Action `json:"actions"`
}

// Action is a move at a specific time.
type Action struct {
	// At time in milliseconds the action should fire.
	At int64 `json:"at"`
	// Pos is the place in percent to move to.
	Pos int `json:"pos"`

	Slope     float64
	Intensity int64
}

type GradientTable []struct {
	Col colorful.Color
	Pos float64
}

func GenerateHeatmaps() {
	if !models.CheckLock("heatmaps") {
		models.CreateLock("heatmaps")

		db, _ := models.GetDB()
		defer db.Close()

		var scriptfiles []models.File
		db.Model(&models.File{}).Preload("Volume").Where("type = ?", "script").Where("has_heatmap = ?", false).Find(&scriptfiles)

		for _, file := range scriptfiles {
			if file.Exists() {
				log.Infof("Rendering %v", file.Filename)
				destFile := filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%d.png", file.ID))
				err := RenderHeatmap(
					file.GetPath(),
					destFile,
					1000,
					10,
					250,
				)
				if err == nil {
					file.HasHeatmap = true
					file.Save()
				} else {
					log.Warn(err)
				}
			}
		}
	}

	models.RemoveLock("heatmaps")
}

func RenderHeatmap(inputFile string, destFile string, width, height, numSegments int) error {
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return err
	}

	var funscript Script
	json.Unmarshal(data, &funscript)
	sort.SliceStable(funscript.Actions, func(i, j int) bool { return funscript.Actions[i].At < funscript.Actions[j].At })

	funscript.UpdateIntesity()
	gradient := funscript.getGradientTable(numSegments)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		c := gradient.GetInterpolatedColorFor(float64(x) / float64(width))
		draw.Draw(img, image.Rect(x, 0, x+1, height), &image.Uniform{c}, image.Point{}, draw.Src)
	}

	outpng, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("Error storing png: " + err.Error())
	}
	defer outpng.Close()

	png.Encode(outpng, img)
	return nil
}

func (gt GradientTable) GetInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(gt)-1; i++ {
		c1 := gt[i]
		c2 := gt[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}

	// Nothing found? Means we're at (or past) the last gradient keypoint.
	return gt[len(gt)-1].Col
}

func (funscript Script) UpdateIntesity() {

	var t1, t2 int64
	var p1, p2 int
	var slope float64
	for i := range funscript.Actions {
		if i == 0 {
			continue
		}
		t1 = funscript.Actions[i].At
		t2 = funscript.Actions[i-1].At
		p1 = funscript.Actions[i].Pos
		p2 = funscript.Actions[i-1].Pos

		slope = math.Min(math.Max(1/(2*float64(t1-t2)/1000), 0), 20)

		funscript.Actions[i].Slope = slope
		funscript.Actions[i].Intensity = int64(slope * math.Abs((float64)(p1-p2)))
	}
}

func getSegmentColor(intensity float64) colorful.Color {
	colorBlue, _ := colorful.Hex("#1e90ff")   // DodgerBlue
	colorGreen, _ := colorful.Hex("#228b22")  // ForestGreen
	colorYellow, _ := colorful.Hex("#ffd700") // Gold
	colorRed, _ := colorful.Hex("#dc143c")    // Crimson
	colorPurple, _ := colorful.Hex("#800080") // Purple
	colorBlack, _ := colorful.Hex("#0f001e")
	colorWhite, _ := colorful.Hex("#ffffff")

	var stepSize float64 = 60.0
	var f float64
	var c colorful.Color

	if intensity <= 0.001 {
		c = colorWhite
	} else if intensity <= 1*stepSize {
		f = (intensity - 0*stepSize) / stepSize
		c = colorBlue.BlendLab(colorGreen, f)
	} else if intensity <= 2*stepSize {
		f = (intensity - 1*stepSize) / stepSize
		c = colorGreen.BlendLab(colorYellow, f)
	} else if intensity <= 3*stepSize {
		f = (intensity - 2*stepSize) / stepSize
		c = colorYellow.BlendLab(colorRed, f)
	} else if intensity <= 4*stepSize {
		f = (intensity - 3*stepSize) / stepSize
		c = colorRed.BlendRgb(colorPurple, f)
	} else {
		f = (intensity - 4*stepSize) / (5 * stepSize)
		f = math.Min(f, 1.0)
		c = colorPurple.BlendLab(colorBlack, f)
	}
	return c
}

func (funscript Script) getGradientTable(numSegments int) GradientTable {
	segments := make([]struct {
		count     int
		intensity int
	}, numSegments)
	gradient := make(GradientTable, numSegments)

	var maxts int64 = 0
	for _, a := range funscript.Actions {
		if a.At > maxts {
			maxts = a.At
		}
	}

	for _, a := range funscript.Actions {
		segment := int(float64(a.At) / float64(maxts+1) * float64(numSegments))
		segments[segment].count = segments[segment].count + 1
		segments[segment].intensity = segments[segment].intensity + int(a.Intensity)
	}

	for i := 0; i < numSegments; i++ {
		gradient[i].Pos = float64(i) / float64(numSegments-1)
		if segments[i].count > 0 {
			gradient[i].Col = getSegmentColor(float64(segments[i].intensity) / float64(segments[i].count))
		} else {
			gradient[i].Col = getSegmentColor(0.0)
		}
	}

	return gradient
}
