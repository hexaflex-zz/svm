package main

import (
	"bytes"
	"fmt"
	"io"
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
	createImage(config.Output, config.Inputs)
}

// createImage creates an image file from zero or more input files.
func createImage(file string, inputs []string) {
	var buf bytes.Buffer
	for _, file := range inputs {
		copyFile(file, &buf)
	}

	diff := FloppySize - buf.Len()
	if diff < 0 {
		buf.Truncate(FloppySize)
	} else if diff > 0 {
		if _, err := buf.Write(make([]byte, diff)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	w, close := makeWriter(file)
	defer close()

	if _, err := io.Copy(w, &buf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// copyFile copies the contents of file into dst.
func copyFile(file string, dst io.Writer) {
	src, err := os.Open(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer src.Close()

	if _, err := io.Copy(dst, src); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// makeWriter opens the given file for writing.
// Returns stdout if the name is empty.
func makeWriter(file string) (io.Writer, func()) {
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

	return fd, func() { fd.Close() }
}
