package ar

import (
	"bytes"
	"reflect"
	"testing"
)

func TestAR(t *testing.T) {
	ar := New()
	ar.Debug.Files = append(ar.Debug.Files,
		"path/to/file1.asm",
		"path/to/file2.asm")
	ar.Debug.Symbols = append(ar.Debug.Symbols,
		DebugData{0, 10, 20, 30, 40, Breakpoint},
		DebugData{50, 60, 70, 80, 90, 0})
	ar.Instructions = append(ar.Instructions,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

	var buf bytes.Buffer
	if err := ar.Save(&buf); err != nil {
		t.Fatal(err)
	}

	br := New()
	if err := br.Load(&buf); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(ar, br) {
		t.Fatalf("archive mismatch:\nhave: %v\nwant: %v", br, ar)
	}
}
