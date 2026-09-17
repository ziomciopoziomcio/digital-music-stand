package plugins

import (
	"errors"
	"fmt"
	"sync"
)

var ErrNotImplemented = errors.New("method not implemented by this mixer plugin")

type MixerPlugin interface {
	Name() string
	Connect(ipAddress string) error
	Disconnect() error
	GetConnectionStatus() bool

	GetChannelCount() int
	GetBusCount() int

	GetMainVolume() (float64, error)
	SetMainVolume(level float64) error
	MuteMain(mute bool) error

	SetChannelVolume(channel int, level float64) error
	MuteChannel(channel int, mute bool) error

	SetBusVolume(bus int, level float64) error
	SetChannelSendVolume(channel int, bus int, level float64) error

	SetMonitorVolume(level float64) error
	MuteMonitor(mute bool) error

	SetChannelPan(channel int, pan float64) error
	SetChannelName(channel int, name string) error

	mustEmbedUnimplementedMixerPlugin()
}

type UnimplementedMixerPlugin struct{}

func (UnimplementedMixerPlugin) Name() string                      { return "Unknown Mixer" }
func (UnimplementedMixerPlugin) Connect(ipAddress string) error    { return ErrNotImplemented }
func (UnimplementedMixerPlugin) Disconnect() error                 { return ErrNotImplemented }
func (UnimplementedMixerPlugin) GetConnectionStatus() bool         { return false }
func (UnimplementedMixerPlugin) GetChannelCount() int              { return 0 }
func (UnimplementedMixerPlugin) GetBusCount() int                  { return 0 }
func (UnimplementedMixerPlugin) GetMainVolume() (float64, error)   { return 0, ErrNotImplemented }
func (UnimplementedMixerPlugin) SetMainVolume(level float64) error { return ErrNotImplemented }
func (UnimplementedMixerPlugin) MuteMain(mute bool) error          { return ErrNotImplemented }
func (UnimplementedMixerPlugin) SetChannelVolume(channel int, level float64) error {
	return ErrNotImplemented
}
func (UnimplementedMixerPlugin) MuteChannel(channel int, mute bool) error  { return ErrNotImplemented }
func (UnimplementedMixerPlugin) SetBusVolume(bus int, level float64) error { return ErrNotImplemented }
func (UnimplementedMixerPlugin) SetChannelSendVolume(channel int, bus int, level float64) error {
	return ErrNotImplemented
}
func (UnimplementedMixerPlugin) SetMonitorVolume(level float64) error { return ErrNotImplemented }
func (UnimplementedMixerPlugin) MuteMonitor(mute bool) error          { return ErrNotImplemented }
func (UnimplementedMixerPlugin) SetChannelPan(channel int, pan float64) error {
	return ErrNotImplemented
}
func (UnimplementedMixerPlugin) SetChannelName(channel int, name string) error {
	return ErrNotImplemented
}
func (UnimplementedMixerPlugin) mustEmbedUnimplementedMixerPlugin() {}

var (
	mixersMu    sync.RWMutex
	mixers      = make(map[string]MixerPlugin)
	activeMixer MixerPlugin
)

func RegisterMixer(plugin MixerPlugin) {
	mixersMu.Lock()
	defer mixersMu.Unlock()
	if plugin == nil {
		panic("plugins: Register mixer is nil")
	}
	name := plugin.Name()
	if _, dup := mixers[name]; dup {
		panic("plugins: Register called twice for mixer " + name)
	}
	mixers[name] = plugin
}

func GetAvailableMixers() []string {
	mixersMu.RLock()
	defer mixersMu.RUnlock()
	var list []string
	for name := range mixers {
		list = append(list, name)
	}
	return list
}

func GetMixer(name string) (MixerPlugin, error) {
	mixersMu.RLock()
	defer mixersMu.RUnlock()
	m, ok := mixers[name]
	if !ok {
		return nil, fmt.Errorf("mixer plugin %q not found", name)
	}
	return m, nil
}

func SetActiveMixer(plugin MixerPlugin) {
	mixersMu.Lock()
	defer mixersMu.Unlock()
	activeMixer = plugin
}

func GetActiveMixer() MixerPlugin {
	mixersMu.RLock()
	defer mixersMu.RUnlock()
	return activeMixer
}
