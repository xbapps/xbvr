package transcode

import (
	"sync"
	"testing"
)

func fakeGPUs(t *testing.T, gpus ...nvidiaGPU) {
	t.Helper()
	saved := listGPUs
	listGPUs = func() []nvidiaGPU { return append([]nvidiaGPU(nil), gpus...) }
	t.Cleanup(func() { listGPUs = saved })
}

func nvencHardware(gpu, free int) Hardware {
	b := Backend{Name: BackendNVENC, GPUIndex: gpu, FreeVRAMMiB: free, Encoders: map[string]string{CodecHEVC: "hevc_nvenc"}}
	return Hardware{Preferred: b, Available: []Backend{b}}
}

func TestRefreshVRAMCountsReservedMemoryAsUsed(t *testing.T) {
	fakeGPUs(t, nvidiaGPU{index: 0, free: 8000})
	plan := Plan{Codec: CodecHEVC, Backend: Backend{Name: BackendNVENC, GPUIndex: 0}, Width: 3840, Height: 1920}
	need := vramNeedMiB(plan.Codec, plan.Width, plan.Height)
	if need <= 0 {
		t.Fatal("test setup: the plan should need GPU memory")
	}

	release := plan.ReserveVRAM()
	if got := RefreshVRAM(nvencHardware(0, 0)).Preferred.FreeVRAMMiB; got != 8000-need {
		t.Fatalf("free = %d while reserved, want %d", got, 8000-need)
	}
	release()
	release() // a second release must not free the memory twice
	if got := RefreshVRAM(nvencHardware(0, 0)).Preferred.FreeVRAMMiB; got != 8000 {
		t.Fatalf("free = %d after release, want 8000", got)
	}
}

func TestReservationOnFullGPUStillReadsAsKnown(t *testing.T) {
	fakeGPUs(t, nvidiaGPU{index: 0, free: 100})
	release := Plan{Codec: CodecHEVC, Backend: Backend{Name: BackendNVENC}, Width: 3840, Height: 1920}.ReserveVRAM()
	defer release()
	if got := RefreshVRAM(nvencHardware(0, 0)).Preferred.FreeVRAMMiB; got != 1 {
		t.Fatalf("free = %d, want 1: zero would read as unknown and skip the memory check", got)
	}
}

func TestReserveVRAMIgnoresPlansWithoutNVENC(t *testing.T) {
	fakeGPUs(t, nvidiaGPU{index: 0, free: 8000})
	release := Plan{Codec: CodecH264, Backend: Backend{Name: BackendCPU}, Width: 3840, Height: 1920}.ReserveVRAM()
	defer release()
	if got := RefreshVRAM(nvencHardware(0, 0)).Preferred.FreeVRAMMiB; got != 8000 {
		t.Fatalf("free = %d, want 8000", got)
	}
}

func TestRefreshVRAMLeavesItsArgumentUnchanged(t *testing.T) {
	fakeGPUs(t, nvidiaGPU{index: 0, free: 8000})
	shared := nvencHardware(0, 1234)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RefreshVRAM(shared)
		}()
	}
	wg.Wait()
	if got := shared.Available[0].FreeVRAMMiB; got != 1234 {
		t.Fatalf("shared result changed to %d; it is read by other requests", got)
	}
}
