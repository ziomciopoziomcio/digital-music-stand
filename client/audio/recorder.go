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
	isPlaying   bool
	file        *os.File
	ticker      *time.Ticker

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

	if r.isRecording || r.isPlaying {
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

	if r.isRecording || r.isPlaying {
		return nil
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	r.isPlaying = true
	offset := 0

	playConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	playConfig.Playback.Format = malgo.FormatS16
	playConfig.Playback.Channels = 1
	playConfig.SampleRate = r.sampleRate

	onSendFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		bytesToRead := int(framecount * 2)
		if offset >= len(fileData) {
			r.StopPlayback()
			if onFinish != nil {
				go onFinish()
			}
			return
		}

		end := offset + bytesToRead
		if end > len(fileData) {
			end = len(fileData)
		}
		copy(pOutputSample, fileData[offset:end])
		offset = end
	}

	r.playDevice, _ = malgo.InitDevice(r.ctx.Context, playConfig, malgo.DeviceCallbacks{
		Data: onSendFrames,
	})

	r.playDevice.Start()
	return nil
}

func (r *RecorderAudio) StopPlayback() {
	if r.playDevice != nil {
		r.playDevice.Stop()
		r.playDevice.Uninit()
		r.playDevice = nil
	}
	r.isPlaying = false
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
