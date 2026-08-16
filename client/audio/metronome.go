package audio

import (
	"math"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

type MetronomeAudio struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	sampleRate uint32
	mu         sync.Mutex

	phase      float64
	framesLeft int
	isAccent   bool
}

func NewMetronomeAudio() (*MetronomeAudio, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	m := &MetronomeAudio{
		ctx:        ctx,
		sampleRate: 44100,
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatF32
	deviceConfig.Playback.Channels = 1
	deviceConfig.SampleRate = m.sampleRate

	onSendFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		samples := unsafe.Slice((*float32)(unsafe.Pointer(&pOutputSample[0])), framecount)

		m.mu.Lock()
		defer m.mu.Unlock()

		for i := uint32(0); i < framecount; i++ {
			if m.framesLeft > 0 {
				freq := 1000.0
				if m.isAccent {
					freq = 2000.0
				}

				env := float32(m.framesLeft) / float32(int(m.sampleRate)/20)
				samples[i] = float32(math.Sin(m.phase)) * env * 0.8

				m.phase += 2 * math.Pi * freq / float64(m.sampleRate)
				m.framesLeft--
			} else {
				samples[i] = 0
				m.phase = 0
			}
		}
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onSendFrames,
	})
	if err != nil {
		ctx.Free()
		return nil, err
	}
	m.device = device

	err = device.Start()
	if err != nil {
		device.Uninit()
		ctx.Free()
		return nil, err
	}

	return m, nil
}

func (m *MetronomeAudio) PlayTick(accent bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.framesLeft = int(m.sampleRate) / 20
	m.phase = 0
	m.isAccent = accent
}

func (m *MetronomeAudio) Close() {
	if m.device != nil {
		m.device.Uninit()
	}
	if m.ctx != nil {
		m.ctx.Free()
	}
}
