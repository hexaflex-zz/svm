// Package clock implements a simple clock and timer mechanism.
package clock

import (
	"time"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

// Known interrupt operations.
const (
	SetIntID = iota
	Uptime
	SetTimer
)

// Device defines all internal doodads for the display.
type Device struct {
	intFunc  devices.IntFunc    // Hardware interrupt handler.
	start    time.Time          // Startup time.
	endPoll  chan struct{}      // poll exit signaller.
	newTimer chan time.Duration // new timer channel.
	intID    int                // interrupt Id.
}

var _ devices.Device = &Device{}

// New creates a new device instance.
func New() *Device {
	return &Device{}
}

// ID returns the device id.
func (d *Device) ID() devices.ID {
	return devices.NewID(0xfffe, 0x0005)
}

// Startup initializes device resources.
func (d *Device) Startup(f devices.IntFunc) error {
	d.intFunc = f
	d.start = time.Now()
	d.intID = 0
	d.endPoll = make(chan struct{}, 1)
	d.newTimer = make(chan time.Duration, 1)
	go d.poll()
	return nil
}

// Shutdown clears device resources.
func (d *Device) Shutdown() error {
	close(d.endPoll)

	d.intFunc = nil
	d.intID = 0

	return nil
}

// Int triggers an interrupt on the device. The device can read from- and write to system memory.
func (d *Device) Int(mem devices.Memory) {
	switch mem.U16(cpu.R0) {
	case SetIntID:
		d.intID = mem.U16(cpu.R1)
	case Uptime:
		ms := int(time.Since(d.start).Milliseconds())
		addr := mem.U16(cpu.R1)
		mem.SetU16(addr, (ms>>16)&0xffff)
		mem.SetU16(addr+2, (ms & 0xffff))
	case SetTimer:
		d.newTimer <- time.Millisecond * time.Duration(mem.U16(cpu.R1))
	}
}

// poll triggers periodic hardware interrupts if a timer is running.
func (d *Device) poll() {
	timer := time.NewTicker(time.Minute * 1e6)
	defer timer.Stop()

	for {
		select {
		case <-d.endPoll:
			return
		case interval := <-d.newTimer:
			timer = time.NewTicker(interval)
		case <-timer.C:
			if d.intID > 0 {
				d.intFunc(d.intID)
			}
		}
	}
}
