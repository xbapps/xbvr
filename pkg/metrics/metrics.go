package metrics

import (
	"os"
	"path/filepath"
	"time"

	"github.com/go-graphite/go-whisper"
	"github.com/xbapps/xbvr/pkg/common"
)

func GetMetric(name string) (*whisper.Whisper, error) {
	retentions, err := whisper.ParseRetentionDefs("1m:1d,1h:60d,12h:20y")
	path := filepath.Join(common.MetricsDir, name+".wsp")

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

func WritePoint(name string, value float64) error {
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
