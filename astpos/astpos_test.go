package astpos

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/imports"
)

func TestAstPos(t *testing.T) {
	src := `package example
	
	// comment 0
	type MyStruct struct {
		// field comment 0
		name, address string
		// field comment 1
		age int
		level int
	}
	
	// comment 1
	// comment 2
	var (
		a int = 2
		b int = 12
	)

	// comment 3
	// comment 4
	func (s *MyStruct) PrintSome() {
		fmt.Println(s.name)
		fmt.Println(s.address)
		fmt.Println(s.age)
		if len(s.name) == 0 {
			fmt.Println("I am nameless!")
		}
		a, b, c, d, efg := 1, 2, 3, 4, 9000
		_ = a + b - (c * d) / efg

		for _ = range a {
			_ = "hi!"
		}
		for i := 0; i < b; i++ {
			_ = MyStruct{}
		}

		switch a {
		case 0:
			fmt.Println(a)
		default:
			a++
			a *= 10
			fmt.Println(a)
		}

		l := []*MyStruct{
			{name: "bob"},
			{name: "carl"},
		}
		_ = l

		// comment 5
		var m = "commented var"
		_ = m

		// comment 6
		// comment 7
		//
		// comment 8
		var o = 42
		_ = o
	}

	// comment 9
	func hello() int {
		fmt.Println("hello?"[:3])
		return 777
	}

	// comment 10
	var _ = 9000
	`

	expected := `package example

import "fmt"

// comment 0
type MyStruct struct {
	// field comment 0
	name, address string
	// field comment 1
	age   int
	level int
}

// comment 1
// comment 2
var (
	a int = 2
	b int = 12
)

// comment 3
// comment 4
func (s *MyStruct) PrintSome() {
	fmt.Println(s.name)
	fmt.Println(s.address)
	fmt.Println(s.age)
	if len(s.name) == 0 {
		fmt.Println("I am nameless!")
	}
	a, b, c, d, efg := 1, 2, 3, 4, 9000
	_ = a + b - (c*d)/efg
	for _ = range a {
		_ = "hi!"
	}
	for i := 0; i < b; i++ {
		_ = MyStruct{}
	}
	switch a {
	case 0:
		fmt.Println(a)
	default:
		a++
		a *= 10
		fmt.Println(a)
	}
	l := []*MyStruct{{name: "bob"}, {name: "carl"}}
	_ = l
	// comment 5
	var m = "commented var"
	_ = m
	// comment 6
	// comment 7
	//
	// comment 8
	var o = 42
	_ = o
}

// comment 9
func hello() int {
	fmt.Println("hello?"[:3])
	return 777
}

// comment 10
var _ = 9000
`

	fset := token.NewFileSet()
	opts := parser.SkipObjectResolution | parser.ParseComments
	f, err := parser.ParseFile(fset, "x.go", src, opts)
	if err != nil {
		t.Fatal(err)
	}

	f, fset = RewritePositions(f)

	result := writeAST(t, f, fset)
	if result != expected {
		t.Fatal("The re-formatted source code differs from the expected outcome")
	}
}

func writeAST(t *testing.T, f *ast.File, fset *token.FileSet) string {
	formatted := &bytes.Buffer{}
	if err := format.Node(formatted, fset, f); err != nil {
		t.Fatal(err)
	}
	importProcessed, err := imports.Process("", formatted.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}

	return string(importProcessed)
}
