package main

import (
	"flag"
	"fmt"
	"os"
)

// Config defines program configuration.
type Config struct {
	ImportRoot  string // Root path with module and program sources.
	Program     string // Import path for the program to build.
	Output      string // Path to store output in. Filepath or empty for stdout.
	DebugBuild  bool   // Include debug symbols in build?
	DumpAST     bool   // Print a human-readable dump of the unprocessed AST.
	DumpArchive bool   // Print a human-readable dump of the compiled archive and exit.
}

// parseArgs parses command line arguments as applicable.
//
// If an error occurred, this exits the program with an appropriate message.
// When version information is requested, it is printed to stdout and the program ends cleanly.
func parseArgs() *Config {
	var c Config

	flag.Usage = func() {
		fmt.Printf("%s [options] <target import path>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&c.ImportRoot, "import", c.ImportRoot, "Root directory for all source code.")
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

	c.Program = flag.Arg(0)
	return &c
}
