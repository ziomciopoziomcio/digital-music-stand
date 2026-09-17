package plugins

import (
	"fmt"
	"sync"
)

type MixerPlugin interface {
	Name() string
	Connect(ipAddress string) error
	Disconnect() error
	GetConnectionStatus() bool

	SetMainVolume(level float64) error
	MuteMain(mute bool) error

	SetChannelVolume(channel int, level float64) error
	MuteChannel(channel int, mute bool) error

	SetMonitorVolume(level float64) error
	MuteMonitor(mute bool) error

	SetChannelPan(channel int, pan float64) error
	SetChannelName(channel int, name string) error

	mustEmbedUnimplementedMixerPlugin()
}
}

var (
	mixersMu sync.RWMutex
	mixers   = make(map[string]MixerPlugin)
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
