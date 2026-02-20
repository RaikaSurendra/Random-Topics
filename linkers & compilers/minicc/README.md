# minicc — Mini-C Compiler, ELF Writer & Static Linker

A from-scratch compiler toolchain for Mini-C, a small subset of C with integer-only
types, written in Go. Part of a 14-week course on Compiler Design and Linker Internals.

## Components

| Tool | Description |
|------|-------------|
| `minicc` | Mini-C → x86-64 assembly compiler |
| `minilink` | Minimal static ELF linker |

## Mini-C Language

Mini-C supports:
- `int` variables (32-bit signed), global and local
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Boolean: `&&`, `||`, `!`
- Control flow: `if`/`else`, `while`, `return`
- Functions with up to 6 parameters
- Nested block scopes with shadowing
- Single-line comments (`//`)

## Build

```bash
make build   # builds bin/minicc and bin/minilink
make test    # runs all Go tests
make vet     # runs go vet
```

## Usage (macOS, ch07+)

```bash
# Compile Mini-C to assembly
./bin/minicc -platform macos testdata/valid/06_factorial.mc -o /tmp/factorial.s

# Assemble and link with clang (x86_64 via Rosetta 2)
clang /tmp/factorial.s runtime/runtime.c -o /tmp/factorial -arch x86_64

# Run
/tmp/factorial
echo $?  # should print 120
```

## Usage (Linux/Docker, ch09+)

```bash
make docker-build
make docker-test
```

## Project Structure

```
minicc/
├── cmd/minicc/        # Compiler CLI
├── cmd/minilink/      # Linker CLI
├── pkg/
│   ├── lexer/         # Tokenizer
│   ├── parser/        # Recursive descent parser + AST
│   ├── semantic/      # Symbol tables, scope, type checking
│   ├── ir/            # Three-address code IR
│   ├── codegen/       # x86-64 code generation
│   ├── elf/           # ELF format types and writer
│   └── linker/        # ELF reader, symbol resolution, relocation
├── testdata/
│   ├── valid/         # 20 Mini-C test programs
│   ├── invalid/       # 10 error programs
│   └── expected/      # Expected results
├── runtime/           # C runtime (print_int, main wrapper)
├── scripts/           # Test scripts
├── Makefile
├── Dockerfile
└── go.mod
```

## Chapter Progression

| Ch | Branch | What Gets Built |
|----|--------|----------------|
| 01 | `minicc/ch01-project-init` | Project scaffold, test programs |
| 02 | `minicc/ch02-lexer` | Tokenizer |
| 03 | `minicc/ch03-parser` | Parser + AST |
| 04 | `minicc/ch04-semantic` | Semantic analysis |
| 05 | `minicc/ch05-ir` | IR builder |
| 06 | `minicc/ch06-codegen` | x86-64 codegen |
| 07 | `minicc/ch07-macos-pipeline` | macOS end-to-end |
| 08 | `minicc/ch08-elf-types` | ELF type definitions |
| 09 | `minicc/ch09-elf-writer` | ELF object writer |
| 10 | `minicc/ch10-linker-reader` | ELF reader + symbols |
| 11 | `minicc/ch11-linker-reloc` | Relocation + linking |
| 12 | `minicc/ch12-final` | Full pipeline validation |
