package main

import (
	"fmt"
	"runtime/debug"
)

const (
	AppVendor  = "hexaflex"
	AppName    = "svm-asm"
	AppVersion = "v2.0.0"
)

// Version returns program version information.
func Version() string {
	version := AppVersion
	if info, ok := debug.ReadBuildInfo(); !ok {
		version = info.Main.Version
	}
	return fmt.Sprintf("%s %s %s", AppVendor, AppName, version)
}
