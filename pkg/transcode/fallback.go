package transcode

import (
	"errors"
	"io"
)

// WithFallback returns a process that reads primary's output and, if primary exits with an
// error before writing anything, starts the process returned by fallback and reads that one
// instead. Output already written cannot be replaced, so a failure after the first byte is
// returned as is. fallback receives primary's error and is called at most once; if it fails,
// the result ends with primary's error.
func WithFallback(primary *Process, fallback func(cause error) (*Process, error)) *Process {
	f := &fallbackProcess{active: primary, fallback: fallback}
	return &Process{Stdout: f, Wait: f.wait}
}

type fallbackProcess struct {
	active   *Process
	fallback func(error) (*Process, error)
	wrote    bool
	waited   bool
	waitErr  error
}

func (f *fallbackProcess) Read(p []byte) (int, error) {
	for {
		n, err := f.active.Stdout.Read(p)
		if n > 0 {
			f.wrote = true
		}
		if !errors.Is(err, io.EOF) || f.wrote || f.fallback == nil {
			return n, err
		}
		fallback := f.fallback
		f.fallback = nil
		cause := f.active.Wait()
		if cause == nil {
			f.waited = true
			return 0, io.EOF
		}
		next, startErr := fallback(cause)
		if startErr != nil || next == nil {
			f.waited, f.waitErr = true, cause
			return 0, io.EOF
		}
		f.active = next
	}
}

func (f *fallbackProcess) wait() error {
	if f.waited {
		return f.waitErr
	}
	f.waited = true
	f.waitErr = f.active.Wait()
	return f.waitErr
}
