package plugins

import (
	"fmt"
	"net"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type BehringerXAirPlugin struct {
	UnimplementedMixerPlugin
	ip        string
	connected bool
}

func init() {
	RegisterMixer(&BehringerXAirPlugin{})
}

func (p *BehringerXAirPlugin) Name() string {
	return "Behringer X-Air (XR18/OSC)"
}

func (p *BehringerXAirPlugin) sendOSC(address string, arg interface{}) error {
	if !p.connected {
		return fmt.Errorf("not connected")
	}
	client := osc.NewClient(p.ip, 10024)
	msg := osc.NewMessage(address)
	if arg != nil {
		switch v := arg.(type) {
		case float64:
			msg.Append(float32(v))
		case float32:
			msg.Append(v)
		case int:
			msg.Append(int32(v))
		case string:
			msg.Append(v)
		}
	}
	return client.Send(msg)
}

func (p *BehringerXAirPlugin) queryOSC(address string) (*osc.Message, error) {
	if !p.connected {
		return nil, fmt.Errorf("not connected")
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:10024", p.ip))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	msg := osc.NewMessage(address)
	b, err := msg.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(b); err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, fmt.Errorf("mixer timeout or offline: %v", err)
	}

	// Jawne rzutowanie []byte na string
	packet, err := osc.ParsePacket(string(buf[:n]))
	if err != nil {
		return nil, err
	}

	if replyMsg, ok := packet.(*osc.Message); ok {
		return replyMsg, nil
	}
	return nil, fmt.Errorf("invalid response format")
}

func (p *BehringerXAirPlugin) Connect(ipAddress string) error {
	p.ip = ipAddress
	p.connected = true

	_, err := p.GetMainVolume()
	if err != nil {
		p.connected = false
		return fmt.Errorf("failed to connect to XR18 at %s: %v", ipAddress, err)
	}

	return nil
}

func (p *BehringerXAirPlugin) Disconnect() error {
	p.connected = false
	return nil
}

func (p *BehringerXAirPlugin) GetConnectionStatus() bool {
	return p.connected
}

func (p *BehringerXAirPlugin) GetChannelCount() int {
	return 16
}

func (p *BehringerXAirPlugin) GetBusCount() int {
	return 6
}

func (p *BehringerXAirPlugin) GetMainVolume() (float64, error) {
	msg, err := p.queryOSC("/lr/mix/fader")
	if err != nil {
		return 0, err
	}
	if len(msg.Arguments) > 0 {
		if val, ok := msg.Arguments[0].(float32); ok {
			return float64(val), nil
		}
	}
	return 0, fmt.Errorf("unexpected response arguments")
}

func (p *BehringerXAirPlugin) SetMainVolume(level float64) error {
	return p.sendOSC("/lr/mix/fader", float32(level))
}

func (p *BehringerXAirPlugin) MuteMain(mute bool) error {
	val := 1
	if mute {
		val = 0
	}
	return p.sendOSC("/lr/mix/on", int32(val))
}

func (p *BehringerXAirPlugin) GetBusVolume(bus int) (float64, error) {
	addr := fmt.Sprintf("/bus/%02d/mix/fader", bus)
	msg, err := p.queryOSC(addr)
	if err != nil {
		return 0, err
	}
	if len(msg.Arguments) > 0 {
		if val, ok := msg.Arguments[0].(float32); ok {
			return float64(val), nil
		}
	}
	return 0, fmt.Errorf("unexpected response arguments")
}

func (p *BehringerXAirPlugin) SetBusVolume(bus int, level float64) error {
	return p.sendOSC(fmt.Sprintf("/bus/%02d/mix/fader", bus), float32(level))
}

func (p *BehringerXAirPlugin) GetChannelSendVolume(channel int, bus int) (float64, error) {
	addr := fmt.Sprintf("/ch/%02d/mix/%02d/level", channel, bus)
	msg, err := p.queryOSC(addr)
	if err != nil {
		return 0, err
	}
	if len(msg.Arguments) > 0 {
		if val, ok := msg.Arguments[0].(float32); ok {
			return float64(val), nil
		}
	}
	return 0, fmt.Errorf("unexpected response arguments")
}

func (p *BehringerXAirPlugin) SetChannelSendVolume(channel int, bus int, level float64) error {
	return p.sendOSC(fmt.Sprintf("/ch/%02d/mix/%02d/level", channel, bus), float32(level))
}

func (p *BehringerXAirPlugin) GetChannelName(channel int) (string, error) {
	addr := fmt.Sprintf("/ch/%02d/config/name", channel)
	msg, err := p.queryOSC(addr)
	if err != nil {
		return "", err
	}
	if len(msg.Arguments) > 0 {
		if val, ok := msg.Arguments[0].(string); ok {
			return val, nil
		}
	}
	return "", fmt.Errorf("unexpected response arguments")
}

func (p *BehringerXAirPlugin) SetChannelName(channel int, name string) error {
	return p.sendOSC(fmt.Sprintf("/ch/%02d/config/name", channel), name)
}

func (p *BehringerXAirPlugin) GetBusName(bus int) (string, error) {
	addr := fmt.Sprintf("/bus/%02d/config/name", bus)
	msg, err := p.queryOSC(addr)
	if err != nil {
		return "", err
	}
	if len(msg.Arguments) > 0 {
		if val, ok := msg.Arguments[0].(string); ok {
			return val, nil
		}
	}
	return "", fmt.Errorf("unexpected response arguments")
}
