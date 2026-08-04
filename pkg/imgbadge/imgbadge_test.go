package imgbadge

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
	"testing"
)

func newCanvas() *image.NRGBA {
	c := image.NewNRGBA(image.Rect(0, 0, 700, 420))
	draw.Draw(c, c.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	return c
}

func TestBuildBadgeItemsCapsAtThree(t *testing.T) {
	items, err := buildBadgeItems([]string{"anal", "milf", "pov", "teen", "dp"}, 700)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"anal", "milf", "pov", "+2"}
	if len(items) != len(want) {
		t.Fatalf("got %v, want %v", items, want)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("got %v, want %v", items, want)
		}
	}
}

func TestBuildBadgeItemsNoOverflow(t *testing.T) {
	items, err := buildBadgeItems([]string{"anal"}, 700)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0] != "anal" {
		t.Fatalf("got %v", items)
	}
}

func TestBuildBadgeItemsEmpty(t *testing.T) {
	items, err := buildBadgeItems(nil, 700)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("got %v", items)
	}
}

func TestBuildBadgeItemsTruncatesLongName(t *testing.T) {
	long := strings.Repeat("x", 50)
	items, err := buildBadgeItems([]string{long}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %v", items)
	}
	if !strings.HasSuffix(items[0], "…") {
		t.Fatalf("expected ellipsis suffix, got %q", items[0])
	}
}

func TestBuildBadgeItemsDropRecountsOverflow(t *testing.T) {
	long := strings.Repeat("a", 30)
	tags := []string{long, long, long, long, long}
	items, err := buildBadgeItems(tags, 250)
	if err != nil {
		t.Fatal(err)
	}
	named := 0
	overflow := 0
	for _, it := range items {
		if strings.HasPrefix(it, "+") {
			n, err := strconv.Atoi(strings.TrimPrefix(it, "+"))
			if err != nil {
				t.Fatalf("bad overflow badge %q", it)
			}
			overflow = n
		} else {
			named++
		}
	}
	if named+overflow != len(tags) {
		t.Fatalf("named(%d)+overflow(%d)=%d, want %d", named, overflow, named+overflow, len(tags))
	}
}

func TestBuildBadgeItemsPlusNamedTag(t *testing.T) {
	count := func(tags, items []string) (named, overflow int) {
		for _, it := range items {
			isNamed := false
			for _, tag := range tags {
				if it == tag {
					isNamed = true
					break
				}
			}
			if isNamed {
				named++
				continue
			}
			n, err := strconv.Atoi(strings.TrimPrefix(it, "+"))
			if err != nil {
				t.Fatalf("bad overflow badge %q", it)
			}
			overflow = n
		}
		return named, overflow
	}

	items, err := buildBadgeItems([]string{"+3", "anal"}, 700)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"+3", "anal"}
	if len(items) != len(want) {
		t.Fatalf("got %v, want %v", items, want)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("got %v, want %v", items, want)
		}
	}
	named, overflow := count([]string{"+3", "anal"}, items)
	if named != 2 || overflow != 0 {
		t.Fatalf("literal \"+3\" must be drawn as a named badge, got %v", items)
	}
	if named+overflow != 2 {
		t.Fatalf("named(%d)+overflow(%d)=%d, want %d", named, overflow, named+overflow, 2)
	}

	items, err = buildBadgeItems([]string{"+3", "anal", "pov", "milf"}, 700)
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"+3", "anal", "pov", "+1"}
	if len(items) != len(want) {
		t.Fatalf("got %v, want %v", items, want)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("got %v, want %v", items, want)
		}
	}
	named, overflow = count([]string{"+3", "anal", "pov", "milf"}, items)
	if named != 3 || overflow != 1 {
		t.Fatalf("got %v, want 3 named badges + a separate \"+1\" overflow badge", items)
	}
	if named+overflow != 4 {
		t.Fatalf("named(%d)+overflow(%d)=%d, want %d", named, overflow, named+overflow, 4)
	}
}

func TestDrawPromotedBadgesPaintsTopLeft(t *testing.T) {
	c := newCanvas()
	err := DrawPromotedBadges(c, []string{"anal"})
	if err != nil {
		t.Fatal(err)
	}
	col := c.NRGBAAt(50, 12)
	if col.R == 0 && col.G == 0 && col.B == 0 {
		t.Fatal("expected non-black pixel in top-left badge area")
	}
	col = c.NRGBAAt(350, 415)
	if col.R != 0 || col.G != 0 || col.B != 0 {
		t.Fatal("expected bottom of canvas untouched")
	}
}

func TestDrawPromotedBadgesNoTags(t *testing.T) {
	c := newCanvas()
	err := DrawPromotedBadges(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	col := c.NRGBAAt(50, 12)
	if col.R != 0 || col.G != 0 || col.B != 0 {
		t.Fatal("expected canvas untouched when no promoted tags")
	}
}
