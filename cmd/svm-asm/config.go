package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config defines program configuration.
type Config struct {
	Includes    []string // Include search paths.
	Input       string   // Input source file to build.
	Output      string   // Path to store output in.
	DebugBuild  bool     // Include debug symbols in build?
	DumpAST     bool     // Print a human-readable dump of the unprocessed AST.
	DumpArchive bool     // Print a human-readable dump of the compiled archive and exit.
}

// parseArgs parses command line arguments as applicable.
//
// If an error occurred, this exits the program with an appropriate message.
// When version information is requested, it is printed to stdout and the program ends cleanly.
func parseArgs() *Config {
	var c Config
	c.Output = "out.a"

	flag.Usage = func() {
		fmt.Printf("%s [options] <input source file>\n", os.Args[0])
		flag.PrintDefaults()
	}

	includes := flag.String("include", "", "Colon-separated list of include search paths.")
	flag.StringVar(&c.Output, "out", c.Output, "Output file.")
	flag.BoolVar(&c.DebugBuild, "debug", c.DebugBuild, "Include debug symbols in the build. Creates an extra <out>.dbg file as output.")
	flag.BoolVar(&c.DumpAST, "dump-ast", c.DumpAST, "Print a human-readable version of the unprocessed AST to stdout.")
	flag.BoolVar(&c.DumpArchive, "dump-ar", c.DumpArchive, "Print a human-readable version of the compiled binary to stdout.")
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

	if len(*includes) > 0 {
		c.Includes = filteredSplit(*includes, ":")
	}

	c.Input = flag.Arg(0)
	return &c
}

// filteredSplit splits value by sep and returns the resulting list, minus empty entries.
func filteredSplit(value, sep string) []string {
	out := strings.Split(value, sep)
	for i := 0; i < len(out); i++ {
		out[i] = strings.TrimSpace(out[i])
		if len(out[i]) == 0 {
			copy(out[i:], out[i+1:])
			out = out[:len(out)-1]
		}
	}
	return out
}
