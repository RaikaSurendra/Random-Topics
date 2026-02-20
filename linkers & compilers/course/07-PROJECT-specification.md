===============================================================================
  FINAL PROJECT SPECIFICATION
  Capstone Deliverable — Due Week 14
===============================================================================


PROJECT OVERVIEW
===============================================================================

Build a complete toolchain from source code to executable binary.

Deliverables:
  1. minicc    — Mini-C compiler (Go), emitting x86-64 assembly
  2. elfwriter — ELF object file writer (Go)
  3. minilink  — Minimal static linker (Go)
  4. Test suite of Mini-C programs
  5. Documentation and project report


REQUIRED FUNCTIONALITY
===============================================================================

The compiler (minicc) must:
  [R1] Tokenize valid Mini-C programs without errors
  [R2] Parse Mini-C into a correct AST
  [R3] Perform semantic analysis (scope, type, arity checks)
  [R4] Generate three-address code IR
  [R5] Emit correct x86-64 assembly (AT&T syntax)
  [R6] Support both macOS (with _ prefix) and Linux assembly output
  [R7] Handle: int variables, arithmetic, if/else, while, functions,
       globals, locals, return, nested scopes

The ELF writer (elfwriter) must:
  [R8]  Write a valid ELF64 header
  [R9]  Write section headers for .text, .data, .symtab, .strtab, .shstrtab
  [R10] Write relocation entries (.rela.text)
  [R11] Produce .o files that pass `readelf -a` validation

The linker (minilink) must:
  [R12] Read multiple ELF relocatable object files
  [R13] Merge .text and .data sections
  [R14] Resolve global symbols across object files
  [R15] Apply R_X86_64_PC32 and R_X86_64_PLT32 relocations
  [R16] Emit a valid ELF executable with program headers
  [R17] Produce executables that run correctly on Linux (Docker)


TEST PROGRAMS
===============================================================================

The following programs must compile, link, and produce the correct result:

```
  Program                  Expected Exit Code    Tests
  -----------------------  --------------------  -------------------------
  empty_main.mc            0                     Minimal function
  arithmetic.mc            54                    All arithmetic operators
  comparison.mc            3                     All comparison operators
  if_else.mc               42                    Conditional branching
  while_loop.mc            55                    Loop execution
  factorial.mc             120                   Iteration + multiplication
  fibonacci.mc             55                    Recursion
  nested_scopes.mc         1                     Block scoping / shadowing
  globals.mc               6                     Global variable access
  multi_func.mc            25                    Multiple function calls
  boolean_logic.mc         3                     && || ! operators
  unary_minus.mc           42                    Negation
  complex_expr.mc          17                    Nested expressions
  mutual_recursion.mc      1                     isEven/isOdd pattern
  many_args.mc             21                    6-argument function call
  deeply_nested.mc         100                   5-level nested blocks
  chain_calls.mc           10                    f(g(h(x))) pattern
  global_mutation.mc       15                    Global state across calls
  shadow_param.mc          7                     Param shadowed by local
  modulo.mc                3                     % operator
```

Additionally, provide at least 10 invalid programs that the compiler must
reject with clear error messages (undeclared variables, duplicate names,
wrong argument counts, missing main, etc.).


DIRECTORY LAYOUT
===============================================================================

```
minicc/
+-- cmd/
|   +-- minicc/main.go            # Compiler CLI
|   +-- elfwriter/main.go         # ELF writer CLI (optional standalone)
|   +-- minilink/main.go          # Linker CLI
+-- pkg/
|   +-- lexer/
|   |   +-- token.go              # Token types
|   |   +-- lexer.go              # Scanner implementation
|   |   +-- lexer_test.go
|   +-- parser/
|   |   +-- ast.go                # AST node definitions
|   |   +-- parser.go             # Recursive descent parser
|   |   +-- parser_test.go
|   +-- semantic/
|   |   +-- symtable.go           # Scope and symbol table
|   |   +-- analyzer.go           # Semantic analysis
|   |   +-- semantic_test.go
|   +-- ir/
|   |   +-- ir.go                 # IR instruction types
|   |   +-- builder.go            # AST-to-IR translation
|   |   +-- ir_test.go
|   +-- codegen/
|   |   +-- x86_64.go             # x86-64 code generator
|   |   +-- codegen_test.go
|   +-- elf/
|   |   +-- types.go              # ELF struct definitions and constants
|   |   +-- writer.go             # ELF object file writer
|   |   +-- elf_test.go
|   +-- linker/
|       +-- reader.go             # ELF object file reader/parser
|       +-- symbol.go             # Symbol resolution
|       +-- relocation.go         # Relocation application
|       +-- linker.go             # Section merging and layout
|       +-- output.go             # Executable ELF writer
|       +-- linker_test.go
+-- testdata/
|   +-- valid/                    # Test programs (should compile + run)
|   |   +-- empty_main.mc
|   |   +-- factorial.mc
|   |   +-- fibonacci.mc
|   |   +-- ... (20 programs)
|   +-- invalid/                  # Error programs (should fail gracefully)
|   |   +-- undeclared_var.mc
|   |   +-- duplicate_decl.mc
|   |   +-- ... (10 programs)
|   +-- expected/                 # Expected exit codes
|       +-- results.txt           # "factorial.mc 120\nfibonacci.mc 55\n..."
+-- runtime/
|   +-- runtime.c                 # C runtime (print_int, main wrapper)
+-- scripts/
|   +-- test_macos.sh             # End-to-end test on macOS (via clang)
|   +-- test_linux.sh             # End-to-end test in Docker
|   +-- test_linker.sh            # Test the linker pipeline
+-- Makefile
+-- Dockerfile
+-- go.mod
+-- README.md                     # Architecture, build, and usage docs
```


MAKEFILE
===============================================================================

```makefile
.PHONY: all build test test-macos test-linux clean

all: build test

build:
	go build -o bin/minicc ./cmd/minicc
	go build -o bin/minilink ./cmd/minilink

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/minicc-linux ./cmd/minicc
	GOOS=linux GOARCH=amd64 go build -o bin/minilink-linux ./cmd/minilink

test:
	go test ./pkg/...

test-macos: build
	@echo "=== Testing on macOS (via clang) ==="
	@for f in testdata/valid/*.mc; do \
		name=$$(basename $$f .mc); \
		expected=$$(grep $$name testdata/expected/results.txt | awk '{print $$2}'); \
		bin/minicc -platform macos $$f -o /tmp/$$name.s && \
		clang -arch x86_64 /tmp/$$name.s -o /tmp/$$name && \
		result=$$(/tmp/$$name; echo $$?); \
		if [ "$$result" = "$$expected" ]; then \
			echo "PASS: $$name ($$result)"; \
		else \
			echo "FAIL: $$name (expected $$expected, got $$result)"; \
		fi; \
	done

test-linux: build-linux
	@echo "=== Testing in Docker (via linker) ==="
	docker run --platform linux/amd64 --rm -v $(PWD):/work -w /work \
		minicc-linux bash scripts/test_linux.sh

clean:
	rm -rf bin/ /tmp/*.s /tmp/*.o
```


DEBUGGING WORKFLOW
===============================================================================

When a compiled program produces the wrong result:

```
  Step 1: Check the token stream
  $ bin/minicc -dump-tokens testdata/valid/factorial.mc

  Step 2: Check the AST
  $ bin/minicc -dump-ast testdata/valid/factorial.mc

  Step 3: Check the IR
  $ bin/minicc -dump-ir testdata/valid/factorial.mc

  Step 4: Check the assembly
  $ bin/minicc -platform macos testdata/valid/factorial.mc -o /tmp/test.s
  $ cat /tmp/test.s

  Step 5: Assemble and debug
  $ clang -g -arch x86_64 /tmp/test.s -o /tmp/test
  $ lldb /tmp/test
  (lldb) b _main
  (lldb) run
  (lldb) register read
  (lldb) si     # step through instructions

  Step 6: Compare with reference compiler
  # Write the same logic in C, compile with clang -S, compare assembly
  $ clang -S -O0 reference.c -o reference.s
  $ diff reference.s /tmp/test.s
```


GRADING RUBRIC
===============================================================================

```
  Component                          Points   Criteria
  -----------------------------------  ------  ----------------------------
  Lexer                                10      Tokenizes all test programs
  Parser                               15      Correct AST for all programs
  Semantic Analysis                    10      Catches all error programs
  IR Generation                        10      Correct three-address code
  Code Generation                      15      Correct assembly output
  macOS Pipeline (clang)               10      All test programs pass
  ELF Object Writer                    10      Valid .o files (readelf)
  Static Linker                        15      Links and produces executables
  Linux Pipeline (Docker)              15      All test programs pass
  Test Suite Quality                   10      Coverage, edge cases
  Code Quality                         10      Clean Go code, tests
  Documentation & Report               10      Architecture, reflection
  ---------------------------------------------------------------
  Total                               140
  (Bonus points possible for stretch goals)
```


STRETCH GOALS (BONUS)
===============================================================================

Each stretch goal is worth up to 10 bonus points:

  SG1: Register Allocation
    Implement a simple register allocator (linear scan or graph coloring)
    that keeps frequently-used values in registers instead of always
    spilling to the stack. Demonstrate speedup.

  SG2: Optimization Passes
    Implement at least two optimization passes on the IR:
    constant folding + dead code elimination. Show before/after IR.

  SG3: String Support
    Add string literals to Mini-C. Emit them in .rodata. Implement
    print_str as a runtime function.

  SG4: Static Library Support
    Implement reading .a archive files in the linker.

  SG5: DWARF Debug Info
    Emit DWARF debug sections so that gdb/lldb can show source-level
    debugging (line numbers, variable names) for Mini-C programs.

  SG6: ARM64 Backend
    Add an ARM64 code generator. Compile Mini-C programs to run natively
    on Apple Silicon without Rosetta 2.
