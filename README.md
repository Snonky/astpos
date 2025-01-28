# Go AST Position Rewrite for Code Generation

When parsing a go source file with the `go/ast` package the result is the AST itself and a FileSet struct that stores file line information.

However when the goal is to generate go source programmatically from scratch there is evidently no way to get a FileSet as there is no file.
The `go/format` and `go/printer` packages can still produce correct code with
an empty FileSet but the code looks compacted and can not contain comments.

This module is able to produce a FileSet directly from an AST to preserve
doc comments and make the code a bit more readable.

## Usage

Call `astpos.RewritePositions`

```
func RewritePositions(f *ast.File) (*ast.File, *token.FileSet)
```

All nodes will have their position(s) set and the FileSet can be used in the formatting step.

## Demo

<table>
<tr>
<th> This is from an AST with all its positions set to <code>0</code> and an empty FileSet</th>
<th> This is from the same AST with the Positions and FileSet filled in by <code>astpos.RewritePositions</code></th>
</tr>
<tr>
<td>

```go
package // Example file
// Documentation for MyStruct
// It does it all!
// The name
// The status
// This method has all the power!
// It returns nothing.
// Returns the name
// This is a package-level variable
//
// It contains a greeting
astpos

import "fmt"

type MyStruct struct {
	name   string
	status int
}

func (m *MyStruct) PowerMethod() {
	if m.name == "" {
		fmt.Println("I am nameless!")
	}
	m.name = m.name + m.name
}

func (m *MyStruct) GetName() string {
	return m.name
}

var PackageVar = "hello!"
```

</td>
<td>

```go
// Example file
package astpos

import "fmt"

// Documentation for MyStruct
// It does it all!
type MyStruct struct {
	// The name
	name string
	// The status
	status int
}

// This method has all the power!
// It returns nothing.
func (m *MyStruct) PowerMethod() {
	if m.name == "" {
		fmt.Println("I am nameless!")
	}
	m.name = m.name + m.name
}

// Returns the name
func (m *MyStruct) GetName() string {
	return m.name
}

// This is a package-level variable
//
// It contains a greeting
var PackageVar = "hello!"
```

</td>
</tr>
</table>

## Example

Minimal example on how to put `astpos` to use:
```go
// Programmatically create a tiny go AST:
variableDeclaration := &ast.GenDecl{
	Tok: token.VAR,
	Specs: []ast.Spec{&ast.ValueSpec{
		Names:  []*ast.Ident{ast.NewIdent("myString")},
		Type:   ast.NewIdent("string"),
		Values: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"My String\""}},
	}},
}
f := &ast.File{
	Name: ast.NewIdent("packagename"),
	Doc: &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// File doc comment"}},
	},
	Decls: []ast.Decl{variableDeclaration},
}

// Use go/format + astpos to print the AST as code:

// Not like this: Demonstrate behaviour without positions and empty fset
var sb strings.Builder
fset := token.NewFileSet()
format.Node(&sb, fset, f)
malformedCode := sb.String()

// Make it work: Synthesize the FileSet and rewrite the positions
f, fset = astpos.RewritePositions(f) //                                    <--- The package call
sb.Reset()
format.Node(&sb, fset, f)
sourceCode := sb.String()

fmt.Println("---Malformed Source Code---")
fmt.Println(malformedCode)
fmt.Println("---Good Source Code---")
fmt.Println(sourceCode)
```

The output of this snippet is

```
---Malformed Source Code---
package// File doc comment
packagename

var myString string = "My String"

---Good Source Code---
// File doc comment
package packagename

var myString string = "My String"
```