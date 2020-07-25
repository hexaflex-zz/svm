package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/pkg/errors"

	"github.com/hexaflex/svm/arch"
	"github.com/hexaflex/svm/asm/ar"
	"github.com/hexaflex/svm/devices/fffe/cpu"
	"github.com/hexaflex/svm/devices/fffe/gp14"
	"github.com/hexaflex/svm/devices/fffe/sprdi"
)

// App defines application context.
type App struct {
	config       *Config        // Application configuration.
	window       *glfw.Window   // OpenGL/GLFW context.
	cpu          *CPUController // VM with program to be run.
	display      *sprdi.Device  // Virtual display peripheral.
	gamepad      *gp14.Device   // Virtual gamepad peripheral.
	debugData    *ar.Debug      // Debug data stored in an archive.
	titleUpdated time.Time      // Value used to periodically update window title.
	lastRendered time.Time      // Last time a frame was rendered.
}

// NewApp creates a new application instance using the given configuration.
func NewApp(config *Config) *App {
	var a App
	a.config = config
	a.display = sprdi.New()
	a.gamepad = gp14.New()
	a.cpu = NewCPUController(a.printTrace, a.display, a.gamepad)
	return &a
}

// Run runs the application and does not return until it is finished
// or an error occured during initialization.
func (a *App) Run() error {
	if err := a.initGL(); err != nil {
		return err
	}

	defer a.dispose()

	log.Println(Version())
	printHelp()
	a.loadProgram()

	if !a.config.Debug {
		a.cpu.Start()
	}

	for !a.window.ShouldClose() {
		a.mainLoop()
	}

	return nil
}

// mainLoop performs all main loop operations.
func (a *App) mainLoop() {
	a.gamepad.Update()

	if a.cpu.Running() {
		err := a.cpu.Step()
		if err != nil {
			log.Println(err)
		}
	}

	// Periodically render display contents.
	if time.Since(a.lastRendered) >= time.Second/60 {
		a.lastRendered = time.Now()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.display.Draw()
		a.window.SwapBuffers()
	}

	// Periodically update the window title to show the current cpu clock frequency.
	if time.Since(a.titleUpdated) >= time.Second*2 {
		a.titleUpdated = time.Now()
		freq := prettyFrequency(a.cpu.Frequency())
		a.window.SetTitle(fmt.Sprintf("%s %s - %s", AppName, AppVersion, freq))
	}

	glfw.PollEvents()
}

// dispose ensures openGL/GLFW and other resources are cleaned up.
func (a *App) dispose() {
	a.cpu.Stop()
	a.cpu.Shutdown()

	if a.window != nil {
		a.window.Destroy()
		a.window = nil
	}

	glfw.Terminate()
}

func (a *App) keyCallback(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action != glfw.Press {
		return
	}

	var err error

	switch key {
	case glfw.KeyEscape:
		a.window.SetShouldClose(true)
	case glfw.KeyF1:
		printHelp()
	case glfw.KeyF2:
		a.config.Debug = !a.config.Debug
	case glfw.KeyF5:
		err = a.loadProgram()
	case glfw.KeyQ:
		a.cpu.ToggleRun()
	case glfw.KeyE:
		err = a.cpu.Step()
	case glfw.KeyD:
		a.config.PrintTrace = !a.config.PrintTrace
	}

	if err != nil {
		log.Println(err)
	}
}

// initGL initializes GLFW and openGL.
func (a *App) initGL() error {
	err := glfw.Init()
	if err != nil {
		return errors.Wrapf(err, "glfw.Init failed")
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.True)
	glfw.WindowHint(glfw.Focused, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 6)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	var monitor *glfw.Monitor

	width := sprdi.DisplayWidth * a.config.ScaleFactor
	height := sprdi.DisplayHeight * a.config.ScaleFactor

	if a.config.Fullscreen {
		monitor = glfw.GetPrimaryMonitor()
		mode := monitor.GetVideoMode()

		width = mode.Width
		height = mode.Height

		glfw.WindowHint(glfw.Decorated, glfw.False)
		glfw.WindowHint(glfw.Maximized, glfw.True)
	} else {
		glfw.WindowHint(glfw.Decorated, glfw.True)
		glfw.WindowHint(glfw.Maximized, glfw.False)
	}

	a.window, err = glfw.CreateWindow(width, height, "", monitor, nil)
	if err != nil {
		a.dispose()
		return errors.Wrapf(err, "glfw.CreateWindow failed")
	}

	a.window.MakeContextCurrent()
	a.window.SetKeyCallback(a.keyCallback)

	glfw.SwapInterval(0)

	err = gl.Init()
	if err != nil {
		a.dispose()
		return errors.Wrapf(err, "gl.Init failed")
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0, 0, 0, 1.0)
	return nil
}

// loadProgram loads the current program from disk and restarts the cpu.
func (a *App) loadProgram() error {
	log.Println("loading", a.config.Program)

	fd, err := os.Open(a.config.Program)
	if err != nil {
		return err
	}

	defer fd.Close()

	ar := ar.New()
	if err = ar.Load(fd); err != nil {
		return err
	}

	a.debugData = &ar.Debug

	a.cpu.Shutdown()
	a.cpu.Startup(ar.Instructions, ar.Entrypoint)
	return nil
}

// printTrace prints instruction trace data. This can be toggled
// on off through a.config.PrintTrace.
//
// It also ensures execution is stopped if the given instruction has a breakpoint
// associated with it. This only happens if a.config.Debug is true.
func (a *App) printTrace(i *cpu.Instruction) {
	var dbg *ar.DebugData

	// Pause execution if we are in debug mode and this instruction has a breakpoint.
	if a.config.Debug && a.debugData != nil {
		dbg = a.debugData.Find(i.IP)
		if dbg != nil {
			if dbg.Flags&ar.Breakpoint != 0 {
				a.cpu.Stop()
			}
		}
	}

	// Print instruction trace data if applicable.
	if !a.config.PrintTrace {
		return
	}

	var sb strings.Builder
	sb.Grow(120)

	name, _ := arch.Name(i.Opcode)
	argc := arch.Argc(i.Opcode)

	for j := 0; j < argc; j++ {
		argv := i.Args[j]

		if argv.Mode == cpu.Register {
			index := (argv.Address - cpu.UserMemoryCapacity) / 2
			fmt.Fprintf(&sb, "%4s %04x", arch.RegiserName(index), uint16(argv.Value))

		} else if argv.Mode == cpu.Address {
			fmt.Fprintf(&sb, "%04x %04x", argv.Address, uint16(argv.Value))

		} else {
			fmt.Fprintf(&sb, "%04x", uint16(argv.Value))
		}

		if j < argc-1 {
			fmt.Fprintf(&sb, ", ")
		}
	}

	// Add source context of it is available.
	if dbg != nil {
		pad(&sb, 40)
		file := a.debugData.Files[dbg.File]
		fmt.Fprintf(&sb, " %s:%d:%d", file, dbg.Line, dbg.Col)
	}

	fmt.Printf("%04x %5s  %s\n", i.IP, name, sb.String())
}

// printHelp writes a short voerview of supported shortcut keys to stdout.
func printHelp() {
	var sb strings.Builder
	sb.WriteString("shortcut keys:\n")
	sb.WriteString(" ESC      Exit the cpu.\n")
	sb.WriteString(" F1       Display this help.\n")
	sb.WriteString(" F2       Enable/Disable debug mode.\n")
	sb.WriteString(" F5       (re)load the program from disk and reset the cpu.\n")
	sb.WriteString(" Q        Start/Stop program execution.\n")
	sb.WriteString(" E        Perform a single execution step.\n")
	sb.WriteString(" D        Enable/Disable debug trace output.\n")
	sb.WriteString(" V        Enable/Disable VSync.")
	log.Println(sb.String())
}

// pad padds sb with spaces until it reaches the given size.
var pad = func() func(*strings.Builder, int) {
	set := strings.Repeat(" ", 80)
	return func(sb *strings.Builder, size int) {
		if sb.Len() >= size {
			return
		}
		if size > len(set) {
			size = len(set)
		}
		if size < sb.Len() {
			size = sb.Len()
		}
		sb.WriteString(set[:size-sb.Len()])
	}
}()

// prettyFrequency returns a human-readable version of the given clock frequency in herz.
func prettyFrequency(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.2f GHz", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2f MHz", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.2f KHz", v/1e3)
	default:
		return fmt.Sprintf("%.2f Hz", v)
	}
}

func _bool(v bool) int {
	if v {
		return 1
	}
	return 0
}
