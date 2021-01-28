package ffprobe

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrBinNotFound is returned when the ffprobe binary was not found
	ErrBinNotFound = errors.New("ffprobe bin not found")
	// ErrTimeout is returned when the ffprobe process did not succeed within the given time
	ErrTimeout = errors.New("process timeout exceeded")

	binPath = "ffprobe"
)

// SetFFProbeBinPath sets the global path to find and execute the ffprobe program
func SetFFProbeBinPath(newBinPath string) {
	binPath = newBinPath
}

// GetProbeData is used for probing the given media file using ffprobe with a set timeout.
// The timeout can be provided to kill the process if it takes too long to determine
// the files information.
// Note: It is probably better to use Context with GetProbeDataContext() these days as it is more flexible.
func GetProbeData(filePath string, timeout time.Duration) (data *ProbeData, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return GetProbeDataContext(ctx, filePath)
}
