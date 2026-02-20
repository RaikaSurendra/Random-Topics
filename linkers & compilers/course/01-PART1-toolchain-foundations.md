===============================================================================
  PART 1 — TOOLCHAIN & BINARY FOUNDATIONS
  Weeks 1–2
===============================================================================


███████████████████████████████████████████████████████████████████████████████
  WEEK 1: THE COMPILATION PIPELINE
███████████████████████████████████████████████████████████████████████████████


1.1  WHAT IS A COMPILER?
===============================================================================

Formal Definition:

  A compiler is a program that reads a program written in one language
  (the source language) and translates it into an equivalent program in
  another language (the target language).

  Formally: A compiler is a function  C: L_source → L_target  such that
  for all programs P in L_source, the observable behavior of C(P) in
  L_target is equivalent to the defined semantics of P in L_source.

The key word is "equivalent." The compiled program must preserve the
observable behavior defined by the source language's semantics — not
necessarily its structure, variable names, or execution strategy.

A compiler differs from:
  - An interpreter: executes source directly, no target program produced
  - A transpiler: source and target are at similar abstraction levels
  - An assembler: translates 1:1 from assembly mnemonics to machine code

Historical Context:

  1957  FORTRAN compiler (John Backus, IBM) — first optimizing compiler
  1962  LISP 1.5 — introduced garbage collection and self-hosting
  1970s C language and PCC (Portable C Compiler)
  1987  GCC 1.0 (Richard Stallman) — GNU Compiler Collection
  2000  LLVM project begins (Chris Lattner, University of Illinois)
  2007  Clang — C/C++/ObjC frontend for LLVM
  2009  Go 1.0 — initially compiled via C, later self-hosting (Go 1.5+)

Why this matters to us:
  GCC and Clang are the toolchains we will use to assemble and link our
  compiler's output. Understanding their architecture informs our design.


1.2  THE COMPILATION PIPELINE
===============================================================================

The transformation from source code to running binary happens in four
distinct phases. Each phase has a dedicated tool.

```
                +-----------------------------------------------------+
                |              THE COMPILATION PIPELINE                |
                +-----------------------------------------------------+

  Source Code (.c)
       |
       v
  +---------------+    Expands #include, #define, #ifdef
  | PREPROCESSOR  |    Tool: cpp, or clang -E
  |   (cpp)       |    Output: translation unit (.i)
  +-------+-------+
          |
          v
  +---------------+    Lexing -> Parsing -> Semantic Analysis -> Optimization
  |  COMPILER     |    -> Code Generation
  |  (cc1)        |    Tool: clang -S
  +-------+-------+    Output: assembly (.s)
          |
          v
  +---------------+    Translates mnemonics to machine code
  |  ASSEMBLER    |    Tool: as, or clang -c
  |  (as)         |    Output: object file (.o)
  +-------+-------+
          |
          v
  +---------------+    Combines .o files, resolves symbols, applies relocations
  |   LINKER      |    Tool: ld, or clang (which invokes ld internally)
  |   (ld)        |    Output: executable (a.out, ELF, Mach-O)
  +-------+-------+
          |
          v
    Executable Binary
```


Practical Demonstration — Observing Each Phase:

  Given this source file:

  // hello.c
  #include <stdio.h>
  int main() {
      int x = 42;
      printf("x = %d\n", x);
      return 0;
  }

  Phase 1 — Preprocessing:
  ─────────────────────────

  $ clang -E hello.c -o hello.i
  $ wc -l hello.i        # Thousands of lines — stdio.h expanded
  $ head -50 hello.i     # See the expanded headers

  The preprocessor:
  - Expands all #include directives (recursively)
  - Substitutes all #define macros
  - Evaluates #ifdef / #ifndef conditional compilation
  - Strips comments
  - Output is pure C with no preprocessor directives


  Phase 2 — Compilation (Source → Assembly):
  ──────────────────────────────────────────

  $ clang -S hello.c -o hello.s
  $ cat hello.s

  On macOS (AT&T syntax), you'll see something like:

      .section    __TEXT,__text
      .globl      _main
  _main:
      pushq   %rbp
      movq    %rsp, %rbp
      subq    $16, %rsp
      movl    $42, -4(%rbp)
      movl    -4(%rbp), %esi
      leaq    L_.str(%rip), %rdi
      xorl    %eax, %eax
      callq   _printf
      xorl    %eax, %eax
      addq    $16, %rsp
      popq    %rbp
      retq

      .section    __TEXT,__cstring
  L_.str:
      .asciz  "x = %d\n"

  On Linux:
  $ gcc -S hello.c -o hello.s
  # Output uses .text section, no underscore prefix on symbols

  Key observations:
  - _main on macOS vs main on Linux (Mach-O prefixes C symbols with _)
  - Section names differ: __TEXT,__text (Mach-O) vs .text (ELF)
  - String literals in __TEXT,__cstring (Mach-O) vs .rodata (ELF)
  - callq _printf — an external symbol, unresolved at this stage


  Phase 3 — Assembly (Assembly → Object File):
  ─────────────────────────────────────────────

  $ clang -c hello.c -o hello.o
  $ file hello.o

  macOS output:
    hello.o: Mach-O 64-bit object x86_64

  Linux output:
    hello.o: ELF 64-bit LSB relocatable, x86-64, version 1 (SYSV)

  The assembler:
  - Converts mnemonics to encoded machine instructions
  - Assigns offsets within sections
  - Creates a symbol table (defined and undefined symbols)
  - Creates relocation entries for unresolved references
  - Does NOT assign final memory addresses — that's the linker's job


  Phase 4 — Linking (Object Files → Executable):
  ──────────────────────────────────────────────

  $ clang hello.o -o hello
  $ file hello

  macOS:
    hello: Mach-O 64-bit executable x86_64

  Linux:
    hello: ELF 64-bit LSB executable, x86-64, version 1 (SYSV),
    dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2

  The linker:
  - Reads all input object files
  - Merges sections (.text from all .o files into one .text)
  - Resolves symbol references (connects callq printf to libc's printf)
  - Applies relocations (patches machine code with correct addresses)
  - Writes the final executable with program headers for the OS loader


  End-to-End — Verbose Mode:
  ──────────────────────────

  $ clang -v hello.c -o hello 2>&1 | head -30

  This reveals every subprocess clang invokes:
  1. /usr/bin/clang -cc1 ...       (compiler proper)
  2. /usr/bin/as ...                (assembler — on Linux)
  3. /usr/bin/ld ...                (linker)

  On macOS, clang uses an integrated assembler, so steps 1+2 are combined.


1.3  INSIDE AN OBJECT FILE
===============================================================================

Object files are the intermediate artifacts between compilation and linking.
They contain machine code but are NOT executable — they contain unresolved
references that the linker must fix.

An object file contains:

```
  +--------------------------------------------------+
  |                  OBJECT FILE                      |
  +--------------------------------------------------+
  |  Header          Format magic, architecture      |
  +--------------------------------------------------+
  |  .text           Machine code (instructions)     |
  +--------------------------------------------------+
  |  .data           Initialized global variables    |
  +--------------------------------------------------+
  |  .bss            Uninitialized globals (no bytes)|
  +--------------------------------------------------+
  |  .rodata         Read-only data (string literals)|
  +--------------------------------------------------+
  |  Symbol Table    Names, addresses, types, scope  |
  +--------------------------------------------------+
  |  Relocation      Patches needed by the linker    |
  |  Entries                                         |
  +--------------------------------------------------+
  |  Section Table   Index of all sections           |
  +--------------------------------------------------+
```


Inspecting Object Files — The Essential Tools:

  nm — List Symbols:
  ──────────────────

  $ nm hello.o

  macOS output:
    0000000000000000 T _main
                     U _printf

  Linux output:
    0000000000000000 T main
                     U printf

  Symbol types:
    T = defined in .text (code)
    D = defined in .data (initialized data)
    B = defined in .bss (uninitialized data)
    U = undefined (must be resolved by linker)
    t = local (static) text symbol
    d = local data symbol

  Key insight: _main is defined at offset 0 within this object's .text
  section. _printf is undefined — the linker must find it.


  objdump — Disassemble:
  ──────────────────────

  Linux:
  $ objdump -d hello.o

  hello.o:     file format elf64-x86-64
  Disassembly of section .text:

  0000000000000000 <main>:
     0: 55                    push   %rbp
     1: 48 89 e5              mov    %rsp,%rbp
     4: 48 83 ec 10           sub    $0x10,%rsp
     8: c7 45 fc 2a 00 00 00  movl   $0x2a,-0x4(%rbp)
     f: 8b 75 fc              mov    -0x4(%rbp),%esi
    12: 48 8d 3d 00 00 00 00  lea    0x0(%rip),%rdi
    19: b8 00 00 00 00        mov    $0x0,%eax
    1e: e8 00 00 00 00        call   0x23 <main+0x23>
    23: 31 c0                 xor    %eax,%eax
    25: 48 83 c4 10           add    $0x10,%rsp
    29: 5d                    pop    %rbp
    2a: c3                    ret

  CRITICAL OBSERVATION:
  At offset 0x12, the lea instruction loads the address of the format string.
  The operand is 0x0(%rip) — that's a PLACEHOLDER. The real address is unknown.

  At offset 0x1e, the call instruction targets offset 0x23 (the next
  instruction!) — clearly wrong. This is also a PLACEHOLDER.

  These placeholders are what relocation entries fix.


  readelf — ELF Structure (Linux only):
  ──────────────────────────────────────

  $ readelf -h hello.o        # ELF header
  $ readelf -S hello.o        # Section headers
  $ readelf -s hello.o        # Symbol table
  $ readelf -r hello.o        # Relocation entries

  Relocation output:
  Relocation section '.rela.text' at offset 0x...:
    Offset          Info           Type              Sym. Value    Sym. Name + Addend
  000000000015  000500000002 R_X86_64_PC32     0000000000000000 .rodata - 4
  00000000001f  000600000004 R_X86_64_PLT32    0000000000000000 printf - 4

  This tells the linker:
  - At offset 0x15 in .text, patch in the PC-relative address of .rodata
  - At offset 0x1f in .text, patch in the PLT entry for printf


  otool — Mach-O Analysis (macOS only):
  ──────────────────────────────────────

  $ otool -h hello.o          # Mach-O header
  $ otool -l hello.o          # Load commands
  $ otool -t hello.o          # Text section (hex)
  $ otool -tv hello.o         # Disassemble text
  $ otool -r hello.o          # Relocation entries


1.4  SYMBOL TABLES IN DEPTH
===============================================================================

A symbol table maps names to properties. Every object file has one.

Each symbol entry contains:
  - Name:    The identifier (e.g., "main", "printf", "global_count")
  - Value:   Offset within its section (or absolute address after linking)
  - Size:    Size in bytes (for data symbols)
  - Type:    FUNC (function), OBJECT (variable), NOTYPE, SECTION, FILE
  - Binding: LOCAL (file-scoped), GLOBAL (visible to linker), WEAK
  - Section: Which section it belongs to (or UNDEF if external)

Example — Two Source Files:

  // math.c
  int square(int x) {
      return x * x;
  }

  // main.c
  extern int square(int x);
  int result;
  int main() {
      result = square(7);
      return 0;
  }

  $ gcc -c math.c main.c
  $ nm math.o
  0000000000000000 T square

  $ nm main.o
                   U square        ← undefined, needs linking
  0000000000000004 C result        ← common symbol (uninitialized global)
  0000000000000000 T main

  After linking:
  $ gcc math.o main.o -o program
  $ nm program | grep -E "square|result|main"
  0000000000401136 T main
  0000000000404030 B result        ← placed in .bss
  0000000000401126 T square        ← now has final address

  Symbol resolution: The linker matched main.o's undefined "square"
  with math.o's defined "square" and patched the call instruction.


1.5  STATIC VS DYNAMIC LINKING
===============================================================================

  STATIC LINKING                        DYNAMIC LINKING
  ─────────────────────────────────     ─────────────────────────────────
  All code copied into executable       Only references stored in binary
  Larger binary                         Smaller binary
  No runtime dependencies               Requires .so/.dylib at runtime
  Faster startup (no loading)           Slower startup (loader resolves)
  One version, frozen at link time      Can update library independently
  Tool: ld (with archives .a)          Tool: ld (with shared libs .so)

  Static Library = Archive of .o files:
  $ ar rcs libmath.a math.o            # Create static library
  $ gcc main.o -L. -lmath -o program   # Link against it

  Dynamic Library:
  $ gcc -shared -o libmath.so math.o   # Create shared library (Linux)
  $ gcc main.o -L. -lmath -o program   # Link against it
  $ ldd program                        # Shows runtime dependencies

  macOS equivalents:
  $ clang -dynamiclib -o libmath.dylib math.o
  $ otool -L program                   # Shows dynamic dependencies

  We will build a STATIC linker in this course. Dynamic linking is covered
  theoretically in Part 6.


███████████████████████████████████████████████████████████████████████████████
  WEEK 2: BINARY FORMATS AND MEMORY LAYOUT
███████████████████████████████████████████████████████████████████████████████


2.1  ELF vs MACH-O — THE TWO BINARY FORMATS
===============================================================================

Every OS needs a standard binary format that tells the kernel how to load
and execute a program. The two we care about:

  ELF (Executable and Linkable Format):
  - Used by: Linux, FreeBSD, Solaris, most Unixes
  - File types: Relocatable (.o), Executable, Shared object (.so), Core dump
  - Specification: TIS ELF v1.2 + OS/arch supplements

  Mach-O (Mach Object):
  - Used by: macOS, iOS, watchOS, tvOS
  - File types: Object, Executable, Dynamic library (.dylib), Bundle
  - Specification: Apple developer documentation

```
  +--------------+------------------------+--------------------------+
  |   Feature    |         ELF            |        Mach-O            |
  +--------------+------------------------+--------------------------+
  | Magic bytes  | 7f 45 4c 46 (\x7fELF) | fe ed fa cf (64-bit)     |
  | Sections     | .text, .data, .bss     | __TEXT,__text etc.       |
  | Segments     | LOAD, DYNAMIC, INTERP  | __TEXT, __DATA, __LINKEDIT|
  | Symbol pfx   | main                   | _main (underscore)       |
  | Reloc types  | R_X86_64_PC32 etc.     | X86_64_RELOC_BRANCH etc. |
  | Loader       | ld-linux-x86-64.so.2   | /usr/lib/dyld            |
  | Inspect tool | readelf, objdump       | otool, MachOView         |
  | Linker       | ld (GNU/gold/mold/lld) | ld64 (Apple)             |
  +--------------+------------------------+--------------------------+
```


2.2  ELF FORMAT OVERVIEW
===============================================================================

An ELF file has this structure:

```
  Offset 0
  +------------------------+
  |      ELF Header        |  64 bytes (for ELF64)
  |  magic, class, type,   |
  |  machine, entry point, |
  |  phdr offset, shdr off |
  +------------------------+
  |   Program Headers      |  Optional in .o files
  |   (Segment Table)      |  Required in executables
  +------------------------+
  |      .text             |  Machine code
  +------------------------+
  |      .rodata           |  Read-only data
  +------------------------+
  |      .data             |  Initialized data
  +------------------------+
  |      .bss              |  (No bytes -- just a size)
  +------------------------+
  |      .symtab           |  Symbol table
  +------------------------+
  |      .strtab           |  String table for symbols
  +------------------------+
  |      .shstrtab         |  String table for section names
  +------------------------+
  |      .rela.text        |  Relocations for .text
  +------------------------+
  |   Section Headers      |  Table of all sections
  +------------------------+
```

  ELF Header Fields (we will write these byte-by-byte in Part 5):
  - e_ident[16]: Magic number + class + endianness + OS ABI
  - e_type:      ET_REL (1) for .o, ET_EXEC (2) for executable
  - e_machine:   EM_X86_64 (0x3E)
  - e_entry:     Entry point virtual address (0 for .o files)
  - e_phoff:     Program header table offset
  - e_shoff:     Section header table offset
  - e_shnum:     Number of section headers
  - e_shstrndx:  Index of the section name string table


2.3  MACH-O FORMAT OVERVIEW
===============================================================================

```
  Offset 0
  +------------------------+
  |    Mach-O Header       |  32 bytes
  |  magic, cputype,       |
  |  filetype, ncmds       |
  +------------------------+
  |    Load Commands       |  Variable length, sequential
  |  LC_SEGMENT_64         |  -> contains sections
  |  LC_SYMTAB             |  -> symbol table info
  |  LC_DYSYMTAB           |  -> dynamic symbol info
  |  LC_MAIN               |  -> entry point
  |  ...                   |
  +------------------------+
  |    Segment: __TEXT      |
  |      Section: __text   |  Machine code
  |      Section: __cstring|  String literals
  +------------------------+
  |    Segment: __DATA     |
  |      Section: __data   |  Initialized data
  |      Section: __bss    |  Uninitialized data
  +------------------------+
  |    Link Edit Data      |
  |      Symbol table      |
  |      String table      |
  |      Relocations       |
  +------------------------+
```

  Mach-O uses "load commands" instead of ELF's dual header tables.
  Each LC_SEGMENT_64 command describes a memory region and contains
  section descriptors within it.

  Practical inspection:
  $ otool -h /bin/ls                  # Header
  $ otool -l /bin/ls | head -80       # Load commands
  $ size /bin/ls                      # Section sizes


2.4  MEMORY LAYOUT OF A RUNNING PROGRAM
===============================================================================

When the OS loads an executable, it maps the binary into virtual memory:

```
  High Address
  +------------------------------+  0x7FFF_FFFF_FFFF (Linux user space top)
  |         Kernel Space         |  (not accessible from user mode)
  +------------------------------+
  |                              |
  |          Stack  (v)          |  Grows downward
  |    (local vars, frames,      |  RSP points to current top
  |     return addresses)        |
  |                              |
  + - - - - - - - - - - - - - - +  <-- Stack growth limit
  |                              |
  |       (unmapped gap)         |
  |                              |
  + - - - - - - - - - - - - - - +
  |                              |
  |          Heap  (^)           |  Grows upward (malloc/brk)
  |    (dynamic allocations)     |
  |                              |
  +------------------------------+
  |          .bss                |  Uninitialized globals (zeroed)
  +------------------------------+
  |          .data               |  Initialized globals
  +------------------------------+
  |          .rodata             |  Read-only data (string literals)
  +------------------------------+
  |          .text               |  Executable code
  +------------------------------+
  Low Address                       0x0040_0000 (typical Linux start)
```

  Notes:
  - .text is mapped read+execute (no write)
  - .data is mapped read+write
  - .rodata is mapped read-only
  - .bss occupies no space in the file — the OS zeros the memory
  - Stack starts near the top of user space and grows down
  - On macOS, the base address and layout differ slightly but the
    concept is identical

  Verifying layout:
  $ cat /proc/self/maps    # Linux — shows all memory mappings
  $ vmmap <pid>            # macOS — shows memory regions


2.5  STACK FRAMES AND CALLING CONVENTIONS
===============================================================================

When a function is called, a "stack frame" is created:

```
  High Address
  +----------------------+
  |  Caller's frame      |
  +----------------------+
  |  Return address      |  <-- pushed by 'call' instruction
  +----------------------+  <-- RBP points here (after push rbp; mov rsp,rbp)
  |  Saved RBP           |
  +----------------------+
  |  Local var 1         |  -8(%rbp)
  +----------------------+
  |  Local var 2         |  -16(%rbp)
  +----------------------+
  |  ...                 |
  +----------------------+  <-- RSP points here (current stack top)
  Low Address
```

  The System V AMD64 ABI (used by Linux AND macOS):

  Argument Passing:
    Integer args 1-6:  RDI, RSI, RDX, RCX, R8, R9  (in order)
    Additional args:   Pushed onto stack (right-to-left)
    Return value:      RAX (and RDX for 128-bit returns)

  Caller-saved registers (volatile — may be destroyed by callee):
    RAX, RCX, RDX, RSI, RDI, R8, R9, R10, R11

  Callee-saved registers (must be preserved across calls):
    RBX, RBP, R12, R13, R14, R15

  Stack alignment: RSP must be 16-byte aligned BEFORE the call instruction.
  After call pushes the return address (8 bytes), RSP is misaligned by 8.
  This is why function prologues often sub $8 or $24 (to realign to 16).

  Function call sequence:
    1. Caller places args in RDI, RSI, RDX, RCX, R8, R9
    2. Caller executes CALL instruction (pushes return addr, jumps)
    3. Callee prologue: push %rbp; mov %rsp, %rbp; sub $N, %rsp
    4. Callee body executes
    5. Callee places return value in RAX
    6. Callee epilogue: mov %rbp, %rsp; pop %rbp; ret
    (Or equivalently: leave; ret)

  Example:

  int add(int a, int b) { return a + b; }
  int main() { return add(3, 4); }

  Assembly for main:
      push    %rbp
      mov     %rsp, %rbp
      mov     $3, %edi          # First arg → RDI
      mov     $4, %esi          # Second arg → RSI
      call    add
      pop     %rbp
      ret

  Assembly for add:
      push    %rbp
      mov     %rsp, %rbp
      mov     %edi, -4(%rbp)    # Save first arg
      mov     %esi, -8(%rbp)    # Save second arg
      mov     -4(%rbp), %eax
      add     -8(%rbp), %eax    # Result in EAX
      pop     %rbp
      ret

  THIS IS EXACTLY THE CODE OUR COMPILER WILL GENERATE.


2.6  RELOCATION — THE LINKER'S CORE OPERATION
===============================================================================

Relocation is the process of adjusting addresses in machine code after
the linker decides where each section will be placed in the final binary.

Why is it necessary?

  When the assembler processes a single .o file, it doesn't know:
  1. Where the .text section will be in the final executable
  2. Where symbols from OTHER .o files will be
  3. What the final address of global variables will be

  So the assembler leaves placeholders (usually 0x00000000) and creates
  relocation entries that tell the linker what to patch and how.

  Relocation Entry (ELF):
```
  +------------------------------------------------------+
  |  Offset    |  Where in the section to patch          |
  |  Symbol    |  Which symbol's address to use          |
  |  Type      |  How to compute the patched value       |
  |  Addend    |  Constant to add to the computed value  |
  +------------------------------------------------------+
```

  Common x86-64 relocation types:
  ─────────────────────────────────────────────────────────────────────
  R_X86_64_PC32     Relative 32-bit: S + A - P
                    (S = symbol addr, A = addend, P = patch location)
                    Used for: call, jmp, lea with RIP-relative addressing

  R_X86_64_32S      Absolute 32-bit (sign-extended)
                    Used for: direct address references in small code model

  R_X86_64_64       Absolute 64-bit
                    Used for: .data section pointers

  R_X86_64_PLT32    PC-relative call to PLT entry
                    Used for: calls to external functions
  ─────────────────────────────────────────────────────────────────────

  Example — Relocation in action:

  Before linking (in hello.o):
    1e: e8 00 00 00 00    call <placeholder>
    Relocation: offset=0x1f, type=R_X86_64_PLT32, sym=printf, addend=-4

  After linking (in hello executable):
    40113e: e8 cd fe ff ff    call 0x401010 <printf@plt>

  The linker computed:
    target = address_of_printf_plt
    patch  = target - (0x40113e + 1) + (-4)   [PC-relative, from next insn]
    Result = 0xFFFFFFCD (signed -307 decimal)
    Written as little-endian: cd fe ff ff


2.7  THE TOOLCHAIN AS A SYSTEM
===============================================================================

```
  +--------+     +---------+     +---------+     +--------+
  | Source | --> |Compiler | --> |Assembler| --> | Linker | --> Executable
  | (.c)   |     | (-S)    |     | (-c)    |     |        |
  +--------+     +---------+     +---------+     +--------+
                      |               |               |
                      v               v               v
                  Assembly (.s)   Object (.o)    Binary (ELF/Mach-O)
                      |               |               |
                      v               v               v
                  Human-readable  Machine code    Runnable
                  mnemonics       + relocations   program
```

  In our course:
  - WE build the compiler (Go)     → emits .s files
  - clang/as assembles for us      → produces .o files  (Weeks 4-8)
  - WE build the ELF writer (Go)   → produces .o files  (Week 10)
  - WE build the linker (Go)       → produces ELF exe   (Weeks 11-12)


WEEK 1 EXERCISES
===============================================================================

Exercise 1.1: Pipeline Observation
  Write a C program with two functions in separate files. Use clang -v
  to compile and link them. Record every subprocess invoked.

Exercise 1.2: Symbol Investigation
  Create three .c files where file A calls functions from B and C.
  Compile each to .o files. Use nm on each. Draw a diagram showing
  which symbols are defined (T) and undefined (U) in each file.
  Then link them and verify all U symbols become resolved.

Exercise 1.3: Relocation Discovery
  Compile a .c file with a function call and a global variable access.
  Use objdump -dr (Linux) or otool -rv (macOS) to find the relocation
  entries. For each, identify: (a) the instruction being patched,
  (b) the target symbol, (c) the relocation type.

Exercise 1.4: Mach-O vs ELF
  Compile the same hello.c on macOS and Linux (Docker). Compare:
  - Object file sizes
  - Number of sections (readelf -S vs otool -l)
  - Symbol names (underscore prefix?)
  - Relocation entry formats


WEEK 2 EXERCISES
===============================================================================

Exercise 2.1: Memory Map
  Write a C program that prints the addresses of: a global variable,
  a local variable, a heap allocation (malloc), a function, and a
  string literal. Run it and map each address to the memory layout
  diagram from section 2.4.

Exercise 2.2: Stack Frame Tracing
  Write a recursive factorial function. Use gdb/lldb to:
  (a) Set a breakpoint at the recursive call
  (b) Print the stack frames at recursion depth 5
  (c) Record RSP and RBP at each frame
  (d) Verify the frame layout matches section 2.5

Exercise 2.3: Manual Linking
  Create two .o files. Use ld directly (not via gcc/clang) to link them.
  Linux:  ld -o program main.o math.o -lc -dynamic-linker /lib64/ld-linux-x86-64.so.2
  macOS:  ld -o program main.o math.o -lSystem -syslibroot $(xcrun --sdk macosx --show-sdk-path)
  Observe what happens when you omit a required .o file.

Exercise 2.4: Binary Comparison
  Use hexdump or xxd on both an ELF and Mach-O object file.
  Find and annotate: (a) the magic bytes, (b) the machine type field,
  (c) the beginning of the .text section.


WEEK 2 READING
===============================================================================

Required:
  - Dragon Book Ch. 1 (Introduction to Compilers)
  - ELF Specification §1-3 (Header, Sections, Program Headers)
  - Ian Lance Taylor "Linkers" Part 1-3

Recommended:
  - "Computer Systems: A Programmer's Perspective" Ch. 7 (Linking)
  - man elf (Linux)
  - Apple Mach-O documentation
