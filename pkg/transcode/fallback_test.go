package transcode

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func fakeProcess(output string, exit error) *Process {
	return &Process{Stdout: strings.NewReader(output), Wait: func() error { return exit }}
}

func TestWithFallbackUsesPrimaryOutput(t *testing.T) {
	called := false
	p := WithFallback(fakeProcess("primary", nil), func(error) (*Process, error) {
		called = true
		return fakeProcess("fallback", nil), nil
	})
	out, err := io.ReadAll(p.Stdout)
	if err != nil || string(out) != "primary" {
		t.Fatalf("output = %q, %v; want primary", out, err)
	}
	if err := p.Wait(); err != nil {
		t.Errorf("Wait = %v", err)
	}
	if called {
		t.Error("fallback ran although primary succeeded")
	}
}

func TestWithFallbackReplacesSilentFailure(t *testing.T) {
	cause := errors.New("hwaccel initialisation returned error")
	var got error
	p := WithFallback(fakeProcess("", cause), func(err error) (*Process, error) {
		got = err
		return fakeProcess("fallback", nil), nil
	})
	out, err := io.ReadAll(p.Stdout)
	if err != nil || string(out) != "fallback" {
		t.Fatalf("output = %q, %v; want fallback", out, err)
	}
	if !errors.Is(got, cause) {
		t.Errorf("fallback cause = %v, want %v", got, cause)
	}
	if err := p.Wait(); err != nil {
		t.Errorf("Wait = %v, want the fallback's result", err)
	}
}

func TestWithFallbackKeepsFailureAfterOutput(t *testing.T) {
	cause := errors.New("out of memory")
	p := WithFallback(fakeProcess("partial", cause), func(error) (*Process, error) {
		t.Error("fallback ran after output was written")
		return fakeProcess("fallback", nil), nil
	})
	out, _ := io.ReadAll(p.Stdout)
	if string(out) != "partial" {
		t.Fatalf("output = %q, want partial", out)
	}
	if err := p.Wait(); !errors.Is(err, cause) {
		t.Errorf("Wait = %v, want %v", err, cause)
	}
}

func TestWithFallbackReportsPrimaryErrorWhenFallbackCannotStart(t *testing.T) {
	cause := errors.New("decoder failed")
	p := WithFallback(fakeProcess("", cause), func(error) (*Process, error) {
		return nil, errors.New("context canceled")
	})
	out, err := io.ReadAll(p.Stdout)
	if err != nil || len(out) != 0 {
		t.Fatalf("output = %q, %v; want empty", out, err)
	}
	if err := p.Wait(); !errors.Is(err, cause) {
		t.Errorf("Wait = %v, want %v", err, cause)
	}
}

func TestWithFallbackRunsOnce(t *testing.T) {
	calls := 0
	p := WithFallback(fakeProcess("", errors.New("first")), func(error) (*Process, error) {
		calls++
		return fakeProcess("", errors.New("second")), nil
	})
	if _, err := io.ReadAll(p.Stdout); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("fallback calls = %d, want 1", calls)
	}
	if err := p.Wait(); err == nil || err.Error() != "second" {
		t.Errorf("Wait = %v, want the fallback's error", err)
	}
}

func TestWithFallbackCleanExitWithoutOutput(t *testing.T) {
	p := WithFallback(fakeProcess("", nil), func(error) (*Process, error) {
		t.Error("fallback ran after a clean exit")
		return nil, nil
	})
	if _, err := io.ReadAll(p.Stdout); err != nil {
		t.Fatal(err)
	}
	if err := p.Wait(); err != nil {
		t.Errorf("Wait = %v", err)
	}
}
