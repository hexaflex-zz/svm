package main

import (
	"flag"
	"fmt"
	"os"
)

// Config defines program configuration.
type Config struct {
	File string // Target image file.
	New  bool   // Create a new, blank image file.
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

	flag.BoolVar(&c.New, "new", c.New, "Create a new, empty image file.")
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

	c.File = flag.Arg(0)
	return &c
}
