package main

import (
	"flag"
	"fmt"
	"os"
)

// Config defines program configuration.
type Config struct {
	Image       string // Path to the image file to load.
	ScaleFactor int    // Amount by which each pixel is scaled (virtual resolution)
	Fullscreen  bool   // Run in fullscreen?
	Debug       bool   // Enable debug mode? This handles breakpoints if enabled.
	PrintTrace  bool   // Print instruction trace data?
	Readonly    bool   // Is the image read-only?
}

// parseArgs parses command line arguments as applicable.
//
// If an error occurred, this exits the program with an appropriate message.
// When version information is requested, it is printed to stdout and the program ends cleanly.
func parseArgs() *Config {
	var c Config
	c.ScaleFactor = 2
	c.Fullscreen = false
	c.Debug = false
	c.PrintTrace = false

	flag.Usage = func() {
		fmt.Printf("%s [options] <image file>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVar(&c.Debug, "debug", c.Debug, "Run in debug mode.")
	flag.BoolVar(&c.Readonly, "readonly", c.Readonly, "Is the loaded image file write protected?")
	flag.IntVar(&c.ScaleFactor, "scale-factor", c.ScaleFactor, "Pixel scale factor for the display.")
	flag.BoolVar(&c.Fullscreen, "fullscreen", c.Fullscreen, "Run the display in fullscreen or windowed mode.")

	version := flag.Bool("version", false, "Display version information.")
	flag.Parse()

	if *version {
		fmt.Println(Version())
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	c.Image = flag.Arg(0)
	c.PrintTrace = c.Debug
	return &c
}
