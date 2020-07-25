package parser

import (
	"fmt"
	"testing"
)

const File = "../../testdata/test/main.svm"

func TestAST(t *testing.T) {
	ast := NewAST()
	err := ast.ParseFile(File)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(ast.Dump())
}
