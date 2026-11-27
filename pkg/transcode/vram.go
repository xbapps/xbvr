package transcode

import "sync"

// pendingVRAM is GPU memory promised to encodes that have started but may not have allocated
// it yet, by GPU index. nvidia-smi only shows an encode's memory once its encoder is open, so
// without this two streams starting together would both be planned against the same memory.
var pendingVRAM = struct {
	sync.Mutex
	mib map[int]int
}{mib: map[int]int{}}

// ReserveVRAM counts the GPU memory this plan's encode (and GPU decode) needs as used until release is called,
// which should happen once the encoder has allocated it or has exited. release may be called
// more than once. Plans that use no NVENC memory reserve nothing.
func (p Plan) ReserveVRAM() (release func()) {
	need := 0
	if p.Backend.Name == BackendNVENC {
		need = vramNeedMiB(p.Codec, p.Width, p.Height) + p.decodeVRAMMiB
	}
	if need <= 0 {
		return func() {}
	}
	gpu := p.Backend.GPUIndex

	pendingVRAM.Lock()
	pendingVRAM.mib[gpu] += need
	pendingVRAM.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			pendingVRAM.Lock()
			defer pendingVRAM.Unlock()
			pendingVRAM.mib[gpu] -= need
			if pendingVRAM.mib[gpu] <= 0 {
				delete(pendingVRAM.mib, gpu)
			}
		})
	}
}

func pendingVRAMByGPU() map[int]int {
	pendingVRAM.Lock()
	defer pendingVRAM.Unlock()
	out := make(map[int]int, len(pendingVRAM.mib))
	for gpu, mib := range pendingVRAM.mib {
		out[gpu] = mib
	}
	return out
}
