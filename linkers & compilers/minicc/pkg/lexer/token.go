package lexer

import "fmt"

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Literals
	IntLiteral TokenType = iota
	Identifier

	// Keywords
	KwInt
	KwIf
	KwElse
	KwWhile
	KwReturn

	// Arithmetic operators
	Plus
	Minus
	Star
	Slash
	Percent

	// Assignment and comparison
	Assign
	Equal
	NotEqual
	Less
	Greater
	LessEq
	GreaterEq

	// Logical operators
	And
	Or
	Not

	// Delimiters
	LParen
	RParen
	LBrace
	RBrace
	Semicolon
	Comma

	// Special
	EOF
	Illegal
)

var tokenNames = [...]string{
	IntLiteral: "INT_LITERAL",
	Identifier: "IDENTIFIER",
	KwInt:      "int",
	KwIf:       "if",
	KwElse:     "else",
	KwWhile:    "while",
	KwReturn:   "return",
	Plus:       "+",
	Minus:      "-",
	Star:       "*",
	Slash:      "/",
	Percent:    "%",
	Assign:     "=",
	Equal:      "==",
	NotEqual:   "!=",
	Less:       "<",
	Greater:    ">",
	LessEq:     "<=",
	GreaterEq:  ">=",
	And:        "&&",
	Or:         "||",
	Not:        "!",
	LParen:     "(",
	RParen:     ")",
	LBrace:     "{",
	RBrace:     "}",
	Semicolon:  ";",
	Comma:      ",",
	EOF:        "EOF",
	Illegal:    "ILLEGAL",
}

func (t TokenType) String() string {
	if int(t) < len(tokenNames) {
		return tokenNames[t]
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// keywords maps keyword strings to their token types.
var keywords = map[string]TokenType{
	"int":    KwInt,
	"if":     KwIf,
	"else":   KwElse,
	"while":  KwWhile,
	"return": KwReturn,
}

// LookupIdent returns the keyword TokenType for ident if it is a keyword,
// or Identifier otherwise.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return Identifier
}

// Token represents a single lexical token with its position in the source.
type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %d:%d", t.Type, t.Lexeme, t.Line, t.Column)
}
