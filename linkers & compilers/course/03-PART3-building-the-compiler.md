===============================================================================
  PART 3 — BUILDING THE COMPILER IN GO
  Weeks 4–8
===============================================================================

This is the core of the course. We build five modules, each corresponding
roughly to one week. Every module includes theory, data structures, Go
implementation, testing, and exercises.

```
  +----------+    +----------+    +----------+    +----------+    +----------+
  |  Source   |--->|  LEXER   |--->|  PARSER  |--->| SEMANTIC |--->|    IR    |
  |  Code     |    |  Tokens  |    |   AST    |    | Analysis |    |  3-Addr  |
  +----------+    +----------+    +----------+    +----------+    +----------+
                                                                       |
                                                                       v
                                                                  +----------+
                                                                  | CODEGEN  |
                                                                  | x86-64   |
                                                                  | Assembly |
                                                                  +----------+
```


███████████████████████████████████████████████████████████████████████████████
  MODULE 1 — THE LEXER (Week 4)
███████████████████████████████████████████████████████████████████████████████


M1.1  THEORY: LEXICAL ANALYSIS
===============================================================================

The lexer (also called scanner or tokenizer) is the first phase of
compilation. It reads a stream of characters and produces a stream
of tokens.

```
  Characters:  i n t   m a i n ( )   { \n   r e t u r n   0 ; \n }
                           |
                           v
  Tokens:     [INT] [IDENT:"main"] [LPAREN] [RPAREN] [LBRACE]
              [RETURN] [INT_LIT:0] [SEMICOLON] [RBRACE] [EOF]
```

Formal Foundation:

  Tokens are defined by regular expressions. Regular expressions can be
  recognized by Deterministic Finite Automata (DFA).

  A DFA is a 5-tuple (Q, Σ, δ, q₀, F) where:
    Q  = finite set of states
    Σ  = input alphabet (ASCII characters)
    δ  = transition function: Q × Σ → Q
    q₀ = start state
    F  = set of accepting (final) states

  Example -- DFA for integer literals [0-9]+:

```
     +-------------+  [0-9]   +-------------+
     |   START     |--------->|  ACCEPT     |--+
     |   (q0)      |          |  (q1)       |  | [0-9]
     +-------------+          +-------------+--+
```

     State q0: not yet seen any digit
     State q1: seen at least one digit (accepting)
     On any non-digit in q1: emit INT_LIT token, return to q0

  Example -- DFA for identifiers and keywords:

```
     +-----+ [a-zA-Z_] +---------+
     | q0  |---------->|  q1     |--+
     +-----+           |(accept) |  | [a-zA-Z0-9_]
                       +---------+--+
```

     After accepting, check if the lexeme is a keyword.

  Why manual scanning over regex libraries?
  - Full control over error messages (line, column)
  - Performance (no regex engine overhead)
  - Handles edge cases cleanly (maximal munch)
  - Educational: you understand exactly what happens


M1.2  DATA STRUCTURES
===============================================================================

  // pkg/lexer/token.go

  package lexer

  import "fmt"

  type TokenType int

  const (
      // Literals
      TOKEN_INT_LITERAL TokenType = iota
      TOKEN_IDENTIFIER

      // Keywords
      TOKEN_INT
      TOKEN_IF
      TOKEN_ELSE
      TOKEN_WHILE
      TOKEN_RETURN

      // Single-character operators
      TOKEN_PLUS
      TOKEN_MINUS
      TOKEN_STAR
      TOKEN_SLASH
      TOKEN_PERCENT
      TOKEN_ASSIGN
      TOKEN_NOT
      TOKEN_LESS
      TOKEN_GREATER
      TOKEN_LPAREN
      TOKEN_RPAREN
      TOKEN_LBRACE
      TOKEN_RBRACE
      TOKEN_SEMICOLON
      TOKEN_COMMA

      // Two-character operators
      TOKEN_EQUAL       // ==
      TOKEN_NOT_EQUAL   // !=
      TOKEN_LESS_EQ     // <=
      TOKEN_GREATER_EQ  // >=
      TOKEN_AND         // &&
      TOKEN_OR          // ||

      // Special
      TOKEN_EOF
      TOKEN_ILLEGAL
  )

  // String returns the human-readable name for debugging.
  func (t TokenType) String() string {
      names := [...]string{
          "INT_LITERAL", "IDENTIFIER",
          "int", "if", "else", "while", "return",
          "+", "-", "*", "/", "%", "=", "!", "<", ">",
          "(", ")", "{", "}", ";", ",",
          "==", "!=", "<=", ">=", "&&", "||",
          "EOF", "ILLEGAL",
      }
      if int(t) < len(names) {
          return names[t]
      }
      return fmt.Sprintf("TokenType(%d)", t)
  }

  type Token struct {
      Type   TokenType
      Lexeme string
      Line   int
      Column int
  }

  func (t Token) String() string {
      return fmt.Sprintf("(%s, %q, %d:%d)", t.Type, t.Lexeme, t.Line, t.Column)
  }

  // keywords maps keyword strings to their token types.
  var keywords = map[string]TokenType{
      "int":    TOKEN_INT,
      "if":     TOKEN_IF,
      "else":   TOKEN_ELSE,
      "while":  TOKEN_WHILE,
      "return": TOKEN_RETURN,
  }


M1.3  GO IMPLEMENTATION
===============================================================================

  // pkg/lexer/lexer.go

  package lexer

  import (
      "fmt"
      "unicode"
  )

  type Lexer struct {
      source  []rune   // Full source as runes (Unicode-safe)
      tokens  []Token  // Accumulated tokens
      start   int      // Start of current lexeme
      current int      // Current position in source
      line    int      // Current line number (1-based)
      column  int      // Current column (1-based)
      errors  []string // Accumulated error messages
  }

  func New(source string) *Lexer {
      return &Lexer{
          source: []rune(source),
          line:   1,
          column: 1,
      }
  }

  // Tokenize scans the entire source and returns all tokens.
  func (l *Lexer) Tokenize() ([]Token, []string) {
      for !l.isAtEnd() {
          l.start = l.current
          l.scanToken()
      }
      l.tokens = append(l.tokens, Token{
          Type:   TOKEN_EOF,
          Lexeme: "",
          Line:   l.line,
          Column: l.column,
      })
      return l.tokens, l.errors
  }

  // scanToken reads one token starting at l.current.
  func (l *Lexer) scanToken() {
      ch := l.advance()
      switch ch {
      case '+':
          l.addToken(TOKEN_PLUS)
      case '-':
          l.addToken(TOKEN_MINUS)
      case '*':
          l.addToken(TOKEN_STAR)
      case '/':
          if l.match('/') {
              // Comment: consume until end of line
              for !l.isAtEnd() && l.peek() != '\n' {
                  l.advance()
              }
          } else {
              l.addToken(TOKEN_SLASH)
          }
      case '%':
          l.addToken(TOKEN_PERCENT)
      case '(':
          l.addToken(TOKEN_LPAREN)
      case ')':
          l.addToken(TOKEN_RPAREN)
      case '{':
          l.addToken(TOKEN_LBRACE)
      case '}':
          l.addToken(TOKEN_RBRACE)
      case ';':
          l.addToken(TOKEN_SEMICOLON)
      case ',':
          l.addToken(TOKEN_COMMA)

      // Two-character operators
      case '=':
          if l.match('=') {
              l.addToken(TOKEN_EQUAL)
          } else {
              l.addToken(TOKEN_ASSIGN)
          }
      case '!':
          if l.match('=') {
              l.addToken(TOKEN_NOT_EQUAL)
          } else {
              l.addToken(TOKEN_NOT)
          }
      case '<':
          if l.match('=') {
              l.addToken(TOKEN_LESS_EQ)
          } else {
              l.addToken(TOKEN_LESS)
          }
      case '>':
          if l.match('=') {
              l.addToken(TOKEN_GREATER_EQ)
          } else {
              l.addToken(TOKEN_GREATER)
          }
      case '&':
          if l.match('&') {
              l.addToken(TOKEN_AND)
          } else {
              l.error("unexpected character '&'; did you mean '&&'?")
          }
      case '|':
          if l.match('|') {
              l.addToken(TOKEN_OR)
          } else {
              l.error("unexpected character '|'; did you mean '||'?")
          }

      // Whitespace
      case ' ', '\t', '\r':
          // Skip
      case '\n':
          l.line++
          l.column = 1

      default:
          if unicode.IsDigit(ch) {
              l.number()
          } else if unicode.IsLetter(ch) || ch == '_' {
              l.identifier()
          } else {
              l.error(fmt.Sprintf("unexpected character %q", ch))
          }
      }
  }

  // number scans an integer literal.
  func (l *Lexer) number() {
      for !l.isAtEnd() && unicode.IsDigit(l.peek()) {
          l.advance()
      }
      // Check for invalid suffix (e.g., "123abc")
      if !l.isAtEnd() && (unicode.IsLetter(l.peek()) || l.peek() == '_') {
          l.error("invalid number literal")
          return
      }
      l.addToken(TOKEN_INT_LITERAL)
  }

  // identifier scans an identifier or keyword.
  func (l *Lexer) identifier() {
      for !l.isAtEnd() && (unicode.IsLetter(l.peek()) || unicode.IsDigit(l.peek()) || l.peek() == '_') {
          l.advance()
      }
      text := string(l.source[l.start:l.current])
      if kwType, ok := keywords[text]; ok {
          l.addToken(kwType)
      } else {
          l.addToken(TOKEN_IDENTIFIER)
      }
  }

  // ─── Helper Methods ───────────────────────────────────────────────

  func (l *Lexer) advance() rune {
      ch := l.source[l.current]
      l.current++
      l.column++
      return ch
  }

  func (l *Lexer) peek() rune {
      return l.source[l.current]
  }

  func (l *Lexer) match(expected rune) bool {
      if l.isAtEnd() || l.source[l.current] != expected {
          return false
      }
      l.current++
      l.column++
      return true
  }

  func (l *Lexer) isAtEnd() bool {
      return l.current >= len(l.source)
  }

  func (l *Lexer) addToken(tokenType TokenType) {
      text := string(l.source[l.start:l.current])
      l.tokens = append(l.tokens, Token{
          Type:   tokenType,
          Lexeme: text,
          Line:   l.line,
          Column: l.column - len(text),
      })
  }

  func (l *Lexer) error(msg string) {
      l.errors = append(l.errors, fmt.Sprintf(
          "lexer error at %d:%d: %s", l.line, l.column, msg,
      ))
      l.addToken(TOKEN_ILLEGAL)
  }


M1.4  TESTING STRATEGY
===============================================================================

  // pkg/lexer/lexer_test.go

  package lexer

  import (
      "testing"
  )

  func TestSimpleTokens(t *testing.T) {
      input := "(){};,+-*/%"
      lex := New(input)
      tokens, errs := lex.Tokenize()
      if len(errs) > 0 {
          t.Fatalf("unexpected errors: %v", errs)
      }

      expected := []TokenType{
          TOKEN_LPAREN, TOKEN_RPAREN, TOKEN_LBRACE, TOKEN_RBRACE,
          TOKEN_SEMICOLON, TOKEN_COMMA, TOKEN_PLUS, TOKEN_MINUS,
          TOKEN_STAR, TOKEN_SLASH, TOKEN_PERCENT, TOKEN_EOF,
      }

      if len(tokens) != len(expected) {
          t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
      }
      for i, exp := range expected {
          if tokens[i].Type != exp {
              t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Type)
          }
      }
  }

  func TestKeywordsAndIdentifiers(t *testing.T) {
      input := "int main if else while return foo bar123 _x"
      lex := New(input)
      tokens, _ := lex.Tokenize()

      expected := []struct {
          typ    TokenType
          lexeme string
      }{
          {TOKEN_INT, "int"},
          {TOKEN_IDENTIFIER, "main"},
          {TOKEN_IF, "if"},
          {TOKEN_ELSE, "else"},
          {TOKEN_WHILE, "while"},
          {TOKEN_RETURN, "return"},
          {TOKEN_IDENTIFIER, "foo"},
          {TOKEN_IDENTIFIER, "bar123"},
          {TOKEN_IDENTIFIER, "_x"},
          {TOKEN_EOF, ""},
      }

      for i, exp := range expected {
          if tokens[i].Type != exp.typ || tokens[i].Lexeme != exp.lexeme {
              t.Errorf("token %d: expected (%s, %q), got (%s, %q)",
                  i, exp.typ, exp.lexeme, tokens[i].Type, tokens[i].Lexeme)
          }
      }
  }

  func TestTwoCharOperators(t *testing.T) {
      input := "== != <= >= && ||"
      lex := New(input)
      tokens, _ := lex.Tokenize()

      expected := []TokenType{
          TOKEN_EQUAL, TOKEN_NOT_EQUAL, TOKEN_LESS_EQ,
          TOKEN_GREATER_EQ, TOKEN_AND, TOKEN_OR, TOKEN_EOF,
      }

      for i, exp := range expected {
          if tokens[i].Type != exp {
              t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i].Type)
          }
      }
  }

  func TestFullProgram(t *testing.T) {
      input := `int factorial(int n) {
          int result = 1;
          while (n > 1) {
              result = result * n;
              n = n - 1;
          }
          return result;
      }`
      lex := New(input)
      tokens, errs := lex.Tokenize()
      if len(errs) > 0 {
          t.Fatalf("unexpected errors: %v", errs)
      }
      // Verify no ILLEGAL tokens
      for _, tok := range tokens {
          if tok.Type == TOKEN_ILLEGAL {
              t.Errorf("illegal token: %s", tok)
          }
      }
      // Verify last token is EOF
      if tokens[len(tokens)-1].Type != TOKEN_EOF {
          t.Error("expected EOF as last token")
      }
  }

  func TestComments(t *testing.T) {
      input := "int x; // this is a comment\nint y;"
      lex := New(input)
      tokens, _ := lex.Tokenize()

      // Comments should be skipped entirely
      types := make([]TokenType, 0)
      for _, tok := range tokens {
          types = append(types, tok.Type)
      }
      expected := []TokenType{
          TOKEN_INT, TOKEN_IDENTIFIER, TOKEN_SEMICOLON,
          TOKEN_INT, TOKEN_IDENTIFIER, TOKEN_SEMICOLON,
          TOKEN_EOF,
      }
      if len(types) != len(expected) {
          t.Fatalf("expected %d tokens, got %d", len(expected), len(types))
      }
  }


M1.5  EXERCISES
===============================================================================

Exercise 4.1: Line/Column Tracking
  Modify the lexer to correctly track column numbers through multi-line
  input. Write a test with a deliberate error on line 3, column 10, and
  verify the error message reports the correct location.

Exercise 4.2: Hex Literals
  Extend the lexer to support hexadecimal integer literals (0xFF).
  Write the DFA diagram for this extension.

Exercise 4.3: Performance
  Tokenize a 10,000-line Mini-C file (generate one programmatically).
  Measure time. What is the throughput in MB/s?

Exercise 4.4: Error Recovery
  Currently, one illegal character produces one error and one ILLEGAL
  token. Modify the lexer to skip illegal characters and continue
  scanning, accumulating multiple errors.


███████████████████████████████████████████████████████████████████████████████
  MODULE 2 — THE PARSER (Week 5)
███████████████████████████████████████████████████████████████████████████████


M2.1  THEORY: PARSING
===============================================================================

The parser takes a flat stream of tokens and builds a tree that captures
the hierarchical structure of the program — the Abstract Syntax Tree (AST).

  Token Stream:
    [INT] [IDENT:factorial] [LPAREN] [INT] [IDENT:n] [RPAREN] [LBRACE]
    [INT] [IDENT:result] [ASSIGN] [INT_LIT:1] [SEMICOLON] ...

  AST:
```
    Program
    +-- FuncDecl: "factorial"
        +-- Params: [("n", int)]
        +-- Body: Block
            +-- VarDecl: "result" = IntLit(1)
            +-- While
                +-- Condition: BinaryExpr(>, Ident("n"), IntLit(1))
                +-- Body: Block
                    +-- ExprStmt: Assign("result", BinaryExpr(*, ...))
                    +-- ExprStmt: Assign("n", BinaryExpr(-, ...))
```

Context-Free Grammars:

  A context-free grammar (CFG) G = (V, Σ, R, S) where:
    V = set of non-terminals (e.g., <expression>, <statement>)
    Σ = set of terminals (tokens)
    R = set of production rules (e.g., <statement> → "return" <expression> ";")
    S = start symbol (e.g., <program>)

  A string of tokens is "in the language" if it can be derived from the
  start symbol by applying production rules.

Parsing Strategies:

  Top-Down (LL parsers):
    Start from the start symbol, try to derive the input.
    Recursive descent is the simplest top-down approach.
    Each non-terminal becomes a function.

  Bottom-Up (LR parsers):
    Start from the input, try to reduce to the start symbol.
    More powerful but harder to implement by hand.
    Used by parser generators (yacc, bison).

  We use RECURSIVE DESCENT — the most intuitive approach:
    - One function per grammar rule
    - Functions call each other following the grammar structure
    - Easy to implement, easy to debug
    - Efficient: O(n) for LL(1) grammars

Parse Tree vs AST:

  Parse Tree: mirrors the grammar exactly, includes all intermediate
  non-terminals. Verbose.

  AST: simplified tree that captures only the essential structure.
  Removes syntactic sugar ({, }, ;, etc.) and intermediate rules.

  We build an AST directly — no parse tree intermediate.


M2.2  AST DATA STRUCTURES
===============================================================================

  // pkg/parser/ast.go

  package parser

  import "github.com/user/minicc/pkg/lexer"

  // ─── Top-Level ────────────────────────────────────────────────────

  // Program is the root AST node.
  type Program struct {
      Declarations []Declaration
  }

  // Declaration is either a variable or function declaration.
  type Declaration interface {
      declNode()
  }

  // ─── Declarations ─────────────────────────────────────────────────

  type VarDecl struct {
      Name     string
      Init     Expression  // nil if no initializer
      Line     int
  }
  func (*VarDecl) declNode()  {}
  func (*VarDecl) stmtNode()  {}  // VarDecl can appear as a statement

  type FuncDecl struct {
      Name     string
      Params   []Param
      Body     *BlockStmt
      Line     int
  }
  func (*FuncDecl) declNode() {}

  type Param struct {
      Name string
  }

  // ─── Statements ───────────────────────────────────────────────────

  type Statement interface {
      stmtNode()
  }

  type BlockStmt struct {
      Stmts []Statement   // Can include VarDecls
  }
  func (*BlockStmt) stmtNode() {}

  type ExprStmt struct {
      Expr Expression
  }
  func (*ExprStmt) stmtNode() {}

  type IfStmt struct {
      Condition Expression
      Then      Statement
      Else      Statement   // nil if no else
  }
  func (*IfStmt) stmtNode() {}

  type WhileStmt struct {
      Condition Expression
      Body      Statement
  }
  func (*WhileStmt) stmtNode() {}

  type ReturnStmt struct {
      Value Expression
      Line  int
  }
  func (*ReturnStmt) stmtNode() {}

  // ─── Expressions ──────────────────────────────────────────────────

  type Expression interface {
      exprNode()
  }

  type IntLiteral struct {
      Value int
  }
  func (*IntLiteral) exprNode() {}

  type Identifier struct {
      Name string
      Line int
  }
  func (*Identifier) exprNode() {}

  type BinaryExpr struct {
      Op    lexer.TokenType
      Left  Expression
      Right Expression
  }
  func (*BinaryExpr) exprNode() {}

  type UnaryExpr struct {
      Op      lexer.TokenType
      Operand Expression
  }
  func (*UnaryExpr) exprNode() {}

  type AssignExpr struct {
      Name  string
      Value Expression
      Line  int
  }
  func (*AssignExpr) exprNode() {}

  type CallExpr struct {
      Name string
      Args []Expression
      Line int
  }
  func (*CallExpr) exprNode() {}


M2.3  GO IMPLEMENTATION — RECURSIVE DESCENT PARSER
===============================================================================

  // pkg/parser/parser.go

  package parser

  import (
      "fmt"
      "strconv"
      "github.com/user/minicc/pkg/lexer"
  )

  type Parser struct {
      tokens  []lexer.Token
      current int
      errors  []string
  }

  func New(tokens []lexer.Token) *Parser {
      return &Parser{tokens: tokens}
  }

  // Parse is the entry point. Parses the token stream into a Program AST.
  func (p *Parser) Parse() (*Program, []string) {
      prog := &Program{}
      for !p.isAtEnd() {
          decl := p.declaration()
          if decl != nil {
              prog.Declarations = append(prog.Declarations, decl)
          }
      }
      return prog, p.errors
  }

  // ─── Declarations ─────────────────────────────────────────────────

  // declaration parses a top-level variable or function declaration.
  // Both start with "int IDENTIFIER", so we look ahead to distinguish.
  func (p *Parser) declaration() Declaration {
      if !p.check(lexer.TOKEN_INT) {
          p.error("expected type 'int'")
          p.advance() // skip bad token
          return nil
      }
      p.advance() // consume "int"
      name := p.consume(lexer.TOKEN_IDENTIFIER, "expected identifier")

      if p.check(lexer.TOKEN_LPAREN) {
          // Function declaration: int name(...)
          return p.funcDecl(name.Lexeme, name.Line)
      }
      // Variable declaration: int name; or int name = expr;
      return p.varDecl(name.Lexeme, name.Line)
  }

  func (p *Parser) funcDecl(name string, line int) *FuncDecl {
      p.consume(lexer.TOKEN_LPAREN, "expected '('")
      params := p.paramList()
      p.consume(lexer.TOKEN_RPAREN, "expected ')'")
      body := p.blockStmt()
      return &FuncDecl{Name: name, Params: params, Body: body, Line: line}
  }

  func (p *Parser) paramList() []Param {
      var params []Param
      if p.check(lexer.TOKEN_RPAREN) {
          return params // empty parameter list
      }
      for {
          p.consume(lexer.TOKEN_INT, "expected 'int' in parameter")
          name := p.consume(lexer.TOKEN_IDENTIFIER, "expected parameter name")
          params = append(params, Param{Name: name.Lexeme})
          if !p.match(lexer.TOKEN_COMMA) {
              break
          }
      }
      return params
  }

  func (p *Parser) varDecl(name string, line int) *VarDecl {
      var init Expression
      if p.match(lexer.TOKEN_ASSIGN) {
          init = p.expression()
      }
      p.consume(lexer.TOKEN_SEMICOLON, "expected ';' after variable declaration")
      return &VarDecl{Name: name, Init: init, Line: line}
  }

  // ─── Statements ───────────────────────────────────────────────────

  func (p *Parser) statement() Statement {
      switch {
      case p.check(lexer.TOKEN_LBRACE):
          return p.blockStmt()
      case p.check(lexer.TOKEN_IF):
          return p.ifStmt()
      case p.check(lexer.TOKEN_WHILE):
          return p.whileStmt()
      case p.check(lexer.TOKEN_RETURN):
          return p.returnStmt()
      case p.check(lexer.TOKEN_INT):
          // Local variable declaration
          p.advance() // consume "int"
          name := p.consume(lexer.TOKEN_IDENTIFIER, "expected identifier")
          return p.varDecl(name.Lexeme, name.Line)
      default:
          return p.exprStmt()
      }
  }

  func (p *Parser) blockStmt() *BlockStmt {
      p.consume(lexer.TOKEN_LBRACE, "expected '{'")
      var stmts []Statement
      for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
          stmts = append(stmts, p.statement())
      }
      p.consume(lexer.TOKEN_RBRACE, "expected '}'")
      return &BlockStmt{Stmts: stmts}
  }

  func (p *Parser) ifStmt() *IfStmt {
      p.advance() // consume "if"
      p.consume(lexer.TOKEN_LPAREN, "expected '('")
      condition := p.expression()
      p.consume(lexer.TOKEN_RPAREN, "expected ')'")
      then := p.statement()
      var elseStmt Statement
      if p.match(lexer.TOKEN_ELSE) {
          elseStmt = p.statement()
      }
      return &IfStmt{Condition: condition, Then: then, Else: elseStmt}
  }

  func (p *Parser) whileStmt() *WhileStmt {
      p.advance() // consume "while"
      p.consume(lexer.TOKEN_LPAREN, "expected '('")
      condition := p.expression()
      p.consume(lexer.TOKEN_RPAREN, "expected ')'")
      body := p.statement()
      return &WhileStmt{Condition: condition, Body: body}
  }

  func (p *Parser) returnStmt() *ReturnStmt {
      tok := p.advance() // consume "return"
      value := p.expression()
      p.consume(lexer.TOKEN_SEMICOLON, "expected ';' after return")
      return &ReturnStmt{Value: value, Line: tok.Line}
  }

  func (p *Parser) exprStmt() *ExprStmt {
      expr := p.expression()
      p.consume(lexer.TOKEN_SEMICOLON, "expected ';'")
      return &ExprStmt{Expr: expr}
  }

  // ─── Expressions (Precedence Climbing) ────────────────────────────
  //
  // Each precedence level is a function that calls the next higher level.
  // This naturally implements left-to-right, lowest-to-highest precedence.
  //
  //  expression     → assignment
  //  assignment     → logicalOr ( "=" assignment )?     (right-associative)
  //  logicalOr      → logicalAnd ( "||" logicalAnd )*
  //  logicalAnd     → equality ( "&&" equality )*
  //  equality       → comparison ( ("==" | "!=") comparison )*
  //  comparison     → additive ( ("<" | ">" | "<=" | ">=") additive )*
  //  additive       → multiplicative ( ("+" | "-") multiplicative )*
  //  multiplicative → unary ( ("*" | "/" | "%") unary )*
  //  unary          → ("-" | "!") unary | primary
  //  primary        → INT_LITERAL | IDENTIFIER | IDENTIFIER "(" args ")" | "(" expr ")"

  func (p *Parser) expression() Expression {
      return p.assignment()
  }

  func (p *Parser) assignment() Expression {
      // We need to handle: x = expr
      // Since assignment is right-associative, we parse the left side,
      // check if it's an identifier followed by '=', then recurse.
      expr := p.logicalOr()

      if p.match(lexer.TOKEN_ASSIGN) {
          value := p.assignment() // right-associative: recurse
          if ident, ok := expr.(*Identifier); ok {
              return &AssignExpr{Name: ident.Name, Value: value, Line: ident.Line}
          }
          p.error("invalid assignment target")
      }
      return expr
  }

  func (p *Parser) logicalOr() Expression {
      left := p.logicalAnd()
      for p.match(lexer.TOKEN_OR) {
          right := p.logicalAnd()
          left = &BinaryExpr{Op: lexer.TOKEN_OR, Left: left, Right: right}
      }
      return left
  }

  func (p *Parser) logicalAnd() Expression {
      left := p.equality()
      for p.match(lexer.TOKEN_AND) {
          right := p.equality()
          left = &BinaryExpr{Op: lexer.TOKEN_AND, Left: left, Right: right}
      }
      return left
  }

  func (p *Parser) equality() Expression {
      left := p.comparison()
      for p.check(lexer.TOKEN_EQUAL) || p.check(lexer.TOKEN_NOT_EQUAL) {
          op := p.advance()
          right := p.comparison()
          left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
      }
      return left
  }

  func (p *Parser) comparison() Expression {
      left := p.additive()
      for p.check(lexer.TOKEN_LESS) || p.check(lexer.TOKEN_GREATER) ||
          p.check(lexer.TOKEN_LESS_EQ) || p.check(lexer.TOKEN_GREATER_EQ) {
          op := p.advance()
          right := p.additive()
          left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
      }
      return left
  }

  func (p *Parser) additive() Expression {
      left := p.multiplicative()
      for p.check(lexer.TOKEN_PLUS) || p.check(lexer.TOKEN_MINUS) {
          op := p.advance()
          right := p.multiplicative()
          left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
      }
      return left
  }

  func (p *Parser) multiplicative() Expression {
      left := p.unary()
      for p.check(lexer.TOKEN_STAR) || p.check(lexer.TOKEN_SLASH) ||
          p.check(lexer.TOKEN_PERCENT) {
          op := p.advance()
          right := p.unary()
          left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
      }
      return left
  }

  func (p *Parser) unary() Expression {
      if p.check(lexer.TOKEN_MINUS) || p.check(lexer.TOKEN_NOT) {
          op := p.advance()
          operand := p.unary() // right-associative
          return &UnaryExpr{Op: op.Type, Operand: operand}
      }
      return p.primary()
  }

  func (p *Parser) primary() Expression {
      tok := p.peek()
      switch tok.Type {
      case lexer.TOKEN_INT_LITERAL:
          p.advance()
          val, _ := strconv.Atoi(tok.Lexeme)
          return &IntLiteral{Value: val}

      case lexer.TOKEN_IDENTIFIER:
          p.advance()
          if p.match(lexer.TOKEN_LPAREN) {
              // Function call
              args := p.argList()
              p.consume(lexer.TOKEN_RPAREN, "expected ')' after arguments")
              return &CallExpr{Name: tok.Lexeme, Args: args, Line: tok.Line}
          }
          return &Identifier{Name: tok.Lexeme, Line: tok.Line}

      case lexer.TOKEN_LPAREN:
          p.advance()
          expr := p.expression()
          p.consume(lexer.TOKEN_RPAREN, "expected ')'")
          return expr

      default:
          p.error(fmt.Sprintf("unexpected token %s", tok.Type))
          p.advance()
          return &IntLiteral{Value: 0} // error recovery: return dummy
      }
  }

  func (p *Parser) argList() []Expression {
      var args []Expression
      if p.check(lexer.TOKEN_RPAREN) {
          return args
      }
      args = append(args, p.expression())
      for p.match(lexer.TOKEN_COMMA) {
          args = append(args, p.expression())
      }
      return args
  }

  // ─── Utility Methods ──────────────────────────────────────────────

  func (p *Parser) peek() lexer.Token {
      return p.tokens[p.current]
  }

  func (p *Parser) advance() lexer.Token {
      tok := p.tokens[p.current]
      if !p.isAtEnd() {
          p.current++
      }
      return tok
  }

  func (p *Parser) check(t lexer.TokenType) bool {
      return p.peek().Type == t
  }

  func (p *Parser) match(t lexer.TokenType) bool {
      if p.check(t) {
          p.advance()
          return true
      }
      return false
  }

  func (p *Parser) consume(t lexer.TokenType, msg string) lexer.Token {
      if p.check(t) {
          return p.advance()
      }
      p.error(fmt.Sprintf("%s (got %s at %d:%d)",
          msg, p.peek().Type, p.peek().Line, p.peek().Column))
      return p.peek()
  }

  func (p *Parser) isAtEnd() bool {
      return p.peek().Type == lexer.TOKEN_EOF
  }

  func (p *Parser) error(msg string) {
      tok := p.peek()
      p.errors = append(p.errors, fmt.Sprintf(
          "parse error at %d:%d: %s", tok.Line, tok.Column, msg,
      ))
  }


M2.4  UNDERSTANDING OPERATOR PRECEDENCE
===============================================================================

The chain of functions naturally encodes precedence:

  expression() → assignment() → logicalOr() → logicalAnd() → equality()
               → comparison() → additive() → multiplicative() → unary()
               → primary()

Lower precedence = called first (outermost in the tree).
Higher precedence = called last (innermost in the tree).

Example: Parsing "1 + 2 * 3"

  1. additive() is called
  2. It calls multiplicative() → unary() → primary() → returns IntLit(1)
  3. Sees '+', consumes it
  4. Calls multiplicative() again
     a. multiplicative() calls primary() → returns IntLit(2)
     b. Sees '*', consumes it
     c. Calls primary() → returns IntLit(3)
     d. Returns BinaryExpr(*, IntLit(2), IntLit(3))
  5. Returns BinaryExpr(+, IntLit(1), BinaryExpr(*, IntLit(2), IntLit(3)))

  AST:
       (+)
      /   \
   1      (*)
         /   \
        2     3

  This correctly gives * higher precedence than +.


M2.5  EXERCISES
===============================================================================

Exercise 5.1: AST Printer
  Write a function that pretty-prints an AST with indentation:
    Program
      FuncDecl: main ()
        Block
          VarDecl: x = IntLit(42)
          ReturnStmt
            Ident: x

Exercise 5.2: Error Recovery
  Implement "panic mode" error recovery: when a parse error occurs,
  skip tokens until a synchronization point (';' or '}') and continue.
  Test with a program containing multiple errors.

Exercise 5.3: Parse Tree Visualization
  For the expression "a + b * c - d", draw both the parse tree
  (following the grammar rules literally) and the AST. Count nodes
  in each. Why is the AST preferred?

Exercise 5.4: Dangling Else
  Parse "if (a) if (b) x = 1; else x = 2;" by hand following our
  recursive descent. Which "if" does "else" bind to? Why?


███████████████████████████████████████████████████████████████████████████████
  MODULE 3 — SEMANTIC ANALYSIS (Week 6)
███████████████████████████████████████████████████████████████████████████████


M3.1  THEORY: WHAT SEMANTIC ANALYSIS DOES
===============================================================================

The parser ensures the program is syntactically valid. Semantic analysis
ensures it is MEANINGFUL. Specifically:

  1. Variable resolution: Every used name is declared.
  2. Duplicate detection: No name is declared twice in the same scope.
  3. Function validation: Calls match declarations in argument count.
  4. Type checking: (Trivial for Mini-C since we only have int.)
  5. Main function: Program must have a main() with no parameters.

These checks require information that context-free grammars cannot express.
They require the CONTEXT of where names are declared — hence, symbol tables.


M3.2  SYMBOL TABLE
===============================================================================

A symbol table maps names to their properties within a scope.

```
  +-----------------------------------------+
  |  Scope Chain (Linked List of Scopes)    |
  |                                         |
  |  Global Scope                           |
  |  +-----------------------------------+  |
  |  | "counter" -> {kind:var, offset:0} |  |
  |  | "increment" -> {kind:func, arity:0}| |
  |  | "main" -> {kind:func, arity:0}    |  |
  |  +--------------+--------------------+  |
  |                 | parent                |
  |  Function Scope (main)                  |
  |  +--------------+--------------------+  |
  |  | "x" -> {kind:var, offset:-8}      |  |
  |  | "y" -> {kind:var, offset:-16}     |  |
  |  +--------------+--------------------+  |
  |                 | parent                |
  |  Block Scope (inside if)                |
  |  +--------------+--------------------+  |
  |  | "z" -> {kind:var, offset:-24}     |  |
  |  +-----------------------------------+  |
  +-----------------------------------------+
```

  Lookup: search current scope first, then parent, then grandparent, etc.


M3.3  GO IMPLEMENTATION
===============================================================================

  // pkg/semantic/symtable.go

  package semantic

  import "fmt"

  type SymbolKind int

  const (
      SymVar  SymbolKind = iota
      SymFunc
  )

  type Symbol struct {
      Name   string
      Kind   SymbolKind
      Arity  int   // Number of parameters (for functions)
      Offset int   // Stack offset (for local variables, assigned during codegen)
      Global bool  // Whether this is a global variable
  }

  type Scope struct {
      symbols map[string]*Symbol
      parent  *Scope
  }

  func NewScope(parent *Scope) *Scope {
      return &Scope{
          symbols: make(map[string]*Symbol),
          parent:  parent,
      }
  }

  // Define adds a symbol to the current scope.
  // Returns an error if the name is already defined in THIS scope.
  func (s *Scope) Define(sym *Symbol) error {
      if _, exists := s.symbols[sym.Name]; exists {
          return fmt.Errorf("'%s' already declared in this scope", sym.Name)
      }
      s.symbols[sym.Name] = sym
      return nil
  }

  // Resolve looks up a name, searching outward through parent scopes.
  func (s *Scope) Resolve(name string) (*Symbol, bool) {
      if sym, ok := s.symbols[name]; ok {
          return sym, true
      }
      if s.parent != nil {
          return s.parent.Resolve(name)
      }
      return nil, false
  }


  // pkg/semantic/analyzer.go

  package semantic

  import (
      "fmt"
      "github.com/user/minicc/pkg/parser"
  )

  type Analyzer struct {
      scope  *Scope
      errors []string
  }

  func New() *Analyzer {
      return &Analyzer{
          scope: NewScope(nil), // global scope
      }
  }

  func (a *Analyzer) Analyze(prog *parser.Program) []string {
      // First pass: register all top-level declarations
      // This allows functions to call each other regardless of order
      // (only at global level — local vars still require declaration before use)
      for _, decl := range prog.Declarations {
          switch d := decl.(type) {
          case *parser.FuncDecl:
              a.defineFunc(d)
          case *parser.VarDecl:
              a.defineGlobalVar(d)
          }
      }

      // Second pass: analyze all function bodies
      for _, decl := range prog.Declarations {
          if fd, ok := decl.(*parser.FuncDecl); ok {
              a.analyzeFunc(fd)
          }
      }

      // Check for main function
      sym, ok := a.scope.Resolve("main")
      if !ok || sym.Kind != SymFunc {
          a.error("no 'main' function defined")
      } else if sym.Arity != 0 {
          a.error("'main' must have no parameters")
      }

      return a.errors
  }

  func (a *Analyzer) defineFunc(fd *parser.FuncDecl) {
      sym := &Symbol{Name: fd.Name, Kind: SymFunc, Arity: len(fd.Params)}
      if err := a.scope.Define(sym); err != nil {
          a.errorAt(fd.Line, err.Error())
      }
  }

  func (a *Analyzer) defineGlobalVar(vd *parser.VarDecl) {
      sym := &Symbol{Name: vd.Name, Kind: SymVar, Global: true}
      if err := a.scope.Define(sym); err != nil {
          a.errorAt(vd.Line, err.Error())
      }
  }

  func (a *Analyzer) analyzeFunc(fd *parser.FuncDecl) {
      // Enter function scope
      a.scope = NewScope(a.scope)

      // Define parameters
      for _, param := range fd.Params {
          sym := &Symbol{Name: param.Name, Kind: SymVar}
          if err := a.scope.Define(sym); err != nil {
              a.errorAt(fd.Line, err.Error())
          }
      }

      // Analyze body
      a.analyzeBlock(fd.Body)

      // Exit function scope
      a.scope = a.scope.parent
  }

  func (a *Analyzer) analyzeBlock(block *parser.BlockStmt) {
      a.scope = NewScope(a.scope)
      for _, stmt := range block.Stmts {
          a.analyzeStmt(stmt)
      }
      a.scope = a.scope.parent
  }

  func (a *Analyzer) analyzeStmt(stmt parser.Statement) {
      switch s := stmt.(type) {
      case *parser.VarDecl:
          if s.Init != nil {
              a.analyzeExpr(s.Init)
          }
          sym := &Symbol{Name: s.Name, Kind: SymVar}
          if err := a.scope.Define(sym); err != nil {
              a.errorAt(s.Line, err.Error())
          }
      case *parser.ExprStmt:
          a.analyzeExpr(s.Expr)
      case *parser.IfStmt:
          a.analyzeExpr(s.Condition)
          a.analyzeStmt(s.Then)
          if s.Else != nil {
              a.analyzeStmt(s.Else)
          }
      case *parser.WhileStmt:
          a.analyzeExpr(s.Condition)
          a.analyzeStmt(s.Body)
      case *parser.ReturnStmt:
          a.analyzeExpr(s.Value)
      case *parser.BlockStmt:
          a.analyzeBlock(s)
      }
  }

  func (a *Analyzer) analyzeExpr(expr parser.Expression) {
      switch e := expr.(type) {
      case *parser.IntLiteral:
          // Always valid
      case *parser.Identifier:
          if _, ok := a.scope.Resolve(e.Name); !ok {
              a.errorAt(e.Line, fmt.Sprintf("'%s' undeclared", e.Name))
          }
      case *parser.BinaryExpr:
          a.analyzeExpr(e.Left)
          a.analyzeExpr(e.Right)
      case *parser.UnaryExpr:
          a.analyzeExpr(e.Operand)
      case *parser.AssignExpr:
          if _, ok := a.scope.Resolve(e.Name); !ok {
              a.errorAt(e.Line, fmt.Sprintf("'%s' undeclared", e.Name))
          }
          a.analyzeExpr(e.Value)
      case *parser.CallExpr:
          sym, ok := a.scope.Resolve(e.Name)
          if !ok {
              a.errorAt(e.Line, fmt.Sprintf("'%s' undeclared", e.Name))
          } else if sym.Kind != SymFunc {
              a.errorAt(e.Line, fmt.Sprintf("'%s' is not a function", e.Name))
          } else if len(e.Args) != sym.Arity {
              a.errorAt(e.Line, fmt.Sprintf(
                  "'%s' expects %d arguments, got %d",
                  e.Name, sym.Arity, len(e.Args)))
          }
          for _, arg := range e.Args {
              a.analyzeExpr(arg)
          }
      }
  }

  func (a *Analyzer) error(msg string) {
      a.errors = append(a.errors, "semantic error: "+msg)
  }

  func (a *Analyzer) errorAt(line int, msg string) {
      a.errors = append(a.errors, fmt.Sprintf("semantic error at line %d: %s", line, msg))
  }


M3.4  EXERCISES
===============================================================================

Exercise 6.1: Scope Visualization
  For the "Nested Scopes" program from Part 2 (Program 7), draw the
  scope chain at each point where 'x' is referenced. Show which
  Symbol each reference resolves to.

Exercise 6.2: Unused Variable Warning
  Add a warning (not error) for variables that are declared but never
  used. Track usage counts in the Symbol struct.

Exercise 6.3: Forward Reference Functions
  Our current two-pass approach allows mutual recursion at global scope:
    int isEven(int n) { if (n==0) return 1; return isOdd(n-1); }
    int isOdd(int n)  { if (n==0) return 0; return isEven(n-1); }
  Test this. Why does two-pass make it work?


███████████████████████████████████████████████████████████████████████████████
  MODULE 4 — INTERMEDIATE REPRESENTATION (Week 7)
███████████████████████████████████████████████████████████████████████████████


M4.1  THEORY: WHY AN IR?
===============================================================================

We could generate assembly directly from the AST. But an intermediate
representation (IR) provides critical benefits:

  1. DECOUPLING: The IR separates the front-end (language-specific) from
     the back-end (architecture-specific). Supporting a new architecture
     means writing a new code generator, not a new parser.

  2. OPTIMIZATION: Many optimizations (constant folding, dead code
     elimination, common subexpression elimination) are easier to
     express and implement on a linear IR than on a tree.

  3. ANALYSIS: Control flow analysis, liveness analysis, and register
     allocation all operate on the IR's Control Flow Graph (CFG).

Our IR: Three-Address Code (TAC)

  Every instruction has at most three operands:
    result = operand1  op  operand2

  Temporary variables are generated for subexpressions:

  Source:    return a + b * c - d;

  Three-Address Code:
    t1 = b * c
    t2 = a + t1
    t3 = t2 - d
    return t3


M4.2  IR DATA STRUCTURES
===============================================================================

  // pkg/ir/ir.go

  package ir

  import "fmt"

  // OpCode represents the operation type.
  type OpCode int

  const (
      OpAdd     OpCode = iota  // dst = left + right
      OpSub                    // dst = left - right
      OpMul                    // dst = left * right
      OpDiv                    // dst = left / right
      OpMod                    // dst = left % right
      OpNeg                    // dst = -left
      OpNot                    // dst = !left
      OpEq                     // dst = (left == right)
      OpNeq                    // dst = (left != right)
      OpLt                     // dst = (left < right)
      OpGt                     // dst = (left > right)
      OpLte                    // dst = (left <= right)
      OpGte                    // dst = (left >= right)
      OpAnd                    // dst = left && right (short-circuit via jumps)
      OpOr                     // dst = left || right (short-circuit via jumps)
      OpCopy                   // dst = left
      OpLoadImm                // dst = immediate value
      OpCall                   // dst = call func(args...)
      OpParam                  // push parameter for upcoming call
      OpReturn                 // return left
      OpJump                   // unconditional jump to label
      OpJumpIfZero             // if left == 0, jump to label
      OpJumpIfNotZero          // if left != 0, jump to label
      OpLabel                  // label definition
      OpLoadGlobal             // dst = global variable
      OpStoreGlobal            // global variable = left
  )

  // Operand represents a value in the IR.
  type Operand struct {
      Kind  OperandKind
      Name  string   // variable name or label
      Value int      // immediate value
      Temp  int      // temporary number (t0, t1, ...)
  }

  type OperandKind int

  const (
      OpndNone   OperandKind = iota
      OpndTemp               // Compiler-generated temporary
      OpndVar                // Named variable
      OpndImm                // Immediate integer
      OpndLabel              // Label reference
      OpndFunc               // Function name (for calls)
  )

  func (o Operand) String() string {
      switch o.Kind {
      case OpndTemp:
          return fmt.Sprintf("t%d", o.Temp)
      case OpndVar:
          return o.Name
      case OpndImm:
          return fmt.Sprintf("%d", o.Value)
      case OpndLabel:
          return o.Name
      case OpndFunc:
          return o.Name
      default:
          return "_"
      }
  }

  // Instruction is a single three-address code instruction.
  type Instruction struct {
      Op    OpCode
      Dst   Operand
      Left  Operand
      Right Operand
      // For calls: function name and argument count
      FuncName string
      ArgCount int
  }

  func (i Instruction) String() string {
      switch i.Op {
      case OpAdd, OpSub, OpMul, OpDiv, OpMod,
           OpEq, OpNeq, OpLt, OpGt, OpLte, OpGte:
          return fmt.Sprintf("  %s = %s %s %s",
              i.Dst, i.Left, opSymbol(i.Op), i.Right)
      case OpNeg:
          return fmt.Sprintf("  %s = -%s", i.Dst, i.Left)
      case OpNot:
          return fmt.Sprintf("  %s = !%s", i.Dst, i.Left)
      case OpCopy:
          return fmt.Sprintf("  %s = %s", i.Dst, i.Left)
      case OpLoadImm:
          return fmt.Sprintf("  %s = %s", i.Dst, i.Left)
      case OpCall:
          return fmt.Sprintf("  %s = call %s (%d args)", i.Dst, i.FuncName, i.ArgCount)
      case OpParam:
          return fmt.Sprintf("  param %s", i.Left)
      case OpReturn:
          return fmt.Sprintf("  return %s", i.Left)
      case OpJump:
          return fmt.Sprintf("  jump %s", i.Dst)
      case OpJumpIfZero:
          return fmt.Sprintf("  if %s == 0 jump %s", i.Left, i.Dst)
      case OpJumpIfNotZero:
          return fmt.Sprintf("  if %s != 0 jump %s", i.Left, i.Dst)
      case OpLabel:
          return fmt.Sprintf("%s:", i.Dst)
      case OpLoadGlobal:
          return fmt.Sprintf("  %s = load_global %s", i.Dst, i.Left)
      case OpStoreGlobal:
          return fmt.Sprintf("  store_global %s = %s", i.Dst, i.Left)
      default:
          return fmt.Sprintf("  ??? %v", i.Op)
      }
  }

  func opSymbol(op OpCode) string {
      switch op {
      case OpAdd: return "+"
      case OpSub: return "-"
      case OpMul: return "*"
      case OpDiv: return "/"
      case OpMod: return "%"
      case OpEq:  return "=="
      case OpNeq: return "!="
      case OpLt:  return "<"
      case OpGt:  return ">"
      case OpLte: return "<="
      case OpGte: return ">="
      default:    return "?"
      }
  }

  // Function represents a function's IR.
  type Function struct {
      Name         string
      Params       []string
      Instructions []Instruction
      LocalCount   int  // number of local variables (for stack allocation)
  }

  // Program is the top-level IR container.
  type IRProgram struct {
      Globals   []string
      Functions []*Function
  }


M4.3  IR BUILDER — AST TO THREE-ADDRESS CODE
===============================================================================

  // pkg/ir/builder.go

  package ir

  import (
      "fmt"
      "github.com/user/minicc/pkg/parser"
      "github.com/user/minicc/pkg/lexer"
  )

  type Builder struct {
      tempCount  int
      labelCount int
      current    *Function
      program    *IRProgram
      localVars  map[string]bool  // track which vars are local
  }

  func NewBuilder() *Builder {
      return &Builder{
          program: &IRProgram{},
      }
  }

  func (b *Builder) Build(prog *parser.Program) *IRProgram {
      for _, decl := range prog.Declarations {
          switch d := decl.(type) {
          case *parser.VarDecl:
              b.program.Globals = append(b.program.Globals, d.Name)
          case *parser.FuncDecl:
              b.buildFunc(d)
          }
      }
      return b.program
  }

  func (b *Builder) buildFunc(fd *parser.FuncDecl) {
      b.current = &Function{Name: fd.Name}
      b.localVars = make(map[string]bool)

      for _, p := range fd.Params {
          b.current.Params = append(b.current.Params, p.Name)
          b.localVars[p.Name] = true
      }

      b.buildBlock(fd.Body)
      b.program.Functions = append(b.program.Functions, b.current)
  }

  func (b *Builder) buildBlock(block *parser.BlockStmt) {
      for _, stmt := range block.Stmts {
          b.buildStmt(stmt)
      }
  }

  func (b *Builder) buildStmt(stmt parser.Statement) {
      switch s := stmt.(type) {
      case *parser.VarDecl:
          b.localVars[s.Name] = true
          b.current.LocalCount++
          if s.Init != nil {
              val := b.buildExpr(s.Init)
              b.emit(Instruction{
                  Op:  OpCopy,
                  Dst: Operand{Kind: OpndVar, Name: s.Name},
                  Left: val,
              })
          }

      case *parser.ExprStmt:
          b.buildExpr(s.Expr)

      case *parser.ReturnStmt:
          val := b.buildExpr(s.Value)
          b.emit(Instruction{Op: OpReturn, Left: val})

      case *parser.IfStmt:
          cond := b.buildExpr(s.Condition)
          elseLabel := b.newLabel("else")
          endLabel := b.newLabel("endif")

          b.emit(Instruction{
              Op:   OpJumpIfZero,
              Left: cond,
              Dst:  Operand{Kind: OpndLabel, Name: elseLabel},
          })
          b.buildStmt(s.Then)
          if s.Else != nil {
              b.emit(Instruction{
                  Op:  OpJump,
                  Dst: Operand{Kind: OpndLabel, Name: endLabel},
              })
          }
          b.emit(Instruction{
              Op:  OpLabel,
              Dst: Operand{Kind: OpndLabel, Name: elseLabel},
          })
          if s.Else != nil {
              b.buildStmt(s.Else)
              b.emit(Instruction{
                  Op:  OpLabel,
                  Dst: Operand{Kind: OpndLabel, Name: endLabel},
              })
          }

      case *parser.WhileStmt:
          loopLabel := b.newLabel("while")
          endLabel := b.newLabel("endwhile")

          b.emit(Instruction{
              Op:  OpLabel,
              Dst: Operand{Kind: OpndLabel, Name: loopLabel},
          })
          cond := b.buildExpr(s.Condition)
          b.emit(Instruction{
              Op:   OpJumpIfZero,
              Left: cond,
              Dst:  Operand{Kind: OpndLabel, Name: endLabel},
          })
          b.buildStmt(s.Body)
          b.emit(Instruction{
              Op:  OpJump,
              Dst: Operand{Kind: OpndLabel, Name: loopLabel},
          })
          b.emit(Instruction{
              Op:  OpLabel,
              Dst: Operand{Kind: OpndLabel, Name: endLabel},
          })

      case *parser.BlockStmt:
          b.buildBlock(s)
      }
  }

  func (b *Builder) buildExpr(expr parser.Expression) Operand {
      switch e := expr.(type) {
      case *parser.IntLiteral:
          dst := b.newTemp()
          b.emit(Instruction{
              Op:   OpLoadImm,
              Dst:  dst,
              Left: Operand{Kind: OpndImm, Value: e.Value},
          })
          return dst

      case *parser.Identifier:
          if b.isGlobal(e.Name) {
              dst := b.newTemp()
              b.emit(Instruction{
                  Op:   OpLoadGlobal,
                  Dst:  dst,
                  Left: Operand{Kind: OpndVar, Name: e.Name},
              })
              return dst
          }
          return Operand{Kind: OpndVar, Name: e.Name}

      case *parser.BinaryExpr:
          left := b.buildExpr(e.Left)
          right := b.buildExpr(e.Right)
          dst := b.newTemp()
          b.emit(Instruction{
              Op:    tokenToOp(e.Op),
              Dst:   dst,
              Left:  left,
              Right: right,
          })
          return dst

      case *parser.UnaryExpr:
          operand := b.buildExpr(e.Operand)
          dst := b.newTemp()
          op := OpNeg
          if e.Op == lexer.TOKEN_NOT {
              op = OpNot
          }
          b.emit(Instruction{Op: op, Dst: dst, Left: operand})
          return dst

      case *parser.AssignExpr:
          val := b.buildExpr(e.Value)
          if b.isGlobal(e.Name) {
              b.emit(Instruction{
                  Op:   OpStoreGlobal,
                  Dst:  Operand{Kind: OpndVar, Name: e.Name},
                  Left: val,
              })
          } else {
              b.emit(Instruction{
                  Op:   OpCopy,
                  Dst:  Operand{Kind: OpndVar, Name: e.Name},
                  Left: val,
              })
          }
          return val

      case *parser.CallExpr:
          // Emit parameters in order
          for _, arg := range e.Args {
              argVal := b.buildExpr(arg)
              b.emit(Instruction{Op: OpParam, Left: argVal})
          }
          dst := b.newTemp()
          b.emit(Instruction{
              Op:       OpCall,
              Dst:      dst,
              FuncName: e.Name,
              ArgCount: len(e.Args),
          })
          return dst

      default:
          return Operand{Kind: OpndImm, Value: 0}
      }
  }

  // ─── Helpers ──────────────────────────────────────────────────────

  func (b *Builder) emit(instr Instruction) {
      b.current.Instructions = append(b.current.Instructions, instr)
  }

  func (b *Builder) newTemp() Operand {
      t := b.tempCount
      b.tempCount++
      return Operand{Kind: OpndTemp, Temp: t}
  }

  func (b *Builder) newLabel(prefix string) string {
      l := b.labelCount
      b.labelCount++
      return fmt.Sprintf(".L_%s_%d", prefix, l)
  }

  func (b *Builder) isGlobal(name string) bool {
      if b.localVars[name] {
          return false
      }
      for _, g := range b.program.Globals {
          if g == name {
              return true
          }
      }
      return false
  }

  func tokenToOp(t lexer.TokenType) OpCode {
      switch t {
      case lexer.TOKEN_PLUS:       return OpAdd
      case lexer.TOKEN_MINUS:      return OpSub
      case lexer.TOKEN_STAR:       return OpMul
      case lexer.TOKEN_SLASH:      return OpDiv
      case lexer.TOKEN_PERCENT:    return OpMod
      case lexer.TOKEN_EQUAL:      return OpEq
      case lexer.TOKEN_NOT_EQUAL:  return OpNeq
      case lexer.TOKEN_LESS:       return OpLt
      case lexer.TOKEN_GREATER:    return OpGt
      case lexer.TOKEN_LESS_EQ:    return OpLte
      case lexer.TOKEN_GREATER_EQ: return OpGte
      case lexer.TOKEN_AND:        return OpAnd
      case lexer.TOKEN_OR:         return OpOr
      default:                     return OpAdd
      }
  }


M4.4  BASIC BLOCKS AND CONTROL FLOW GRAPHS
===============================================================================

A Basic Block is a maximal sequence of instructions with:
  - No jumps except possibly the last instruction
  - No labels except possibly the first instruction
  - Execution enters at the first instruction and exits at the last

Basic blocks are the nodes of the Control Flow Graph (CFG).

  Example — Factorial:

  int factorial(int n) {
      int result = 1;
      while (n > 1) {
          result = result * n;
          n = n - 1;
      }
      return result;
  }

  Three-Address Code:
    B0:  result = 1                     ← Basic Block 0
    .L_while_0:
    B1:  t0 = n > 1                     ← Basic Block 1
         if t0 == 0 jump .L_endwhile_1
    B2:  t1 = result * n                ← Basic Block 2
         result = t1
         t2 = n - 1
         n = t2
         jump .L_while_0
    .L_endwhile_1:
    B3:  return result                   ← Basic Block 3

  CFG:
```
         +-----+
         | B0  |
         +--+--+
            |
            v
         +-----+
    +--->| B1  | (loop header: test condition)
    |    +--+--+
    |       | true        false
    |       v               |
    |    +-----+            |
    |    | B2  |            |
    |    +--+--+            |
    |       | (back edge)   |
    +-------+               |
                            v
                         +-----+
                         | B3  |
                         +-----+
```

The CFG is the foundation for optimization and register allocation.
We don't implement full optimizations in this course, but understanding
the CFG is essential for understanding compilers at a professional level.


M4.5  SSA FORM — INTRODUCTION
===============================================================================

Static Single Assignment (SSA) form is an IR property where every
variable is assigned exactly once.

  Non-SSA:               SSA:
    x = 1                  x1 = 1
    x = x + 1              x2 = x1 + 1
    y = x * 2              y1 = x2 * 2

  At control flow merges, SSA introduces φ (phi) functions:

    if (cond)              if (cond)
      x = 1                  x1 = 1
    else                   else
      x = 2                  x2 = 2
    use(x)                 x3 = φ(x1, x2)
                           use(x3)

SSA is used by LLVM, GCC, and all modern compilers for optimization.
We do NOT implement SSA in this course — but understanding it helps
you read LLVM IR and production compiler internals.


M4.6  EXERCISES
===============================================================================

Exercise 7.1: IR Pretty-Printer
  Print the IR for the factorial program. Verify it matches the
  expected three-address code.

Exercise 7.2: Constant Folding
  Write an optimization pass that folds constant expressions:
    t0 = 3 + 4   →   t0 = 7
  Apply it to the IR before code generation.

Exercise 7.3: Dead Code Elimination
  Write a pass that removes instructions whose results are never used.

Exercise 7.4: CFG Builder
  Write a function that takes a list of IR instructions and produces
  a list of BasicBlock structs. Each block has a list of instructions
  and lists of successor/predecessor blocks.


███████████████████████████████████████████████████████████████████████████████
  MODULE 5 — CODE GENERATION (Week 8)
███████████████████████████████████████████████████████████████████████████████


M5.1  THEORY: CODE GENERATION
===============================================================================

Code generation translates IR into target machine instructions.
Our target is x86-64 assembly in AT&T syntax.

Key decisions:
  1. Register allocation: Which values go in registers vs on the stack?
  2. Instruction selection: Which x86 instruction implements each IR op?
  3. Calling convention: How do we call and return from functions?
  4. Stack frame layout: Where are local variables?

We use a SIMPLE strategy: stack-based allocation.
  - Every local variable and temporary lives on the stack.
  - We use RAX as the primary accumulator.
  - We load operands from the stack into registers, compute, and store back.

This is "correct but slow." Production compilers do register allocation
to keep values in registers as much as possible. We optimize for
simplicity and correctness.


M5.2  x86-64 REGISTER REFERENCE
===============================================================================

```
  64-bit   32-bit   16-bit   8-bit    Purpose (System V ABI)
  -------  -------  -------  -------  -------------------------
  RAX      EAX      AX       AL       Return value, accumulator
  RBX      EBX      BX       BL       Callee-saved
  RCX      ECX      CX       CL       4th integer argument
  RDX      EDX      DX       DL       3rd integer argument
  RSI      ESI      SI       SIL      2nd integer argument
  RDI      EDI      DI       DIL      1st integer argument
  RBP      EBP      BP       BPL      Base pointer (frame pointer)
  RSP      ESP      SP       SPL      Stack pointer
  R8       R8D      R8W      R8B      5th integer argument
  R9       R9D      R9W      R9B      6th integer argument
  R10      R10D     R10W     R10B     Caller-saved
  R11      R11D     R11W     R11B     Caller-saved
  R12-R15  R12D..   R12W..   R12B..   Callee-saved
```

  For 32-bit int operations, we use the 32-bit register names (EAX, etc.)
  Operations on 32-bit registers automatically zero the upper 32 bits.

  Key x86-64 instructions we'll emit:
```
  movl  src, dst        Move 32-bit value
  addl  src, dst        dst += src
  subl  src, dst        dst -= src
  imull src, dst        dst *= src
  cdq                   Sign-extend EAX into EDX:EAX (for division)
  idivl src             EDX:EAX / src → EAX (quotient), EDX (remainder)
  negl  dst             dst = -dst
  cmpl  src, dst        Compare (sets flags)
  sete  dst             Set byte if equal (ZF=1)
  setne dst             Set byte if not equal
  setl  dst             Set byte if less (SF≠OF)
  setg  dst             Set byte if greater
  setle dst             Set byte if less or equal
  setge dst             Set byte if greater or equal
  movzbl src, dst       Zero-extend byte to 32-bit
  je    label           Jump if equal
  jne   label           Jump if not equal
  jmp   label           Unconditional jump
  pushq src             Push 64-bit value to stack
  popq  dst             Pop 64-bit value from stack
  call  label           Push return address, jump to label
  ret                   Pop return address, jump to it
```


M5.3  STACK FRAME LAYOUT FOR OUR COMPILER
===============================================================================

  For a function with N parameters and M local variables + temporaries:

```
  High Address
  +----------------------+
  |  Argument 7+         |  (if more than 6 args -- we support up to 6)
  +----------------------+
  |  Return Address      |  <-- pushed by CALL
  +----------------------+  <-- RBP points here
  |  Saved RBP           |  -0(%rbp)
  +----------------------+
  |  Local var 1         |  -8(%rbp)    [or -4 for int with tight packing]
  +----------------------+
  |  Local var 2         |  -16(%rbp)
  +----------------------+
  |  Temp 0              |  -24(%rbp)
  +----------------------+
  |  Temp 1              |  -32(%rbp)
  +----------------------+
  |  ...                 |
  +----------------------+  <-- RSP (after sub $N, %rsp)
```
  Low Address

  We allocate 8 bytes per variable/temporary for simplicity (wastes 4 bytes
  per int, but keeps alignment simple and avoids mixing 4-byte/8-byte offsets).

  Parameters (up to 6):
    Param 1 (RDI) → stored at first local slot after prologue
    Param 2 (RSI) → stored at second local slot
    ... and so on

  Function prologue:
    pushq   %rbp
    movq    %rsp, %rbp
    subq    $FRAME_SIZE, %rsp   # FRAME_SIZE = 8 * (params + locals + temps)
    # Spill parameters from registers to stack
    movl    %edi, -8(%rbp)      # param 1
    movl    %esi, -16(%rbp)     # param 2
    ...

  Function epilogue:
    movq    %rbp, %rsp
    popq    %rbp
    retq


M5.4  GO IMPLEMENTATION — CODE GENERATOR
===============================================================================

  // pkg/codegen/x86_64.go

  package codegen

  import (
      "fmt"
      "strings"
      "github.com/user/minicc/pkg/ir"
  )

  // Target platform affects symbol naming and directives.
  type Platform int

  const (
      PlatformMacOS Platform = iota
      PlatformLinux
  )

  type CodeGen struct {
      output   strings.Builder
      platform Platform
      // Stack layout for current function
      varOffset map[string]int  // variable/temp name → stack offset from RBP
      nextOffset int            // next available stack offset (negative)
  }

  func New(platform Platform) *CodeGen {
      return &CodeGen{platform: platform}
  }

  func (g *CodeGen) Generate(prog *ir.IRProgram) string {
      g.output.Reset()

      // Emit global variables in .data / .bss
      if len(prog.Globals) > 0 {
          g.emitGlobals(prog.Globals)
      }

      // Emit each function
      g.emitDirective(".text")
      for _, fn := range prog.Functions {
          g.emitFunction(fn)
      }

      return g.output.String()
  }

  func (g *CodeGen) emitGlobals(globals []string) {
      g.emitDirective(".data")
      for _, name := range globals {
          sym := g.symbolName(name)
          g.emitf(".globl %s", sym)
          g.emitf("%s:", sym)
          g.emitf("    .long 0")   // 4 bytes, initialized to 0
      }
      g.emit("")
  }

  func (g *CodeGen) emitFunction(fn *ir.Function) {
      // Reset stack layout
      g.varOffset = make(map[string]int)
      g.nextOffset = -8

      // Allocate stack slots for parameters
      for _, param := range fn.Params {
          g.allocVar(param)
      }

      // Allocate stack slots for all local vars and temporaries
      // We scan instructions to find all destinations
      for _, instr := range fn.Instructions {
          if instr.Dst.Kind == ir.OpndVar && !g.hasVar(instr.Dst.Name) {
              g.allocVar(instr.Dst.Name)
          }
          if instr.Dst.Kind == ir.OpndTemp {
              name := fmt.Sprintf("t%d", instr.Dst.Temp)
              if !g.hasVar(name) {
                  g.allocVar(name)
              }
          }
      }

      frameSize := -g.nextOffset + 8  // +8 for alignment padding if needed
      // Ensure 16-byte alignment (after push %rbp, RSP is already -8)
      if frameSize%16 != 0 {
          frameSize += 16 - (frameSize % 16)
      }

      // Function header
      sym := g.symbolName(fn.Name)
      g.emitf(".globl %s", sym)
      g.emitf("%s:", sym)

      // Prologue
      g.emit("    pushq   %rbp")
      g.emit("    movq    %rsp, %rbp")
      g.emitf("    subq    $%d, %%rsp", frameSize)

      // Spill parameters from registers to stack
      paramRegs := []string{"%edi", "%esi", "%edx", "%ecx", "%r8d", "%r9d"}
      for i, param := range fn.Params {
          if i < len(paramRegs) {
              g.emitf("    movl    %s, %d(%%rbp)", paramRegs[i], g.varOffset[param])
          }
      }

      // Emit instructions
      for _, instr := range fn.Instructions {
          g.emitInstruction(instr)
      }

      // Safety: if function falls through without return
      g.emit("    xorl    %eax, %eax")
      g.emit("    movq    %rbp, %rsp")
      g.emit("    popq    %rbp")
      g.emit("    retq")
      g.emit("")
  }

  func (g *CodeGen) emitInstruction(instr ir.Instruction) {
      switch instr.Op {
      case ir.OpLoadImm:
          offset := g.getOffset(instr.Dst)
          g.emitf("    movl    $%d, %d(%%rbp)", instr.Left.Value, offset)

      case ir.OpCopy:
          g.loadToEAX(instr.Left)
          offset := g.getOffset(instr.Dst)
          g.emitf("    movl    %%eax, %d(%%rbp)", offset)

      case ir.OpAdd:
          g.loadToEAX(instr.Left)
          g.emitf("    addl    %s, %%eax", g.operandRef(instr.Right))
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpSub:
          g.loadToEAX(instr.Left)
          g.emitf("    subl    %s, %%eax", g.operandRef(instr.Right))
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpMul:
          g.loadToEAX(instr.Left)
          g.emitf("    imull   %s, %%eax", g.operandRef(instr.Right))
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpDiv:
          g.loadToEAX(instr.Left)
          g.emit("    cdq")  // sign-extend EAX → EDX:EAX
          g.loadToECX(instr.Right)
          g.emit("    idivl   %ecx")  // EAX = quotient
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpMod:
          g.loadToEAX(instr.Left)
          g.emit("    cdq")
          g.loadToECX(instr.Right)
          g.emit("    idivl   %ecx")
          g.emitf("    movl    %%edx, %d(%%rbp)", g.getOffset(instr.Dst))  // remainder in EDX

      case ir.OpNeg:
          g.loadToEAX(instr.Left)
          g.emit("    negl    %eax")
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpNot:
          g.loadToEAX(instr.Left)
          g.emit("    cmpl    $0, %eax")
          g.emit("    sete    %al")
          g.emit("    movzbl  %al, %eax")
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpEq, ir.OpNeq, ir.OpLt, ir.OpGt, ir.OpLte, ir.OpGte:
          g.loadToEAX(instr.Left)
          g.emitf("    cmpl    %s, %%eax", g.operandRef(instr.Right))
          setInstr := map[ir.OpCode]string{
              ir.OpEq: "sete", ir.OpNeq: "setne",
              ir.OpLt: "setl", ir.OpGt: "setg",
              ir.OpLte: "setle", ir.OpGte: "setge",
          }[instr.Op]
          g.emitf("    %s    %%al", setInstr)
          g.emit("    movzbl  %al, %eax")
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpLabel:
          g.emitf("%s:", instr.Dst.Name)

      case ir.OpJump:
          g.emitf("    jmp     %s", instr.Dst.Name)

      case ir.OpJumpIfZero:
          g.loadToEAX(instr.Left)
          g.emit("    cmpl    $0, %eax")
          g.emitf("    je      %s", instr.Dst.Name)

      case ir.OpJumpIfNotZero:
          g.loadToEAX(instr.Left)
          g.emit("    cmpl    $0, %eax")
          g.emitf("    jne     %s", instr.Dst.Name)

      case ir.OpParam:
          // Parameters are collected and placed in registers before call.
          // We use a simple approach: push params onto a param stack,
          // and the OpCall handler pops them into registers.
          // For simplicity here, we emit movl instructions in OpCall.
          // The param instructions just mark the values.

      case ir.OpCall:
          // For simplicity, we assume params were emitted immediately before.
          // In a real compiler, we'd track them. Here, we use the convention
          // that arguments are already in the right positions.
          // (A full implementation would collect OpParam operands.)
          fnSym := g.symbolName(instr.FuncName)
          g.emitf("    callq   %s", fnSym)
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpReturn:
          g.loadToEAX(instr.Left)
          g.emit("    movq    %rbp, %rsp")
          g.emit("    popq    %rbp")
          g.emit("    retq")

      case ir.OpLoadGlobal:
          sym := g.symbolName(instr.Left.Name)
          g.emitf("    movl    %s(%%rip), %%eax", sym)
          g.emitf("    movl    %%eax, %d(%%rbp)", g.getOffset(instr.Dst))

      case ir.OpStoreGlobal:
          g.loadToEAX(instr.Left)
          sym := g.symbolName(instr.Dst.Name)
          g.emitf("    movl    %%eax, %s(%%rip)", sym)
      }
  }

  // ─── Helpers ──────────────────────────────────────────────────────

  func (g *CodeGen) symbolName(name string) string {
      if g.platform == PlatformMacOS {
          return "_" + name  // Mach-O prefixes C symbols with _
      }
      return name
  }

  func (g *CodeGen) allocVar(name string) {
      g.varOffset[name] = g.nextOffset
      g.nextOffset -= 8
  }

  func (g *CodeGen) hasVar(name string) bool {
      _, ok := g.varOffset[name]
      return ok
  }

  func (g *CodeGen) getOffset(op ir.Operand) int {
      switch op.Kind {
      case ir.OpndVar:
          return g.varOffset[op.Name]
      case ir.OpndTemp:
          name := fmt.Sprintf("t%d", op.Temp)
          return g.varOffset[name]
      }
      return 0
  }

  func (g *CodeGen) loadToEAX(op ir.Operand) {
      switch op.Kind {
      case ir.OpndImm:
          g.emitf("    movl    $%d, %%eax", op.Value)
      case ir.OpndVar:
          g.emitf("    movl    %d(%%rbp), %%eax", g.varOffset[op.Name])
      case ir.OpndTemp:
          name := fmt.Sprintf("t%d", op.Temp)
          g.emitf("    movl    %d(%%rbp), %%eax", g.varOffset[name])
      }
  }

  func (g *CodeGen) loadToECX(op ir.Operand) {
      switch op.Kind {
      case ir.OpndImm:
          g.emitf("    movl    $%d, %%ecx", op.Value)
      case ir.OpndVar:
          g.emitf("    movl    %d(%%rbp), %%ecx", g.varOffset[op.Name])
      case ir.OpndTemp:
          name := fmt.Sprintf("t%d", op.Temp)
          g.emitf("    movl    %d(%%rbp), %%ecx", g.varOffset[name])
      }
  }

  func (g *CodeGen) operandRef(op ir.Operand) string {
      switch op.Kind {
      case ir.OpndImm:
          return fmt.Sprintf("$%d", op.Value)
      case ir.OpndVar:
          return fmt.Sprintf("%d(%%rbp)", g.varOffset[op.Name])
      case ir.OpndTemp:
          name := fmt.Sprintf("t%d", op.Temp)
          return fmt.Sprintf("%d(%%rbp)", g.varOffset[name])
      }
      return "$0"
  }

  func (g *CodeGen) emit(line string) {
      g.output.WriteString(line + "\n")
  }

  func (g *CodeGen) emitf(format string, args ...any) {
      g.output.WriteString(fmt.Sprintf(format, args...) + "\n")
  }

  func (g *CodeGen) emitDirective(dir string) {
      g.output.WriteString(dir + "\n")
  }


M5.5  EXAMPLE — COMPLETE ASSEMBLY OUTPUT
===============================================================================

Mini-C Input:

  int add(int a, int b) {
      return a + b;
  }

  int main() {
      int x = add(3, 4);
      return x;
  }

Generated Assembly (macOS):

      .text
      .globl _add
  _add:
      pushq   %rbp
      movq    %rsp, %rbp
      subq    $32, %rsp
      movl    %edi, -8(%rbp)          # a = param 1
      movl    %esi, -16(%rbp)         # b = param 2
      movl    -8(%rbp), %eax          # load a
      addl    -16(%rbp), %eax         # eax = a + b
      movl    %eax, -24(%rbp)         # t0 = a + b
      movl    -24(%rbp), %eax         # return t0
      movq    %rbp, %rsp
      popq    %rbp
      retq

      .globl _main
  _main:
      pushq   %rbp
      movq    %rsp, %rbp
      subq    $32, %rsp
      movl    $3, %edi                # arg 1 = 3
      movl    $4, %esi                # arg 2 = 4
      callq   _add
      movl    %eax, -8(%rbp)          # x = return value
      movl    -8(%rbp), %eax          # return x
      movq    %rbp, %rsp
      popq    %rbp
      retq

  Instruction-by-instruction breakdown for _add:
  ─────────────────────────────────────────────────────────────────
  pushq %rbp       Save caller's frame pointer (8 bytes on stack)
  movq %rsp, %rbp  Set our frame pointer = stack pointer
  subq $32, %rsp   Reserve 32 bytes for locals/temps
  movl %edi, -8    Store param 'a' at RBP-8
  movl %esi, -16   Store param 'b' at RBP-16
  movl -8, %eax    Load 'a' into EAX
  addl -16, %eax   EAX = a + b
  movl %eax, -24   Store result in temp slot
  movl -24, %eax   Load result for return (could optimize out)
  movq %rbp, %rsp  Restore stack pointer
  popq %rbp        Restore caller's frame pointer
  retq             Return to caller (pops return address into RIP)
  ─────────────────────────────────────────────────────────────────


M5.6  TESTING THE CODE GENERATOR
===============================================================================

End-to-end test workflow:

  1. Write Mini-C source
  2. Run through lexer → parser → semantic → IR → codegen
  3. Write assembly to .s file
  4. Assemble and link with clang
  5. Run the program
  6. Check the exit code

  // In a test:
  func TestFactorial(t *testing.T) {
      source := `
      int factorial(int n) {
          int result = 1;
          while (n > 1) {
              result = result * n;
              n = n - 1;
          }
          return result;
      }
      int main() {
          return factorial(5);
      }`

      asm := compile(source, codegen.PlatformMacOS)
      os.WriteFile("/tmp/test.s", []byte(asm), 0644)

      // Assemble and link
      cmd := exec.Command("clang", "/tmp/test.s", "-o", "/tmp/test_bin")
      if out, err := cmd.CombinedOutput(); err != nil {
          t.Fatalf("clang failed: %s\n%s", err, out)
      }

      // Run and check exit code
      cmd = exec.Command("/tmp/test_bin")
      err := cmd.Run()
      exitCode := cmd.ProcessState.ExitCode()

      if exitCode != 120 {
          t.Errorf("expected exit code 120, got %d", exitCode)
      }
  }


M5.7  EXERCISES
===============================================================================

Exercise 8.1: Trace Execution
  For the factorial(5) program, trace through every assembly instruction
  from _main through the entire execution of factorial. Track RAX, RBP,
  RSP at each step.

Exercise 8.2: Function Arguments
  Write a Mini-C program that calls a function with 6 arguments.
  Verify the assembly uses RDI, RSI, RDX, RCX, R8D, R9D correctly.

Exercise 8.3: Optimization Opportunities
  The generated assembly for "return a + b" produces redundant loads
  and stores. Identify 3 specific redundancies and describe how a
  peephole optimizer could eliminate them.

Exercise 8.4: Boolean Short-Circuit
  Implement short-circuit evaluation for && and ||:
    a && b  → if a is 0, don't evaluate b (result is 0)
    a || b  → if a is non-zero, don't evaluate b (result is 1)
  Generate the appropriate conditional jumps.


WEEK 4-8 READING
===============================================================================

Required:
  - Dragon Book Ch. 3 (Lexical Analysis)
  - Dragon Book Ch. 4.1-4.4 (Syntax Analysis — top-down)
  - Dragon Book Ch. 6.1-6.4 (Intermediate Code Generation)
  - Dragon Book Ch. 8.1-8.6 (Code Generation)
  - System V AMD64 ABI specification (calling convention)

Recommended:
  - Cooper & Torczon Ch. 1-7
  - Appel "Modern Compiler Implementation" Ch. 1-9
  - Intel 64 and IA-32 Architectures Software Developer's Manual, Vol. 2
    (Instruction Set Reference)
