package imgbadge

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	badgeHeight    = 36
	badgePaddingX  = 15
	badgeGap       = 9
	badgeMargin    = 9
	iconSize       = 15
	iconGap        = 8
	maxNamedBadges = 3
	textSize       = 21.0
)

var (
	fontFace font.Face
	fontOnce sync.Once
	fontErr  error
)

func face() (font.Face, error) {
	fontOnce.Do(func() {
		ttf, err := opentype.Parse(gobold.TTF)
		if err != nil {
			fontErr = err
			return
		}
		fontFace, fontErr = opentype.NewFace(ttf, &opentype.FaceOptions{
			Size:    textSize,
			DPI:     72,
			Hinting: font.HintingFull,
		})
	})
	return fontFace, fontErr
}

func badgeWidth(f font.Face, s string) int {
	return badgePaddingX*2 + iconSize + iconGap + font.MeasureString(f, s).Round()
}

func totalWidth(f font.Face, items []string) int {
	total := 0
	for _, s := range items {
		total += badgeWidth(f, s)
	}
	total += badgeGap * (len(items) - 1)
	return total
}

func truncateName(f font.Face, s string, maxW int) string {
	if badgeWidth(f, s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		cand := string(runes[:len(runes)-1]) + "…"
		if badgeWidth(f, cand) <= maxW {
			return cand
		}
		runes = runes[:len(runes)-1]
	}
	return "…"
}

func buildBadgeItems(tagNames []string, maxWidth int) ([]string, error) {
	f, err := face()
	if err != nil {
		return nil, err
	}
	if len(tagNames) == 0 {
		return []string{}, nil
	}

	names := tagNames
	if len(names) > maxNamedBadges {
		names = names[:maxNamedBadges]
	}
	more := len(tagNames) - len(names)

	maxTotal := maxWidth - 2*badgeMargin

	// Drop the last named badge (moving it into the overflow count) until
	// the line fits. The overflow badge is tracked structurally via the
	// `more` counter, never by matching "+" on the string, so a tag
	// literally named "+x" cannot be confused with it. At least one named
	// badge is always kept.
	for len(names) > 1 {
		width := totalWidth(f, names)
		if more > 0 {
			width += badgeGap + badgeWidth(f, fmt.Sprintf("+%d", more))
		}
		if width <= maxTotal {
			break
		}
		names = names[:len(names)-1]
		more++
	}

	// Reserve space for the overflow badge (short, must stay readable),
	// then truncate the named badges to fit the space that remains.
	reserved := 0
	if more > 0 {
		reserved = badgeGap + badgeWidth(f, fmt.Sprintf("+%d", more))
	}
	available := maxTotal - reserved
	for i := range names {
		if i > 0 {
			available -= badgeGap
		}
		names[i] = truncateName(f, names[i], available)
		available -= badgeWidth(f, names[i])
	}

	items := append([]string{}, names...)
	if more > 0 {
		items = append(items, fmt.Sprintf("+%d", more))
	}
	return items, nil
}

func fillCircle(img *image.NRGBA, cx, cy, r int, col color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				img.Set(cx+dx, cy+dy, col)
			}
		}
	}
}

func drawPill(img *image.NRGBA, x, y, w, h, r int, col color.RGBA) {
	draw.Draw(img, image.Rect(x, y, x+w, y+h), image.NewUniform(col), image.Point{}, draw.Over)
	fillCircle(img, x+r, y+r, r, col)
	fillCircle(img, x+w-r, y+r, r, col)
	fillCircle(img, x+r, y+h-r, r, col)
	fillCircle(img, x+w-r, y+h-r, r, col)
}

// fillRightTriangle fills a right-pointing triangle: vertical edge at x=bx
// from y0..y1, apex at (bx+tip, (y0+y1)/2).
func fillRightTriangle(img *image.NRGBA, bx, y0, y1, tip int, col color.RGBA) {
	cy := (y0 + y1) / 2
	half := (y1 - y0) / 2
	if half <= 0 || tip <= 0 {
		return
	}
	for yy := y0; yy < y1; yy++ {
		frac := float64(yy-cy) / float64(half)
		if frac < 0 {
			frac = -frac
		}
		w := int(float64(tip) * (1 - frac))
		for xx := bx; xx < bx+w; xx++ {
			img.Set(xx, yy, col)
		}
	}
}

// drawLabelIcon draws an MDI-style "label" icon: a rounded square body with a
// triangular point on its right side, centered at cy, `size` px wide. The
// body is 14/19 of the glyph width, matching the mdi-label proportions.
func drawLabelIcon(img *image.NRGBA, x, cy, size int, col color.RGBA) {
	body := size * 11 / 15
	tip := size - body
	y0 := cy - body/2
	drawPill(img, x, y0, body, body, body/4, col)
	fillRightTriangle(img, x+body, y0, y0+body, tip, col)
}

// DrawPromotedBadges draws one mauve pill badge per promoted tag name,
// top-left on the canvas, capped at maxNamedBadges names plus a "+N"
// overflow badge.
func DrawPromotedBadges(canvas *image.NRGBA, tagNames []string) error {
	f, err := face()
	if err != nil {
		return err
	}
	items, err := buildBadgeItems(tagNames, canvas.Bounds().Dx())
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	white := color.RGBA{255, 255, 255, 255}
	mauve := color.RGBA{121, 87, 213, 255}

	x := badgeMargin
	y := badgeMargin
	for _, s := range items {
		w := badgeWidth(f, s)
		drawPill(canvas, x, y, w, badgeHeight, badgeHeight/4, mauve)
		drawLabelIcon(canvas, x+badgePaddingX, y+badgeHeight/2, iconSize, white)
		d := &font.Drawer{Dst: canvas, Src: image.NewUniform(white), Face: f}
		m := f.Metrics()
		baseline := y + (badgeHeight-m.Ascent.Ceil()-m.Descent.Ceil())/2 + m.Ascent.Ceil()
		d.Dot = fixed.P(x+badgePaddingX+iconSize+iconGap, baseline)
		d.DrawString(s)
		x += w + badgeGap
	}
	return nil
}
