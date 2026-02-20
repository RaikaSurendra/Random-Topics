===============================================================================
  PART 4 — macOS NATIVE EXECUTABLE PHASE
  Week 8 (Second Half)
===============================================================================

At this point, the compiler emits x86-64 assembly. This section covers
how to turn that assembly into a running macOS executable, and how to
inspect and debug the result using macOS-specific tools.


4.1  macOS ASSEMBLY CONVENTIONS
===============================================================================

macOS uses the Mach-O binary format and has several differences from Linux:

  1. Symbol Prefixing:
     All C-level symbols are prefixed with an underscore.
       C function "main"  → assembly symbol "_main"
       C function "add"   → assembly symbol "_add"
       C variable "count" → assembly symbol "_count"

  2. Section Names:
     ELF:    .text, .data, .bss, .rodata
     Mach-O: __TEXT,__text   __DATA,__data   __DATA,__bss   __TEXT,__cstring

     Our compiler can use the simplified AT&T-syntax directives:
       .text     → maps to __TEXT,__text
       .data     → maps to __DATA,__data
     clang's integrated assembler handles the mapping.

  3. Position-Independent Code (PIC):
     macOS REQUIRES position-independent code for executables since
     macOS 10.7 (Lion). All data access must be RIP-relative.

     WRONG:  movl _counter, %eax        ← absolute addressing
     RIGHT:  movl _counter(%rip), %eax  ← RIP-relative addressing

     Our code generator already emits RIP-relative references for globals.

  4. Stack Alignment:
     The System V AMD64 ABI requires 16-byte stack alignment at the
     point of a CALL instruction. This is the same on macOS and Linux.

  5. System Calls:
     macOS system calls differ from Linux. But since we're using clang
     to link against libSystem, we don't make raw syscalls — we call
     libc functions (printf, exit, etc.) through the C runtime.

  6. No Direct System Call Numbers:
     On Linux you can write: mov $60, %rax; syscall  (exit)
     On macOS the syscall numbers are different and Apple discourages
     direct use — always go through libSystem.


4.2  BUILDING AND RUNNING ON macOS
===============================================================================

Complete workflow from Mini-C source to running program:

  Step 1 — Compile (our compiler):
  $ go run ./cmd/minicc -platform macos testdata/factorial.mc -o /tmp/factorial.s

  Step 2 — Inspect the assembly:
  $ cat /tmp/factorial.s

  Expected output:
      .text
      .globl _factorial
  _factorial:
      pushq   %rbp
      movq    %rsp, %rbp
      subq    $48, %rsp
      movl    %edi, -8(%rbp)
      movl    $1, -16(%rbp)
  .L_while_0:
      movl    -8(%rbp), %eax
      cmpl    $1, %eax
      setg    %al
      movzbl  %al, %eax
      movl    %eax, -24(%rbp)
      movl    -24(%rbp), %eax
      cmpl    $0, %eax
      je      .L_endwhile_1
      movl    -16(%rbp), %eax
      imull   -8(%rbp), %eax
      movl    %eax, -32(%rbp)
      movl    -32(%rbp), %eax
      movl    %eax, -16(%rbp)
      movl    -8(%rbp), %eax
      subl    $1, %eax
      movl    %eax, -40(%rbp)
      movl    -40(%rbp), %eax
      movl    %eax, -8(%rbp)
      jmp     .L_while_0
  .L_endwhile_1:
      movl    -16(%rbp), %eax
      movq    %rbp, %rsp
      popq    %rbp
      retq

      .globl _main
  _main:
      pushq   %rbp
      movq    %rsp, %rbp
      subq    $16, %rsp
      movl    $5, %edi
      callq   _factorial
      movl    %eax, -8(%rbp)
      movl    -8(%rbp), %eax
      movq    %rbp, %rsp
      popq    %rbp
      retq

  Step 3 — Assemble and link with clang:
  $ clang /tmp/factorial.s -o /tmp/factorial -arch x86_64

  Note: On Apple Silicon, we cross-compile to x86_64 using -arch x86_64.
  The binary runs under Rosetta 2 translation.

  Step 4 — Verify the binary:
  $ file /tmp/factorial
  /tmp/factorial: Mach-O 64-bit executable x86_64

  Step 5 — Run and check result:
  $ /tmp/factorial; echo $?
  120

  Step 6 — Using the runtime for output:
  If we want print_int support:

  // runtime.c
  #include <stdio.h>
  extern int mc_main();
  void print_int(int x) { printf("%d\n", x); }
  int main() { return mc_main(); }

  # Compile main as mc_main and link with runtime:
  $ clang /tmp/factorial.s runtime.c -o /tmp/factorial -arch x86_64
  (Adjust symbol: main → mc_main in assembly)


4.3  MACH-O STRUCTURE DEEP DIVE
===============================================================================

Let's inspect our compiled binary:

  $ otool -h /tmp/factorial

  Mach header
        magic  cputype cpusubtype  caps    filetype ncmds sizeofcmds  flags
   0xfeedfacf 16777223          3  0x00           2    16       1368 0x00200085

  Fields:
    magic:      0xFEEDFACF = 64-bit Mach-O (0xFEEDFACE = 32-bit)
    cputype:    0x01000007 = CPU_TYPE_X86_64
    cpusubtype: 3 = CPU_SUBTYPE_ALL
    filetype:   2 = MH_EXECUTE (executable)
    ncmds:      16 = number of load commands
    flags:      MH_PIE (position-independent executable)


  $ otool -l /tmp/factorial | head -60

  Load command 0
        cmd LC_SEGMENT_64
    cmdsize 72
    segname __PAGEZERO
     vmaddr 0x0000000000000000
     vmsize 0x0000000100000000
    fileoff 0
   filesize 0
    maxprot 0x00000000
   initprot 0x00000000
     nsects 0
      flags 0x0

  Load command 1
        cmd LC_SEGMENT_64
    cmdsize 232
    segname __TEXT
     vmaddr 0x0000000100000000
     vmsize 0x0000000000001000
    fileoff 0
   filesize 4096
    maxprot 0x00000005
   initprot 0x00000005
     nsects 2
      flags 0x0
    Section
      sectname __text
       segname __TEXT
          addr 0x0000000100000f50
          size 0x0000000000000065
        offset 3920
         align 2^4 (16)
        reloff 0
        nreloc 0
         flags 0x80000400
     reserved1 0
     reserved2 0

  Key observations:

  __PAGEZERO segment:
    - Virtual size = 4 GB, file size = 0
    - Catches NULL pointer dereferences (any access below 0x100000000 faults)
    - This is why pointers on macOS 64-bit start at 0x100000000

  __TEXT segment:
    - Contains the executable code
    - maxprot = 5 = r-x (read + execute, no write)
    - initprot = 5 = same
    - __text section: our machine code starts at 0x100000f50


  $ otool -tv /tmp/factorial

  /tmp/factorial:
  (__TEXT,__text) section
  _factorial:
  0000000100000f50    pushq   %rbp
  0000000100000f51    movq    %rsp, %rbp
  0000000100000f54    subq    $0x30, %rsp
  ...
  _main:
  0000000100000f96    pushq   %rbp
  0000000100000f97    movq    %rsp, %rbp
  ...

  Now we see FINAL addresses — the linker (ld64) resolved everything.

  $ nm /tmp/factorial
  0000000100000f96 T _main
  0000000100000f50 T _factorial

  Both symbols are in the Text section with final addresses.


4.4  DEBUGGING WITH LLDB
===============================================================================

lldb is the macOS debugger (part of LLVM/Xcode). It is the macOS
equivalent of gdb.

  Compile with debug info:
  $ clang -g /tmp/factorial.s -o /tmp/factorial -arch x86_64

  Start lldb:
  $ lldb /tmp/factorial

  Essential commands:

  (lldb) breakpoint set --name factorial    # Break at function entry
  (lldb) b _factorial                       # Same, shorter form
  (lldb) run                                # Start execution
  (lldb) register read                      # Show all registers
  (lldb) register read rax rbp rsp rdi      # Show specific registers
  (lldb) disassemble --frame                # Disassemble current function
  (lldb) si                                 # Step one instruction
  (lldb) ni                                 # Step over calls
  (lldb) memory read --size 4 --count 8 $rbp-64  # Read stack
  (lldb) x/8xw $rbp-64                     # Same, GDB-style format
  (lldb) bt                                 # Backtrace (show call stack)
  (lldb) frame info                         # Current frame details
  (lldb) continue                           # Resume until next breakpoint

  Debugging session for factorial(5):

  (lldb) b _factorial
  (lldb) run
  Process stopped at _factorial:
  (lldb) register read rdi
       rdi = 0x0000000000000005    ← n = 5 (first argument)

  (lldb) si   # pushq %rbp
  (lldb) si   # movq %rsp, %rbp
  (lldb) si   # subq $48, %rsp
  (lldb) si   # movl %edi, -8(%rbp)

  (lldb) memory read --size 4 --count 1 $rbp-8
  0x...: 05 00 00 00              ← n stored on stack

  (lldb) continue  # Run to completion
  Process exited with status = 120 (0x00000078)

  Using lldb to verify our calling convention:

  Set a breakpoint right before "callq _factorial" in _main:
  (lldb) b _main
  (lldb) run
  (lldb) disassemble --frame
  (lldb) b 0x100000fa3    # Address of callq instruction
  (lldb) continue
  (lldb) register read rdi
       rdi = 5   ← Correct: argument passed in RDI


4.5  COMMON ISSUES AND DEBUGGING
===============================================================================

Issue 1: "dyld: Symbol not found: _main"
  Cause: Your assembly defines _mc_main but the linker expects _main.
  Fix: Either name it _main in assembly, or link with runtime.c.

Issue 2: Segfault on startup
  Cause: Stack misalignment. The CALL instruction pushes 8 bytes (return
  address). If RSP wasn't 16-byte aligned before CALL, it won't be after.
  Fix: Ensure subq in prologue aligns the stack. After pushq %rbp (8 bytes)
  and subq $N, %rsp, the total must be a multiple of 16.

  Debug: (lldb) register read rsp
  Check: RSP should be 0x...0 or 0x...8 at function entry (after CALL)
  and 0x...0 after prologue.

Issue 3: "Illegal instruction"
  Cause: Fell through past retq into garbage bytes.
  Fix: Ensure every code path ends with retq. Our codegen adds a safety
  retq at the end of every function.

Issue 4: Wrong return value
  Debug: (lldb) si through the epilogue and check EAX before retq.
  The return value must be in EAX (lower 32 bits of RAX).

Issue 5: Calling convention mismatch
  Symptom: Function receives wrong argument values.
  Debug: Check that arguments are placed in the correct registers
  (RDI, RSI, RDX, RCX, R8, R9) BEFORE the CALL instruction.


4.6  EXERCISES
===============================================================================

Exercise 8.5: Full Pipeline Test
  Run all 8 sample programs from Part 2 through the complete pipeline
  (compile → assemble → link → run) on macOS. Verify each return code.
  Record any that fail and diagnose the issue.

Exercise 8.6: Mach-O Inspection
  For the factorial binary, use otool to:
  (a) List all load commands
  (b) Find the __TEXT,__text section offset and size
  (c) Find the entry point address
  (d) Disassemble the binary and compare with your .s file

Exercise 8.7: Debugger Walkthrough
  Using lldb, step through factorial(5) instruction by instruction.
  At each step, record: the instruction, RBP, RSP, RAX, and the
  contents of the first 6 stack slots below RBP.

Exercise 8.8: Runtime Integration
  Write a runtime.c that provides print_int. Modify the compiler to
  emit calls to _print_int. Compile and link a program that prints
  "120" to stdout using this mechanism.
