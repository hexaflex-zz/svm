// Package fd35 implements a generic 3.5" floppy disk drive.
package fd35

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

const (
	TrackCount         = 80
	SectorsPerTrack    = 18
	BytesPerSector     = 1024
	FloppySize         = TrackCount * SectorsPerTrack * BytesPerSector
	BytesPerSecond     = BytesPerSector * 10
	TrackSeekTime      = time.Millisecond * 3
	SectorTransferTime = (time.Second * BytesPerSector) / BytesPerSecond
)

// Known interrupt operations.
const (
	ReadState = iota
	ReadSector
	WriteSector
)

// Known device states.
const (
	StateNoMedia = iota
	StateReady
	StateReadyWP
	StateBusy
)

// Known error conditions.
const (
	ErrorNone = iota
	ErrorBusy
	ErrorNoMedia
	ErrorProtected
	ErrorEject
	ErrorBadSector
	ErrorBroken
)

// Device defines all internal doodads for the display.
type Device struct {
	m        sync.Mutex
	file     string // Backing file for disk data.
	data     []byte // Floppy disk data.
	state    int    // Current device state.
	error    int    // Last error that occurred.
	track    int    // Current track we are at.
	readonly bool   // Disk is readonly?
}

var _ devices.Device = &Device{}

func New(file string, readonly bool) *Device {
	return &Device{
		file:     file,
		readonly: readonly,
	}
}

func (d *Device) Id() devices.Id {
	return devices.NewId(0xfffe, 0x0004)
}

func (d *Device) Startup(devices.IntFunc) error {
	d.m.Lock()

	d.state = StateNoMedia
	d.error = ErrorNone
	d.track = 0

	if len(d.file) == 0 {
		d.m.Unlock()
		return nil
	}

	log.Println(d.Id(), "reading", d.file)
	fd, err := os.Open(d.file)
	if err != nil {
		d.error = ErrorBroken
		d.m.Unlock()
		return err
	}

	d.data, err = ioutil.ReadAll(fd)
	fd.Close()

	if err != nil {
		d.error = ErrorBroken
		d.m.Unlock()
		return err
	}

	if len(d.data) != FloppySize {
		d.error = ErrorBroken
		d.m.Unlock()
		return fmt.Errorf("invalid disk size; expected %d, have %d", FloppySize, len(d.data))
	}

	d.m.Unlock()
	d.setReady()
	return nil
}

func (d *Device) Shutdown() error {
	d.m.Lock()
	defer d.m.Unlock()

	d.state = StateNoMedia
	d.error = ErrorNone
	d.track = 0

	if len(d.file) > 0 {
		log.Println(d.Id(), "writing", d.file)
		if fd, err := os.Create(d.file); err == nil {
			fd.Write(d.data)
			fd.Close()
		}
	}

	d.data = nil
	return nil
}

// Int triggers an interrupt on the device. The device can read from- and write to system memory.
func (d *Device) Int(mem devices.Memory) {
	switch mem.U16(cpu.R0) {
	case ReadState:
		d.m.Lock()
		mem.SetU16(cpu.R0, d.state)
		mem.SetU16(cpu.R1, d.error)
		d.m.Unlock()
	case ReadSector:
		d.readSector(mem)
	case WriteSector:
		d.writeSector(mem)
	}
}

func (d *Device) readSector(mem devices.Memory) {
	if !d.setBusy(true, mem) {
		return
	}

	dst := mem.U16(cpu.R1)
	sector := mem.U16(cpu.R2)

	if sector >= TrackCount*SectorsPerTrack {
		d.error = ErrorBadSector
		return
	}

	go func() {
		src := sector * BytesPerSector
		d.seek(sector % SectorsPerTrack)
		mem.Write(dst, d.data[src:src+BytesPerSector])
		<-time.After(SectorTransferTime)
		d.setReady()
	}()
}

func (d *Device) writeSector(mem devices.Memory) {
	if !d.setBusy(false, mem) {
		return
	}

	sector := mem.U16(cpu.R1)
	src := mem.U16(cpu.R2)

	if sector >= TrackCount*SectorsPerTrack {
		d.error = ErrorBadSector
		return
	}

	go func() {
		dst := sector * BytesPerSector
		d.seek(sector % SectorsPerTrack)
		mem.Read(src, d.data[dst:dst+BytesPerSector])
		<-time.After(SectorTransferTime)
		d.setReady()
	}()
}

// seek fakes moving the read/write head to the given track if needed.
//
// Seek time delay is simulated by multiplying the TrackSeekTime with the
// number of tracks we are shifting.
func (d *Device) seek(track int) {
	if d.track == track {
		return
	}

	delta := max(track, d.track) - min(track, d.track)
	<-time.After(time.Duration(delta) * TrackSeekTime)
	d.track = track
}

// setReady sets the device to its appropriate ready state.
func (d *Device) setReady() {
	d.m.Lock()
	defer d.m.Unlock()

	if d.readonly {
		d.state = StateReadyWP
	} else {
		d.state = StateReady
	}

	d.error = ErrorNone
}

// setBusy returns true if the device is ready for reading or writing.
// Sets device state and errors as needed.
// Additionally sets the RST/compare flag if it can successfully begin
// reading/writing.
func (d *Device) setBusy(read bool, mem devices.Memory) bool {
	d.m.Lock()
	defer d.m.Unlock()

	if d.state == StateReady || (d.state == StateReadyWP && read) {
		d.state = StateBusy
		mem.SetRSTCompare(true)
		return true
	}

	mem.SetRSTCompare(false)

	if d.state == StateReadyWP && !read {
		d.error = ErrorProtected
		return false
	}

	if d.state == StateNoMedia {
		d.error = ErrorNoMedia
		return false
	}

	d.error = ErrorBusy
	return false
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
