package parser

import "minicc/pkg/lexer"

// Program is the root AST node — a list of top-level declarations.
type Program struct {
	Declarations []Declaration
}

// Declaration is implemented by VarDecl and FuncDecl.
type Declaration interface {
	declNode()
}

// Statement is implemented by all statement nodes.
type Statement interface {
	stmtNode()
}

// Expression is implemented by all expression nodes.
type Expression interface {
	exprNode()
}

// --- Declarations ---

type VarDecl struct {
	Name string
	Init Expression // nil if no initializer
	Line int
}

type FuncDecl struct {
	Name   string
	Params []Param
	Body   *BlockStmt
	Line   int
}

type Param struct {
	Name string
	Line int
}

func (*VarDecl) declNode()  {}
func (*FuncDecl) declNode() {}

// VarDecl can also appear as a statement inside blocks.
func (*VarDecl) stmtNode() {}

// --- Statements ---

type BlockStmt struct {
	Items []Statement
	Line  int
}

type ExprStmt struct {
	Expr Expression
	Line int
}

type IfStmt struct {
	Condition Expression
	Then      Statement
	Else      Statement // nil if no else
	Line      int
}

type WhileStmt struct {
	Condition Expression
	Body      Statement
	Line      int
}

type ReturnStmt struct {
	Value Expression
	Line  int
}

func (*BlockStmt) stmtNode()  {}
func (*ExprStmt) stmtNode()   {}
func (*IfStmt) stmtNode()     {}
func (*WhileStmt) stmtNode()  {}
func (*ReturnStmt) stmtNode() {}

// --- Expressions ---

type IntLiteral struct {
	Value int
	Line  int
}

type Ident struct {
	Name string
	Line int
}

type BinaryExpr struct {
	Op    lexer.TokenType
	Left  Expression
	Right Expression
	Line  int
}

type UnaryExpr struct {
	Op      lexer.TokenType
	Operand Expression
	Line    int
}

type AssignExpr struct {
	Name  string
	Value Expression
	Line  int
}

type CallExpr struct {
	Name string
	Args []Expression
	Line int
}

func (*IntLiteral) exprNode() {}
func (*Ident) exprNode()      {}
func (*BinaryExpr) exprNode() {}
func (*UnaryExpr) exprNode()  {}
func (*AssignExpr) exprNode() {}
func (*CallExpr) exprNode()   {}
