package lexer

import (
	"fmt"
	"unicode"
)

// Lexer scans Mini-C source code into a sequence of tokens.
type Lexer struct {
	source  []rune
	tokens  []Token
	start   int // start of current lexeme
	current int // current scan position
	line    int // 1-based line number
	column  int // 1-based column at start of current lexeme
	curCol  int // 1-based column at current scan position
	errors  []string
}

// New creates a lexer for the given source string.
func New(source string) *Lexer {
	return &Lexer{
		source: []rune(source),
		line:   1,
		column: 1,
		curCol: 1,
	}
}

// Tokenize scans the entire source and returns all tokens and any errors.
// The token list always ends with an EOF token.
func (l *Lexer) Tokenize() ([]Token, []string) {
	for !l.isAtEnd() {
		l.start = l.current
		l.column = l.curCol
		l.scanToken()
	}
	l.tokens = append(l.tokens, Token{
		Type:   EOF,
		Lexeme: "",
		Line:   l.line,
		Column: l.curCol,
	})
	return l.tokens, l.errors
}

func (l *Lexer) scanToken() {
	ch := l.advance()
	switch ch {
	case '(':
		l.addToken(LParen)
	case ')':
		l.addToken(RParen)
	case '{':
		l.addToken(LBrace)
	case '}':
		l.addToken(RBrace)
	case ';':
		l.addToken(Semicolon)
	case ',':
		l.addToken(Comma)
	case '+':
		l.addToken(Plus)
	case '-':
		l.addToken(Minus)
	case '*':
		l.addToken(Star)
	case '%':
		l.addToken(Percent)

	case '/':
		if l.match('/') {
			// Single-line comment: skip to end of line
			for !l.isAtEnd() && l.peek() != '\n' {
				l.advance()
			}
		} else {
			l.addToken(Slash)
		}

	case '=':
		if l.match('=') {
			l.addToken(Equal)
		} else {
			l.addToken(Assign)
		}

	case '!':
		if l.match('=') {
			l.addToken(NotEqual)
		} else {
			l.addToken(Not)
		}

	case '<':
		if l.match('=') {
			l.addToken(LessEq)
		} else {
			l.addToken(Less)
		}

	case '>':
		if l.match('=') {
			l.addToken(GreaterEq)
		} else {
			l.addToken(Greater)
		}

	case '&':
		if l.match('&') {
			l.addToken(And)
		} else {
			l.errorf("unexpected character '&' — did you mean '&&'?")
		}

	case '|':
		if l.match('|') {
			l.addToken(Or)
		} else {
			l.errorf("unexpected character '|' — did you mean '||'?")
		}

	case ' ', '\t', '\r':
		// skip whitespace

	case '\n':
		l.line++
		l.curCol = 1

	default:
		if unicode.IsDigit(ch) {
			l.number()
		} else if isIdentStart(ch) {
			l.identifier()
		} else {
			l.errorf("unexpected character %q", ch)
		}
	}
}

// number scans an integer literal [0-9]+.
func (l *Lexer) number() {
	for !l.isAtEnd() && unicode.IsDigit(l.peek()) {
		l.advance()
	}
	// Check that the number isn't immediately followed by a letter/underscore
	// (e.g., "123abc" is invalid).
	if !l.isAtEnd() && isIdentStart(l.peek()) {
		// Consume the bad suffix for a better error message
		for !l.isAtEnd() && isIdentPart(l.peek()) {
			l.advance()
		}
		l.errorf("invalid number literal %q", l.lexeme())
		return
	}
	l.addToken(IntLiteral)
}

// identifier scans an identifier or keyword [a-zA-Z_][a-zA-Z0-9_]*.
func (l *Lexer) identifier() {
	for !l.isAtEnd() && isIdentPart(l.peek()) {
		l.advance()
	}
	text := l.lexeme()
	l.addToken(LookupIdent(text))
}

// advance consumes the current rune and returns it.
func (l *Lexer) advance() rune {
	ch := l.source[l.current]
	l.current++
	if ch != '\n' {
		l.curCol++
	}
	return ch
}

// peek returns the current rune without consuming it.
func (l *Lexer) peek() rune {
	return l.source[l.current]
}

// match consumes the current rune if it equals expected, returns true/false.
func (l *Lexer) match(expected rune) bool {
	if l.isAtEnd() || l.source[l.current] != expected {
		return false
	}
	l.current++
	l.curCol++
	return true
}

// isAtEnd returns true if the scanner has consumed all source runes.
func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

// lexeme returns the text of the current token being scanned.
func (l *Lexer) lexeme() string {
	return string(l.source[l.start:l.current])
}

// addToken appends a token of the given type to the token list.
func (l *Lexer) addToken(typ TokenType) {
	l.tokens = append(l.tokens, Token{
		Type:   typ,
		Lexeme: l.lexeme(),
		Line:   l.line,
		Column: l.column,
	})
}

// errorf records an error at the current position.
func (l *Lexer) errorf(format string, args ...any) {
	msg := fmt.Sprintf("lexer error at %d:%d: %s", l.line, l.column, fmt.Sprintf(format, args...))
	l.errors = append(l.errors, msg)
	// Also emit an ILLEGAL token so the parser sees something
	l.tokens = append(l.tokens, Token{
		Type:   Illegal,
		Lexeme: l.lexeme(),
		Line:   l.line,
		Column: l.column,
	})
}

func isIdentStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentPart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}
