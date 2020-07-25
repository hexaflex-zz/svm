package vm

import (
	"log"

	"github.com/pkg/errors"
)

// Device represents a peripheral device.
// It can interact with a program through interrupts.
type Device interface {
	// Id yields the manufacturer and serial number for the device.
	Id() Id

	// Startup initializes internal resources.
	Startup() error

	// Shutdown cleans up internal resources.
	Shutdown() error

	// Int triggers an interrupt on the device and is called
	// through a program's INT instruction. It accepts the system's
	// memory bank.
	Int(Memory)
}

// DeviceMap contains a list of registered peripherals.
type DeviceMap []Device

// Connect adds the given device to the device map.
// Returns false if the device type is already present in the set.
func (dm *DeviceMap) Connect(dev Device) bool {
	if (*dm).Find(dev.Id()) > -1 {
		return false
	}

	*dm = append(*dm, dev)
	return true
}

// Int triggers an interrupt on the device with the given index.
// Returns false if the index is not valid.
func (dm DeviceMap) Int(index int, m Memory) bool {
	if index < 0 || index >= len(dm) {
		return false
	}
	dm[index].Int(m)
	return true
}

// Startup initializes internal resources.
func (dm DeviceMap) Startup() error {
	var errorset ErrorSet

	for _, dev := range dm {
		log.Println(dev.Id(), "startup")
		if err := dev.Startup(); err != nil {
			errorset.Append(errors.Wrapf(err, "%s", dev.Id()))
		}
	}

	if errorset.Len() == 0 {
		return nil
	}

	return errorset
}

// Shutdown cleans up internal resources.
func (dm DeviceMap) Shutdown() error {
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
func (dm DeviceMap) Find(id Id) int {
	for i, dev := range dm {
		if dev.Id() == id {
			return i
		}
	}
	return -1
}
