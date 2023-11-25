package common

import (
	"os"
	"path/filepath"
	"time"

	"github.com/lomik/go-whisper"
)

func GetMetric(name string) (*whisper.Whisper, error) {
	retentions, _ := whisper.ParseRetentionDefs("1m:1d,1h:60d,12h:20y")
	path := filepath.Join(MetricsDir, name+".wsp")

	wsp, err := whisper.Create(path, retentions, whisper.Last, 0.5)

	if err == os.ErrExist {
		wsp, err = whisper.Open(path)
		if err != nil {
			return nil, err
		}
		return wsp, nil
	}

	if err != nil {
		return nil, err
	}

	return wsp, nil
}

func AddMetricPoint(name string, value float64) error {
	db, err := GetMetric(name)
	defer db.Close()
	if err != nil {
		return err
	}

	err = db.Update(value, int(time.Now().Unix()))
	if err != nil {
		return err
	}

	return nil
}
