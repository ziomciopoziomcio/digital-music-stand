package plugins

import (
	"fmt"
	"sync"
)

type MixerPlugin interface {
	Name() string
	Connect(ipAddress string) error
	Disconnect() error
	SetMainVolume(level float64) error
	MuteChannel(channel string, mute bool) error
}

var (
	mixersMu sync.RWMutex
	mixers   = make(map[string]MixerPlugin)
)

func RegisterMixer(plugin MixerPlugin) {
	mixersMu.Lock()
	defer mixersMu.Unlock()
	if plugin == nil {
		panic("mixer: RegisterMixer called with nil plugin")
	}
	name := plugin.Name()
	if _, dup := mixers[name]; dup {
		panic("mixer: RegisterMixer called twice for plugin " + name)
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
		return nil, fmt.Errorf("mixer: no plugin with name %s found", name)
	}
	return m, nil
}
