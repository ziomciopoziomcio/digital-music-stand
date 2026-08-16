package audio

import (
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/gen2brain/malgo"
)

type MetronomeAudio struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	sampleRate uint32
	mu         sync.Mutex

	playing         bool
	bpm             float64
	beatsPerMeasure int

	startTime time.Time
	nextBeat  int

	phase      float64
	framesLeft int
	isAccent   bool

	OnBeat func(accent bool)
}

func NewMetronomeAudio() (*MetronomeAudio, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	m := &MetronomeAudio{
		ctx:             ctx,
		sampleRate:      44100,
		bpm:             120,
		beatsPerMeasure: 4,
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatF32
	deviceConfig.Playback.Channels = 1
	deviceConfig.SampleRate = m.sampleRate

	onSendFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		samples := unsafe.Slice((*float32)(unsafe.Pointer(&pOutputSample[0])), framecount)

		m.mu.Lock()
		defer m.mu.Unlock()

		var triggerBeat bool

		if m.playing {
			elapsed := time.Since(m.startTime).Seconds()

			expectedBeats := int(elapsed * (m.bpm / 60.0))

			if expectedBeats >= m.nextBeat {
				m.nextBeat = expectedBeats + 1
				triggerBeat = true

				beatInMeasure := expectedBeats % m.beatsPerMeasure
				m.isAccent = (m.beatsPerMeasure > 1 && beatInMeasure == 0)

				m.framesLeft = int(m.sampleRate) / 20
				m.phase = 0
			}
		}

		if triggerBeat && m.OnBeat != nil {
			accent := m.isAccent
			go m.OnBeat(accent)
		}

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

func (m *MetronomeAudio) SetBPM(bpm float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.playing {
		m.startTime = time.Now()
		m.nextBeat = 0
	}
	m.bpm = bpm
}

func (m *MetronomeAudio) SetTimeSignature(beats int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beatsPerMeasure = beats
}

func (m *MetronomeAudio) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playing = true
	m.startTime = time.Now()
	m.nextBeat = 0
	m.framesLeft = 0
}

func (m *MetronomeAudio) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playing = false
	m.framesLeft = 0
}

func (m *MetronomeAudio) Close() {
	if m.device != nil {
		m.device.Uninit()
	}
	if m.ctx != nil {
		m.ctx.Free()
	}
}
