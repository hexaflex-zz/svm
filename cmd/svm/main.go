package main

import (
	"log"
	"os"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	os.Args = append(os.Args, "-debug", "../../testdata/test.a")

	err := NewApp(parseArgs()).Run()
	if err != nil {
		log.Fatal(err)
	}
}
