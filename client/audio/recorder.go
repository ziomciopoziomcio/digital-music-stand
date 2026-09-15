package audio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
)

type RecorderAudio struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	playDevice *malgo.Device
	sampleRate uint32
	mu         sync.Mutex

	isRecording bool

	isPlaying      bool
	isPaused       bool
	playbackFile   string
	playbackData   []byte
	playbackOffset int

	file   *os.File
	ticker *time.Ticker

	OnRecordPulse func(active bool)
}

func NewRecorderAudio() (*RecorderAudio, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	r := &RecorderAudio{
		ctx:        ctx,
		sampleRate: 44100,
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = r.sampleRate

	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.isRecording && r.file != nil {
			binary.Write(r.file, binary.LittleEndian, pInputSamples)
		}
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		ctx.Free()
		return nil, err
	}
	r.device = device
	return r, nil
}

func (r *RecorderAudio) StartRecording(outputDir, fileName string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording {
		return "", nil
	}

	os.MkdirAll(outputDir, os.ModePerm)
	fullPath := filepath.Join(outputDir, fileName+".pcm")
	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	r.file = file
	r.isRecording = true
	r.device.Start()

	r.ticker = time.NewTicker(500 * time.Millisecond)
	go func() {
		pulse := true
		for range r.ticker.C {
			if r.OnRecordPulse != nil {
				r.OnRecordPulse(pulse)
			}
			pulse = !pulse
		}
	}()

	return fullPath, nil
}

func (r *RecorderAudio) StopRecording() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRecording {
		return
	}

	r.isRecording = false
	r.device.Stop()

	if r.ticker != nil {
		r.ticker.Stop()
	}
	if r.file != nil {
		r.file.Close()
		r.file = nil
	}
	if r.OnRecordPulse != nil {
		r.OnRecordPulse(false)
	}
}

func (r *RecorderAudio) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRecording
}

func (r *RecorderAudio) PlayRecording(filePath string, onFinish func()) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording {
		return nil
	}

	if r.isPlaying {
		r.stopPlaybackLocked()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	r.playbackData = data
	r.playbackOffset = 0
	r.playbackFile = filePath
	r.isPlaying = true
	r.isPaused = false

	playConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	playConfig.Playback.Format = malgo.FormatS16
	playConfig.Playback.Channels = 1
	playConfig.SampleRate = r.sampleRate

	onSendFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.isPlaying || r.isPaused {
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
			return
		}

		bytesToRead := len(pOutputSample)
		if r.playbackOffset >= len(r.playbackData) {
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
			return
		}

		end := r.playbackOffset + bytesToRead
		if end > len(r.playbackData) {
			end = len(r.playbackData)
		}
		copied := copy(pOutputSample, r.playbackData[r.playbackOffset:end])
		for i := copied; i < len(pOutputSample); i++ {
			pOutputSample[i] = 0
		}
		r.playbackOffset = end
	}

	device, err := malgo.InitDevice(r.ctx.Context, playConfig, malgo.DeviceCallbacks{
		Data: onSendFrames,
	})

	if err != nil {
		r.isPlaying = false
		return err
	}

	r.playDevice = device
	return r.playDevice.Start()
}

func (r *RecorderAudio) stopPlaybackLocked() {
	if r.playDevice != nil {
		r.playDevice.Stop()
		r.playDevice.Uninit()
		r.playDevice = nil
	}
	r.isPlaying = false
	r.isPaused = false
}

func (r *RecorderAudio) StopPlayback() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopPlaybackLocked()
}

func (r *RecorderAudio) TogglePause() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isPlaying {
		r.isPaused = !r.isPaused
	}
	return r.isPaused
}

func (r *RecorderAudio) SeekRelative(seconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isPlaying {
		return
	}
	bytesOffset := int(seconds*float64(r.sampleRate)) * 2
	r.playbackOffset += bytesOffset
	if r.playbackOffset < 0 {
		r.playbackOffset = 0
	}
	if r.playbackOffset >= len(r.playbackData) {
		r.playbackOffset = len(r.playbackData) - 2
	}
}

func (r *RecorderAudio) SeekAbsolute(seconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isPlaying {
		return
	}
	newOffset := int(seconds*float64(r.sampleRate)) * 2
	if newOffset < 0 {
		newOffset = 0
	}
	if newOffset >= len(r.playbackData) {
		newOffset = len(r.playbackData) - 2
	}
	r.playbackOffset = newOffset
}

func (r *RecorderAudio) GetPlaybackState() (playing bool, paused bool, currentSec float64, totalSec float64, filePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isPlaying {
		return false, false, 0, 0, ""
	}
	cSec := float64(r.playbackOffset) / (float64(r.sampleRate) * 2.0)
	tSec := float64(len(r.playbackData)) / (float64(r.sampleRate) * 2.0)
	return r.isPlaying, r.isPaused, cSec, tSec, r.playbackFile
}

func (r *RecorderAudio) Close() {
	r.StopRecording()
	r.StopPlayback()
	if r.device != nil {
		r.device.Uninit()
	}
	if r.ctx != nil {
		r.ctx.Free()
	}
}
