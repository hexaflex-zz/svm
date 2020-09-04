package main

import (
	"fmt"
	"runtime/debug"
)

// Various version related constants.
const (
	AppVendor  = "hexaflex"
	AppName    = "svm"
	AppVersion = "v6.1.3"
)

// Version returns program version information.
func Version() string {
	version := AppVersion
	if info, ok := debug.ReadBuildInfo(); !ok {
		version = info.Main.Version
	}
	return fmt.Sprintf("%s %s %s", AppVendor, AppName, version)
}
