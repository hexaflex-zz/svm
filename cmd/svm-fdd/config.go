package main

import (
	"flag"
	"fmt"
	"os"
)

// Config defines program configuration.
type Config struct {
	Inputs []string // Optional input files.
	Output string   // Target image file.
}

// parseArgs parses command line arguments as applicable.
//
// If an error occurred, this exits the program with an appropriate message.
// When version information is requested, it is printed to stdout and the program ends cleanly.
func parseArgs() *Config {
	var c Config

	flag.Usage = func() {
		fmt.Printf("%s [options] [<input files>]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&c.Output, "out", c.Output, "Output file to generate (Not optional).")
	version := flag.Bool("version", false, "Display version information.")
	flag.Parse()

	if *version {
		fmt.Println(Version())
		os.Exit(0)
	}

	if len(c.Output) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	c.Inputs = flag.Args()
	return &c
}
