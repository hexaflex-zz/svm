package asm

import "testing"

func TestBuild(t *testing.T) {
	_, err := Build("../testdata", "examples/sprites", true)
	if err != nil {
		t.Fatal(err)
	}
}
