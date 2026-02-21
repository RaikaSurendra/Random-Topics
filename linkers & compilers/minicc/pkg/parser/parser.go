package parser

import (
	"fmt"
	"minicc/pkg/lexer"
	"strconv"
)

// Parser converts a token stream into an AST using recursive descent.
type Parser struct {
	tokens  []lexer.Token
	current int
	errors  []string
}

// New creates a parser for the given token stream.
func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse parses the token stream into a Program AST.
func (p *Parser) Parse() (*Program, []string) {
	prog := &Program{}
	for !p.check(lexer.EOF) {
		decl := p.declaration()
		if decl != nil {
			prog.Declarations = append(prog.Declarations, decl)
		}
	}
	return prog, p.errors
}

// --- Top-level declarations ---

// declaration → "int" IDENT ( "(" ... → func | "=" expr ";" | ";" → var )
func (p *Parser) declaration() Declaration {
	if !p.check(lexer.KwInt) {
		p.errorf("expected type 'int' at top level, got %s", p.peek().Type)
		p.advance()
		return nil
	}
	intTok := p.advance() // consume "int"

	name := p.expectIdent()
	if name == "" {
		return nil
	}

	if p.check(lexer.LParen) {
		return p.funcDecl(name, intTok.Line)
	}
	return p.varDeclRest(name, intTok.Line)
}

// funcDecl parses the rest of a function after "int" IDENT.
func (p *Parser) funcDecl(name string, line int) *FuncDecl {
	p.expect(lexer.LParen)
	params := p.paramList()
	p.expect(lexer.RParen)
	body := p.block()
	return &FuncDecl{Name: name, Params: params, Body: body, Line: line}
}

// paramList → ( "int" IDENT ("," "int" IDENT)* )?
func (p *Parser) paramList() []Param {
	var params []Param
	if p.check(lexer.RParen) {
		return params
	}
	for {
		p.expect(lexer.KwInt)
		tok := p.peek()
		name := p.expectIdent()
		if name == "" {
			break
		}
		params = append(params, Param{Name: name, Line: tok.Line})
		if !p.match(lexer.Comma) {
			break
		}
	}
	return params
}

// varDeclRest parses the rest of a var declaration after "int" IDENT.
// Handles: ";" or "=" expr ";"
func (p *Parser) varDeclRest(name string, line int) *VarDecl {
	var init Expression
	if p.match(lexer.Assign) {
		init = p.expression()
	}
	p.expect(lexer.Semicolon)
	return &VarDecl{Name: name, Init: init, Line: line}
}

// --- Statements ---

// statement → block | if | while | return | varDecl | exprStmt
func (p *Parser) statement() Statement {
	if p.check(lexer.LBrace) {
		return p.block()
	}
	if p.check(lexer.KwIf) {
		return p.ifStmt()
	}
	if p.check(lexer.KwWhile) {
		return p.whileStmt()
	}
	if p.check(lexer.KwReturn) {
		return p.returnStmt()
	}
	if p.check(lexer.KwInt) {
		return p.varDeclStmt()
	}
	return p.exprStmt()
}

// block → "{" blockItem* "}"
func (p *Parser) block() *BlockStmt {
	tok := p.peek()
	p.expect(lexer.LBrace)
	var items []Statement
	for !p.check(lexer.RBrace) && !p.check(lexer.EOF) {
		stmt := p.statement()
		if stmt != nil {
			items = append(items, stmt)
		}
	}
	p.expect(lexer.RBrace)
	return &BlockStmt{Items: items, Line: tok.Line}
}

// ifStmt → "if" "(" expr ")" statement ("else" statement)?
func (p *Parser) ifStmt() *IfStmt {
	tok := p.advance() // consume "if"
	p.expect(lexer.LParen)
	cond := p.expression()
	p.expect(lexer.RParen)
	then := p.statement()
	var elseStmt Statement
	if p.match(lexer.KwElse) {
		elseStmt = p.statement()
	}
	return &IfStmt{Condition: cond, Then: then, Else: elseStmt, Line: tok.Line}
}

// whileStmt → "while" "(" expr ")" statement
func (p *Parser) whileStmt() *WhileStmt {
	tok := p.advance() // consume "while"
	p.expect(lexer.LParen)
	cond := p.expression()
	p.expect(lexer.RParen)
	body := p.statement()
	return &WhileStmt{Condition: cond, Body: body, Line: tok.Line}
}

// returnStmt → "return" expr ";"
func (p *Parser) returnStmt() *ReturnStmt {
	tok := p.advance() // consume "return"
	val := p.expression()
	p.expect(lexer.Semicolon)
	return &ReturnStmt{Value: val, Line: tok.Line}
}

// varDeclStmt → "int" IDENT ("=" expr)? ";"
func (p *Parser) varDeclStmt() *VarDecl {
	intTok := p.advance() // consume "int"
	name := p.expectIdent()
	if name == "" {
		return nil
	}
	return p.varDeclRest(name, intTok.Line)
}

// exprStmt → expr ";"
func (p *Parser) exprStmt() *ExprStmt {
	tok := p.peek()
	expr := p.expression()
	p.expect(lexer.Semicolon)
	return &ExprStmt{Expr: expr, Line: tok.Line}
}

// --- Expressions (precedence climbing) ---

// expression → assignment
func (p *Parser) expression() Expression {
	return p.assignment()
}

// assignment → IDENT "=" assignment | logicalOr
// Right-associative: if we see IDENT followed by "=", it's assignment.
func (p *Parser) assignment() Expression {
	// Speculatively try to parse as logicalOr; if result is Ident and next is "=", convert.
	expr := p.logicalOr()

	if p.check(lexer.Assign) {
		p.advance() // consume "="
		ident, ok := expr.(*Ident)
		if !ok {
			p.errorf("invalid assignment target")
			return expr
		}
		value := p.assignment() // right-associative
		return &AssignExpr{Name: ident.Name, Value: value, Line: ident.Line}
	}
	return expr
}

// logicalOr → logicalAnd ("||" logicalAnd)*
func (p *Parser) logicalOr() Expression {
	left := p.logicalAnd()
	for p.check(lexer.Or) {
		tok := p.advance()
		right := p.logicalAnd()
		left = &BinaryExpr{Op: tok.Type, Left: left, Right: right, Line: tok.Line}
	}
	return left
}

// logicalAnd → equality ("&&" equality)*
func (p *Parser) logicalAnd() Expression {
	left := p.equality()
	for p.check(lexer.And) {
		tok := p.advance()
		right := p.equality()
		left = &BinaryExpr{Op: tok.Type, Left: left, Right: right, Line: tok.Line}
	}
	return left
}

// equality → comparison (("==" | "!=") comparison)*
func (p *Parser) equality() Expression {
	left := p.comparison()
	for p.check(lexer.Equal) || p.check(lexer.NotEqual) {
		tok := p.advance()
		right := p.comparison()
		left = &BinaryExpr{Op: tok.Type, Left: left, Right: right, Line: tok.Line}
	}
	return left
}

// comparison → additive (("<" | ">" | "<=" | ">=") additive)*
func (p *Parser) comparison() Expression {
	left := p.additive()
	for p.check(lexer.Less) || p.check(lexer.Greater) ||
		p.check(lexer.LessEq) || p.check(lexer.GreaterEq) {
		tok := p.advance()
		right := p.additive()
		left = &BinaryExpr{Op: tok.Type, Left: left, Right: right, Line: tok.Line}
	}
	return left
}

// additive → multiplicative (("+" | "-") multiplicative)*
func (p *Parser) additive() Expression {
	left := p.multiplicative()
	for p.check(lexer.Plus) || p.check(lexer.Minus) {
		tok := p.advance()
		right := p.multiplicative()
		left = &BinaryExpr{Op: tok.Type, Left: left, Right: right, Line: tok.Line}
	}
	return left
}

// multiplicative → unary (("*" | "/" | "%") unary)*
func (p *Parser) multiplicative() Expression {
	left := p.unary()
	for p.check(lexer.Star) || p.check(lexer.Slash) || p.check(lexer.Percent) {
		tok := p.advance()
		right := p.unary()
		left = &BinaryExpr{Op: tok.Type, Left: left, Right: right, Line: tok.Line}
	}
	return left
}

// unary → ("-" | "!") unary | primary
func (p *Parser) unary() Expression {
	if p.check(lexer.Minus) || p.check(lexer.Not) {
		tok := p.advance()
		operand := p.unary()
		return &UnaryExpr{Op: tok.Type, Operand: operand, Line: tok.Line}
	}
	return p.primary()
}

// primary → INT_LITERAL | IDENT | IDENT "(" args ")" | "(" expr ")"
func (p *Parser) primary() Expression {
	tok := p.peek()

	// Integer literal
	if p.check(lexer.IntLiteral) {
		p.advance()
		val, err := strconv.Atoi(tok.Lexeme)
		if err != nil {
			p.errorf("invalid integer literal %q", tok.Lexeme)
			return &IntLiteral{Value: 0, Line: tok.Line}
		}
		return &IntLiteral{Value: val, Line: tok.Line}
	}

	// Identifier or function call
	if p.check(lexer.Identifier) {
		p.advance()
		if p.check(lexer.LParen) {
			// Function call
			p.advance() // consume "("
			args := p.argList()
			p.expect(lexer.RParen)
			return &CallExpr{Name: tok.Lexeme, Args: args, Line: tok.Line}
		}
		return &Ident{Name: tok.Lexeme, Line: tok.Line}
	}

	// Grouped expression
	if p.match(lexer.LParen) {
		expr := p.expression()
		p.expect(lexer.RParen)
		return expr
	}

	p.errorf("expected expression, got %s %q", tok.Type, tok.Lexeme)
	p.advance()
	return &IntLiteral{Value: 0, Line: tok.Line}
}

// argList → (expr ("," expr)*)?
func (p *Parser) argList() []Expression {
	var args []Expression
	if p.check(lexer.RParen) {
		return args
	}
	args = append(args, p.expression())
	for p.match(lexer.Comma) {
		args = append(args, p.expression())
	}
	return args
}

// --- Helper methods ---

// peek returns the current token without consuming it.
func (p *Parser) peek() lexer.Token {
	return p.tokens[p.current]
}

// advance consumes the current token and returns it.
func (p *Parser) advance() lexer.Token {
	tok := p.tokens[p.current]
	if tok.Type != lexer.EOF {
		p.current++
	}
	return tok
}

// check returns true if the current token has the given type.
func (p *Parser) check(typ lexer.TokenType) bool {
	return p.tokens[p.current].Type == typ
}

// match consumes the current token if it matches, returns true/false.
func (p *Parser) match(typ lexer.TokenType) bool {
	if p.check(typ) {
		p.advance()
		return true
	}
	return false
}

// expect consumes the current token if it matches, or records an error.
func (p *Parser) expect(typ lexer.TokenType) lexer.Token {
	if p.check(typ) {
		return p.advance()
	}
	tok := p.peek()
	p.errorf("expected %s, got %s %q", typ, tok.Type, tok.Lexeme)
	return tok
}

// expectIdent expects an identifier token and returns its name.
func (p *Parser) expectIdent() string {
	tok := p.expect(lexer.Identifier)
	if tok.Type != lexer.Identifier {
		return ""
	}
	return tok.Lexeme
}

func (p *Parser) errorf(format string, args ...any) {
	tok := p.peek()
	msg := fmt.Sprintf("parse error at %d:%d: %s", tok.Line, tok.Column, fmt.Sprintf(format, args...))
	p.errors = append(p.errors, msg)
}
