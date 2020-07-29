package main

import (
	"io"
	"time"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

// CPUController controls the execution of a CPU.
type CPUController struct {
	cpu        *cpu.CPU
	start      time.Time
	cycleCount uint64
	running    bool
}

// NewCPUController creates a new CPU controller.
func NewCPUController(trace cpu.TraceFunc, devices ...devices.Device) *CPUController {
	cpu := cpu.New(trace)

	for _, dev := range devices {
		cpu.Connect(dev)
	}

	return &CPUController{
		cpu: cpu,
	}
}

// Running returns true if the CPU is currently running.
func (c *CPUController) Running() bool {
	return c.running
}

// Frequency returns the current clock frequency in herz.
func (c *CPUController) Frequency() float64 {
	if c.running {
		return float64(c.cycleCount) / time.Since(c.start).Seconds()
	} else {
		return 0
	}
}

// ToggleRun starts or stops program execution.
func (c *CPUController) ToggleRun() {
	c.setRunning(!c.running)
}

// Start begins execution of the program.
func (c *CPUController) Start() {
	c.setRunning(true)
}

// Stop pauses execution of the program.
func (c *CPUController) Stop() {
	c.setRunning(false)
}

// Step performs a single exection step.
func (c *CPUController) Step() error {
	c.cycleCount++

	err := c.cpu.Step()
	if err != nil {
		c.setRunning(false)
		if err != io.EOF {
			return err
		}
	}

	return nil
}

// Memory returns the cpu's internal memory bank.
func (c *CPUController) Memory() devices.Memory {
	return c.cpu.Memory()
}

// Startup loads the given program and initializes the cpu and connected peripherals.
func (c *CPUController) Startup() error {
	return c.cpu.Startup()
}

// Shutdown disposes of CPU and peripheral resources.
func (c *CPUController) Shutdown() error {
	return c.cpu.Shutdown()
}

// setRunning determines of the CPU is running or is paused.
func (c *CPUController) setRunning(v bool) {
	c.running = v
	c.start = time.Now()
	c.cycleCount = 0
}
