===============================================================================
  PART 5 — DOCKER LINUX ELF DEEP DIVE & STATIC LINKER
  Weeks 9-12
===============================================================================

Starting this week, we switch to Linux inside Docker. All ELF work
happens in the container. Our Go code runs on macOS but cross-compiles
or targets the Docker environment.


DOCKER ENVIRONMENT SETUP
===============================================================================

Create a Dockerfile for our development environment:

```dockerfile
# Dockerfile
FROM --platform=linux/amd64 ubuntu:22.04

RUN apt-get update && apt-get install -y \
    build-essential \
    nasm \
    binutils \
    hexdump \
    xxd \
    strace \
    gdb \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /work
```

Build and run:

```bash
$ docker build -t minicc-linux .
$ docker run --platform linux/amd64 -it -v $(pwd):/work minicc-linux bash
```

From macOS, cross-compile the Go tools for Linux:

```bash
$ GOOS=linux GOARCH=amd64 go build -o minicc-linux ./cmd/minicc
$ GOOS=linux GOARCH=amd64 go build -o elfwriter-linux ./cmd/elfwriter
$ GOOS=linux GOARCH=amd64 go build -o minilinker-linux ./cmd/linker
```

These binaries run inside the Docker container natively (x86-64 via Rosetta 2).


===============================================================================
  WEEK 9: ELF FORMAT — BYTE-BY-BYTE DEEP DIVE
===============================================================================


9.1  THE ELF HEADER — ALL 64 BYTES
===============================================================================

Every ELF file starts with a 64-byte header. Here is every field:

```
  Offset  Size  Field            Value (for our .o files)
  ------  ----  ---------------  ----------------------------------------
  0x00    4     e_ident[0..3]    0x7F 'E' 'L' 'F'   (magic number)
  0x04    1     e_ident[4]       2 = ELFCLASS64      (64-bit)
  0x05    1     e_ident[5]       1 = ELFDATA2LSB     (little-endian)
  0x06    1     e_ident[6]       1 = EV_CURRENT      (ELF version)
  0x07    1     e_ident[7]       0 = ELFOSABI_NONE   (System V)
  0x08    8     e_ident[8..15]   0 (padding)
  0x10    2     e_type           1 = ET_REL (.o) or 2 = ET_EXEC (executable)
  0x12    2     e_machine        0x3E = EM_X86_64
  0x14    4     e_version        1 = EV_CURRENT
  0x18    8     e_entry          Entry point (0 for .o, address for exec)
  0x20    8     e_phoff          Program header table offset (0 for .o)
  0x28    8     e_shoff          Section header table offset
  0x30    4     e_flags          0 (processor-specific flags)
  0x34    2     e_ehsize         64 (size of this header)
  0x36    2     e_phentsize      56 (size of one program header entry)
  0x38    2     e_phnum          0 for .o, N for executable
  0x3A    2     e_shentsize      64 (size of one section header entry)
  0x3C    2     e_shnum          Number of section headers
  0x3E    2     e_shstrndx       Index of section name string table
  ------  ----  ---------------  ----------------------------------------
  Total: 64 bytes
```

Practical verification:

```bash
# Inside Docker container:
$ echo 'int main() { return 42; }' > test.c
$ gcc -c test.c -o test.o
$ readelf -h test.o

ELF Header:
  Magic:   7f 45 4c 46 02 01 01 00 00 00 00 00 00 00 00 00
  Class:                             ELF64
  Data:                              2's complement, little endian
  Version:                           1 (current)
  OS/ABI:                            UNIX - System V
  ABI Version:                       0
  Type:                              REL (Relocatable file)
  Machine:                           Advanced Micro Devices X86-64
  Version:                           0x1
  Entry point address:               0x0
  Start of program headers:          0 (bytes into file)
  Start of section headers:          272 (bytes into file)
  Flags:                             0x0
  Size of this header:               64 (bytes)
  Size of program headers:           0 (bytes)
  Number of program headers:         0
  Size of section headers:           64 (bytes)
  Number of section headers:         12
  Section header string table index: 11
```

Raw hex inspection:

```bash
$ xxd -l 64 test.o
00000000: 7f45 4c46 0201 0100 0000 0000 0000 0000  .ELF............
00000010: 0100 3e00 0100 0000 0000 0000 0000 0000  ..>.............
00000020: 0000 0000 0000 0000 1001 0000 0000 0000  ................
00000030: 0000 0000 4000 0000 0000 4000 0c00 0b00  ....@.....@.....
```

Reading this byte by byte:
  7f 45 4c 46  = magic "\x7fELF"
  02           = ELFCLASS64
  01           = little-endian
  01           = ELF version 1
  00           = ELFOSABI_NONE
  (8 padding bytes)
  01 00        = ET_REL (little-endian: 0x0001)
  3e 00        = EM_X86_64 (0x003E)
  01 00 00 00  = version 1
  (8 zero bytes = entry point 0)
  (8 zero bytes = phoff 0 -- no program headers in .o)
  10 01 00 00 00 00 00 00 = shoff = 0x110 = 272 (section headers at byte 272)
  00 00 00 00  = flags = 0
  40 00        = ehsize = 64
  00 00        = phentsize = 0 (no program headers)
  00 00        = phnum = 0
  40 00        = shentsize = 64
  0c 00        = shnum = 12 sections
  0b 00        = shstrndx = 11 (section name strings in section 11)


9.2  SECTION HEADERS
===============================================================================

Each section header is 64 bytes:

```
  Offset  Size  Field         Description
  ------  ----  -----------   -------------------------------------------
  0x00    4     sh_name       Offset into .shstrtab for section name
  0x04    4     sh_type       Section type (see below)
  0x08    8     sh_flags      Section flags (see below)
  0x10    8     sh_addr       Virtual address (0 in .o files)
  0x18    8     sh_offset     File offset to section data
  0x20    8     sh_size       Section size in bytes
  0x28    4     sh_link       Link to related section (varies by type)
  0x2C    4     sh_info       Extra info (varies by type)
  0x30    8     sh_addralign  Alignment requirement
  0x38    8     sh_entsize    Entry size (for tables like symtab)
  ------  ----  -----------   -------------------------------------------
  Total: 64 bytes per entry
```

Section types we care about:
```
  SHT_NULL      = 0   First entry (always null/empty)
  SHT_PROGBITS  = 1   Code or data (.text, .data, .rodata)
  SHT_SYMTAB    = 2   Symbol table
  SHT_STRTAB    = 3   String table (.strtab, .shstrtab)
  SHT_RELA      = 4   Relocation with addends (.rela.text)
  SHT_NOBITS    = 8   .bss (occupies no file space)
```

Section flags:
```
  SHF_WRITE     = 0x1   Writable at runtime (.data, .bss)
  SHF_ALLOC     = 0x2   Occupies memory at runtime (.text, .data, .bss)
  SHF_EXECINSTR = 0x4   Executable instructions (.text)
```

Viewing section headers:

```bash
$ readelf -S test.o
There are 12 section headers, starting at offset 0x110:

Section Headers:
  [Nr] Name              Type             Address           Offset
       Size              EntSize          Flags  Link  Info  Align
  [ 0]                   NULL             0000000000000000  00000000
       0000000000000000  0000000000000000           0     0     0
  [ 1] .text             PROGBITS         0000000000000000  00000040
       0000000000000015  0000000000000000  AX       0     0     1
  [ 2] .data             PROGBITS         0000000000000000  00000055
       0000000000000000  0000000000000000  WA       0     0     1
  [ 3] .bss              NOBITS           0000000000000000  00000055
       0000000000000000  0000000000000000  WA       0     0     1
  ...
  [ 9] .symtab           SYMTAB           0000000000000000  00000068
       0000000000000090  0000000000000018          10     4     8
  [10] .strtab           STRTAB           0000000000000000  000000f8
       0000000000000008  0000000000000000           0     0     1
  [11] .shstrtab         STRTAB           0000000000000000  00000100
       0000000000000059  0000000000000000           0     0     1
```

Key observations:
  - Section [0] is always NULL (required by spec)
  - .text at offset 0x40 with flags AX (Alloc + eXecutable)
  - .data and .bss both empty for our trivial program
  - .symtab has Link=10 (points to .strtab for symbol names)
  - .symtab has entsize=0x18=24 bytes per symbol entry


9.3  SYMBOL TABLE ENTRIES
===============================================================================

Each symbol table entry (.symtab) is 24 bytes:

```
  Offset  Size  Field         Description
  ------  ----  -----------   -----------------------------------
  0x00    4     st_name       Offset into .strtab
  0x04    1     st_info       Type (4 bits) + Binding (4 bits)
  0x05    1     st_other      Visibility (usually 0)
  0x06    2     st_shndx      Section index (or SHN_UNDEF = 0)
  0x08    8     st_value      Value (offset within section)
  0x10    8     st_size       Symbol size (0 for functions usually)
  ------  ----  -----------   -----------------------------------
  Total: 24 bytes per entry
```

st_info encoding:
```
  st_info = (binding << 4) | type

  Binding:
    STB_LOCAL  = 0    Not visible outside this object file
    STB_GLOBAL = 1    Visible to linker across all files
    STB_WEAK   = 2    Like global but can be overridden

  Type:
    STT_NOTYPE  = 0   Unspecified
    STT_OBJECT  = 1   Data object (variable)
    STT_FUNC    = 2   Function
    STT_SECTION = 3   Section symbol
    STT_FILE    = 4   Source filename
```

Viewing symbols:

```bash
$ readelf -s test.o
Symbol table '.symtab' contains 6 entries:
   Num:    Value          Size Type    Bind   Vis      Ndx Name
     0: 0000000000000000     0 NOTYPE  LOCAL  DEFAULT  UND
     1: 0000000000000000     0 FILE    LOCAL  DEFAULT  ABS test.c
     2: 0000000000000000     0 SECTION LOCAL  DEFAULT    1
     3: 0000000000000000     0 SECTION LOCAL  DEFAULT    2
     4: 0000000000000000     0 SECTION LOCAL  DEFAULT    3
     5: 0000000000000000    21 FUNC    GLOBAL DEFAULT    1 main
```

  Entry 0: Always a null entry (required)
  Entry 1: Source file name
  Entry 2-4: Section symbols (for .text, .data, .bss)
  Entry 5: "main" - a GLOBAL FUNC in section 1 (.text) at offset 0, size 21 bytes

Important rule: In .symtab, ALL LOCAL symbols come before ALL GLOBAL symbols.
The sh_info field of the .symtab section header gives the index of the
first global symbol.


9.4  RELOCATION ENTRIES
===============================================================================

Each relocation entry (.rela.text) is 24 bytes:

```
  Offset  Size  Field       Description
  ------  ----  ----------  ------------------------------------------
  0x00    8     r_offset    Where to apply the relocation (offset in section)
  0x08    8     r_info      Symbol index (32 bits) + type (32 bits)
  0x10    8     r_addend    Constant to add to the computed value
  ------  ----  ----------  ------------------------------------------
  Total: 24 bytes per entry
```

r_info encoding:
```
  symbol_index = r_info >> 32
  reloc_type   = r_info & 0xFFFFFFFF
```

Key relocation types for x86-64:
```
  R_X86_64_NONE    = 0    No relocation
  R_X86_64_64      = 1    Absolute 64-bit:  S + A
  R_X86_64_PC32    = 2    PC-relative 32-bit: S + A - P
  R_X86_64_32      = 10   Absolute 32-bit: S + A
  R_X86_64_32S     = 11   Absolute 32-bit signed: S + A
  R_X86_64_PLT32   = 4    PC-relative to PLT: L + A - P

  Where:
    S = symbol value (final address)
    A = addend (from relocation entry)
    P = place (address of the byte being patched)
    L = PLT entry address
```

Example with a function call:

```bash
$ cat call_test.c
extern int square(int x);
int main() { return square(7); }

$ gcc -c call_test.c
$ readelf -r call_test.o

Relocation section '.rela.text' at offset 0x...:
  Offset          Info           Type           Sym. Value    Sym. Name + Addend
000000000000000a  000500000004 R_X86_64_PLT32    0000000000000000 square - 4
```

This means:
  - At offset 0x0A in .text, there's a CALL instruction
  - The operand is a 32-bit PC-relative value
  - It should resolve to the PLT entry for "square"
  - The addend is -4 (because PC-relative is computed from the END
    of the instruction, but the offset points to the START of the operand;
    the operand is 4 bytes before the end)


9.5  STRING TABLES
===============================================================================

String tables (.strtab, .shstrtab) are simple: a concatenation of
null-terminated strings. References are byte offsets into the table.

```
  Offset  Bytes                     String
  ------  ----------------------    -------
  0x00    00                        "" (null string, index 0)
  0x01    74 65 73 74 2e 63 00      "test.c"
  0x08    6d 61 69 6e 00            "main"
  0x0d    73 71 75 61 72 65 00      "square"
```

When a symbol's st_name = 0x08, the name is "main" (read from offset 8
until the null terminator).

.shstrtab follows the same format but contains section names:
  "\0.text\0.data\0.bss\0.symtab\0.strtab\0.shstrtab\0.rela.text\0"


===============================================================================
  WEEK 10: WRITING ELF OBJECT FILES IN GO
===============================================================================


10.1  ELF TYPES IN GO
===============================================================================

```go
// pkg/elf/types.go

package elf

import "encoding/binary"

// ELF64 Header
type Elf64Header struct {
    Ident     [16]byte // Magic number and other info
    Type      uint16   // Object file type
    Machine   uint16   // Architecture
    Version   uint32   // Object file version
    Entry     uint64   // Entry point virtual address
    PhOff     uint64   // Program header table file offset
    ShOff     uint64   // Section header table file offset
    Flags     uint32   // Processor-specific flags
    EhSize    uint16   // ELF header size in bytes
    PhEntSize uint16   // Program header table entry size
    PhNum     uint16   // Program header table entry count
    ShEntSize uint16   // Section header table entry size
    ShNum     uint16   // Section header table entry count
    ShStrNdx  uint16   // Section header string table index
}

// ELF64 Section Header
type Elf64SectionHeader struct {
    Name      uint32 // Section name (string table index)
    Type      uint32 // Section type
    Flags     uint64 // Section flags
    Addr      uint64 // Section virtual address at execution
    Offset    uint64 // Section file offset
    Size      uint64 // Section size in bytes
    Link      uint32 // Link to another section
    Info      uint32 // Additional section information
    AddrAlign uint64 // Section alignment
    EntSize   uint64 // Entry size if section holds table
}

// ELF64 Symbol Table Entry
type Elf64Sym struct {
    Name  uint32 // Symbol name (string table index)
    Info  uint8  // Symbol type and binding
    Other uint8  // Symbol visibility
    Shndx uint16 // Section index
    Value uint64 // Symbol value
    Size  uint64 // Symbol size
}

// ELF64 Relocation Entry with Addend
type Elf64Rela struct {
    Offset uint64 // Address
    Info   uint64 // Relocation type and symbol index
    Addend int64  // Addend
}

// ELF constants
const (
    // e_ident indices
    EI_MAG0    = 0
    EI_MAG1    = 1
    EI_MAG2    = 2
    EI_MAG3    = 3
    EI_CLASS   = 4
    EI_DATA    = 5
    EI_VERSION = 6
    EI_OSABI   = 7

    // ELF magic
    ELFMAG0 = 0x7f
    ELFMAG1 = 'E'
    ELFMAG2 = 'L'
    ELFMAG3 = 'F'

    // ELF class
    ELFCLASS64 = 2

    // Data encoding
    ELFDATA2LSB = 1 // Little-endian

    // ELF version
    EV_CURRENT = 1

    // Object file types
    ET_REL  = 1 // Relocatable file
    ET_EXEC = 2 // Executable file

    // Machine types
    EM_X86_64 = 0x3E

    // Section types
    SHT_NULL     = 0
    SHT_PROGBITS = 1
    SHT_SYMTAB   = 2
    SHT_STRTAB   = 3
    SHT_RELA     = 4
    SHT_NOBITS   = 8

    // Section flags
    SHF_WRITE     = 0x1
    SHF_ALLOC     = 0x2
    SHF_EXECINSTR = 0x4

    // Symbol binding
    STB_LOCAL  = 0
    STB_GLOBAL = 1

    // Symbol type
    STT_NOTYPE  = 0
    STT_OBJECT  = 1
    STT_FUNC    = 2
    STT_SECTION = 3
    STT_FILE    = 4

    // Special section indices
    SHN_UNDEF = 0
    SHN_ABS   = 0xFFF1

    // Relocation types
    R_X86_64_NONE  = 0
    R_X86_64_64    = 1
    R_X86_64_PC32  = 2
    R_X86_64_PLT32 = 4
    R_X86_64_32    = 10
    R_X86_64_32S   = 11
)

// Helper to encode st_info
func ElfSymInfo(binding, symType uint8) uint8 {
    return (binding << 4) | (symType & 0x0F)
}

// Helper to encode r_info
func ElfRelaInfo(symIdx uint32, relType uint32) uint64 {
    return (uint64(symIdx) << 32) | uint64(relType)
}

// Byte order for ELF (always little-endian for x86-64)
var ByteOrder = binary.LittleEndian
```


10.2  THE ELF OBJECT FILE WRITER
===============================================================================

```go
// pkg/elf/writer.go

package elf

import (
    "bytes"
    "encoding/binary"
    "fmt"
)

// Section represents a section to be written.
type Section struct {
    Name    string
    Type    uint32
    Flags   uint64
    Data    []byte
    Align   uint64
    EntSize uint64
    Link    uint32
    Info    uint32
}

// Symbol represents a symbol to be written.
type Symbol struct {
    Name    string
    Binding uint8
    Type    uint8
    Section int    // Index into our section list (-1 = UNDEF)
    Value   uint64
    Size    uint64
}

// Relocation represents a relocation entry.
type Relocation struct {
    Offset    uint64
    SymbolIdx int    // Index into our symbol list
    Type      uint32
    Addend    int64
}

// ObjectWriter builds an ELF relocatable object file.
type ObjectWriter struct {
    sections    []Section
    symbols     []Symbol
    relocations map[int][]Relocation // section index -> relocations
    strtab      *StringTable
    shstrtab    *StringTable
}

// StringTable builds a null-terminated string table.
type StringTable struct {
    data    bytes.Buffer
    offsets map[string]uint32
}

func NewStringTable() *StringTable {
    st := &StringTable{
        offsets: make(map[string]uint32),
    }
    st.data.WriteByte(0) // First byte is always null
    return st
}

func (st *StringTable) Add(s string) uint32 {
    if off, ok := st.offsets[s]; ok {
        return off
    }
    off := uint32(st.data.Len())
    st.data.WriteString(s)
    st.data.WriteByte(0)
    st.offsets[s] = off
    return off
}

func (st *StringTable) Bytes() []byte {
    return st.data.Bytes()
}

func NewObjectWriter() *ObjectWriter {
    return &ObjectWriter{
        relocations: make(map[int][]Relocation),
        strtab:      NewStringTable(),
        shstrtab:    NewStringTable(),
    }
}

// AddSection adds a section and returns its index.
func (w *ObjectWriter) AddSection(s Section) int {
    idx := len(w.sections)
    w.sections = append(w.sections, s)
    return idx
}

// AddSymbol adds a symbol and returns its index.
func (w *ObjectWriter) AddSymbol(s Symbol) int {
    idx := len(w.symbols)
    w.symbols = append(w.symbols, s)
    return idx
}

// AddRelocation adds a relocation for a section.
func (w *ObjectWriter) AddRelocation(sectionIdx int, r Relocation) {
    w.relocations[sectionIdx] = append(w.relocations[sectionIdx], r)
}

// Write produces the complete ELF object file bytes.
func (w *ObjectWriter) Write() ([]byte, error) {
    var buf bytes.Buffer

    // Phase 1: Prepare string tables
    // Add section names to shstrtab
    w.shstrtab.Add("")       // Null section name
    for _, s := range w.sections {
        w.shstrtab.Add(s.Name)
    }
    w.shstrtab.Add(".symtab")
    w.shstrtab.Add(".strtab")
    w.shstrtab.Add(".shstrtab")

    // Add relocation section names
    for secIdx := range w.relocations {
        w.shstrtab.Add(".rela" + w.sections[secIdx].Name)
    }

    // Add symbol names to strtab
    w.strtab.Add("") // Null symbol name
    for _, sym := range w.symbols {
        if sym.Name != "" {
            w.strtab.Add(sym.Name)
        }
    }

    // Phase 2: Calculate layout
    //
    // Layout:
    //   [ELF Header]           64 bytes
    //   [Section data]         variable
    //   [Section headers]      64 * numSections bytes

    ehdrSize := uint64(64)
    offset := ehdrSize

    // Build complete section list:
    // [0] = NULL
    // [1..N] = user sections
    // [N+1..M] = .rela sections
    // [M+1] = .symtab
    // [M+2] = .strtab
    // [M+3] = .shstrtab

    type layoutEntry struct {
        shdr Elf64SectionHeader
        data []byte
    }

    var layout []layoutEntry

    // Section 0: NULL
    layout = append(layout, layoutEntry{
        shdr: Elf64SectionHeader{},
        data: nil,
    })

    // User sections
    userSectionMap := make(map[int]int) // original index -> layout index
    for i, s := range w.sections {
        align := s.Align
        if align == 0 {
            align = 1
        }
        // Align offset
        if offset%align != 0 {
            offset += align - (offset % align)
        }

        nameOff := w.shstrtab.Add(s.Name)
        shdr := Elf64SectionHeader{
            Name:      nameOff,
            Type:      s.Type,
            Flags:     s.Flags,
            Addr:      0,
            Offset:    offset,
            Size:      uint64(len(s.Data)),
            Link:      s.Link,
            Info:      s.Info,
            AddrAlign: align,
            EntSize:   s.EntSize,
        }

        layoutIdx := len(layout)
        userSectionMap[i] = layoutIdx
        layout = append(layout, layoutEntry{shdr: shdr, data: s.Data})
        offset += uint64(len(s.Data))
    }

    // Build symbol table data
    // Sort: locals first, then globals (ELF requirement)
    var localSyms, globalSyms []Symbol
    var localIdxMap, globalIdxMap []int // original index -> new position
    localIdxMap = make([]int, len(w.symbols))
    globalIdxMap = make([]int, len(w.symbols))

    for i, sym := range w.symbols {
        if sym.Binding == STB_LOCAL {
            localIdxMap[i] = len(localSyms)
            localSyms = append(localSyms, sym)
        }
    }
    firstGlobal := len(localSyms) + 1 // +1 for null symbol
    for i, sym := range w.symbols {
        if sym.Binding == STB_GLOBAL {
            globalIdxMap[i] = len(globalSyms) + firstGlobal
            globalSyms = append(globalSyms, sym)
        }
    }

    // Build symtab bytes
    var symtabBuf bytes.Buffer
    // Null symbol entry
    binary.Write(&symtabBuf, ByteOrder, Elf64Sym{})
    // Local symbols
    for _, sym := range localSyms {
        shndx := uint16(SHN_UNDEF)
        if sym.Section >= 0 {
            shndx = uint16(userSectionMap[sym.Section])
        }
        entry := Elf64Sym{
            Name:  w.strtab.Add(sym.Name),
            Info:  ElfSymInfo(sym.Binding, sym.Type),
            Other: 0,
            Shndx: shndx,
            Value: sym.Value,
            Size:  sym.Size,
        }
        binary.Write(&symtabBuf, ByteOrder, entry)
    }
    // Global symbols
    for _, sym := range globalSyms {
        shndx := uint16(SHN_UNDEF)
        if sym.Section >= 0 {
            shndx = uint16(userSectionMap[sym.Section])
        }
        entry := Elf64Sym{
            Name:  w.strtab.Add(sym.Name),
            Info:  ElfSymInfo(sym.Binding, sym.Type),
            Other: 0,
            Shndx: shndx,
            Value: sym.Value,
            Size:  sym.Size,
        }
        binary.Write(&symtabBuf, ByteOrder, entry)
    }
    symtabData := symtabBuf.Bytes()

    // Build relocation section data and map symbol indices
    symIdxRemap := func(origIdx int) uint32 {
        sym := w.symbols[origIdx]
        if sym.Binding == STB_LOCAL {
            return uint32(localIdxMap[origIdx] + 1) // +1 for null
        }
        return uint32(globalIdxMap[origIdx])
    }

    type relaSection struct {
        targetSecIdx int
        data         []byte
    }
    var relaSections []relaSection

    for secIdx, relocs := range w.relocations {
        var relaBuf bytes.Buffer
        for _, r := range relocs {
            entry := Elf64Rela{
                Offset: r.Offset,
                Info:   ElfRelaInfo(symIdxRemap(r.SymbolIdx), r.Type),
                Addend: r.Addend,
            }
            binary.Write(&relaBuf, ByteOrder, entry)
        }
        relaSections = append(relaSections, relaSection{
            targetSecIdx: secIdx,
            data:         relaBuf.Bytes(),
        })
    }

    // Emit rela sections
    // (symtab index will be determined after we add it)
    symtabLayoutIdx := 0 // filled in below
    for _, rs := range relaSections {
        if offset%8 != 0 {
            offset += 8 - (offset % 8)
        }
        nameOff := w.shstrtab.Add(".rela" + w.sections[rs.targetSecIdx].Name)
        shdr := Elf64SectionHeader{
            Name:      nameOff,
            Type:      SHT_RELA,
            Flags:     0,
            Offset:    offset,
            Size:      uint64(len(rs.data)),
            Link:      0, // will be patched to symtab index
            Info:      uint32(userSectionMap[rs.targetSecIdx]),
            AddrAlign: 8,
            EntSize:   24, // sizeof(Elf64Rela)
        }
        layout = append(layout, layoutEntry{shdr: shdr, data: rs.data})
        offset += uint64(len(rs.data))
    }

    // .symtab section
    if offset%8 != 0 {
        offset += 8 - (offset % 8)
    }
    symtabLayoutIdx = len(layout)
    strtabLayoutIdx := symtabLayoutIdx + 1
    shstrtabLayoutIdx := symtabLayoutIdx + 2

    symtabNameOff := w.shstrtab.Add(".symtab")
    layout = append(layout, layoutEntry{
        shdr: Elf64SectionHeader{
            Name:      symtabNameOff,
            Type:      SHT_SYMTAB,
            Flags:     0,
            Offset:    offset,
            Size:      uint64(len(symtabData)),
            Link:      uint32(strtabLayoutIdx),
            Info:      uint32(firstGlobal),
            AddrAlign: 8,
            EntSize:   24,
        },
        data: symtabData,
    })
    offset += uint64(len(symtabData))

    // Patch rela sections to point to symtab
    for i := range layout {
        if layout[i].shdr.Type == SHT_RELA {
            layout[i].shdr.Link = uint32(symtabLayoutIdx)
        }
    }

    // .strtab section
    strtabData := w.strtab.Bytes()
    strtabNameOff := w.shstrtab.Add(".strtab")
    layout = append(layout, layoutEntry{
        shdr: Elf64SectionHeader{
            Name:      strtabNameOff,
            Type:      SHT_STRTAB,
            Flags:     0,
            Offset:    offset,
            Size:      uint64(len(strtabData)),
            AddrAlign: 1,
        },
        data: strtabData,
    })
    offset += uint64(len(strtabData))

    // .shstrtab section
    shstrtabData := w.shstrtab.Bytes()
    shstrtabNameOff := w.shstrtab.Add(".shstrtab")
    layout = append(layout, layoutEntry{
        shdr: Elf64SectionHeader{
            Name:      shstrtabNameOff,
            Type:      SHT_STRTAB,
            Flags:     0,
            Offset:    offset,
            Size:      uint64(len(shstrtabData)),
            AddrAlign: 1,
        },
        data: shstrtabData,
    })
    offset += uint64(len(shstrtabData))

    // Section header table
    if offset%8 != 0 {
        offset += 8 - (offset % 8)
    }
    shdrOffset := offset

    // Phase 3: Write everything
    // ELF Header
    ehdr := Elf64Header{
        Type:      ET_REL,
        Machine:   EM_X86_64,
        Version:   EV_CURRENT,
        Entry:     0,
        PhOff:     0,
        ShOff:     shdrOffset,
        Flags:     0,
        EhSize:    64,
        PhEntSize: 0,
        PhNum:     0,
        ShEntSize: 64,
        ShNum:     uint16(len(layout)),
        ShStrNdx:  uint16(shstrtabLayoutIdx),
    }
    ehdr.Ident[EI_MAG0] = ELFMAG0
    ehdr.Ident[EI_MAG1] = ELFMAG1
    ehdr.Ident[EI_MAG2] = ELFMAG2
    ehdr.Ident[EI_MAG3] = ELFMAG3
    ehdr.Ident[EI_CLASS] = ELFCLASS64
    ehdr.Ident[EI_DATA] = ELFDATA2LSB
    ehdr.Ident[EI_VERSION] = EV_CURRENT

    binary.Write(&buf, ByteOrder, ehdr)

    // Pad and write section data
    for _, entry := range layout {
        if entry.data == nil {
            continue
        }
        // Pad to section offset
        for uint64(buf.Len()) < entry.shdr.Offset {
            buf.WriteByte(0)
        }
        buf.Write(entry.data)
    }

    // Pad to section header table
    for uint64(buf.Len()) < shdrOffset {
        buf.WriteByte(0)
    }

    // Write section headers
    for _, entry := range layout {
        binary.Write(&buf, ByteOrder, entry.shdr)
    }

    return buf.Bytes(), nil
}
```


10.3  USING THE ELF WRITER
===============================================================================

Example: writing an object file for a function that returns 42.

```go
func main() {
    w := elf.NewObjectWriter()

    // Machine code for: mov $42, %eax; ret
    textCode := []byte{
        0xB8, 0x2A, 0x00, 0x00, 0x00, // mov $42, %eax
        0xC3,                           // ret
    }

    textIdx := w.AddSection(elf.Section{
        Name:  ".text",
        Type:  elf.SHT_PROGBITS,
        Flags: elf.SHF_ALLOC | elf.SHF_EXECINSTR,
        Data:  textCode,
        Align: 16,
    })

    w.AddSymbol(elf.Symbol{
        Name:    "main",
        Binding: elf.STB_GLOBAL,
        Type:    elf.STT_FUNC,
        Section: textIdx,
        Value:   0,
        Size:    uint64(len(textCode)),
    })

    data, err := w.Write()
    if err != nil {
        log.Fatal(err)
    }
    os.WriteFile("test.o", data, 0644)
}
```

Validation inside Docker:

```bash
$ readelf -h test.o       # Should show valid ELF header
$ readelf -S test.o       # Should show .text, .symtab, .strtab, .shstrtab
$ readelf -s test.o       # Should show 'main' as GLOBAL FUNC
$ objdump -d test.o       # Should disassemble: mov $0x2a,%eax; ret
$ gcc test.o -o test -nostartfiles -e main
$ ./test; echo $?         # Should print 42
```


10.4  EXERCISES
===============================================================================

Exercise 10.1: Hand-Written ELF
  Using xxd or a hex editor, create a minimal valid ELF relocatable
  file by hand (just the header + one empty .text section + section
  headers). Verify with readelf -h and readelf -S.

Exercise 10.2: Two-Symbol Object
  Use the ELF writer to create an object file with two functions:
  "add" (returns the sum of its two arguments) and "main" (calls add).
  Include the proper relocation entry for the call. Verify with readelf -r.

Exercise 10.3: Global Data
  Add a .data section containing a global variable initialized to 100.
  Add the symbol. Verify with readelf -s and objdump -s (section contents).

Exercise 10.4: Integration Test
  Modify the compiler's code generator to emit raw x86-64 machine code
  bytes (instead of assembly text). Feed the bytes directly to the ELF
  writer. Produce a .o file and verify with objdump -d.


===============================================================================
  WEEK 11: STATIC LINKER — READING AND SYMBOL RESOLUTION
===============================================================================


11.1  WHAT THE LINKER MUST DO
===============================================================================

The static linker takes multiple .o files and produces one executable.

```
  Input:                              Output:

  main.o                              executable
  +--------+                          +--------------+
  | .text  | (main code)              | ELF Header   |
  | .data  | (main globals)      +--> | Prog Headers |
  | .symtab| (main, square:UNDEF)|    | .text        | (merged)
  +--------+                     |    | .data        | (merged)
                                 |    | .bss         | (merged)
  math.o                        |    +--------------+
  +--------+                     |
  | .text  | (square code)       |
  | .symtab| (square:GLOBAL) ----+
  +--------+

  Steps:
  1. Read all input object files
  2. Parse their ELF headers, sections, symbols, relocations
  3. Merge sections (all .text -> one .text, all .data -> one .data)
  4. Build global symbol table (resolve undefined references)
  5. Calculate final addresses for all sections and symbols
  6. Apply relocations (patch machine code with final addresses)
  7. Write executable ELF with program headers
```


11.2  ELF READER IN GO
===============================================================================

```go
// pkg/linker/reader.go

package linker

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "github.com/user/minicc/pkg/elf"
)

// ObjectFile represents a parsed ELF relocatable object file.
type ObjectFile struct {
    Name     string
    Header   elf.Elf64Header
    Sections []ParsedSection
    Symbols  []ParsedSymbol
}

type ParsedSection struct {
    Header elf.Elf64SectionHeader
    Name   string
    Data   []byte
    Relocs []ParsedReloc
}

type ParsedSymbol struct {
    Name    string
    Binding uint8
    Type    uint8
    Section int   // section index in this object (-1 for UNDEF)
    Value   uint64
    Size    uint64
}

type ParsedReloc struct {
    Offset    uint64
    SymbolIdx int    // index into ObjectFile.Symbols
    Type      uint32
    Addend    int64
}

func ReadObject(name string, data []byte) (*ObjectFile, error) {
    r := bytes.NewReader(data)
    obj := &ObjectFile{Name: name}

    // Read ELF header
    if err := binary.Read(r, elf.ByteOrder, &obj.Header); err != nil {
        return nil, fmt.Errorf("reading ELF header: %w", err)
    }

    // Validate magic
    if obj.Header.Ident[0] != 0x7F ||
        obj.Header.Ident[1] != 'E' ||
        obj.Header.Ident[2] != 'L' ||
        obj.Header.Ident[3] != 'F' {
        return nil, fmt.Errorf("%s: not an ELF file", name)
    }
    if obj.Header.Type != elf.ET_REL {
        return nil, fmt.Errorf("%s: not a relocatable file (type=%d)", name, obj.Header.Type)
    }

    // Read section headers
    shdrs := make([]elf.Elf64SectionHeader, obj.Header.ShNum)
    r.Seek(int64(obj.Header.ShOff), 0)
    for i := range shdrs {
        binary.Read(r, elf.ByteOrder, &shdrs[i])
    }

    // Read section name string table
    shstrtabHdr := shdrs[obj.Header.ShStrNdx]
    shstrtab := make([]byte, shstrtabHdr.Size)
    r.Seek(int64(shstrtabHdr.Offset), 0)
    r.Read(shstrtab)

    // Parse sections
    obj.Sections = make([]ParsedSection, len(shdrs))
    for i, shdr := range shdrs {
        obj.Sections[i].Header = shdr
        obj.Sections[i].Name = readString(shstrtab, shdr.Name)

        if shdr.Type != elf.SHT_NOBITS && shdr.Size > 0 {
            obj.Sections[i].Data = make([]byte, shdr.Size)
            r.Seek(int64(shdr.Offset), 0)
            r.Read(obj.Sections[i].Data)
        }
    }

    // Find and parse symbol table
    for i, shdr := range shdrs {
        if shdr.Type == elf.SHT_SYMTAB {
            // Read associated string table
            strtabData := obj.Sections[shdr.Link].Data

            // Parse symbol entries
            numSyms := shdr.Size / shdr.EntSize
            symReader := bytes.NewReader(obj.Sections[i].Data)
            obj.Symbols = make([]ParsedSymbol, numSyms)

            for j := uint64(0); j < numSyms; j++ {
                var sym elf.Elf64Sym
                binary.Read(symReader, elf.ByteOrder, &sym)

                secIdx := -1
                if sym.Shndx != elf.SHN_UNDEF && sym.Shndx < 0xFF00 {
                    secIdx = int(sym.Shndx)
                }

                obj.Symbols[j] = ParsedSymbol{
                    Name:    readString(strtabData, sym.Name),
                    Binding: sym.Info >> 4,
                    Type:    sym.Info & 0x0F,
                    Section: secIdx,
                    Value:   sym.Value,
                    Size:    sym.Size,
                }
            }
            break
        }
    }

    // Parse relocation sections
    for i, shdr := range shdrs {
        if shdr.Type == elf.SHT_RELA {
            targetSec := shdr.Info // section this rela applies to
            numRelocs := shdr.Size / shdr.EntSize
            relaReader := bytes.NewReader(obj.Sections[i].Data)

            for j := uint64(0); j < numRelocs; j++ {
                var rela elf.Elf64Rela
                binary.Read(relaReader, elf.ByteOrder, &rela)

                symIdx := rela.Info >> 32
                relType := uint32(rela.Info & 0xFFFFFFFF)

                obj.Sections[targetSec].Relocs = append(
                    obj.Sections[targetSec].Relocs,
                    ParsedReloc{
                        Offset:    rela.Offset,
                        SymbolIdx: int(symIdx),
                        Type:      relType,
                        Addend:    rela.Addend,
                    },
                )
            }
        }
    }

    return obj, nil
}

func readString(strtab []byte, offset uint32) string {
    start := offset
    for i := offset; i < uint32(len(strtab)); i++ {
        if strtab[i] == 0 {
            return string(strtab[start:i])
        }
    }
    return ""
}
```


11.3  SYMBOL RESOLUTION
===============================================================================

Symbol resolution is the process of matching every undefined symbol reference
with exactly one definition.

Algorithm:

```
  globalSymbols = {}   // name -> (objectFile, symbolEntry)

  for each object file:
      for each symbol:
          if symbol is GLOBAL and DEFINED:
              if name already in globalSymbols:
                  ERROR: duplicate symbol
              else:
                  globalSymbols[name] = (this file, this symbol)

  for each object file:
      for each symbol:
          if symbol is UNDEFINED:
              if name not in globalSymbols:
                  ERROR: undefined reference to "name"
```

```go
// pkg/linker/symbol.go

package linker

import "fmt"

// GlobalSymbol tracks a resolved symbol.
type GlobalSymbol struct {
    Name       string
    ObjFile    *ObjectFile
    SymIdx     int     // index into ObjFile.Symbols
    FinalAddr  uint64  // computed during layout
}

// ResolveSymbols performs symbol resolution across all object files.
func ResolveSymbols(objects []*ObjectFile) (map[string]*GlobalSymbol, error) {
    globals := make(map[string]*GlobalSymbol)

    // Pass 1: Collect all global definitions
    for _, obj := range objects {
        for i, sym := range obj.Symbols {
            if sym.Binding != STB_GLOBAL {
                continue
            }
            if sym.Section < 0 {
                continue // undefined - skip for now
            }
            // Defined global symbol
            if existing, ok := globals[sym.Name]; ok {
                return nil, fmt.Errorf(
                    "duplicate symbol '%s': defined in %s and %s",
                    sym.Name, existing.ObjFile.Name, obj.Name,
                )
            }
            globals[sym.Name] = &GlobalSymbol{
                Name:    sym.Name,
                ObjFile: obj,
                SymIdx:  i,
            }
        }
    }

    // Pass 2: Verify all undefined references can be resolved
    for _, obj := range objects {
        for _, sym := range obj.Symbols {
            if sym.Binding != STB_GLOBAL {
                continue
            }
            if sym.Section >= 0 {
                continue // defined - skip
            }
            // Undefined global symbol - must be in globals map
            if _, ok := globals[sym.Name]; !ok {
                return nil, fmt.Errorf(
                    "undefined reference to '%s' in %s",
                    sym.Name, obj.Name,
                )
            }
        }
    }

    return globals, nil
}
```


11.4  EXERCISES
===============================================================================

Exercise 11.1: Read and Dump
  Write a Go program that reads an ELF .o file (produced by gcc -c)
  and prints: header fields, section names and sizes, symbol table,
  and relocation entries. Compare output with readelf.

Exercise 11.2: Symbol Resolution Test
  Create three .c files with cross-references. Compile to .o files.
  Feed them to your ELF reader + symbol resolver. Verify it correctly
  matches all undefined references to definitions.

Exercise 11.3: Error Cases
  Test your resolver with: (a) duplicate definitions, (b) undefined
  references, (c) a symbol defined in one file and used in two others.


===============================================================================
  WEEK 12: STATIC LINKER — RELOCATION AND EXECUTABLE OUTPUT
===============================================================================


12.1  SECTION MERGING AND LAYOUT
===============================================================================

The linker merges same-named sections from all input files:

```
  main.o .text:  [main code, 48 bytes]     --> merged .text offset 0
  math.o .text:  [square code, 24 bytes]   --> merged .text offset 48

  main.o .data:  [counter, 4 bytes]        --> merged .data offset 0
  math.o .data:  [table, 16 bytes]         --> merged .data offset 4
```

Then it assigns virtual addresses:

```
  Typical Linux executable layout:

  Virtual Address    Contents
  -----------------  ------------------
  0x400000           ELF Header + Program Headers
  0x401000           .text segment (r-x)
  0x402000           .rodata (r--)
  0x403000           .data segment (rw-)
  0x404000           .bss (rw-, zeroed)
```

```go
// pkg/linker/linker.go

package linker

import (
    "github.com/user/minicc/pkg/elf"
)

const (
    BaseAddr    = 0x400000
    TextStart   = 0x401000
    DataStart   = 0x402000
    PageSize    = 0x1000
)

type MergedSection struct {
    Name    string
    Type    uint32
    Flags   uint64
    Data    []byte
    Addr    uint64   // Final virtual address
    Pieces  []SectionPiece
}

type SectionPiece struct {
    ObjFile    *ObjectFile
    SecIdx     int      // section index in the original object
    MergeOff   uint64   // offset within the merged section
    OrigSize   uint64
}

// MergeSections combines same-named sections from all objects.
func MergeSections(objects []*ObjectFile) []*MergedSection {
    mergeMap := make(map[string]*MergedSection)
    var mergeOrder []string

    for _, obj := range objects {
        for secIdx, sec := range obj.Sections {
            // Only merge PROGBITS and NOBITS sections
            if sec.Header.Type != elf.SHT_PROGBITS &&
                sec.Header.Type != elf.SHT_NOBITS {
                continue
            }
            if sec.Name == "" {
                continue
            }

            merged, ok := mergeMap[sec.Name]
            if !ok {
                merged = &MergedSection{
                    Name:  sec.Name,
                    Type:  sec.Header.Type,
                    Flags: sec.Header.Flags,
                }
                mergeMap[sec.Name] = merged
                mergeOrder = append(mergeOrder, sec.Name)
            }

            // Align the piece
            align := sec.Header.AddrAlign
            if align == 0 {
                align = 1
            }
            currentSize := uint64(len(merged.Data))
            if currentSize%align != 0 {
                padding := align - (currentSize % align)
                merged.Data = append(merged.Data, make([]byte, padding)...)
                currentSize += padding
            }

            merged.Pieces = append(merged.Pieces, SectionPiece{
                ObjFile:  obj,
                SecIdx:   secIdx,
                MergeOff: currentSize,
                OrigSize: uint64(len(sec.Data)),
            })

            merged.Data = append(merged.Data, sec.Data...)
        }
    }

    var result []*MergedSection
    for _, name := range mergeOrder {
        result = append(result, mergeMap[name])
    }
    return result
}
```


12.2  APPLYING RELOCATIONS
===============================================================================

After we know the final address of every symbol and section, we patch
the machine code.

```go
// pkg/linker/relocation.go

package linker

import (
    "encoding/binary"
    "fmt"
    "github.com/user/minicc/pkg/elf"
)

// ApplyRelocations patches machine code with final addresses.
func ApplyRelocations(
    objects []*ObjectFile,
    merged []*MergedSection,
    globals map[string]*GlobalSymbol,
    sectionAddr map[string]uint64,
) error {
    // Build a helper to find the merged offset for a section piece
    pieceOffset := func(obj *ObjectFile, secIdx int) (uint64, string) {
        for _, ms := range merged {
            for _, piece := range ms.Pieces {
                if piece.ObjFile == obj && piece.SecIdx == secIdx {
                    return ms.Addr + piece.MergeOff, ms.Name
                }
            }
        }
        return 0, ""
    }

    for _, obj := range objects {
        for secIdx, sec := range obj.Sections {
            if len(sec.Relocs) == 0 {
                continue
            }

            // Find this section's merged data
            var mergedSec *MergedSection
            var pieceOff uint64
            for _, ms := range merged {
                for _, piece := range ms.Pieces {
                    if piece.ObjFile == obj && piece.SecIdx == secIdx {
                        mergedSec = ms
                        pieceOff = piece.MergeOff
                        break
                    }
                }
            }
            if mergedSec == nil {
                continue
            }

            for _, reloc := range sec.Relocs {
                // Resolve the target symbol's final address
                sym := obj.Symbols[reloc.SymbolIdx]
                var symAddr uint64

                if sym.Section >= 0 {
                    // Local/defined symbol: base of its section + value
                    baseAddr, _ := pieceOffset(obj, sym.Section)
                    symAddr = baseAddr + sym.Value
                } else if sym.Binding == STB_GLOBAL {
                    // Global undefined: look up in resolved globals
                    gs, ok := globals[sym.Name]
                    if !ok {
                        return fmt.Errorf("unresolved symbol: %s", sym.Name)
                    }
                    symAddr = gs.FinalAddr
                } else {
                    return fmt.Errorf("cannot resolve symbol: %s", sym.Name)
                }

                // P = address of the byte being patched
                P := mergedSec.Addr + pieceOff + reloc.Offset
                S := symAddr
                A := reloc.Addend

                // Offset into merged data
                dataOff := pieceOff + reloc.Offset

                switch reloc.Type {
                case elf.R_X86_64_PC32, elf.R_X86_64_PLT32:
                    // S + A - P, truncated to 32 bits
                    val := int32(int64(S) + A - int64(P))
                    binary.LittleEndian.PutUint32(
                        mergedSec.Data[dataOff:], uint32(val))

                case elf.R_X86_64_32S:
                    // S + A, truncated to signed 32 bits
                    val := int32(int64(S) + A)
                    binary.LittleEndian.PutUint32(
                        mergedSec.Data[dataOff:], uint32(val))

                case elf.R_X86_64_64:
                    // S + A, full 64 bits
                    val := uint64(int64(S) + A)
                    binary.LittleEndian.PutUint64(
                        mergedSec.Data[dataOff:], val)

                default:
                    return fmt.Errorf(
                        "unsupported relocation type %d", reloc.Type)
                }
            }
        }
    }

    return nil
}
```


12.3  WRITING THE EXECUTABLE ELF
===============================================================================

The executable needs:
  1. An ELF header with e_type = ET_EXEC and a valid e_entry
  2. Program headers (segments) that tell the OS loader what to map
  3. The merged section data
  4. Section headers (optional but helpful for debugging)

```
  Executable Layout:

  +---------------------------+  offset 0
  | ELF Header (64 bytes)     |
  +---------------------------+
  | Program Header 1 (LOAD)   |  .text segment: r-x
  | Program Header 2 (LOAD)   |  .data segment: rw-
  +---------------------------+
  | .text data                |  at file offset aligned to page
  +---------------------------+
  | .data data                |  at next page-aligned offset
  +---------------------------+
  | Section Headers (optional)|
  +---------------------------+
```

```go
// pkg/linker/output.go (simplified)

func WriteExecutable(
    merged []*MergedSection,
    entryAddr uint64,
) ([]byte, error) {
    var buf bytes.Buffer

    // We create two LOAD segments:
    //   Segment 1: .text (read + execute)
    //   Segment 2: .data (read + write)

    var textSec, dataSec *MergedSection
    for _, ms := range merged {
        switch ms.Name {
        case ".text":
            textSec = ms
        case ".data":
            dataSec = ms
        }
    }

    phdrCount := 0
    if textSec != nil { phdrCount++ }
    if dataSec != nil { phdrCount++ }

    ehdrSize := uint64(64)
    phdrSize := uint64(56) // size of one Elf64_Phdr
    phdrsSize := phdrSize * uint64(phdrCount)
    headerSize := ehdrSize + phdrsSize

    // Align sections to page boundaries
    textFileOff := align(headerSize, PageSize)
    textVAddr := TextStart
    textSize := uint64(0)
    if textSec != nil {
        textSize = uint64(len(textSec.Data))
        textSec.Addr = uint64(textVAddr)
    }

    dataFileOff := align(textFileOff + textSize, PageSize)
    dataVAddr := align(uint64(textVAddr) + textSize, PageSize)
    dataSize := uint64(0)
    if dataSec != nil {
        dataSize = uint64(len(dataSec.Data))
        dataSec.Addr = dataVAddr
    }

    // ELF Header
    ehdr := elf.Elf64Header{
        Type:      elf.ET_EXEC,
        Machine:   elf.EM_X86_64,
        Version:   elf.EV_CURRENT,
        Entry:     entryAddr,
        PhOff:     ehdrSize,
        ShOff:     0, // no section headers for simplicity
        EhSize:    64,
        PhEntSize: 56,
        PhNum:     uint16(phdrCount),
        ShEntSize: 64,
        ShNum:     0,
        ShStrNdx:  0,
    }
    ehdr.Ident[0] = 0x7F
    ehdr.Ident[1] = 'E'
    ehdr.Ident[2] = 'L'
    ehdr.Ident[3] = 'F'
    ehdr.Ident[4] = elf.ELFCLASS64
    ehdr.Ident[5] = elf.ELFDATA2LSB
    ehdr.Ident[6] = elf.EV_CURRENT

    binary.Write(&buf, elf.ByteOrder, ehdr)

    // Program Headers
    if textSec != nil {
        phdr := Elf64Phdr{
            Type:   PT_LOAD,
            Flags:  PF_R | PF_X,
            Offset: textFileOff,
            VAddr:  uint64(textVAddr),
            PAddr:  uint64(textVAddr),
            FileSz: textSize,
            MemSz:  textSize,
            Align:  PageSize,
        }
        binary.Write(&buf, elf.ByteOrder, phdr)
    }
    if dataSec != nil {
        phdr := Elf64Phdr{
            Type:   PT_LOAD,
            Flags:  PF_R | PF_W,
            Offset: dataFileOff,
            VAddr:  dataVAddr,
            PAddr:  dataVAddr,
            FileSz: dataSize,
            MemSz:  dataSize,
            Align:  PageSize,
        }
        binary.Write(&buf, elf.ByteOrder, phdr)
    }

    // Pad to .text offset and write .text data
    for uint64(buf.Len()) < textFileOff {
        buf.WriteByte(0)
    }
    if textSec != nil {
        buf.Write(textSec.Data)
    }

    // Pad to .data offset and write .data data
    for uint64(buf.Len()) < dataFileOff {
        buf.WriteByte(0)
    }
    if dataSec != nil {
        buf.Write(dataSec.Data)
    }

    return buf.Bytes(), nil
}

// Program header constants
const (
    PT_LOAD = 1
    PF_X    = 0x1
    PF_W    = 0x2
    PF_R    = 0x4
)

// Elf64 Program Header
type Elf64Phdr struct {
    Type   uint32
    Flags  uint32
    Offset uint64
    VAddr  uint64
    PAddr  uint64
    FileSz uint64
    MemSz  uint64
    Align  uint64
}

func align(val, alignment uint64) uint64 {
    if val%alignment == 0 {
        return val
    }
    return val + alignment - (val % alignment)
}
```


12.4  END-TO-END TEST
===============================================================================

Complete workflow inside Docker:

```bash
# Step 1: Compile Mini-C sources to assembly (our compiler, macOS)
$ ./minicc testdata/factorial.mc -platform linux -o /tmp/factorial.s

# Step 2: Assemble with gcc (inside Docker)
$ gcc -c /tmp/factorial.s -o /tmp/factorial.o

# Step 3: Verify object file
$ readelf -h /tmp/factorial.o
$ readelf -s /tmp/factorial.o
$ readelf -r /tmp/factorial.o

# Step 4: Link with our linker (inside Docker)
$ ./minilinker-linux /tmp/factorial.o -o /tmp/factorial

# Step 5: Verify executable
$ readelf -h /tmp/factorial         # Type should be EXEC
$ readelf -l /tmp/factorial         # Should show LOAD segments
$ objdump -d /tmp/factorial         # Disassemble

# Step 6: Run!
$ chmod +x /tmp/factorial
$ /tmp/factorial; echo $?
120
```

Alternative: use our ELF writer to skip the assembler entirely:

```bash
# Compiler produces machine code bytes -> ELF writer -> .o file
$ ./minicc testdata/factorial.mc -emit-obj -o /tmp/factorial.o

# Link
$ ./minilinker-linux /tmp/factorial.o -o /tmp/factorial
$ /tmp/factorial; echo $?
120
```


12.5  COMMON FAILURE CASES
===============================================================================

Failure 1: "Segmentation fault" on startup
  Cause: Entry point address is wrong. The ELF header's e_entry must
  point to the first instruction of main().
  Debug: readelf -h program | grep "Entry point"

Failure 2: "No such file or directory" (but file exists)
  Cause: Missing PT_INTERP program header for dynamically linked executables.
  For our static linker, ensure we use -nostdlib or statically link.
  Debug: file program (should say "statically linked")

Failure 3: "Bus error"
  Cause: Section data not aligned to the alignment specified in the program
  header. Ensure Align field in Elf64Phdr matches the actual file alignment.

Failure 4: Relocation overflow
  Cause: A 32-bit PC-relative relocation can't reach the target because
  sections are placed too far apart in memory. Keep .text small or use
  a base address that keeps everything within 2 GB of each other.

Failure 5: Wrong instruction encoding after relocation
  Debug: objdump -d program and check the bytes at the relocation offset.
  Compare with the expected computation: S + A - P for R_X86_64_PC32.


12.6  EXERCISES
===============================================================================

Exercise 12.1: Multi-File Link
  Compile three Mini-C files to .o objects. Link them with your linker.
  Run the executable.

Exercise 12.2: Relocation Trace
  Add verbose logging to your relocation code. For each relocation,
  print: section, offset, type, symbol, S, A, P, computed value.
  Compare with objdump -dr output.

Exercise 12.3: Entry Point
  The linker must find the symbol "main" (or "_start") to set the
  entry point. What happens if the symbol is missing? Implement a
  clear error message.

Exercise 12.4: Static Library Support (Stretch)
  Implement reading .a (ar archive) files. An archive is a simple
  format: a header followed by concatenated .o files. Parse the
  archive, extract the objects, and link them.


WEEKS 9-12 READING
===============================================================================

Required:
  - ELF Specification (full read of sections 1-5)
  - System V AMD64 ABI Supplement (relocation types)
  - Ian Lance Taylor "Linkers" Parts 1-10

Recommended:
  - Levine "Linkers and Loaders" Chapters 1-7
  - "Computer Systems: A Programmer's Perspective" Ch. 7
  - mold linker source: https://github.com/rui314/mold (for inspiration)
