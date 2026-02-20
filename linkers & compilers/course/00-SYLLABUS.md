===============================================================================
  COMPILER DESIGN AND LINKER INTERNALS
  Cross-Platform Implementation in Go

  A 14-Week University Course
  Prerequisites: Go proficiency, Data Structures & Algorithms, OS Basics
===============================================================================

COURSE OVERVIEW
===============================================================================

This course takes you from source code to running binary — every byte explained.

You will build:
  1. A complete compiler for a Mini-C language, written in Go
  2. A code generator targeting x86-64 assembly
  3. An ELF object file writer in Go
  4. A minimal static linker in Go
  5. Working executables on both macOS (via clang) and Linux (via your linker)

You will understand:
  - How text becomes machine code
  - How object files are structured (ELF and Mach-O)
  - How linkers resolve symbols and apply relocations
  - How operating system loaders map binaries into memory
  - How the entire toolchain fits together


COURSE SCHEDULE
===============================================================================

```
  Week | Part | Topic                                    | Deliverable
  -----+------+------------------------------------------+---------------------
    1  |  1   | Compilation Pipeline & Toolchains        | Lab 1: Toolchain
    2  |  1   | Binary Formats, Memory Layout, ABI       | Lab 2: Binary Analysis
    3  |  2   | Mini-C Language Design & Grammar          | Language Spec
    4  |  3   | Lexer                                    | Working Lexer
    5  |  3   | Parser & AST                             | Working Parser
    6  |  3   | Semantic Analysis                        | Type Checker
    7  |  3   | Intermediate Representation              | IR Generator
    8  |  3/4 | Code Generation + macOS Native Phase     | Assembly Output
    9  |  5   | ELF Format Deep Dive                     | ELF Analysis Lab
   10  |  5   | ELF Object File Writer in Go             | .o File Writer
   11  |  5   | Static Linker -- Symbol Resolution       | Linker Phase 1
   12  |  5   | Static Linker -- Relocation & Output     | Working Linker
   13  |  6   | Dynamic Linking, GOT/PLT, Loaders        | Analysis Report
   14  |  6   | Go Internals, LTO, JIT, Course Wrap-Up   | Final Project
  -----+------+------------------------------------------+---------------------
```


DEVELOPMENT ENVIRONMENT
===============================================================================

Primary Machine: macOS (Apple Silicon)
Linux Environment: Docker container (Ubuntu 22.04, x86-64 via Rosetta 2)

Required Tools:
  - Go 1.24+
  - Docker Desktop with Rosetta 2 enabled
  - clang (Xcode Command Line Tools)
  - NASM (brew install nasm) — optional, for experimentation
  - lldb (ships with Xcode)

Docker Setup (used from Week 9 onward):
  $ docker run --platform linux/amd64 -it -v $(pwd):/work -w /work \
      ubuntu:22.04 bash
  # Inside container:
  $ apt update && apt install -y build-essential nasm binutils

Verification:
  $ go version           # Go 1.24+
  $ clang --version      # Apple clang 16+
  $ docker --version     # Docker 29+
  $ otool --version      # Mach-O analysis tool
  $ file /bin/ls         # Shows Mach-O on macOS


PROJECT DIRECTORY LAYOUT
===============================================================================

```
  minicc/
  +-- cmd/
  |   +-- minicc/
  |       +-- main.go              # Compiler entry point
  +-- pkg/
  |   +-- lexer/
  |   |   +-- lexer.go
  |   |   +-- token.go
  |   |   +-- lexer_test.go
  |   +-- parser/
  |   |   +-- parser.go
  |   |   +-- ast.go
  |   |   +-- parser_test.go
  |   +-- semantic/
  |   |   +-- analyzer.go
  |   |   +-- symtable.go
  |   |   +-- semantic_test.go
  |   +-- ir/
  |   |   +-- ir.go
  |   |   +-- builder.go
  |   |   +-- ir_test.go
  |   +-- codegen/
  |   |   +-- x86_64.go
  |   |   +-- codegen.go
  |   |   +-- codegen_test.go
  |   +-- elf/
  |   |   +-- writer.go
  |   |   +-- types.go
  |   |   +-- elf_test.go
  |   +-- linker/
  |       +-- linker.go
  |       +-- symbol.go
  |       +-- relocation.go
  |       +-- linker_test.go
  +-- testdata/
  |   +-- hello.mc                 # Mini-C test programs
  |   +-- factorial.mc
  |   +-- fibonacci.mc
  |   +-- globals.mc
  +-- go.mod
  +-- go.sum
  +-- Makefile
  +-- Dockerfile
```


MILESTONE BREAKDOWN
===============================================================================

Milestone 1 (Week 4):  Lexer tokenizes all Mini-C programs correctly
Milestone 2 (Week 5):  Parser produces valid ASTs for all test programs
Milestone 3 (Week 6):  Semantic analyzer catches all type/scope errors
Milestone 4 (Week 7):  IR generator produces three-address code
Milestone 5 (Week 8):  Codegen emits x86-64 assembly; runs on macOS via clang
Milestone 6 (Week 10): ELF writer produces valid .o files (verified by readelf)
Milestone 7 (Week 12): Static linker produces runnable ELF executables
Milestone 8 (Week 14): Final project submission with documentation


GRADING
===============================================================================

```
  Component                          Weight
  ------------------------------------------
  Weekly Labs & Exercises             20%
  Compiler Implementation             30%
  ELF Writer + Linker Implementation  25%
  Final Project Report                15%
  Code Quality & Tests                10%
  ------------------------------------------
```


READING LIST
===============================================================================

Primary Texts:
  [1] Aho, Lam, Sethi, Ullman — "Compilers: Principles, Techniques, & Tools"
      (The Dragon Book), 2nd Edition, Pearson, 2006
  [2] Cooper, Torczon — "Engineering a Compiler", 3rd Edition, Morgan Kaufmann
  [3] Tool Interface Standard (TIS) — "Executable and Linkable Format (ELF)
      Specification", Version 1.2
  [4] Levine — "Linkers and Loaders", Morgan Kaufmann, 1999

Supplementary:
  [5] Nisan, Schocken — "The Elements of Computing Systems" (Nand2Tetris)
  [6] Appel — "Modern Compiler Implementation in C/Java/ML"
  [7] System V Application Binary Interface — AMD64 Architecture Supplement
  [8] Apple — "Mach-O Programming Topics" (developer.apple.com)
  [9] LLVM Language Reference Manual (llvm.org/docs/LangRef.html)
  [10] Ian Lance Taylor — "Linkers" blog series (20 parts)
       https://lwn.net/Articles/276782/

Online References:
  - ELF spec:    https://refspecs.linuxfoundation.org/elf/elf.pdf
  - x86-64 ABI:  https://gitlab.com/x86-psABIs/x86-64-ABI
  - Go source:   https://github.com/golang/go (cmd/link, cmd/compile)
