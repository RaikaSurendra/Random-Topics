# jq Source Code Walkthrough — Annotated Guide

> **Source:** jq 1.8.1 (MIT License) — full source included in `jq-1.8.1/`
> **Companion to:** [brew-source-build-guide.md](./brew-source-build-guide.md)
> **Purpose:** Understand what the source code looks like and how it all fits together

---

## Table of Contents

- [Chapter 1: Project Layout — What's in the Tarball](#chapter-1-project-layout--whats-in-the-tarball)
- [Chapter 2: The Build System Files](#chapter-2-the-build-system-files)
- [Chapter 3: The Entry Point — main.c](#chapter-3-the-entry-point--mainc)
- [Chapter 4: The Type System — jv.h and jv.c](#chapter-4-the-type-system--jvh-and-jvc)
- [Chapter 5: The Public API — jq.h](#chapter-5-the-public-api--jqh)
- [Chapter 6: The Lexer — lexer.l](#chapter-6-the-lexer--lexerl)
- [Chapter 7: The Parser — parser.y](#chapter-7-the-parser--parsery)
- [Chapter 8: The Compiler — compile.c](#chapter-8-the-compiler--compilec)
- [Chapter 9: The Virtual Machine — execute.c](#chapter-9-the-virtual-machine--executec)
- [Chapter 10: Built-in Functions — builtin.c and builtin.jq](#chapter-10-built-in-functions--builtinc-and-builtinjq)
- [Chapter 11: How It All Connects — The Execution Pipeline](#chapter-11-how-it-all-connects--the-execution-pipeline)
- [Chapter 12: The configure.ac — Build Configuration Source](#chapter-12-the-configureac--build-configuration-source)

---

## Chapter 1: Project Layout — What's in the Tarball

When you extract `jq-1.8.1.tar.gz`, you get this structure:

```
jq-1.8.1/                              # 26,265 lines of C source + headers
├── configure                           # 23,661-line shell script (auto-generated)
├── configure.ac                        # 290 lines — the REAL configure source
├── Makefile.in                         # auto-generated from Makefile.am
├── Makefile.am                         # the REAL Makefile source
├── src/                                # ★ ALL THE SOURCE CODE
│   ├── main.c          (722 lines)     # CLI entry point
│   ├── jv.h            (301 lines)     # JSON value type system (public header)
│   ├── jv.c            (1,200+ lines)  # JSON value implementation
│   ├── jq.h            (81 lines)      # Public API header
│   ├── lexer.l         (193 lines)     # Tokenizer (Flex source)
│   ├── lexer.c         (2,000+ lines)  # Auto-generated from lexer.l
│   ├── parser.y        (700+ lines)    # Grammar (Bison source)
│   ├── parser.c        (4,000+ lines)  # Auto-generated from parser.y
│   ├── compile.c       (1,200+ lines)  # jq filter → bytecode compiler
│   ├── execute.c       (1,000+ lines)  # Bytecode interpreter (the VM)
│   ├── bytecode.h      (97 lines)      # Bytecode format definition
│   ├── opcode_list.h   (52 lines)      # All VM opcodes
│   ├── builtin.c       (1,800+ lines)  # Built-in C functions (length, keys, +, etc.)
│   ├── builtin.jq      (244 lines)     # Built-in functions written IN jq itself
│   ├── jv_parse.c      (750+ lines)    # JSON parser
│   ├── jv_print.c      (350+ lines)    # JSON pretty-printer
│   ├── jv_aux.c        (650+ lines)    # JSON helper operations
│   ├── jv_unicode.c    (130+ lines)    # Unicode handling
│   ├── jv_dtoa.c       (2,400+ lines)  # Float ↔ string conversion (David Gay's dtoa)
│   ├── jv_alloc.c      (120+ lines)    # Memory allocator wrapper
│   ├── jv_file.c       (65+ lines)     # File loading
│   ├── linker.c        (430+ lines)    # Module/library linker
│   ├── locfile.c       (70+ lines)     # Source location tracking
│   ├── util.c          (1,000+ lines)  # Utilities
│   └── version.h       (1 line)        # Just: #define JQ_VERSION "1.8.1"
├── tests/
│   └── jq.test                         # Test suite
├── docs/                               # Documentation sources
├── config/                             # Autoconf helper scripts
├── m4/                                 # Autoconf macros
├── vendor/                             # Bundled oniguruma (optional fallback)
├── COPYING                             # MIT license
├── README.md                           # Project readme
├── NEWS.md                             # Changelog
└── AUTHORS                             # Contributors
```

### Key insight: Hand-written vs. Generated

| File | Hand-written or Generated? | What it does |
|---|---|---|
| `configure.ac` | Hand-written (290 lines) | Source for the build configuration |
| `configure` | **Generated** (23,661 lines) | Output of `autoconf` processing `configure.ac` |
| `Makefile.am` | Hand-written | Source for the build rules |
| `Makefile.in` | **Generated** | Output of `automake` processing `Makefile.am` |
| `Makefile` | **Generated at build time** | Output of `./configure` processing `Makefile.in` |
| `lexer.l` | Hand-written (193 lines) | Tokenizer rules |
| `lexer.c` | **Generated** (2,000+ lines) | Output of `flex` processing `lexer.l` |
| `parser.y` | Hand-written (700+ lines) | Grammar rules |
| `parser.c` | **Generated** (4,000+ lines) | Output of `bison` processing `parser.y` |

The generated files are included in release tarballs so you don't need `autoconf`, `flex`, or `bison` installed. You only need them when building from Git HEAD.

---

## Chapter 2: The Build System Files

### 2.1 configure.ac — What the Build Probes For

`configure.ac` is the source for the `./configure` script. It's written in m4 macros processed by GNU Autoconf.

Key sections (from `jq-1.8.1/configure.ac`):

```m4
AC_INIT([jq],[jq_version],[https://github.com/jqlang/jq/issues],[jq],[https://jqlang.org])
```
Defines the project name, version, bug tracker URL, and homepage.

```m4
AC_PROG_CC            # Find a C compiler
AC_PROG_YACC          # Find bison/yacc (parser generator)
```
Detects the compiler and parser generator.

```m4
AC_FIND_FUNC([isatty], [c], [#include <unistd.h>], [0])
AC_FIND_FUNC([strptime], [c], [#include <time.h>], [0, 0, 0])
AC_FIND_FUNC([setlocale], [c], [#include <locale.h>], [0,0])
```
Probes for optional system functions. If found, defines like `HAVE_ISATTY` are set in `config.h`, and the source code uses `#ifdef HAVE_ISATTY` to conditionally include that functionality.

```m4
AC_CHECK_MATH_FUNC(sin)
AC_CHECK_MATH_FUNC(cos)
AC_CHECK_MATH_FUNC(sqrt)
# ... 40+ math functions checked
```
Checks which math functions are available — these become jq built-in functions.

```m4
AC_ARG_WITH([oniguruma], ...)    # --with-oniguruma flag
AC_CHECK_HEADER("oniguruma.h", ...)
AC_CHECK_LIB([onig],[onig_version])
```
Looks for the oniguruma regex library. If found, defines `HAVE_LIBONIG` and enables regex support in jq.

### 2.2 How configure.ac Becomes configure

```
configure.ac  →  autoconf  →  configure  (23,661 lines of shell)
```

That 290-line m4 file expands to a 23,661-line shell script. This is why release tarballs include the pre-generated `configure` — so users don't need autoconf installed.

### 2.3 Makefile.am — The Build Rules Source

`Makefile.am` defines what to compile and how. The key parts:

```makefile
# What source files go into the jq binary
jq_SOURCES = src/main.c src/version.h

# What source files go into libjq (the library)
libjq_la_SOURCES = src/compile.c src/execute.c src/builtin.c \
                   src/jv.c src/jv_parse.c src/jv_print.c ...

# Link jq against libjq and oniguruma
jq_LDADD = libjq.la -lm $(onig_LDFLAGS) -lonig
```

---

## Chapter 3: The Entry Point — main.c

**File:** `jq-1.8.1/src/main.c` (722 lines)

This is where execution begins when you run `jq`. Let's trace through it.

### 3.1 Includes and Setup

```c
#include <assert.h>
#include <ctype.h>
#include <errno.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

#ifdef HAVE_LIBONIG
#include <oniguruma.h>       // Regex support — only if detected by configure
#endif

#include "jv.h"              // JSON value types
#include "jq.h"              // jq API (compile, execute)
#include "util.h"            // Utility functions
#include "src/version.h"     // Just: #define JQ_VERSION "1.8.1"
```

Notice the `#ifdef HAVE_LIBONIG` — this is a conditional compilation flag set by `./configure`. If oniguruma wasn't found, this include is skipped and jq is built without regex support.

### 3.2 The main() Function

```c
int main(int argc, char* argv[]) {
  jq_state *jq = NULL;                   // The jq interpreter state
  int ret = JQ_OK_NO_OUTPUT;
  int options = 0;                        // Bit flags for CLI options

  // Initialize the jq interpreter
  jq = jq_init();

  // Parse command-line arguments
  // (450+ lines of option parsing: -r, -c, -S, --slurp, --arg, etc.)
  for (int i=1; i<argc; i++) {
    // ... option parsing ...
  }

  // Compile the jq filter program
  compiled = jq_compile_args(jq, program, program_arguments);

  // Read input JSON and process it
  while (jv_is_valid(value = jq_util_input_next_input(input_state))) {
    ret = process(jq, value, jq_flags, dumpopts, options);
  }

  // Cleanup
  jq_teardown(&jq);
}
```

### 3.3 The process() Function — Core Execution Loop

This is where each JSON input is filtered:

```c
static int process(jq_state *jq, jv value, int flags, int dumpopts, int options) {
  jq_start(jq, value, flags);             // Load input into the VM
  jv result;
  while (jv_is_valid(result = jq_next(jq))) {  // Get each output
    if ((options & RAW_OUTPUT) && jv_get_kind(result) == JV_KIND_STRING) {
      // -r flag: print strings without quotes
      priv_fwrite(jv_string_value(result), ...);
    } else {
      // Normal output: pretty-print as JSON
      jv_dump(result, dumpopts);
    }
    priv_fwrite("\n", ...);               // Newline after each output
  }
  return ret;
}
```

**The call chain is:** `main()` → `jq_compile_args()` → `jq_start()` → `jq_next()` (loop) → `jv_dump()`

---

## Chapter 4: The Type System — jv.h and jv.c

**File:** `jq-1.8.1/src/jv.h` (301 lines)

jq's internal representation of JSON values. Every piece of data in jq is a `jv`.

### 4.1 The JSON Value Types

```c
typedef enum {
  JV_KIND_INVALID,    // Error/sentinel value
  JV_KIND_NULL,       // JSON null
  JV_KIND_FALSE,      // JSON false
  JV_KIND_TRUE,       // JSON true
  JV_KIND_NUMBER,     // JSON number (stored as double)
  JV_KIND_STRING,     // JSON string
  JV_KIND_ARRAY,      // JSON array
  JV_KIND_OBJECT      // JSON object
} jv_kind;
```

### 4.2 The jv Struct — How Values Are Stored

```c
typedef struct {
  unsigned char kind_flags;    // Which jv_kind + flags
  unsigned char pad_;
  unsigned short offset;       // Array offset (for slices)
  int size;
  union {
    struct jv_refcnt* ptr;     // Pointer to heap data (strings, arrays, objects)
    double number;             // Inline number storage (no heap allocation!)
  } u;
} jv;
```

**Key design decisions:**
- `jv` is a **16-byte value type** — small enough to pass by value (no pointer indirection)
- Numbers are stored **inline** in the union — no heap allocation for numbers
- Strings, arrays, and objects use **reference counting** (`jv_refcnt*`)
- The comment in the source says: *"All of the fields of this struct are private. Really. Do not play with them."*

### 4.3 The Memory Model

```c
/*
 * All jv_* functions consume (decref) input and produce (incref) output
 * Except jv_copy
 */
jv jv_copy(jv);       // Increment reference count
void jv_free(jv);     // Decrement reference count, free if zero
```

This is a **linear ownership model**. Every jv value is either:
- Consumed by a function (the function takes ownership)
- Explicitly copied with `jv_copy()` to share it

This prevents memory leaks without a garbage collector.

### 4.4 The API Surface

```c
// Constructors
jv jv_null(void);
jv jv_true(void);
jv jv_false(void);
jv jv_number(double);
jv jv_string(const char*);
jv jv_array(void);
jv jv_object(void);

// Operations
int jv_array_length(jv);
jv jv_array_get(jv, int);
jv jv_array_append(jv, jv);
jv jv_object_get(jv object, jv key);
jv jv_object_set(jv object, jv key, jv value);

// Inspection
jv_kind jv_get_kind(jv);
static int jv_is_valid(jv x) { return jv_get_kind(x) != JV_KIND_INVALID; }
const char* jv_string_value(jv);
double jv_number_value(jv);
```

---

## Chapter 5: The Public API — jq.h

**File:** `jq-1.8.1/src/jq.h` (81 lines)

This is the API that programs linking against `libjq` would use:

```c
typedef struct jq_state jq_state;     // Opaque interpreter state

// Lifecycle
jq_state *jq_init(void);              // Create interpreter
void jq_teardown(jq_state **);        // Destroy interpreter

// Compilation
int jq_compile(jq_state *, const char*);         // Compile a filter string
int jq_compile_args(jq_state *, const char*, jv); // Compile with named args

// Execution
void jq_start(jq_state *, jv value, int);   // Load input value
jv jq_next(jq_state *);                     // Get next output (iterator)

// Debugging
void jq_dump_disassembly(jq_state *, int);  // Print bytecode
```

**Usage pattern:**
```c
jq_state *jq = jq_init();
jq_compile(jq, ".foo | length");      // Compile the filter
jq_start(jq, input_json, 0);          // Feed it input
jv result;
while (jv_is_valid(result = jq_next(jq))) {
    jv_dump(result, 0);               // Print each output
}
jq_teardown(&jq);
```

---

## Chapter 6: The Lexer — lexer.l

**File:** `jq-1.8.1/src/lexer.l` (193 lines)

The lexer (tokenizer) breaks the jq filter string into tokens. It's written in **Flex** notation.

### 6.1 How Flex Works

```
lexer.l  →  flex  →  lexer.c  (2,000+ lines of generated C)
```

You write rules in `lexer.l`; Flex generates the C implementation.

### 6.2 Token Rules

```lex
"!="     { return NEQ; }
"=="     { return EQ; }
"as"     { return AS; }
"def"    { return DEF; }
"if"     { return IF; }
"then"   { return THEN; }
"else"   { return ELSE; }
"and"    { return AND; }
"or"     { return OR; }
"end"    { return END; }
"reduce" { return REDUCE; }
"try"    { return TRY; }
"catch"  { return CATCH; }
".."     { return REC; }           // Recursive descent operator
"|="     { return SETPIPE; }       // Update operator
"+="     { return SETPLUS; }
"//"     { return DEFINEDOR; }     // Alternative operator
"?//"    { return ALTERNATION; }   // Try-alternative
```

These map jq syntax directly to parser tokens.

### 6.3 Number Parsing

```lex
([0-9]+(\.[0-9]*)?|\.[0-9]+)([eE][+-]?[0-9]+)? {
   yylval->literal = jv_parse_sized(yytext, yyleng);
   return LITERAL;
}
```

Numbers are parsed by the JSON parser (`jv_parse`) and stored as `jv` values right in the lexer.

### 6.4 String Interpolation

```lex
"\"" {
  yy_push_state(IN_QQSTRING, yyscanner);
  return QQSTRING_START;
}
```

jq supports string interpolation: `"Hello \(.name)"`. The lexer uses **state stacking** — when it sees `\(`, it pushes into an interpolation state, parses the expression, then returns to string mode.

### 6.5 Identifiers and Variables

```lex
[a-zA-Z_][a-zA-Z_0-9]*            { return IDENT; }     // function names
\.[a-zA-Z_][a-zA-Z_0-9]*          { return FIELD; }     // .foo
\$[a-zA-Z_][a-zA-Z_0-9]*          { return BINDING; }   // $var
```

---

## Chapter 7: The Parser — parser.y

**File:** `jq-1.8.1/src/parser.y` (700+ lines)

The parser takes tokens from the lexer and builds an intermediate representation (IR). Written in **Bison** notation.

### 7.1 How Bison Works

```
parser.y  →  bison  →  parser.c + parser.h  (4,000+ lines of generated C)
```

### 7.2 Token Declarations

```yacc
%token INVALID_CHARACTER
%token <literal> IDENT
%token <literal> FIELD
%token <literal> BINDING
%token <literal> LITERAL
%token <literal> FORMAT
%token REC ".."
%token EQ "=="
%token NEQ "!="
%token DEFINEDOR "//"
%token AS "as"
%token DEF "def"
%token IF "if"
%token THEN "then"
%token ELSE "else"
%token REDUCE "reduce"
%token TRY "try"
%token CATCH "catch"
```

These match the tokens the lexer produces (Chapter 6).

### 7.3 Grammar Rules (Conceptual)

The grammar defines what valid jq programs look like. Some examples:

```
filter:  '.' FIELD              →  .foo
       | filter '|' filter      →  .foo | .bar
       | filter '+' filter      →  .a + .b
       | 'if' filter 'then' filter 'else' filter 'end'
       | 'def' IDENT ':' filter ';' filter
       | 'reduce' filter 'as' BINDING '(' filter ';' filter ')'
       | filter '?' '//' filter →  try-alternative
```

The parser produces **blocks** — an intermediate representation that the compiler (Chapter 8) converts to bytecode.

---

## Chapter 8: The Compiler — compile.c

**File:** `jq-1.8.1/src/compile.c` (1,200+ lines)

The compiler takes the parser's IR (blocks of `struct inst`) and produces bytecode.

### 8.1 The Intermediate Representation

```c
struct inst {
  struct inst* next;           // Doubly-linked list
  struct inst* prev;
  opcode op;                   // The operation

  struct {
    uint16_t intval;
    struct inst* target;       // Branch target
    jv constant;               // Constant value
    const struct cfunction* cfunc;  // C function pointer
  } imm;                       // Immediate operand

  struct inst* bound_by;       // Variable binding reference
  char* symbol;                // Variable/function name
  block subfn;                 // Function body (for closures)
};
```

Each `struct inst` is one instruction in the IR. The compiler resolves variable bindings, performs optimizations, and serializes this into bytecode.

### 8.2 Variable Binding

The comments in the source explain the binding model:

```c
// An instruction requiring binding is in one of three states:
//   inst->bound_by = NULL  - Unbound free variable
//   inst->bound_by = inst  - This instruction binds a variable
//   inst->bound_by = other - Uses variable bound by other instruction
```

This is how `$x` in `. as $x | $x + $x` gets resolved — the compiler binds both references to `$x` back to the `as $x` instruction.

---

## Chapter 9: The Virtual Machine — execute.c

**File:** `jq-1.8.1/src/execute.c` (1,000+ lines)

jq doesn't interpret the filter directly — it compiles to **bytecode** and runs it on a stack-based virtual machine.

### 9.1 The jq_state — Interpreter State

```c
struct jq_state {
  struct bytecode* bc;         // The compiled program

  struct stack stk;            // Value stack
  stack_ptr curr_frame;        // Current function call frame
  stack_ptr stk_top;           // Top of stack
  stack_ptr fork_top;          // Backtracking point

  jv path;                     // Current path (for path expressions)
  int subexp_nest;             // Nesting depth
  int debug_trace_enabled;

  int halted;                  // Has halt/halt_error been called?
  jv exit_code;
  jv error_message;

  jq_input_cb input_cb;       // Callback to read more input
  jq_msg_cb debug_cb;         // Callback for debug output
};
```

### 9.2 The Opcodes

All VM operations are defined in `opcode_list.h` (52 lines):

```c
OP(LOADK, CONSTANT, 1, 1)     // Load a constant onto the stack
OP(DUP,   NONE,     1, 2)     // Duplicate top of stack
OP(POP,   NONE,     1, 0)     // Pop top of stack
OP(LOADV, VARIABLE, 1, 1)     // Load a variable
OP(STOREV, VARIABLE, 1, 0)    // Store to a variable
OP(INDEX, NONE,     2, 1)     // Object/array indexing: .foo or .[0]
OP(EACH,  NONE,     1, 1)     // Iterator: .[]
OP(FORK,  BRANCH,   0, 0)     // Create a backtracking point
OP(JUMP,  BRANCH,   0, 0)     // Unconditional jump
OP(JUMP_F,BRANCH,   1, 0)     // Jump if false
OP(BACKTRACK, NONE, 0, 0)     // Backtrack to last FORK
OP(CALL_BUILTIN, CFUNC, -1, 1) // Call a C built-in function
OP(CALL_JQ, UFUNC, 1, 1)      // Call a jq-defined function
OP(RET, NONE, 1, 1)            // Return from function
OP(PATH_BEGIN, NONE, 1, 2)    // Start path tracking
OP(PATH_END,   NONE, 2, 1)    // End path tracking
```

The format is: `OP(name, operand_type, stack_inputs, stack_outputs)`

### 9.3 The Bytecode Format

```c
struct bytecode {
  uint16_t* code;              // Array of 16-bit opcodes + operands
  int codelen;                 // Length of code array
  int nlocals;                 // Number of local variables
  int nclosures;               // Number of closures
  jv constants;                // JSON array of constant values
  struct symbol_table* globals; // C function table
  struct bytecode** subfunctions; // Nested function bytecodes
};
```

### 9.4 Backtracking — jq's Secret Weapon

jq supports **generators** — expressions that produce multiple values. For example, `.[]` iterates over all elements. This is implemented via **backtracking**:

1. `FORK` saves the current state (stack, instruction pointer)
2. When `EACH` runs out of elements, it executes `BACKTRACK`
3. `BACKTRACK` restores the saved state and tries the next alternative

This is why `[.[] | select(. > 3)]` works — `select` uses backtracking to skip non-matching elements.

---

## Chapter 10: Built-in Functions — builtin.c and builtin.jq

jq has two kinds of built-in functions:

### 10.1 C Built-ins (builtin.c — 1,800+ lines)

These are implemented in C for performance or because they need low-level access:

```c
// The + operator — handles multiple types
jv binop_plus(jv a, jv b) {
  if (jv_get_kind(a) == JV_KIND_NULL) {
    jv_free(a);
    return b;                                    // null + x = x
  } else if (jv_get_kind(a) == JV_KIND_NUMBER && jv_get_kind(b) == JV_KIND_NUMBER) {
    jv r = jv_number(jv_number_value(a) + jv_number_value(b));
    jv_free(a);
    jv_free(b);
    return r;                                    // 1 + 2 = 3
  } else if (jv_get_kind(a) == JV_KIND_STRING && jv_get_kind(b) == JV_KIND_STRING) {
    return jv_string_concat(a, b);               // "a" + "b" = "ab"
  } else if (jv_get_kind(a) == JV_KIND_ARRAY && jv_get_kind(b) == JV_KIND_ARRAY) {
    return jv_array_concat(a, b);                // [1] + [2] = [1,2]
  } else if (jv_get_kind(a) == JV_KIND_OBJECT && jv_get_kind(b) == JV_KIND_OBJECT) {
    return jv_object_merge(a, b);                // {a:1} + {b:2} = {a:1,b:2}
  } else {
    return type_error2(a, b, "cannot be added");
  }
}
```

Notice how `+` is **polymorphic** — it does different things based on the types of its operands. This is a core jq design principle.

Other C built-ins include: `length`, `keys`, `values`, `has`, `type`, `empty`, `error`, `debug`, `input`, `path`, `getpath`, `setpath`, `delpaths`, `to_number`, `to_string`, `ascii`, `explode`, `implode`, `split`, `join`, `test`, `match`, `capture`, `now`, `strftime`, `strptime`, `gmtime`, `mktime`, and all the math functions (`sin`, `cos`, `sqrt`, `floor`, etc.).

### 10.2 jq Built-ins (builtin.jq — 244 lines)

Higher-level functions are written in jq itself! These are compiled and included in every jq program:

```jq
def map(f): [.[] | f];
def select(f): if f then . else empty end;
def sort_by(f): _sort_by_impl(map([f]));
def add: reduce .[] as $x (null; . + $x);
def del(f): delpaths([path(f)]);
def flatten: _flatten(-1);
def range($x): range(0;$x);
def reverse: [.[length - 1 - range(0;length)]];
def to_entries: [keys_unsorted[] as $k | {key: $k, value: .[$k]}];
def from_entries: map({(.key // .Key // .name // .Name): ...}) | add // {};
def with_entries(f): to_entries | map(f) | from_entries;

# String operations
def ltrimstr($left): if startswith($left) then .[$left | length:] end;
def rtrimstr($right): if endswith($right) then .[:$right | -length] end;
def ascii_downcase: explode | map(if 65 <= . and . <= 90 then . + 32 else . end) | implode;
def ascii_upcase: explode | map(if 97 <= . and . <= 122 then . - 32 else . end) | implode;

# Iteration
def while(cond; update): def _while: if cond then ., (update | _while) else empty end; _while;
def until(cond; next): def _until: if cond then . else (next|_until) end; _until;
def repeat(exp): def _repeat: exp, _repeat; _repeat;
def recurse(f): def r: ., (f | r); r;

# SQL-like operations
def INDEX(stream; idx_expr): reduce stream as $row ({}; .[$row|idx_expr|tostring] = $row);
def IN(s): any(s == .; .);
```

**Key insight:** `map`, `select`, `any`, `all`, `flatten`, `reverse`, `to_entries`, `from_entries`, `with_entries` — all the functions you use daily in jq — are written in jq. They're not special. You could write them yourself.

---

## Chapter 11: How It All Connects — The Execution Pipeline

When you run:
```bash
echo '{"name":"jq","version":1.8}' | jq '.name | ascii_upcase'
```

Here's what happens inside:

```
Step 1: PARSE INPUT JSON (jv_parse.c)
  '{"name":"jq","version":1.8}'
  → jv object: { "name": jv_string("jq"), "version": jv_number(1.8) }

Step 2: LEX THE FILTER (lexer.l → lexer.c)
  '.name | ascii_upcase'
  → tokens: [FIELD("name"), PIPE, IDENT("ascii_upcase")]

Step 3: PARSE THE FILTER (parser.y → parser.c)
  tokens → IR blocks:
    FIELD("name") | CALL("ascii_upcase")

Step 4: COMPILE TO BYTECODE (compile.c)
  IR → bytecode:
    LOADK "name"     # push field name
    INDEX             # index into input object → "jq"
    CALL_JQ ascii_upcase  # call the jq built-in

  ascii_upcase itself compiles to:
    CALL_BUILTIN explode    # "jq" → [106, 113]
    ... map(if 97<=. ...)   # [106, 113] → [74, 81]
    CALL_BUILTIN implode    # [74, 81] → "JQ"

Step 5: EXECUTE (execute.c)
  VM runs the bytecode on a stack:

  Stack: [{"name":"jq","version":1.8}]     ← input
  LOADK "name"  → [{"name":"jq",...}, "name"]
  INDEX          → ["jq"]
  (call ascii_upcase...)
  → ["JQ"]                                  ← output

Step 6: OUTPUT (jv_print.c, main.c)
  jv_dump(jv_string("JQ"), pretty_flags)
  → prints: "JQ"
```

### Visualized

```
                    ┌──────────┐
 Filter string ──→  │  Lexer   │  tokens
 ".name | ..."      │ lexer.l  │──────────→ ┌──────────┐
                    └──────────┘             │  Parser  │  IR blocks
                                             │ parser.y │──────────→ ┌──────────┐
                                             └──────────┘             │ Compiler │  bytecode
                                                                      │compile.c │──────→ ┌─────┐
                                                                      └──────────┘         │ VM  │
 Input JSON ──→ jv_parse() ──→ jv value ──────────────────────────────────────────────────→│exec │──→ output
                                                                                           └─────┘
```

---

## Chapter 12: The configure.ac — Build Configuration Source

For completeness, here's how `configure.ac` structures the build decisions. This is the "source code" of the build system itself.

### 12.1 Platform Detection

```m4
AC_PROG_CC                    # Find C compiler (clang on macOS, gcc on Linux)
AC_USE_SYSTEM_EXTENSIONS      # Enable platform-specific extensions
AC_SYS_LARGEFILE              # Support files > 2GB
AC_C_BIGENDIAN(...)           # Detect endianness (affects dtoa.c)
```

### 12.2 Feature Detection

```m4
# These probes determine #ifdef flags in config.h:
AC_FIND_FUNC([isatty], ...)   → HAVE_ISATTY     → color output detection
AC_FIND_FUNC([strptime], ...) → HAVE_STRPTIME    → date parsing support
AC_FIND_FUNC([setlocale], ...) → HAVE_SETLOCALE  → locale support
AC_CHECK_MATH_FUNC(sin)       → HAVE_SIN         → math built-in available
```

### 12.3 Dependency Detection

```m4
AC_ARG_WITH([oniguruma], ...)
AC_CHECK_HEADER("oniguruma.h", ...)
AC_CHECK_LIB([onig], [onig_version])
# If found: HAVE_LIBONIG=1, regex support enabled
# If not found: falls back to bundled copy in vendor/
```

### 12.4 Output

```m4
AC_CONFIG_FILES([Makefile libjq.pc])    # Generate these files
AC_OUTPUT                                # Write everything out
```

This generates:
- `Makefile` (from `Makefile.in`)
- `libjq.pc` (from `libjq.pc.in`)
- `config.h` (all the `#define HAVE_*` flags)

---

## Quick Reference: File → Purpose

| File | Role | Read this to understand... |
|---|---|---|
| `main.c` | CLI entry point | How jq parses arguments and drives execution |
| `jv.h` / `jv.c` | JSON value type | How JSON data is represented in memory |
| `jq.h` | Public API | How to use jq as a library |
| `lexer.l` | Tokenizer | How jq filter syntax is recognized |
| `parser.y` | Grammar | What constitutes a valid jq program |
| `compile.c` | Compiler | How filters become bytecode |
| `execute.c` | VM / interpreter | How bytecode runs on a stack machine |
| `bytecode.h` | Bytecode format | The instruction set architecture |
| `opcode_list.h` | Opcode definitions | Every operation the VM supports |
| `builtin.c` | C built-ins | How `+`, `length`, `keys`, `type`, regex work |
| `builtin.jq` | jq built-ins | How `map`, `select`, `sort_by`, `group_by` work |
| `jv_parse.c` | JSON parser | How input JSON is parsed into jv values |
| `jv_print.c` | JSON printer | How jv values become formatted output |
| `configure.ac` | Build config source | What the build system checks for |
| `Makefile.am` | Build rules source | What gets compiled and linked |

---

*Source: jq 1.8.1 — MIT License — https://github.com/jqlang/jq*
*Full source code available in the `jq-1.8.1/` directory alongside this document.*
