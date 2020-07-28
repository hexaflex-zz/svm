package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	TrackCount      = 80
	SectorsPerTrack = 18
	BytesPerSector  = 1024
	FloppySize      = TrackCount * SectorsPerTrack * BytesPerSector
)

func main() {
	config := parseArgs()

	switch {
	case config.New:
		createImage(config.File)
	}
}

// createImage creates a new, empty image file.
func createImage(file string) {
	// Ensure the target directory exists.
	dir, _ := filepath.Split(file)

	err := os.MkdirAll(dir, 0744)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fd, err := os.Create(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer fd.Close()

	var buf [FloppySize]byte
	_, err = fd.Write(buf[:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
