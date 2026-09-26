package tasks

import (
	"os"
	"path/filepath"
	"testing"
)

// Focused coverage for the funscript speed feature (upstream #622):
// CalculateFunscriptSpeed derives the median per-action speed from a
// .funscript file. Speeds here are |dPos|/dAt*1000 -> 200, 0, 200,
// plus the untouched first action at 0, so the even-count median is
// (0+200)/2 = 100.
func TestCalculateFunscriptSpeed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speed.funscript")
	data := `{"version":"1.0","actions":[{"at":0,"pos":0},{"at":500,"pos":100},{"at":1500,"pos":100},{"at":2000,"pos":0}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	speed, err := CalculateFunscriptSpeed(path)
	if err != nil {
		t.Fatal(err)
	}
	if speed != 100 {
		t.Errorf("CalculateFunscriptSpeed = %d, want 100", speed)
	}
}

func TestCalculateFunscriptSpeedMissingFile(t *testing.T) {
	if _, err := CalculateFunscriptSpeed(filepath.Join(t.TempDir(), "nope.funscript")); err == nil {
		t.Error("CalculateFunscriptSpeed(missing) = nil error, want non-nil")
	}
}
