package main

import (
	"flag"
	"fmt"
	"os"
)

// Config defines program configuration.
type Config struct {
	Input  string // Input image file.
	Output string // Output file. Leave empty for stdout.
}

// parseArgs parses command line arguments as applicable.
//
// If an error occurred, this exits the program with an appropriate message.
// When version information is requested, it is printed to stdout and the program ends cleanly.
func parseArgs() *Config {
	var c Config

	flag.Usage = func() {
		fmt.Printf("%s [options] <image file>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&c.Output, "out", c.Output, "File path to write output to. Leave empty to use stdout.")
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

	c.Input = flag.Arg(0)
	return &c
}
