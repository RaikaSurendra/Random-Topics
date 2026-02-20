package lexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmptyInput(t *testing.T) {
	tokens, errs := New("").Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Fatalf("expected single EOF token, got %v", tokens)
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tokens, errs := New("  \t\n\r\n  ").Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Fatalf("expected single EOF token, got %v", tokens)
	}
}

func TestSingleCharTokens(t *testing.T) {
	input := "( ) { } ; , + - * %"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{
		LParen, RParen, LBrace, RBrace, Semicolon, Comma,
		Plus, Minus, Star, Percent, EOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestSlash(t *testing.T) {
	tokens, errs := New("5 / 3").Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{IntLiteral, Slash, IntLiteral, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestTwoCharOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"==", Equal},
		{"!=", NotEqual},
		{"<=", LessEq},
		{">=", GreaterEq},
		{"&&", And},
		{"||", Or},
	}
	for _, tt := range tests {
		tokens, errs := New(tt.input).Tokenize()
		if len(errs) != 0 {
			t.Errorf("input %q: unexpected errors: %v", tt.input, errs)
			continue
		}
		if len(tokens) != 2 { // operator + EOF
			t.Errorf("input %q: expected 2 tokens, got %d", tt.input, len(tokens))
			continue
		}
		if tokens[0].Type != tt.expected {
			t.Errorf("input %q: expected %s, got %s", tt.input, tt.expected, tokens[0].Type)
		}
	}
}

func TestSingleVsTwoChar(t *testing.T) {
	// = vs == , ! vs != , < vs <= , > vs >=
	input := "= == ! != < <= > >="
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{
		Assign, Equal, Not, NotEqual, Less, LessEq, Greater, GreaterEq, EOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := "int if else while return"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{KwInt, KwIf, KwElse, KwWhile, KwReturn, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestIdentifiers(t *testing.T) {
	input := "foo bar _x a1 integer iff"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// "integer" and "iff" are identifiers, not keywords
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type != Identifier {
			t.Errorf("token[%d] %q: expected IDENTIFIER, got %s", i, tokens[i].Lexeme, tokens[i].Type)
		}
	}
}

func TestIntegerLiterals(t *testing.T) {
	input := "0 42 1234567890"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []string{"0", "42", "1234567890"}
	if len(tokens) != len(expected)+1 {
		t.Fatalf("expected %d tokens, got %d", len(expected)+1, len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Type != IntLiteral {
			t.Errorf("token[%d]: expected INT_LITERAL, got %s", i, tokens[i].Type)
		}
		if tokens[i].Lexeme != exp {
			t.Errorf("token[%d]: expected lexeme %q, got %q", i, exp, tokens[i].Lexeme)
		}
	}
}

func TestInvalidNumberLiteral(t *testing.T) {
	_, errs := New("123abc").Tokenize()
	if len(errs) == 0 {
		t.Fatal("expected error for '123abc'")
	}
}

func TestComment(t *testing.T) {
	input := "int x; // this is a comment\nreturn x;"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{KwInt, Identifier, Semicolon, KwReturn, Identifier, Semicolon, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestCommentOnly(t *testing.T) {
	tokens, errs := New("// just a comment").Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Fatalf("expected single EOF, got %v", tokens)
	}
}

func TestLineAndColumn(t *testing.T) {
	input := "int x;\nreturn x;"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// Line 1: "int" at 1:1, "x" at 1:5, ";" at 1:6
	// Line 2: "return" at 2:1, "x" at 2:8, ";" at 2:9
	checks := []struct {
		idx    int
		line   int
		column int
		lexeme string
	}{
		{0, 1, 1, "int"},
		{1, 1, 5, "x"},
		{2, 1, 6, ";"},
		{3, 2, 1, "return"},
		{4, 2, 8, "x"},
		{5, 2, 9, ";"},
	}
	for _, c := range checks {
		tok := tokens[c.idx]
		if tok.Line != c.line || tok.Column != c.column {
			t.Errorf("token %q: expected %d:%d, got %d:%d",
				c.lexeme, c.line, c.column, tok.Line, tok.Column)
		}
		if tok.Lexeme != c.lexeme {
			t.Errorf("token[%d]: expected lexeme %q, got %q", c.idx, c.lexeme, tok.Lexeme)
		}
	}
}

func TestUnexpectedCharacter(t *testing.T) {
	_, errs := New("@").Tokenize()
	if len(errs) == 0 {
		t.Fatal("expected error for '@'")
	}
}

func TestSingleAmpersand(t *testing.T) {
	_, errs := New("&").Tokenize()
	if len(errs) == 0 {
		t.Fatal("expected error for single '&'")
	}
}

func TestSinglePipe(t *testing.T) {
	_, errs := New("|").Tokenize()
	if len(errs) == 0 {
		t.Fatal("expected error for single '|'")
	}
}

func TestFullFunction(t *testing.T) {
	input := `int factorial(int n) {
    int result = 1;
    while (n > 1) {
        result = result * n;
        n = n - 1;
    }
    return result;
}`
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{
		KwInt, Identifier, LParen, KwInt, Identifier, RParen, LBrace,       // int factorial(int n) {
		KwInt, Identifier, Assign, IntLiteral, Semicolon,                    // int result = 1;
		KwWhile, LParen, Identifier, Greater, IntLiteral, RParen, LBrace,   // while (n > 1) {
		Identifier, Assign, Identifier, Star, Identifier, Semicolon,        // result = result * n;
		Identifier, Assign, Identifier, Minus, IntLiteral, Semicolon,       // n = n - 1;
		RBrace,                                                              // }
		KwReturn, Identifier, Semicolon,                                     // return result;
		RBrace,                                                              // }
		EOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s (%q)", i, expected[i], tok.Type, tok.Lexeme)
		}
	}
}

func TestMaximalMunch(t *testing.T) {
	// "==" should be one token, not two "=" tokens
	tokens, errs := New("x==y").Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{Identifier, Equal, Identifier, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestNoSpacesBetweenTokens(t *testing.T) {
	input := "int x=5+3;"
	tokens, errs := New(input).Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenType{KwInt, Identifier, Assign, IntLiteral, Plus, IntLiteral, Semicolon, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %s, got %s (%q)", i, expected[i], tok.Type, tok.Lexeme)
		}
	}
}

// TestValidProgramsTokenize ensures all 20 valid test programs tokenize without errors.
func TestValidProgramsTokenize(t *testing.T) {
	validDir := filepath.Join("..", "..", "testdata", "valid")
	entries, err := os.ReadDir(validDir)
	if err != nil {
		t.Fatalf("cannot read testdata/valid: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".mc") {
			continue
		}
		count++
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(validDir, entry.Name()))
			if err != nil {
				t.Fatalf("cannot read %s: %v", entry.Name(), err)
			}
			tokens, errs := New(string(data)).Tokenize()
			if len(errs) != 0 {
				t.Errorf("tokenization errors in %s: %v", entry.Name(), errs)
			}
			// Every valid program should produce at least one non-EOF token
			if len(tokens) < 2 {
				t.Errorf("%s: expected at least 2 tokens (content + EOF), got %d", entry.Name(), len(tokens))
			}
			// Last token must be EOF
			if tokens[len(tokens)-1].Type != EOF {
				t.Errorf("%s: last token should be EOF, got %s", entry.Name(), tokens[len(tokens)-1].Type)
			}
		})
	}
	if count != 20 {
		t.Errorf("expected 20 valid test programs, found %d", count)
	}
}

// TestInvalidTokenProgram ensures the invalid token test program produces errors.
func TestInvalidTokenProgram(t *testing.T) {
	invalidDir := filepath.Join("..", "..", "testdata", "invalid")
	data, err := os.ReadFile(filepath.Join(invalidDir, "10_invalid_token.mc"))
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}
	_, errs := New(string(data)).Tokenize()
	if len(errs) == 0 {
		t.Error("expected lexer errors for invalid_token.mc")
	}
}

func TestMultipleErrors(t *testing.T) {
	input := "@ # $"
	_, errs := New(input).Tokenize()
	if len(errs) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestTokenStringer(t *testing.T) {
	tok := Token{Type: KwInt, Lexeme: "int", Line: 1, Column: 1}
	s := tok.String()
	if !strings.Contains(s, "int") {
		t.Errorf("Token.String() should contain 'int', got %q", s)
	}
}

func TestLeadingZeros(t *testing.T) {
	// Leading zeros are valid in Mini-C (treated as decimal)
	tokens, errs := New("007").Tokenize()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if tokens[0].Type != IntLiteral || tokens[0].Lexeme != "007" {
		t.Errorf("expected INT_LITERAL '007', got %s %q", tokens[0].Type, tokens[0].Lexeme)
	}
}
