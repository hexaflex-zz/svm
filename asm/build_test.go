package asm

import "testing"

func TestBuild(t *testing.T) {
	includes := []string{"../testdata/"}

	_, err := Build("examples/sprites/main.svm", includes, true)
	if err != nil {
		t.Fatal(err)
	}
}
