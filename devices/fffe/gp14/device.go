// Package gp14 implements the gp14 gamepad.
package gp14

import (
	"log"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

// Known interrupt operations.
const (
	isPressed = iota
	isJustPressed
	isJustReleased
)

// Button Ids.
const (
	ButtonA           = glfw.ButtonA
	ButtonB           = glfw.ButtonB
	ButtonX           = glfw.ButtonX
	ButtonY           = glfw.ButtonY
	ButtonUp          = glfw.ButtonDpadUp
	ButtonRight       = glfw.ButtonDpadRight
	ButtonDown        = glfw.ButtonDpadDown
	ButtonLeft        = glfw.ButtonDpadLeft
	ButtonLT          = glfw.ButtonLeftThumb
	ButtonRT          = glfw.ButtonRightThumb
	ButtonLeftBumper  = glfw.ButtonLeftBumper
	ButtonRightBumper = glfw.ButtonRightBumper
	ButtonBack        = glfw.ButtonBack
	ButtonStart       = glfw.ButtonStart
)

type state struct {
	pressed      bool
	justPressed  bool
	justReleased bool
}

// Device defines all internal doodads for the display.
type Device struct {
	joy         glfw.Joystick
	state       [16]state
	initialized bool
}

var _ devices.Device = &Device{}

// New creates a new device.
func New() *Device {
	return &Device{}
}

// Update updates gamepad state.
func (d *Device) Update() {
	state := d.joy.GetGamepadState()
	if state == nil {
		return
	}

	for btn, action := range state.Buttons {
		bs := d.state[btn]
		pressed := action == glfw.Press

		if pressed && !bs.pressed {
			bs.justPressed = true
		}

		if !pressed && bs.pressed {
			bs.justReleased = true
		}

		bs.pressed = pressed

		d.state[btn] = bs
	}
}

// ID returns the device id.
func (d *Device) ID() devices.ID {
	return devices.NewID(0xfffe, 0x0003)
}

// Startup initializes device resources.
// It detects any connected gamepad.
func (d *Device) Startup(devices.IntFunc) error {
	glfw.SetJoystickCallback(d.configure)

	// Check if we have a connected gamepad.
	for joy := glfw.Joystick1; joy <= glfw.JoystickLast; joy++ {
		if joy.Present() && joy.IsGamepad() {
			d.configure(joy, glfw.Connected)
			break
		}
	}

	return nil
}

// Shutdown clears up device resources.
func (d *Device) Shutdown() error {
	glfw.SetJoystickCallback(nil)
	return nil
}

// Int triggers an interrupt on the device. The device can read from- and write to system memory.
func (d *Device) Int(mem devices.Memory) {
	btn := mem.U16(cpu.R1) & 0xf
	state := &d.state[btn]

	switch mem.U16(cpu.R0) {
	case isPressed:
		mem.SetRSTCompare(state.pressed)
	case isJustPressed:
		mem.SetRSTCompare(state.justPressed)
		state.justPressed = false
	case isJustReleased:
		mem.SetRSTCompare(state.justReleased)
		state.justReleased = false
	}
}

// configure is called whenever a joystick is connected or disconnected from the system.
func (d *Device) configure(joy glfw.Joystick, event glfw.PeripheralEvent) {
	d.initialized = event == glfw.Connected && joy.IsGamepad()
	d.joy = joy

	if d.initialized {
		log.Println(d.ID(), "gamepad connected")
	} else {
		log.Println(d.ID(), "gamepad disconnected")
	}

	for btn, state := range d.state {
		state.pressed = false
		state.justPressed = false
		state.justReleased = false
		d.state[btn] = state
	}
}
