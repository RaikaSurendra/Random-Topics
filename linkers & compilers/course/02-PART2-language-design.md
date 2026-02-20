===============================================================================
  PART 2 — DESIGNING OUR MINI LANGUAGE: Mini-C
  Week 3
===============================================================================


3.1  LANGUAGE DESIGN PHILOSOPHY
===============================================================================

We are designing "Mini-C" — a small, imperative, C-like language that is:

  1. Simple enough to compile in one semester
  2. Complex enough to exercise every compiler phase
  3. Close enough to C that assembly output is intuitive
  4. Sufficient to write non-trivial programs (factorial, fibonacci, sorting)

What Mini-C supports:
  ✓ Integer variables (32-bit signed)
  ✓ Arithmetic and comparison expressions
  ✓ if / else statements
  ✓ while loops
  ✓ Function definitions and calls
  ✓ return statements
  ✓ Local variables (stack-allocated)
  ✓ Global variables
  ✓ Nested scopes (blocks with { })
  ✓ Single-line comments (//)

What Mini-C does NOT support:
  ✗ Floating-point numbers
  ✗ Strings (except as a stretch goal)
  ✗ Arrays and pointers
  ✗ Structs
  ✗ Type casting
  ✗ Preprocessor
  ✗ Standard library (we provide a small runtime)
  ✗ Multiple files (single-file compilation)

This subset is carefully chosen: it requires a full lexer, parser, type
checker, IR, and code generator, but avoids the complexity of type systems,
pointer analysis, and memory management.


3.2  FORMAL GRAMMAR (BNF)
===============================================================================

The grammar is specified in Backus-Naur Form (BNF). Non-terminals are
in angle brackets. Terminals are in quotes or CAPS for token categories.

```
<program>       ::= <declaration>*

<declaration>   ::= <var-decl>
                  | <func-decl>

<var-decl>      ::= "int" IDENTIFIER ";"
                  | "int" IDENTIFIER "=" <expression> ";"

<func-decl>     ::= "int" IDENTIFIER "(" <param-list>? ")" <block>

<param-list>    ::= <param> ("," <param>)*

<param>         ::= "int" IDENTIFIER

<block>         ::= "{" <block-item>* "}"

<block-item>    ::= <var-decl>
                  | <statement>

<statement>     ::= <expression-stmt>
                  | <if-stmt>
                  | <while-stmt>
                  | <return-stmt>
                  | <block>

<expression-stmt> ::= <expression> ";"

<if-stmt>       ::= "if" "(" <expression> ")" <statement>
                  | "if" "(" <expression> ")" <statement> "else" <statement>

<while-stmt>    ::= "while" "(" <expression> ")" <statement>

<return-stmt>   ::= "return" <expression> ";"

<expression>    ::= <assignment>

<assignment>    ::= IDENTIFIER "=" <assignment>
                  | <logical-or>

<logical-or>    ::= <logical-and> ("||" <logical-and>)*

<logical-and>   ::= <equality> ("&&" <equality>)*

<equality>      ::= <comparison> (("==" | "!=") <comparison>)*

<comparison>    ::= <additive> (("<" | ">" | "<=" | ">=") <additive>)*

<additive>      ::= <multiplicative> (("+" | "-") <multiplicative>)*

<multiplicative> ::= <unary> (("*" | "/" | "%") <unary>)*

<unary>         ::= "-" <unary>
                  | "!" <unary>
                  | <primary>

<primary>       ::= INTEGER_LITERAL
                  | IDENTIFIER
                  | IDENTIFIER "(" <arg-list>? ")"
                  | "(" <expression> ")"

<arg-list>      ::= <expression> ("," <expression>)*
```

Operator Precedence (lowest to highest):

```
  Level | Operators      | Associativity | Description
  ------+----------------+---------------+------------------
    1   | =              | Right         | Assignment
    2   | ||             | Left          | Logical OR
    3   | &&             | Left          | Logical AND
    4   | ==  !=         | Left          | Equality
    5   | < > <= >=      | Left          | Comparison
    6   | + -            | Left          | Additive
    7   | * / %          | Left          | Multiplicative
    8   | - ! (unary)    | Right (prefix)| Unary
    9   | () f()         | Left          | Grouping, Call
  ------+----------------+---------------+------------------
```


3.3  LEXICAL SPECIFICATION
===============================================================================

Tokens are the atomic units the lexer produces.

Token Categories:

  KEYWORDS:
    int, if, else, while, return

  IDENTIFIERS:
    [a-zA-Z_][a-zA-Z0-9_]*
    Must not be a keyword.

  INTEGER LITERALS:
    [0-9]+
    Range: 0 to 2147483647 (INT32_MAX)
    Leading zeros are allowed but interpreted as decimal (not octal).

  OPERATORS AND PUNCTUATION:
    +   -   *   /   %
    =   ==  !=  <   >   <=  >=
    &&  ||  !
    (   )   {   }   ;   ,

  WHITESPACE:
    Spaces, tabs, newlines — skipped, not tokenized.

  COMMENTS:
    // until end of line -- skipped.

Maximal Munch Rule:
  The lexer always matches the longest possible token.
  Example: "==" is one EQUAL_EQUAL token, not two EQUAL tokens.
  Example: "int32" is an IDENTIFIER, not keyword "int" + "32".

Token Data Structure (Go):

  type TokenType int

  const (
      // Literals
      TOKEN_INT_LITERAL TokenType = iota
      TOKEN_IDENTIFIER

      // Keywords
      TOKEN_INT          // "int"
      TOKEN_IF           // "if"
      TOKEN_ELSE         // "else"
      TOKEN_WHILE        // "while"
      TOKEN_RETURN       // "return"

      // Operators
      TOKEN_PLUS         // +
      TOKEN_MINUS        // -
      TOKEN_STAR         // *
      TOKEN_SLASH        // /
      TOKEN_PERCENT      // %
      TOKEN_ASSIGN       // =
      TOKEN_EQUAL        // ==
      TOKEN_NOT_EQUAL    // !=
      TOKEN_LESS         // <
      TOKEN_GREATER      // >
      TOKEN_LESS_EQ      // <=
      TOKEN_GREATER_EQ   // >=
      TOKEN_AND          // &&
      TOKEN_OR           // ||
      TOKEN_NOT          // !

      // Delimiters
      TOKEN_LPAREN       // (
      TOKEN_RPAREN       // )
      TOKEN_LBRACE       // {
      TOKEN_RBRACE       // }
      TOKEN_SEMICOLON    // ;
      TOKEN_COMMA        // ,

      // Special
      TOKEN_EOF
      TOKEN_ILLEGAL
  )

  type Token struct {
      Type    TokenType
      Lexeme  string     // The actual text
      Line    int        // Source line number
      Column  int        // Source column number
  }


3.4  TYPE SYSTEM
===============================================================================

Mini-C has exactly ONE type: int (32-bit signed integer).

Type Rules:
  1. All variables are of type int.
  2. All function parameters are of type int.
  3. All functions return int.
  4. All expressions evaluate to int.
  5. Conditions (if, while) treat 0 as false, non-zero as true.
  6. Boolean operators (&&, ||, !, ==, !=, <, >, <=, >=) produce
     1 for true and 0 for false.

This simplification means our type checker is straightforward — it mostly
verifies that variables are declared and functions are called with the
correct number of arguments.


3.5  SCOPING RULES
===============================================================================

Mini-C uses lexical (static) scoping with these rules:

  1. Global scope: Variables and functions declared at the top level.
  2. Function scope: Parameters are visible within the function body.
  3. Block scope: Variables declared in a { } block are local to that block.
  4. Shadowing: An inner scope variable shadows an outer scope variable
     with the same name. No error is produced.
  5. Forward references: Functions must be declared before use.
     (This simplifies the compiler — no forward declaration pass needed.)
  6. No nested functions: Functions can only be declared at global scope.

  Scope example:

  int x;                    // Global scope
  int foo(int x) {          // Parameter 'x' shadows global 'x'
      int y = x + 1;        // 'y' in function scope
      if (y > 0) {
          int z = y * 2;    // 'z' in block scope
          x = z;            // Refers to parameter 'x', not global
      }
      // z is not accessible here
      return x;
  }

  Scope chain lookup: When resolving a name, search from innermost
  scope outward. The first match wins.


3.6  SEMANTIC RULES
===============================================================================

These rules are enforced during semantic analysis (Part 3, Module 3):

  VARIABLES:
  S1. Every variable must be declared before use.
  S2. A variable cannot be declared twice in the same scope.
  S3. A variable cannot be used before initialization (warning, not error).

  FUNCTIONS:
  S4. Every function must be declared before it is called.
  S5. A function cannot be declared twice.
  S6. Function calls must have the correct number of arguments.
  S7. Every function must have at least one return statement.
      (We do not require all paths to return — simplification.)
  S8. The program must contain a function named "main" with no parameters.

  EXPRESSIONS:
  S9.  Division by zero is undefined behavior (no compile-time check for
       non-literal divisors; literal /0 produces a warning).
  S10. Integer overflow wraps (two's complement, matching C behavior).

  CONTROL FLOW:
  S11. return can only appear inside a function body.
  S12. break and continue are NOT supported (keeps CFG simple).


3.7  SAMPLE PROGRAMS
===============================================================================

Program 1 — Hello (minimal):

  int main() {
      return 0;
  }

  Expected: Compiles, runs, exits with code 0.


Program 2 — Arithmetic:

  int main() {
      int a = 10;
      int b = 3;
      int sum = a + b;
      int diff = a - b;
      int prod = a * b;
      int quot = a / b;
      int rem = a % b;
      return sum + diff + prod + quot + rem;
  }

  Expected: return value = 13 + 7 + 30 + 3 + 1 = 54
  Verify:   $ ./program ; echo $?   → 54


Program 3 — Conditional:

  int abs(int x) {
      if (x < 0) {
          return -x;
      }
      return x;
  }

  int main() {
      int a = abs(-42);
      int b = abs(17);
      return a + b;
  }

  Expected: 42 + 17 = 59


Program 4 — While Loop (factorial):

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
  }

  Expected: 120
  Note: 120 < 256, so exit code is valid. For larger values, use
  modular arithmetic or print routines.


Program 5 — Recursion (fibonacci):

  int fib(int n) {
      if (n <= 1) {
          return n;
      }
      return fib(n - 1) + fib(n - 2);
  }

  int main() {
      return fib(10);
  }

  Expected: 55


Program 6 — Multiple Functions and Globals:

  int counter;

  int increment() {
      counter = counter + 1;
      return counter;
  }

  int add(int a, int b) {
      return a + b;
  }

  int main() {
      counter = 0;
      int x = increment();
      int y = increment();
      int z = increment();
      return add(add(x, y), z);
  }

  Expected: add(add(1, 2), 3) = 6


Program 7 — Nested Scopes:

  int main() {
      int x = 1;
      {
          int x = 2;
          {
              int x = 3;
              x = x + 10;
              // x is 13 here
          }
          // x is 2 here (inner x is gone)
          x = x + 100;
      }
      // x is 1 here (both inner x's are gone)
      return x;
  }

  Expected: 1
  This tests that scoping and stack allocation are correct.


Program 8 — Boolean Logic:

  int main() {
      int a = 5;
      int b = 10;
      int c = (a < b) && (b > 0);        // 1 && 1 = 1
      int d = (a > b) || (a == 5);        // 0 || 1 = 1
      int e = !(a == b);                   // !0 = 1
      return c + d + e;
  }

  Expected: 3


3.8  EDGE CASES AND ERROR PROGRAMS
===============================================================================

The compiler must reject these programs with clear error messages:

Edge Case 1 — Undeclared Variable:
  int main() {
      x = 5;        // Error: 'x' undeclared
      return x;
  }

Edge Case 2 — Duplicate Declaration:
  int main() {
      int x = 1;
      int x = 2;    // Error: 'x' already declared in this scope
      return x;
  }

Edge Case 3 — Wrong Argument Count:
  int add(int a, int b) { return a + b; }
  int main() {
      return add(1);    // Error: 'add' expects 2 arguments, got 1
  }

Edge Case 4 — Undeclared Function:
  int main() {
      return foo(5);    // Error: 'foo' undeclared
  }

Edge Case 5 — Missing Main:
  int foo() { return 1; }
  // Error: no 'main' function defined

Edge Case 6 — Nested Function (forbidden):
  int main() {
      int inner() {    // Error: nested functions not allowed
          return 1;
      }
      return inner();
  }

Edge Case 7 — Integer Overflow:
  int main() {
      int x = 2147483647;
      int y = x + 1;    // No error — wraps to -2147483648 (C behavior)
      return 0;
  }


3.9  RUNTIME SUPPORT
===============================================================================

Mini-C programs communicate their results via the exit code (return value
from main). For more practical output, we provide a minimal runtime:

  // runtime.c — linked with every Mini-C program
  #include <stdio.h>
  #include <stdlib.h>

  // Called by Mini-C programs to print an integer
  void print_int(int x) {
      printf("%d\n", x);
  }

  // Wrapper: Mini-C main returns int, we call exit()
  extern int mc_main();
  int main() {
      int result = mc_main();
      return result;
  }

Our compiler emits a function called mc_main (or _mc_main on macOS).
The runtime's main() calls it and returns the result.

print_int is an "extern" function — our compiler treats it as an
undefined symbol that the linker resolves against runtime.o.

Usage:
  $ minicc program.mc -o program.s        # Our compiler
  $ clang program.s runtime.c -o program  # Assemble + link with runtime
  $ ./program


3.10  LANGUAGE SPECIFICATION SUMMARY
===============================================================================

```
  +---------------------------------------------------------------------+
  |                    MINI-C LANGUAGE SPECIFICATION                     |
  +-----------------+---------------------------------------------------+
  | Types           | int (32-bit signed)                               |
  | Variables       | Global and local, block-scoped                    |
  | Functions       | Top-level only, int return type, int params       |
  | Operators       | + - * / % = == != < > <= >= && || ! (unary -)    |
  | Statements      | expression, if/else, while, return, block        |
  | Literals        | Decimal integers [0-9]+                           |
  | Comments        | // single-line                                    |
  | Entry point     | main() function, no parameters                    |
  | Scoping         | Lexical, block-scoped, shadowing allowed          |
  | Calling conv.   | System V AMD64 ABI                                |
  | Undef. behavior | Division by zero, integer overflow                |
  | NOT supported   | float, char, string, pointer, array, struct,      |
  |                 | for, break, continue, switch, goto, typedef       |
  +-----------------+---------------------------------------------------+
```


WEEK 3 EXERCISES
===============================================================================

Exercise 3.1: Grammar Extension
  Extend the BNF grammar to support a "for" loop:
    for (init; condition; update) statement
  Write three example programs using it. Identify which grammar
  rules need to change.

Exercise 3.2: Ambiguity Analysis
  The "dangling else" problem: In the statement
    if (a) if (b) x = 1; else x = 2;
  does "else" bind to the inner or outer "if"? Show that our grammar
  as written is ambiguous. Propose a resolution (hint: always bind
  else to nearest if).

Exercise 3.3: Test Program Suite
  Write 10 Mini-C programs that collectively test every grammar rule
  at least once. For each program, state the expected return value.

Exercise 3.4: Error Catalog
  Write 10 invalid Mini-C programs that should produce compile-time
  errors. For each, state the expected error message and which
  semantic rule (S1-S12) it violates.

Exercise 3.5: Tokenization by Hand
  Given the following program, manually produce the token stream:

  int gcd(int a, int b) {
      while (b != 0) {
          int t = b;
          b = a % b;
          a = t;
      }
      return a;
  }

  List each token as: (TYPE, "lexeme", line, column)


WEEK 3 READING
===============================================================================

Required:
  - Dragon Book Ch. 2 (A Simple Syntax-Directed Translator)
  - Dragon Book Ch. 3.1-3.4 (Lexical Analysis — concepts)

Recommended:
  - Appel Ch. 1-3 (Language design and lexing)
  - Niklaus Wirth — "Compiler Construction" (free PDF), Ch. 1-3
