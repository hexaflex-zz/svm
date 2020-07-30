package devices

import (
	"log"

	"github.com/pkg/errors"
)

// IntFunc represents a Hardware Interrupt handler.
type IntFunc func(int)

// Device represents a peripheral device.
// It can interact with a program through interrupts.
type Device interface {
	// Id yields the manufacturer and serial number for the device.
	Id() Id

	// Startup initializes internal resources.
	//
	// IntFuncrepresents an interrupt handler the device can
	// use to send interrupt requests to the CPU.
	Startup(IntFunc) error

	// Shutdown cleans up internal resources.
	Shutdown() error

	// Int triggers an interrupt on the device and is called
	// through a program's INT instruction. It accepts the system's
	// memory bank.
	Int(Memory)
}

// Map contains a list of registered peripherals.
type Map []Device

// Connect adds the given device to the device map.
// Returns false if the device type is already present in the set.
func (dm *Map) Connect(dev Device) bool {
	if (*dm).Find(dev.Id()) > -1 {
		return false
	}

	*dm = append(*dm, dev)
	return true
}

// Int triggers an interrupt on the device with the given index.
// Returns false if the index is not valid.
func (dm Map) Int(index int, m Memory) bool {
	if index < 0 || index >= len(dm) {
		return false
	}
	dm[index].Int(m)
	return true
}

// Startup initializes internal resources.
func (dm Map) Startup(f IntFunc) error {
	var errorset ErrorSet

	for _, dev := range dm {
		log.Println(dev.Id(), "startup")
		if err := dev.Startup(f); err != nil {
			errorset.Append(errors.Wrapf(err, "%s", dev.Id()))
		}
	}

	if errorset.Len() == 0 {
		return nil
	}

	return errorset
}

// Shutdown cleans up internal resources.
func (dm Map) Shutdown() error {
	var errorset ErrorSet

	for _, dev := range dm {
		log.Println(dev.Id(), "shutdown")
		if err := dev.Shutdown(); err != nil {
			errorset.Append(errors.Wrapf(err, "%s", dev.Id()))
		}
	}

	if errorset.Len() == 0 {
		return nil
	}

	return errorset
}

// Find returns the index for the device with the given id.
// Returns -1 if it can't be found.
func (dm Map) Find(id Id) int {
	for i, dev := range dm {
		if dev.Id() == id {
			return i
		}
	}
	return -1
}
