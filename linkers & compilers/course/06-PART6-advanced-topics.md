===============================================================================
  PART 6 — ADVANCED SYSTEMS TOPICS
  Weeks 13-14
===============================================================================


===============================================================================
  WEEK 13: DYNAMIC LINKING, GOT, PLT, AND OS LOADERS
===============================================================================


13.1  WHY DYNAMIC LINKING EXISTS
===============================================================================

Static linking copies every library function into the executable.
If 100 programs use printf, there are 100 copies of printf on disk
and potentially 100 copies in memory.

Dynamic linking solves this:
  - Shared libraries (.so on Linux, .dylib on macOS) are loaded once
    into memory and shared across all processes
  - Executables contain only references, not copies
  - Libraries can be updated without recompiling programs

Trade-offs:
```
  Static                              Dynamic
  --------------------------------    --------------------------------
  Larger binary                       Smaller binary
  No runtime deps                     Requires .so/.dylib at runtime
  Faster function calls               Indirect calls through GOT/PLT
  Immune to library updates           Can break if library changes ABI
  Simpler to deploy                   "DLL hell" / dependency conflicts
```


13.2  POSITION-INDEPENDENT CODE (PIC)
===============================================================================

Shared libraries must work regardless of where in memory they are loaded.
They cannot use absolute addresses because the load address varies.

PIC rules:
  - All code references must be PC-relative (already true for x86-64 calls)
  - All data references must go through the Global Offset Table (GOT)
  - All function calls to other shared objects go through the PLT

Compiling with PIC:

```bash
$ gcc -fPIC -c math.c -o math.o
$ objdump -d math.o
```

You'll see data accesses using `%rip`-relative addressing to reach the GOT:

```asm
  # Without PIC (absolute address - won't work in shared lib):
  movl  global_var, %eax          # absolute address

  # With PIC (GOT-relative):
  movq  global_var@GOTPCREL(%rip), %rax   # load address from GOT
  movl  (%rax), %eax                       # load the actual value
```


13.3  THE GLOBAL OFFSET TABLE (GOT)
===============================================================================

The GOT is a table of pointers in the .data segment. Each entry holds
the runtime address of a global variable or function from another shared
library.

```
  When the program starts:

  .text (code)                    .got (data)
  +-----------------------+       +--------------------+
  | movq X@GOTPCREL(%rip) | --+   | slot 0: &printf    |  <-- filled by loader
  | ...                   |   |   | slot 1: &global_var |
  +-----------------------+   +-> | slot 2: ...         |
                                  +--------------------+

  The dynamic loader (ld-linux.so / dyld) fills in the GOT entries
  at program startup, resolving each symbol to its actual runtime address.
```

Why a table instead of patching code directly?
  - Code segment (.text) is mapped read-only + execute
  - We can't modify code at runtime (W^X security policy)
  - GOT lives in .data which IS writable
  - One level of indirection through readable .got avoids code patches


13.4  THE PROCEDURE LINKAGE TABLE (PLT)
===============================================================================

The PLT provides lazy binding for function calls. Instead of resolving
every function address at startup, functions are resolved on first call.

```
  Calling printf() through PLT:

  .text                           .plt                         .got.plt
  +-------------------+           +------------------------+   +-----------+
  | callq printf@plt  | -------> | printf@plt:            |   | [slot]    |
  +-------------------+           |   jmp *printf@GOTPLT   |-->| initially |
                                  |   pushq $reloc_index   |   | points    |
                                  |   jmp  plt[0]          |   | back to   |
                                  +------------------------+   | pushq     |
                                                               +-----------+

  First call:
  1. callq printf@plt              -> jumps to PLT entry
  2. jmp *printf@GOTPLT            -> GOT entry points back to next PLT insn
  3. pushq $reloc_index            -> push relocation index onto stack
  4. jmp plt[0]                    -> jump to PLT[0] (the resolver stub)
  5. PLT[0] calls _dl_runtime_resolve()
  6. Resolver looks up "printf" in loaded shared libraries
  7. Resolver patches the GOT entry with the REAL address of printf
  8. Resolver jumps to printf

  Second call:
  1. callq printf@plt              -> jumps to PLT entry
  2. jmp *printf@GOTPLT            -> GOT now has REAL address -> jumps directly!

  After the first call, there's zero overhead from lazy binding.
```

Viewing PLT/GOT:

```bash
$ gcc -o hello hello.c
$ objdump -d hello | grep -A5 'printf@plt'
0000000000401030 <printf@plt>:
  401030:  ff 25 e2 2f 00 00    jmp    *0x2fe2(%rip)  # 404018 <printf@GLIBC>
  401036:  68 00 00 00 00       push   $0x0
  40103b:  e9 e0 ff ff ff       jmp    401020 <_init+0x20>

$ readelf -r hello | grep printf
0000000000404018  R_X86_64_JUMP_SLOT  printf@GLIBC_2.2.5

$ objdump -s -j .got.plt hello
```


13.5  HOW THE LINUX LOADER WORKS
===============================================================================

When you run ./program, the kernel doesn't directly execute your code.
Instead, it reads the ELF header to find the "interpreter":

```bash
$ readelf -l hello | grep interpreter
  [Requesting program interpreter: /lib64/ld-linux-x86-64.so.2]
```

The kernel loads ld-linux-x86-64.so.2 (the dynamic loader) and transfers
control to IT. The loader then:

```
  1. Maps the executable's LOAD segments into memory
  2. Reads the DYNAMIC segment to find needed shared libraries
  3. Loads each shared library (recursively loading their dependencies)
  4. Resolves GOT entries for eagerly-bound symbols (BIND_NOW)
  5. Sets up PLT for lazily-bound symbols
  6. Runs initialization functions (.init, .init_array)
  7. Transfers control to the executable's entry point (e_entry)
```

The DYNAMIC segment contains entries like:

```bash
$ readelf -d hello

Dynamic section at offset 0x2de8 contains 27 entries:
  Tag        Type              Name/Value
  0x0000001  (NEEDED)          Shared library: [libc.so.6]
  0x000000c  (INIT)            0x401000
  0x000000d  (FINI)            0x4011c4
  0x0000019  (INIT_ARRAY)      0x403dd8
  ...
```

Tracing the loader:

```bash
$ LD_DEBUG=all ./hello 2>&1 | head -50   # Verbose loader output
$ strace ./hello 2>&1 | head -30         # System calls during loading
```


13.6  HOW THE macOS LOADER (dyld) DIFFERS
===============================================================================

macOS uses /usr/lib/dyld as its dynamic loader.

Key differences:
```
  Linux ld.so                         macOS dyld
  -----------------------------------  -----------------------------------
  Uses .dynamic section                Uses LC_LOAD_DYLIB load commands
  GOT and PLT                          __got and __stubs / __stub_helper
  .so shared objects                   .dylib dynamic libraries
  RPATH, RUNPATH                       @rpath, @executable_path
  LD_LIBRARY_PATH                      DYLD_LIBRARY_PATH
  LD_DEBUG                             DYLD_PRINT_LIBRARIES
  ld-linux-x86-64.so.2                /usr/lib/dyld
  System V ABI                         System V ABI (same calling convention)
```

macOS uses "two-level namespaces" — each symbol records WHICH library
it came from, not just the symbol name. This prevents symbol conflicts
between libraries.

Inspecting on macOS:

```bash
$ otool -L /bin/ls          # List dynamic dependencies
$ DYLD_PRINT_LIBRARIES=1 /bin/ls  # Show libraries loaded at runtime
$ otool -Iv hello           # Show indirect symbols (GOT/PLT equivalent)
```


13.7  EXERCISES
===============================================================================

Exercise 13.1: PLT Walkthrough
  Compile a program that calls printf. Set a breakpoint at the PLT entry
  using gdb/lldb. Step through the lazy binding process:
  (a) First call: observe the resolution path
  (b) Second call: observe the direct jump

Exercise 13.2: GOT Inspection
  Write a program that accesses a global variable from a shared library.
  Use readelf to find the GOT entry. Use gdb to read the GOT contents
  before and after the first access. Confirm the address changes.

Exercise 13.3: Loader Trace
  Use LD_DEBUG=symbols ./program to trace symbol resolution.
  How many symbols does a simple "hello world" program resolve?

Exercise 13.4: Analysis Report
  Write a 2-page report comparing the dynamic linking model of Linux
  (ld.so) and macOS (dyld). Cover: symbol resolution strategy, lazy vs
  eager binding, security features (RELRO, code signing).


===============================================================================
  WEEK 14: GO INTERNALS, LTO, JIT, AND COURSE WRAP-UP
===============================================================================


14.1  HOW GO BUILDS BINARIES INTERNALLY
===============================================================================

Go has its own compiler and linker — it does NOT use gcc or ld.

The Go toolchain:

```
  .go source
       |
       v
  cmd/compile          Go compiler
       |               - Lexer, parser, type checker, SSA-based optimizer
       v               - Emits Go's own object file format
  .o (Go object)
       |
       v
  cmd/link             Go linker
       |               - Links Go objects + runtime + cgo objects
       v               - Resolves symbols
  executable           - Applies relocations
                       - Writes ELF/Mach-O/PE executable

  Key differences from C toolchain:
  - Go compiler emits a custom object format (not standard ELF .o)
  - Go linker is written in Go (self-hosting since Go 1.5)
  - Go produces STATIC executables by default (no libc dependency!)
  - On Linux, Go uses raw syscalls instead of libc wrappers
  - Go runtime (goroutine scheduler, GC, etc.) is linked into every binary
```

Inspecting Go binaries:

```bash
$ go build -o hello hello.go
$ file hello
hello: ELF 64-bit LSB executable, x86-64, statically linked

$ ls -la hello
-rwxr-xr-x 1 user user 1.8M ...  # 1.8MB for hello world!

$ readelf -h hello | grep Type
  Type:  EXEC (Executable file)

$ readelf -l hello
  ... 7 program headers ...
  LOAD  r-x  (code)
  LOAD  r--  (rodata - includes Go type info, string data)
  LOAD  rw-  (data)
  NOTE  (Go build ID)
  ...

$ nm hello | wc -l
  2847  # Thousands of symbols — the entire Go runtime!

$ nm hello | grep 'T main.main'
  000000000047e420 T main.main
```

Why Go binaries are large:
  - Entire runtime statically linked (~1MB minimum)
  - Goroutine scheduler, garbage collector, stack management
  - Reflection type data
  - DWARF debug info (strip with -ldflags="-s -w")

```bash
$ go build -ldflags="-s -w" -o hello_stripped hello.go
$ ls -la hello_stripped
-rwxr-xr-x 1 user user 1.2M ...  # ~600KB saved by stripping
```

Go's linker source code: https://github.com/golang/go/tree/master/src/cmd/link
  - cmd/link/internal/ld/     Main linker code
  - cmd/link/internal/x86/    x86-64 specific code
  - cmd/link/internal/loader/  Symbol management


14.2  LINK-TIME OPTIMIZATION (LTO)
===============================================================================

LTO extends compiler optimization across translation unit boundaries.

Normal compilation:

```
  a.c  ->  a.o  (optimized in isolation)
  b.c  ->  b.o  (optimized in isolation)
  a.o + b.o -> linker -> executable
```

With LTO:

```
  a.c  ->  a.o  (contains IR, not final machine code)
  b.c  ->  b.o  (contains IR, not final machine code)
  a.o + b.o -> linker -> OPTIMIZER (sees all IR together) -> executable
```

Benefits:
  - Cross-file inlining (inline small functions from other files)
  - Cross-file constant propagation
  - Dead code elimination across files
  - Devirtualization

```bash
$ gcc -flto -c a.c b.c
$ gcc -flto -o program a.o b.o   # Optimizer runs during linking

$ clang -flto=thin -c a.c b.c   # ThinLTO: faster, parallel
$ clang -flto=thin -o program a.o b.o
```

How LTO works in LLVM (Clang):
  1. Compiler emits LLVM IR (bitcode) into object files
  2. Linker detects LTO object files
  3. Linker invokes LLVM optimization passes on the combined IR
  4. LLVM generates final machine code
  5. Linker resolves remaining symbols and produces the executable

ThinLTO (LLVM):
  - Only loads function summaries, not full IR
  - Decides which functions to import for cross-module optimization
  - Parallelizes optimization across modules
  - Much faster than full LTO with most of the benefits


14.3  JIT COMPILATION BASICS
===============================================================================

A JIT (Just-In-Time) compiler generates machine code at runtime,
just before it's needed.

```
  Interpreter:     Source -> execute directly (slow, flexible)
  AOT Compiler:    Source -> machine code -> execute (fast, inflexible)
  JIT Compiler:    Source -> bytecode -> machine code at runtime (fast, flexible)
```

How a JIT works:

```
  1. Parse and compile source to bytecode (intermediate form)
  2. Execute bytecode in an interpreter
  3. Identify "hot" code paths (frequently executed)
  4. Compile hot paths to native machine code at runtime
  5. Patch execution to use native code instead of interpreter
  6. Optionally: de-optimize if assumptions are violated
```

JIT compilation in practice:
  - Java HotSpot VM: bytecode -> interpreter -> C1 (quick compile)
    -> C2 (optimizing compile) based on profiling
  - V8 (JavaScript): source -> bytecode -> Sparkplug (baseline)
    -> Maglev -> TurboFan (optimizing)
  - LuaJIT: one of the fastest JIT compilers ever built
  - .NET CoreCLR: MSIL -> RyuJIT

The key system call for JIT:

```c
// Allocate executable memory
void *mem = mmap(NULL, size,
    PROT_READ | PROT_WRITE | PROT_EXEC,  // rwx
    MAP_PRIVATE | MAP_ANONYMOUS,
    -1, 0);

// Write machine code into it
memcpy(mem, machine_code, code_size);

// Cast to function pointer and call
typedef int (*func_t)();
func_t fn = (func_t)mem;
int result = fn();
```

In Go, JIT is harder because Go doesn't easily let you execute arbitrary
memory. But it's possible using syscall.Mmap:

```go
import "syscall"

// Allocate RWX memory
mem, err := syscall.Mmap(-1, 0, 4096,
    syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC,
    syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS)

// Write: mov $42, %eax; ret
copy(mem, []byte{0xB8, 0x2A, 0x00, 0x00, 0x00, 0xC3})

// Convert to callable function (UNSAFE)
type execFunc func() int32
fn := *(*execFunc)(unsafe.Pointer(&mem))
result := fn()
fmt.Println(result) // 42
```

WARNING: This is extremely unsafe and platform-specific. It's shown for
educational purposes to illustrate how JIT compilers work at the lowest level.


14.4  WHERE TO GO FROM HERE
===============================================================================

You've built a compiler and a linker from scratch. Here's where
each skill extends:

```
  What You Built                What It Leads To
  ----------------------------  ------------------------------------------
  Lexer                         Language server protocols, syntax highlighters
  Parser                        IDEs, refactoring tools, code formatters
  Semantic analysis             Type systems, flow analysis, linters
  IR and optimization           LLVM passes, compiler backends
  Code generation               Architecture-specific compilers, JIT engines
  ELF writer                    Binary instrumentation, malware analysis
  Static linker                 Build systems, dynamic linkers, debuggers
```

Recommended next projects:
  1. Add string support to Mini-C (requires .rodata, pointer semantics)
  2. Add arrays (requires pointer arithmetic, bounds checking)
  3. Implement register allocation (graph coloring or linear scan)
  4. Add an optimization pass (constant folding, dead code elimination)
  5. Build a dynamic linker in Go
  6. Add DWARF debug information to ELF output
  7. Build a minimal JIT compiler in Go
  8. Implement a garbage collector
  9. Port the backend to ARM64 (Apple Silicon native)
  10. Read the Go linker source and understand its architecture


14.5  EXERCISES
===============================================================================

Exercise 14.1: Go Binary Dissection
  Build a Go hello-world. Use readelf to analyze every program header
  and section header. Identify: where the Go runtime symbols are,
  where the GC metadata is, where your main.main function is.

Exercise 14.2: LTO Experiment
  Write two C files where a small function in file B is called in a
  tight loop in file A. Benchmark with and without -flto. Disassemble
  both and confirm the function was inlined with LTO.

Exercise 14.3: Minimal JIT
  Write a Go program that:
  (a) Takes a Mini-C expression as a string
  (b) Compiles it to x86-64 machine code in memory
  (c) Maps the memory as executable
  (d) Calls the generated code and prints the result
  Test with: "3 + 4 * 5" -> should produce 23

Exercise 14.4: Final Project
  Submit the complete minicc project including:
  - All compiler phases (lexer, parser, semantic, IR, codegen)
  - ELF object writer
  - Static linker
  - Test suite (at least 20 Mini-C programs)
  - A README documenting architecture, build instructions, and design decisions
  - A 5-page report reflecting on: what was hardest, what you'd do differently,
    and how this knowledge applies to real-world systems


WEEK 13-14 READING
===============================================================================

Required:
  - ELF Specification: Dynamic Linking chapter
  - Ian Lance Taylor "Linkers" Parts 10-20 (dynamic linking)
  - Drepper "How to Write Shared Libraries" (dsohowto.pdf)

Recommended:
  - "Linkers and Loaders" Ch. 9-10 (Levine)
  - Go source: cmd/link (https://github.com/golang/go/tree/master/src/cmd/link)
  - LLVM LTO documentation (llvm.org/docs/LinkTimeOptimization.html)
  - Mike Pall's LuaJIT design notes
  - V8 blog posts on TurboFan
