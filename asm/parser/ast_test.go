package parser

import (
	"fmt"
	"testing"
)

const File = "../../testdata/examples/sprites/main.svm"

func TestAST(t *testing.T) {
	ast := NewAST()
	err := ast.ParseFile(File)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(ast)
}
