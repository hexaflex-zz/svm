// Package clock implements a simple clock and timer mechanism.
package clock

import (
	"time"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

// Known interrupt operations.
const (
	Uptime = iota
)

// Device defines all internal doodads for the display.
type Device struct {
	start time.Time
}

var _ devices.Device = &Device{}

func New() *Device {
	return &Device{}
}

func (d *Device) Id() devices.Id {
	return devices.NewId(0xfffe, 0x0005)
}

func (d *Device) Startup() error {
	d.start = time.Now()
	return nil
}

func (d *Device) Shutdown() error {
	return nil
}

// Int triggers an interrupt on the device. The device can read from- and write to system memory.
func (d *Device) Int(mem devices.Memory) {
	switch mem.U16(cpu.R0) {
	case Uptime:
		ms := int(time.Since(d.start).Milliseconds())
		addr := mem.U16(cpu.R1)
		mem.SetU16(addr, (ms>>16)&0xffff)
		mem.SetU16(addr+2, (ms & 0xffff))
	}
}
