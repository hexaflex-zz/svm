package asm

import "testing"

func TestBuild(t *testing.T) {
	_, err := Build("../testdata", "test", true)
	if err != nil {
		t.Fatal(err)
	}
}
