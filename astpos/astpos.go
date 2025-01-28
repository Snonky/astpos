package astpos

import (
	"go/ast"
	"go/token"
)

// Rewrites the position values of all AST nodes in the given file.
// The returned *ast.File is the same as the given one and
// the newly created *token.FileSet contains linebreak information
// for go/format or go/print.
//
// Supports doc comments on the lines directly above the
// following: Top of the file, import/const/type/var declarations,
// function declarations and struct fields.
// Block comments (/**/), end of line comments and free floating
// comments will be misplaced when printing the AST but the
// node positions could be used to correct this to some degree
// (see https://github.com/golang/go/issues/18593#issuecomment-295916961).
//
// Adds linebreaks to block-statements/-declarations and the doc
// comments. All other linebreaks should be adequately inserted by
// the formatting of go/format.
func RewritePositions(f *ast.File) (*ast.File, *token.FileSet) {
	p := newPositioner(f)
	p.positionTokens()
	return f, p.fset
}

type astPositioner struct {
	root *ast.File
	*token.File

	fset *token.FileSet

	// Position counter
	p int

	comments []*ast.CommentGroup
}

func newPositioner(root *ast.File) *astPositioner {
	fset := token.NewFileSet()
	maxInt := int(^uint(0) >> 1)
	file := fset.AddFile("x.go", 1, maxInt-2)

	positioner := &astPositioner{
		root:     root,
		File:     file,
		fset:     fset,
		p:        1,
		comments: make([]*ast.CommentGroup, 0),
	}

	return positioner
}

func (p *astPositioner) positionTokens() {
	p.root.FileStart = 1
	p.traverse(p.root)
	p.root.FileEnd = p.pc()
	p.root.Comments = p.comments
}

// Returns the current position counter
func (p *astPositioner) pc() token.Pos {
	return token.Pos(p.p)
}

func (p *astPositioner) newline() {
	p.AddLine(p.p)
	p.moveN(1)
}

func (p *astPositioner) move(t token.Token) {
	p.p += len(t.String())
}

func (p *astPositioner) moveStr(s string) {
	p.p += len(s)
}

func (p *astPositioner) moveN(n int) {
	p.p += n
}

func (p *astPositioner) traverse(node ast.Node) {
	if node == nil {
		return
	}
	ast.Inspect(node, p.down)
}

func traverseList[Slice ~[]E, E ast.Node](p *astPositioner, nodes Slice) {
	// Cannot be a method because of the type params
	for _, n := range nodes {
		p.traverse(n)
	}
}

// Sets the position fields of the encountered node type
// and moves the position counter up accordingly.
//
// It operates in the pre-order of the tree traversal
// (going "down" the tree) but frequently branches off
// when nodes have children that determine their own
// position values.
//
// For maintainability, the switch statement is sorted alphabetically
// and thus ordered the same as documentation page of the go/ast package
// (https://pkg.go.dev/go/ast#pkg-types).
func (p *astPositioner) down(n ast.Node) bool {
	pc := p.pc
	switch n := n.(type) {
	case *ast.ArrayType:
		n.Lbrack = pc()
		p.move(token.LBRACK)
		p.traverse(n.Len)
		p.move(token.RBRACK)
		p.traverse(n.Elt)
		return false

	case *ast.AssignStmt:
		traverseList(p, n.Lhs)
		n.TokPos = pc()
		p.move(n.Tok)
		traverseList(p, n.Rhs)
		return false

	case *ast.BasicLit:
		n.ValuePos = pc()
		p.moveStr(n.Value)

	case *ast.BinaryExpr:
		p.traverse(n.X)
		n.OpPos = pc()
		p.move(n.Op)
		p.traverse(n.Y)
		return false

	case *ast.BlockStmt:
		n.Lbrace = pc()
		p.move(token.LBRACE)
		p.newline()
		traverseList(p, n.List)
		n.Rbrace = pc()
		p.move(token.RBRACE)
		p.newline()
		return false

	case *ast.BranchStmt:
		n.TokPos = pc()
		p.move(n.Tok)

	case *ast.CallExpr:
		p.traverse(n.Fun)
		n.Lparen = pc()
		traverseList(p, n.Args)
		if n.Ellipsis != token.NoPos {
			n.Ellipsis = pc()
			p.move(token.ELLIPSIS)
		}
		n.Rparen = pc()
		return false

	case *ast.CaseClause:
		n.Case = pc()
		if n.List == nil {
			p.move(token.DEFAULT)
		} else {
			p.move(token.CASE)
		}
		traverseList(p, n.List)
		n.Colon = pc()
		p.move(token.COLON)
		p.newline()
		traverseList(p, n.Body)
		return false

	case *ast.ChanType:
		arrowFirst := n.Begin == n.Arrow
		n.Begin = pc()
		if n.Arrow != token.NoPos && arrowFirst {
			n.Arrow = pc()
			p.move(token.ARROW)
		}
		p.move(token.CHAN)
		if n.Arrow != token.NoPos && !arrowFirst {
			n.Arrow = pc()
			p.move(token.ARROW)
		}

	case *ast.CommClause:
		n.Case = pc()
		if n.Comm == nil {
			p.move(token.DEFAULT)
		} else {
			p.move(token.CASE)
		}
		p.traverse(n.Comm)
		n.Colon = pc()
		p.move(token.COLON)
		p.newline()
		traverseList(p, n.Body)
		return false

	// Comments handled separately

	case *ast.CompositeLit:
		p.traverse(n.Type)
		n.Lbrace = pc()
		p.move(token.LBRACE)
		traverseList(p, n.Elts)
		n.Rbrace = pc()
		p.move(token.RBRACE)
		return false

	case *ast.DeferStmt:
		n.Defer = pc()
		p.move(token.DEFER)

	case *ast.Ellipsis:
		n.Ellipsis = pc()
		p.move(token.ELLIPSIS)

	case *ast.EmptyStmt:
		n.Semicolon = pc()
		if !n.Implicit {
			p.move(token.SEMICOLON)
		}

	case *ast.Field:
		p.handleComment(n.Doc)

	case *ast.FieldList:
		if n.Opening != token.NoPos {
			n.Opening = pc()
			p.moveN(1)
		}
		traverseList(p, n.List)
		if n.Closing != token.NoPos {
			n.Closing = pc()
			p.moveN(1)
		}
		return false

	case *ast.File:
		p.handleComment(n.Doc)
		n.Package = pc()
		p.move(token.PACKAGE)
		p.moveStr(" ")
		p.traverse(n.Name)
		p.newline()
		traverseList(p, n.Decls)
		return false

	case *ast.ForStmt:
		n.For = pc()
		p.move(token.FOR)

	case *ast.FuncDecl:
		p.handleComment(n.Doc)
		if n.Recv != nil {
			p.traverse(n.Recv)
		}
		p.traverse(n.Name)
		p.traverse(n.Type)
		p.traverse(n.Body)
		p.newline()
		return false

	case *ast.FuncType:
		n.Func = pc()
		p.move(token.FUNC)

	case *ast.GenDecl:
		p.handleComment(n.Doc)
		n.TokPos = pc()
		p.move(n.Tok)
		if n.Lparen != token.NoPos {
			n.Lparen = pc()
			p.move(token.LPAREN)
			p.newline()
		}
		traverseList(p, n.Specs)
		if n.Rparen != token.NoPos {
			n.Rparen = pc()
			p.move(token.RPAREN)
			p.newline()
		}
		return false

	case *ast.GoStmt:
		n.Go = pc()
		p.move(token.GO)

	case *ast.Ident:
		n.NamePos = pc()
		p.moveStr(n.Name)

	case *ast.IfStmt:
		n.If = pc()
		p.move(token.IF)

	case *ast.ImportSpec:
		p.handleComment(n.Doc)

	case *ast.IncDecStmt:
		p.traverse(n.X)
		n.TokPos = pc()
		p.move(n.Tok)
		return false

	case *ast.IndexExpr:
		p.traverse(n.X)
		n.Lbrack = pc()
		p.move(token.LBRACK)
		p.traverse(n.Index)
		n.Rbrack = pc()
		p.move(token.RBRACK)
		return false

	case *ast.IndexListExpr:
		p.traverse(n.X)
		n.Lbrack = pc()
		p.move(token.LBRACK)
		traverseList(p, n.Indices)
		n.Rbrack = pc()
		p.move(token.RBRACK)
		return false

	case *ast.InterfaceType:
		n.Interface = pc()
		p.move(token.INTERFACE)

	case *ast.KeyValueExpr:
		p.traverse(n.Key)
		n.Colon = pc()
		p.move(token.COLON)
		p.traverse(n.Value)
		return false

	case *ast.LabeledStmt:
		p.traverse(n.Label)
		n.Colon = pc()
		p.move(token.COLON)
		p.traverse(n.Stmt)
		return false

	case *ast.MapType:
		n.Map = pc()
		p.move(token.MAP)

	case *ast.ParenExpr:
		n.Lparen = pc()
		p.move(token.LPAREN)
		p.traverse(n.X)
		n.Rparen = pc()
		p.move(token.RPAREN)
		return false

	case *ast.RangeStmt:
		n.For = pc()
		p.move(token.FOR)
		p.traverse(n.Key)
		p.traverse(n.Value)
		if n.Tok != token.ILLEGAL {
			n.TokPos = pc()
			p.move(n.Tok)
		}
		n.Range = pc()
		p.move(token.RANGE)
		p.traverse(n.X)
		p.traverse(n.Body)
		return false

	case *ast.ReturnStmt:
		n.Return = pc()
		p.move(token.RETURN)

	case *ast.SelectStmt:
		n.Select = pc()
		p.move(token.SELECT)

	case *ast.SendStmt:
		p.traverse(n.Chan)
		n.Arrow = pc()
		p.move(token.ARROW)
		p.traverse(n.Value)
		return false

	case *ast.SliceExpr:
		p.traverse(n.X)
		n.Lbrack = pc()
		p.move(token.LBRACK)
		p.traverse(n.Low)
		p.traverse(n.High)
		p.traverse(n.Max)
		n.Rbrack = pc()
		p.move(token.RBRACK)
		return false

	case *ast.StarExpr:
		n.Star = pc()
		p.moveStr("*")

	case *ast.StructType:
		n.Struct = pc()
		p.move(token.STRUCT)

	case *ast.SwitchStmt:
		n.Switch = pc()
		p.move(token.SWITCH)

	case *ast.TypeAssertExpr:
		p.traverse(n.X)
		n.Lparen = pc()
		p.move(token.LPAREN)
		p.traverse(n.Type)
		n.Rparen = pc()
		p.move(token.RPAREN)
		return false

	case *ast.TypeSpec:
		p.handleComment(n.Doc)
		if n.Assign == token.NoPos {
			return true
		}
		p.traverse(n.Name)
		p.traverse(n.TypeParams)
		n.Assign = pc()
		p.move(token.ASSIGN)
		p.traverse(n.Type)
		return false

	case *ast.TypeSwitchStmt:
		n.Switch = pc()
		p.move(token.SWITCH)

	case *ast.UnaryExpr:
		n.OpPos = pc()
		p.move(n.Op)

	}

	return true
}

func (p *astPositioner) handleComment(c *ast.CommentGroup) {
	if c == nil {
		return
	}

	p.comments = append(p.comments, c)
	p.newline()
	for _, c := range c.List {
		c.Slash = p.pc()
		p.moveStr(c.Text)
		p.newline()
	}
}
