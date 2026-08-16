package audio

import (
	"math"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

type MicrophoneTuner struct {
	ctx           *malgo.AllocatedContext
	device        *malgo.Device
	onPitchDetect func(freq float64)
	running       bool
	buffer        []float32
	mu            sync.Mutex
}

func NewMicrophoneTuner(onPitch func(float64)) *MicrophoneTuner {
	return &MicrophoneTuner{
		onPitchDetect: onPitch,
		buffer:        make([]float32, 0, 8192),
	}
}

func (m *MicrophoneTuner) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return err
	}
	m.ctx = ctx

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 44100

	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		if len(pInputSamples) == 0 {
			return
		}

		samples := unsafe.Slice((*float32)(unsafe.Pointer(&pInputSamples[0])), framecount)

		m.mu.Lock()
		m.buffer = append(m.buffer, samples...)

		if len(m.buffer) >= 4096 {
			processBuf := make([]float32, 4096)
			copy(processBuf, m.buffer[len(m.buffer)-4096:])

			m.buffer = m.buffer[2048:]
			m.mu.Unlock()

			freq := detectPitch(processBuf, 44100)
			if m.onPitchDetect != nil {
				m.onPitchDetect(freq)
			}
		} else {
			m.mu.Unlock()
		}
	}

	callbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, callbacks)
	if err != nil {
		ctx.Free()
		return err
	}
	m.device = device

	err = device.Start()
	if err != nil {
		device.Uninit()
		ctx.Free()
		return err
	}

	m.running = true
	return nil
}

func (m *MicrophoneTuner) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	if m.device != nil {
		m.device.Uninit()
		m.device = nil
	}
	if m.ctx != nil {
		m.ctx.Free()
		m.ctx = nil
	}
	m.running = false
	m.buffer = make([]float32, 0, 8192)
}

func detectPitch(buffer []float32, sampleRate int) float64 {
	n := len(buffer)
	windowSize := n / 2

	var rms float64
	for _, val := range buffer {
		rms += float64(val * val)
	}
	rms = math.Sqrt(rms / float64(n))
	if rms < 0.015 {
		return 0
	}

	minLag := sampleRate / 1000
	maxLag := sampleRate / 60

	if maxLag > windowSize {
		maxLag = windowSize
	}

	diffs := make([]float64, maxLag)
	for lag := minLag; lag < maxLag; lag++ {
		diff := float64(0)
		for i := 0; i < windowSize; i++ {
			delta := float64(buffer[i] - buffer[i+lag])
			diff += delta * delta
		}
		diffs[lag] = diff
	}

	bestLag := -1
	minD := float64(math.MaxFloat64)
	for lag := minLag; lag < maxLag; lag++ {
		if diffs[lag] < minD {
			minD = diffs[lag]
			bestLag = lag
		}
	}

	if bestLag > 0 {
		return float64(sampleRate) / float64(bestLag)
	}
	return 0
}
