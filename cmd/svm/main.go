package main

import (
	"log"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	err := NewApp(parseArgs()).Run()
	if err != nil {
		log.Fatal(err)
	}
}
