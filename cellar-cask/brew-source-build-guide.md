# Homebrew: Building from Source — A Complete Guide

> **Formula used throughout:** `jq` (lightweight JSON processor)
> **Works on:** macOS (Apple Silicon + Intel) and Linux (Homebrew/Linuxbrew)
> **Audience:** Developers who want to understand what happens under the hood

---

## Platform Quick Reference

Before we begin — Homebrew works on three platforms. This guide covers all of them. Where behavior differs, you'll see callout blocks like this:

> **Platform note:** Platform-specific information appears here.

| Detail | macOS (Apple Silicon) | macOS (Intel) | Linux |
|---|---|---|---|
| **Homebrew prefix** | `/opt/homebrew` | `/usr/local` | `/home/linuxbrew/.linuxbrew` |
| **Architecture** | `arm64` | `x86_64` | `x86_64` or `aarch64` |
| **Shared library ext** | `.dylib` | `.dylib` | `.so` |
| **Binary format** | Mach-O arm64 | Mach-O x86_64 | ELF 64-bit |
| **Inspect linked libs** | `otool -L` | `otool -L` | `ldd` |
| **CPU core count** | `sysctl -n hw.ncpu` | `sysctl -n hw.ncpu` | `nproc` |
| **Compiler prereqs** | `xcode-select --install` | `xcode-select --install` | `build-essential` (Debian/Ubuntu) or `gcc` (Fedora/RHEL) |
| **System C library** | `libSystem.B.dylib` | `libSystem.B.dylib` | `libc.so.6` (glibc) |
| **Default compiler** | Apple Clang | Apple Clang | GCC (typically) |

Throughout this guide, `$(brew --prefix)` is used as the portable way to reference your Homebrew prefix. On your system, run `brew --prefix` to see the actual path.

---

## Table of Contents

- [Chapter 1: Homebrew Architecture Recap](#chapter-1-homebrew-architecture-recap)
- [Chapter 2: Anatomy of a Formula](#chapter-2-anatomy-of-a-formula)
- [Chapter 3: The Dependency Chain](#chapter-3-the-dependency-chain)
- [Chapter 4: The Build Pipeline — Step by Step](#chapter-4-the-build-pipeline--step-by-step)
- [Chapter 5: Inside the Keg — What Gets Produced](#chapter-5-inside-the-keg--what-gets-produced)
- [Chapter 6: Linking — Connecting the Keg to Your System](#chapter-6-linking--connecting-the-keg-to-your-system)
- [Chapter 7: DIY — Building It Yourself Without Homebrew](#chapter-7-diy--building-it-yourself-without-homebrew)
- [Chapter 8: Modifying the Source](#chapter-8-modifying-the-source)
- [Chapter 9: Bottles vs Source — What's the Difference?](#chapter-9-bottles-vs-source--whats-the-difference)
- [Chapter 10: Creating Your Own Formula](#chapter-10-creating-your-own-formula)
- [Appendix A: Key Homebrew Commands Reference](#appendix-a-key-homebrew-commands-reference)
- [Appendix B: Glossary](#appendix-b-glossary)

---

## Chapter 1: Homebrew Architecture Recap

Before we build anything, let's ground ourselves in how Homebrew organizes things on disk.

### The Directory Layout

```
$(brew --prefix)/                       # prefix — Homebrew's root
├── bin/                                # symlinks to binaries from linked kegs
├── lib/                                # symlinks to libraries from linked kegs
├── include/                            # symlinks to headers from linked kegs
├── Cellar/                             # where ALL kegs live (the warehouse)
│   ├── bash/5.3.8/                     # one keg = one version of one formula
│   ├── curl/8.17.0/                    # another keg
│   ├── python@3.14/3.14.2/            # versioned formula keg
│   └── ...
├── opt/                                # stable symlinks to latest keg versions
│   ├── jq -> ../Cellar/jq/1.8.1       # always points to the active version
│   └── ...
├── Caskroom/                           # where GUI app casks live (macOS only)
└── Library/
    └── Taps/
        └── homebrew/homebrew-core/     # formula repository (Ruby files)
            └── Formula/
                ├── j/jq.rb            # <-- this is what we'll dissect
                └── ...
```

> **Platform note — prefix paths:**
> - macOS Apple Silicon: `/opt/homebrew`
> - macOS Intel: `/usr/local`
> - Linux: `/home/linuxbrew/.linuxbrew`
>
> Run `brew --prefix` to confirm yours. This guide uses `$(brew --prefix)` as a portable placeholder.

### Discovering Your Own Paths

```bash
brew --prefix          # Homebrew root (e.g., /opt/homebrew)
brew --cellar          # Cellar path  (e.g., /opt/homebrew/Cellar)
brew --cache           # Download cache (e.g., ~/Library/Caches/Homebrew on macOS)
brew --repository      # Git repo location
```

### Key Concepts

| Term       | What It Is                                                   | Example (use `brew --prefix` for your path) |
|------------|--------------------------------------------------------------|----------------------------------------------|
| **Prefix** | Homebrew's root directory                                    | `$(brew --prefix)`                            |
| **Cellar** | Parent directory holding all kegs                            | `$(brew --prefix)/Cellar`                     |
| **Keg**    | One installed version of one formula — isolated, self-contained | `$(brew --prefix)/Cellar/bash/5.3.8/`        |
| **Rack**   | All versions of one formula                                  | `$(brew --prefix)/Cellar/bash/` (may have 5.3.7, 5.3.8) |
| **Tap**    | A Git repository of formula files                            | `homebrew/homebrew-core`                      |
| **Formula**| A Ruby script describing how to download, build, and install | `Formula/j/jq.rb`                            |
| **Bottle** | A pre-compiled binary tarball (skips the build step)         | Downloaded from `ghcr.io/v2/homebrew/core`   |

### The Two Installation Paths

```
Formula → Bottle path (default):
  Download pre-compiled tarball → Extract into Cellar → Link

Formula → Source path (--build-from-source):
  Download source tarball → Extract → Configure → Compile → Install into Cellar → Link
```

This guide focuses entirely on the **source path**.

---

## Chapter 2: Anatomy of a Formula

A formula is a Ruby class that inherits from `Formula`. Here is the **complete, real** `jq` formula from `homebrew-core`:

```ruby
class Jq < Formula
  desc "Lightweight and flexible command-line JSON processor"
  homepage "https://jqlang.github.io/jq/"
  url "https://github.com/jqlang/jq/releases/download/jq-1.8.1/jq-1.8.1.tar.gz"
  sha256 "2be64e7129cecb11d5906290eba10af694fb9e3e7f9fc208a311dc33ca837eb0"
  license "MIT"

  livecheck do
    url :stable
    regex(/^(?:jq[._-])?v?(\d+(?:\.\d+)+)$/i)
  end

  bottle do
    sha256 cellar: :any,                 arm64_tahoe:   "90b0fe4ad..."
    sha256 cellar: :any,                 arm64_sequoia: "d7bce557b..."
    sha256 cellar: :any,                 arm64_sonoma:  "147e51295..."
    sha256 cellar: :any,                 arm64_ventura: "efd141679..."
    sha256 cellar: :any,                 sonoma:        "a1a5f487f..."
    sha256 cellar: :any,                 ventura:       "1b5303b05..."
    sha256 cellar: :any_skip_relocation, arm64_linux:   "274b39102..."
    sha256 cellar: :any_skip_relocation, x86_64_linux:  "82883e1f3..."
  end

  head do
    url "https://github.com/jqlang/jq.git", branch: "master"

    depends_on "autoconf" => :build
    depends_on "automake" => :build
    depends_on "libtool" => :build
  end

  depends_on "oniguruma"

  def install
    system "autoreconf", "--force", "--install", "--verbose" if build.head?
    system "./configure", *std_configure_args,
                          "--disable-silent-rules",
                          "--disable-maintainer-mode"
    system "make", "install"
  end

  test do
    assert_equal "2\n", pipe_output("#{bin}/jq .bar", '{"foo":1, "bar":2}')
  end
end
```

### Section-by-Section Breakdown

#### 2.1 Metadata Block

```ruby
desc "Lightweight and flexible command-line JSON processor"
homepage "https://jqlang.github.io/jq/"
url "https://github.com/jqlang/jq/releases/download/jq-1.8.1/jq-1.8.1.tar.gz"
sha256 "2be64e7129cecb11d5906290eba10af694fb9e3e7f9fc208a311dc33ca837eb0"
license "MIT"
```

| Field      | Purpose                                                         |
|------------|-----------------------------------------------------------------|
| `desc`     | Human-readable description (shown in `brew info`)               |
| `homepage` | Project website                                                 |
| `url`      | **The source tarball URL** — this is what gets downloaded       |
| `sha256`   | Checksum to verify the download wasn't tampered with            |
| `license`  | SPDX license identifier                                        |

**Key insight:** The `url` + `sha256` pair is the anchor of the entire build. Everything starts by downloading this file and verifying its integrity.

#### 2.2 Livecheck Block

```ruby
livecheck do
  url :stable
  regex(/^(?:jq[._-])?v?(\d+(?:\.\d+)+)$/i)
end
```

This tells Homebrew's automated systems how to detect new upstream releases. It scrapes the stable URL and matches version strings with the regex. This is how `brew outdated` knows a newer version exists. **Not used during builds.**

#### 2.3 Bottle Block

```ruby
bottle do
  sha256 cellar: :any, arm64_tahoe: "90b0fe4ad..."
  # ... more platforms
end
```

Pre-compiled binary checksums, one per supported platform. When you run `brew install jq` (without `--build-from-source`), Homebrew downloads one of these bottles instead of compiling. Each bottle is a compressed tarball of the keg directory.

- `cellar: :any` — the bottle can be installed in any Cellar path
- `cellar: :any_skip_relocation` — no path patching needed (common on Linux)

Notice the bottle block includes both macOS (`arm64_sequoia`, `sonoma`, `ventura`) and Linux (`arm64_linux`, `x86_64_linux`) platforms.

**When building from source, this entire block is ignored.**

#### 2.4 Head Block

```ruby
head do
  url "https://github.com/jqlang/jq.git", branch: "master"
  depends_on "autoconf" => :build
  depends_on "automake" => :build
  depends_on "libtool" => :build
end
```

Activated by `brew install --HEAD jq`. Instead of downloading the release tarball, it clones the Git repo's `master` branch. Since the repo won't have a pre-generated `./configure` script, it needs `autoconf`, `automake`, and `libtool` to generate one (`autoreconf`).

#### 2.5 Dependencies

```ruby
depends_on "oniguruma"
```

jq's only runtime dependency. `oniguruma` is a regular expression library — jq uses it for regex support in filters.

Dependency types:
- `depends_on "foo"` — needed at both build-time and runtime
- `depends_on "foo" => :build` — only needed to compile (not linked at runtime)
- `depends_on "foo" => :optional` — user can opt-in with `--with-foo`
- `uses_from_macos "zlib"` — use the macOS system copy; install from source on Linux

> **Platform note:** `uses_from_macos` is how formulas handle libraries that ship with macOS but not Linux. On macOS, the system copy is used. On Linux, Homebrew installs it as a regular dependency.

#### 2.6 Install Method (The Build Instructions)

```ruby
def install
  system "autoreconf", "--force", "--install", "--verbose" if build.head?
  system "./configure", *std_configure_args,
                        "--disable-silent-rules",
                        "--disable-maintainer-mode"
  system "make", "install"
end
```

This is the heart of the formula. Three commands:

1. **`autoreconf`** (HEAD builds only) — regenerates `./configure` from `configure.ac`
2. **`./configure`** — probes the system, generates `Makefile` from `Makefile.in`
3. **`make install`** — compiles the source and copies binaries/libs to the prefix

**What is `std_configure_args`?** It expands to:
```
--disable-debug
--disable-dependency-tracking
--prefix=$(brew --cellar)/jq/1.8.1
--libdir=$(brew --cellar)/jq/1.8.1/lib
```

The `--prefix` is critical. It tells `./configure` to install everything into the **keg directory** inside the Cellar, not into system-wide paths.

#### 2.7 Test Block

```ruby
test do
  assert_equal "2\n", pipe_output("#{bin}/jq .bar", '{"foo":1, "bar":2}')
end
```

Run via `brew test jq`. Pipes JSON into `jq` and verifies the output. This runs **after** installation to confirm the built binary actually works.

---

## Chapter 3: The Dependency Chain

Before jq can be built, all its dependencies must be satisfied. Let's trace the full chain.

### 3.1 jq's Dependency Tree

```
jq (1.8.1)
└── oniguruma (6.9.10) [runtime]
    ├── autoconf [build-only]
    ├── automake [build-only]
    └── libtool  [build-only]
```

- **oniguruma** must be compiled and installed first
- `autoconf`, `automake`, `libtool` are only needed to compile oniguruma (not jq itself, unless using `--HEAD`)

### 3.2 The oniguruma Formula

For completeness, here's what jq depends on:

```ruby
class Oniguruma < Formula
  desc "Regular expressions library"
  homepage "https://github.com/kkos/oniguruma/"
  url "https://github.com/kkos/oniguruma/releases/download/v6.9.10/onig-6.9.10.tar.gz"
  sha256 "2a5cfc5ae259e4e97f86b68dfffc152cdaffe94e2060b770cb827238d769fc05"
  license "BSD-2-Clause"

  depends_on "autoconf" => :build
  depends_on "automake" => :build
  depends_on "libtool" => :build

  def install
    system "autoreconf", "--force", "--install", "--verbose"
    system "./configure", "--disable-dependency-tracking", "--prefix=#{prefix}"
    system "make"
    system "make", "install"
  end

  test do
    assert_match(/#{prefix}/, shell_output("#{bin}/onig-config --prefix"))
  end
end
```

Notice the same pattern: `autoreconf` → `./configure` → `make` → `make install`. This is the classic **GNU Autotools build system** — the most common pattern you'll see in C/C++ formulas.

### 3.3 How Homebrew Resolves Dependencies

When you run `brew install --build-from-source jq`, Homebrew:

```
1. Parse jq formula
2. Discover dependency: oniguruma
3. Check if oniguruma is already installed → if not:
   a. Parse oniguruma formula
   b. Discover its build deps: autoconf, automake, libtool
   c. Check if each build dep is installed → install if missing
   d. Build and install oniguruma from source
4. Now build jq (oniguruma available as a linked library)
```

The resolution is **recursive and depth-first**. Every dependency is resolved before the dependent formula is built.

---

## Chapter 4: The Build Pipeline — Step by Step

Here's exactly what happens when you run:

```bash
brew install --build-from-source jq
```

### Stage 1: Fetch

```
Action:  Download the source tarball
From:    https://github.com/jqlang/jq/releases/download/jq-1.8.1/jq-1.8.1.tar.gz
To:      $(brew --cache)/downloads/<hash>--jq-1.8.1.tar.gz
```

Homebrew downloads the tarball specified in the formula's `url` field. The filename in the cache includes a hash of the URL for deduplication.

> **Platform note — cache locations:**
> - macOS: `~/Library/Caches/Homebrew/downloads/`
> - Linux: `~/.cache/Homebrew/downloads/`
>
> Run `brew --cache` to see yours.

**Integrity check:** After downloading, Homebrew computes the SHA-256 hash and compares it to the `sha256` in the formula:

```
Expected: 2be64e7129cecb11d5906290eba10af694fb9e3e7f9fc208a311dc33ca837eb0
Computed: (hash of downloaded file)
```

If they don't match, the build aborts. This prevents tampered or corrupted downloads from being compiled.

### Stage 2: Extract

```
Action:  Unpack the tarball into a temporary build directory
To:      /tmp/d20260209-XXXXX-jq/jq-1.8.1/
```

The extracted directory contains the project's source code:

```
jq-1.8.1/
├── configure              # autoconf-generated configure script (shell script)
├── configure.ac           # autoconf source (generates configure)
├── Makefile.in            # template that becomes Makefile after configure
├── Makefile.am            # automake source (generates Makefile.in)
├── src/
│   ├── main.c             # jq CLI entry point
│   ├── jv.c               # JSON value implementation
│   ├── jv.h               # JSON value header
│   ├── compile.c          # jq filter compiler
│   ├── execute.c          # jq filter execution engine
│   ├── parser.y           # jq filter parser (yacc/bison grammar)
│   ├── lexer.l            # jq filter lexer (flex)
│   └── builtin.c          # built-in jq functions
├── jq.1                   # man page
├── tests/
│   └── jq.test            # test suite
├── COPYING                # MIT license
└── README.md
```

**This is the source code.** These are the C files that will be compiled into the `jq` binary.

### Stage 3: Configure

```
Action:  Probe the system and generate Makefiles

Command run by the formula:
  ./configure --disable-debug \
              --disable-dependency-tracking \
              --prefix=$(brew --cellar)/jq/1.8.1 \
              --libdir=$(brew --cellar)/jq/1.8.1/lib \
              --disable-silent-rules \
              --disable-maintainer-mode
```

The `./configure` script is a massive shell script (often 10,000+ lines) generated by GNU Autoconf. It:

1. **Detects your compiler:** Finds `cc` / `clang` / `gcc` and tests that it works
2. **Checks for required headers:** `<stdio.h>`, `<stdlib.h>`, etc.
3. **Locates dependencies:** Finds `oniguruma`'s headers and library
4. **Tests system capabilities:** Checks for functions like `mmap`, `strlcpy`
5. **Generates output:**
   - `Makefile` (from `Makefile.in`) — the actual build instructions
   - `config.h` — C header with `#define`s for detected features
   - `config.status` — a script to re-run the configuration

The `--prefix` flag is the most important argument. It tells configure:
> "When `make install` runs, put everything under `$(brew --cellar)/jq/1.8.1/`"

This is how Homebrew keeps each formula version isolated in its own keg.

#### What configure output looks like (abbreviated):

```
checking for a BSD-compatible install... /usr/bin/install -c
checking whether build environment is sane... yes
checking for strip... strip
checking for a C compiler... (clang on macOS, gcc on Linux)
checking whether the C compiler works... yes
checking for oniguruma... yes
checking oniguruma.h usability... yes
...
config.status: creating Makefile
config.status: creating src/Makefile
config.status: creating config.h
```

### Stage 4: Compile

```
Action:  Compile source files into object files, then link into binaries

Command run by the formula:
  make install
  (which internally does make first, then installs)
```

What `make` does internally (using `cc` which resolves to `clang` on macOS or `gcc` on Linux):

```bash
# 1. Compile each .c file into an object file (.o)
cc -O2 -I./src -I$(brew --prefix)/opt/oniguruma/include \
      -c src/main.c -o src/main.o

cc -O2 -I./src -I$(brew --prefix)/opt/oniguruma/include \
      -c src/jv.c -o src/jv.o

cc -O2 -I./src -c src/compile.c -o src/compile.o
cc -O2 -I./src -c src/execute.c -o src/execute.o
cc -O2 -I./src -c src/builtin.c -o src/builtin.o
# ... (more .c files)

# 2. Generate parser from grammar
bison -o src/parser.c src/parser.y
flex -o src/lexer.c src/lexer.l

# 3. Compile generated parser/lexer
cc -O2 -c src/parser.c -o src/parser.o
cc -O2 -c src/lexer.c -o src/lexer.o

# 4. Link everything into the final binary
cc -o jq src/main.o src/jv.o src/compile.o src/execute.o \
      src/builtin.o src/parser.o src/lexer.o \
      -L$(brew --prefix)/opt/oniguruma/lib -lonig
```

Key observations:
- `-I$(brew --prefix)/opt/oniguruma/include` — find oniguruma's header files
- `-L$(brew --prefix)/opt/oniguruma/lib -lonig` — link against oniguruma's library
- `-O2` — optimization level 2 (standard for release builds)

The `-j` flag (e.g., `make -j8`) parallelizes compilation across CPU cores. Homebrew sets this automatically.

### Stage 5: Install into the Keg

```
Action:  Copy compiled artifacts into the Cellar prefix

The make install target copies:
  jq binary             →  $(brew --cellar)/jq/1.8.1/bin/jq
  shared library        →  $(brew --cellar)/jq/1.8.1/lib/libjq.1.<ext>
  jq.h, jv.h            →  $(brew --cellar)/jq/1.8.1/include/
  jq.1 man page         →  $(brew --cellar)/jq/1.8.1/share/man/man1/jq.1
  pkgconfig             →  $(brew --cellar)/jq/1.8.1/lib/pkgconfig/libjq.pc
```

> **Platform note — shared library extension:**
> - macOS: `libjq.1.dylib`
> - Linux: `libjq.so.1`

The resulting keg:

```
$(brew --cellar)/jq/1.8.1/               # THE KEG
├── bin/
│   └── jq                                # the binary you run
├── include/
│   ├── jq.h                              # C header (for programs linking against libjq)
│   └── jv.h
├── lib/
│   ├── libjq.1.dylib (macOS)             # shared library
│   │   or libjq.so.1 (Linux)
│   ├── libjq.dylib or libjq.so           # symlink to current version
│   ├── libjq.a                            # static library
│   └── pkgconfig/
│       └── libjq.pc                       # metadata for pkg-config
├── share/
│   └── man/man1/
│       └── jq.1                           # man page
├── .brew/
│   └── jq.rb                             # snapshot of the formula used to build
├── INSTALL_RECEIPT.json                   # build metadata (compiler, deps, options)
├── COPYING                                # license file
└── sbom.spdx.json                         # software bill of materials
```

### Stage 6: Metadata Recording

Homebrew writes two important files:

**INSTALL_RECEIPT.json** — Records everything about how this keg was produced:

```json
{
  "homebrew_version": "5.x.x",
  "used_options": [],
  "built_as_bottle": false,
  "poured_from_bottle": false,
  "loaded_from_api": true,
  "compiler": "clang",
  "runtime_dependencies": [
    {
      "full_name": "oniguruma",
      "version": "6.9.10",
      "pkg_version": "6.9.10"
    }
  ],
  "source": {
    "spec": "stable",
    "versions": { "stable": "1.8.1" }
  }
}
```

Note: `poured_from_bottle: false` — this confirms it was built from source, not from a bottle.

> **Platform note:** The `compiler` field will show `clang` on macOS and typically `gcc` on Linux.

**.brew/jq.rb** — A snapshot of the exact formula that was used. This is preserved so Homebrew can reference the build instructions later (for `brew postinstall`, `brew test`, etc.).

---

## Chapter 5: Inside the Keg — What Gets Produced

Let's examine each piece of the keg in detail.

### 5.1 Binaries (`bin/`)

```
bin/jq — the executable
```

This is the compiled binary. You can inspect it with platform-appropriate tools:

```bash
# Check binary type
file $(brew --prefix)/Cellar/jq/1.8.1/bin/jq

# macOS output: Mach-O 64-bit executable arm64  (or x86_64 on Intel)
# Linux output: ELF 64-bit LSB pie executable, x86-64  (or aarch64)
```

```bash
# Inspect linked libraries
# macOS:
otool -L $(brew --prefix)/Cellar/jq/1.8.1/bin/jq

# Linux:
ldd $(brew --prefix)/Cellar/jq/1.8.1/bin/jq
```

**macOS output:**
```
jq:
  $(brew --prefix)/opt/oniguruma/lib/libonig.5.dylib   ← oniguruma (Homebrew)
  /usr/lib/libSystem.B.dylib                            ← macOS system library
```

**Linux output:**
```
  libonig.so.5 => $(brew --prefix)/opt/oniguruma/lib/libonig.so.5   ← oniguruma
  libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6                      ← glibc
  libm.so.6 => /lib/x86_64-linux-gnu/libm.so.6                      ← math library
```

### 5.2 Libraries (`lib/`)

```
macOS:                              Linux:
lib/libjq.1.dylib                   lib/libjq.so.1.0.0
lib/libjq.dylib → libjq.1.dylib    lib/libjq.so.1 → libjq.so.1.0.0
lib/libjq.a                         lib/libjq.so → libjq.so.1
lib/pkgconfig/libjq.pc              lib/libjq.a
                                    lib/pkgconfig/libjq.pc
```

**Shared library (`.dylib` on macOS / `.so` on Linux):** Other programs can link against this to use jq's filtering engine as a library, without the CLI.

**Static library (`libjq.a`):** Same code, but compiled into the linking program at build time (no runtime dependency). This file is the same format on both platforms.

**pkgconfig file (`libjq.pc`):** Used by `pkg-config` to find compiler and linker flags:

```
prefix=$(brew --cellar)/jq/1.8.1
libdir=${prefix}/lib
includedir=${prefix}/include

Name: libjq
Description: jq JSON processor library
Version: 1.8.1
Libs: -L${libdir} -ljq
Cflags: -I${includedir}
```

### 5.3 Headers (`include/`)

```
include/jq.h   — public API for libjq
include/jv.h   — JSON value type definitions
```

These are C header files. Any program that wants to use libjq includes them:

```c
#include <jq.h>
#include <jv.h>
```

Header files are platform-independent — they're the same on macOS and Linux.

### 5.4 Man Pages (`share/man/`)

```
share/man/man1/jq.1 — the jq manual page
```

After linking, `man jq` shows this page. Man pages work the same on all platforms.

### 5.5 INSTALL_RECEIPT.json

The full build receipt (covered in Chapter 4, Stage 6). This is how `brew info --json` can report on installed formulas, and how `brew upgrade` knows when a rebuild is needed.

---

## Chapter 6: Linking — Connecting the Keg to Your System

The keg sits isolated in the Cellar. To make `jq` available system-wide, Homebrew creates **symlinks** from the prefix directories into the keg.

### 6.1 How Linking Works

```bash
brew link jq
```

This creates:

```
$(brew --prefix)/bin/jq              →  ../Cellar/jq/1.8.1/bin/jq
$(brew --prefix)/lib/libjq.<ext>     →  ../Cellar/jq/1.8.1/lib/libjq.<ext>
$(brew --prefix)/include/jq.h        →  ../Cellar/jq/1.8.1/include/jq.h
$(brew --prefix)/share/man/man1/jq.1 →  ../Cellar/jq/1.8.1/share/man/man1/jq.1
```

Since `$(brew --prefix)/bin` is in your `$PATH`, you can now just type `jq` from anywhere.

### 6.2 Symlink Examples

```bash
# See symlinks on your own system:
ls -la $(brew --prefix)/bin/ | head -20

# Typical output (paths will match YOUR prefix):
# bash    →  ../Cellar/bash/5.3.8/bin/bash
# python3 →  ../Cellar/python@3.14/3.14.2/bin/python3
```

### 6.3 `keg_only` Formulas

Some formulas are marked `keg_only`:

```ruby
keg_only :provided_by_macos   # macOS ships its own version
keg_only "conflicts with X"   # would conflict with another formula
```

These are **NOT linked** into `$(brew --prefix)/bin` because they would shadow a system or conflicting version.

> **Platform note:** `:provided_by_macos` is only relevant on macOS. On Linux, these formulas are typically linked normally since there's no macOS system version to conflict with.

To use a `keg_only` formula, reference it explicitly:

```bash
# Direct path
$(brew --prefix)/opt/curl/bin/curl --version

# Or via brew --prefix (portable)
$(brew --prefix curl)/bin/curl --version

# Or add to PATH temporarily
export PATH="$(brew --prefix curl)/bin:$PATH"
```

### 6.4 The `opt` Directory — Stable References

```
$(brew --prefix)/opt/jq → ../Cellar/jq/1.8.1
```

The `opt` prefix always points to the currently active version. If you upgrade jq to 1.9.0, the symlink updates:

```
$(brew --prefix)/opt/jq → ../Cellar/jq/1.9.0
```

This is why formulas reference dependencies via `opt_prefix`:

```ruby
-I#{Formula["oniguruma"].opt_include}   # → $(brew --prefix)/opt/oniguruma/include
-L#{Formula["oniguruma"].opt_lib}       # → $(brew --prefix)/opt/oniguruma/lib
```

It avoids hardcoding version numbers.

---

## Chapter 7: DIY — Building It Yourself Without Homebrew

Now that you understand what Homebrew does, here's how to do it manually. This gives you full control — you can modify the source, change compile flags, add debug symbols, etc.

### 7.1 Prerequisites

You need a C compiler and basic build tools.

**macOS:**
```bash
# Install Xcode command-line tools (if not already present)
xcode-select --install

# Verify
cc --version      # Apple clang version ...
make --version
```

**Linux (Debian/Ubuntu):**
```bash
sudo apt update
sudo apt install build-essential autoconf automake libtool curl

# Verify
gcc --version
make --version
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf groupinstall "Development Tools"
sudo dnf install autoconf automake libtool curl

# Verify
gcc --version
make --version
```

### 7.2 Build oniguruma First (jq's dependency)

```bash
# Create a local prefix to install into (keeps your system clean)
mkdir -p ~/local

# Download oniguruma source (same URL from the formula)
curl -LO https://github.com/kkos/oniguruma/releases/download/v6.9.10/onig-6.9.10.tar.gz

# Verify checksum (same sha256 from the formula)
echo "2a5cfc5ae259e4e97f86b68dfffc152cdaffe94e2060b770cb827238d769fc05  onig-6.9.10.tar.gz" | shasum -a 256 -c
# Expected output: onig-6.9.10.tar.gz: OK
# Note: on Linux, use 'sha256sum -c' if 'shasum' is not available

# Extract
tar xf onig-6.9.10.tar.gz
cd onig-6.9.10

# Generate configure (oniguruma requires this step)
autoreconf --force --install --verbose

# Configure — install into YOUR local directory
./configure --prefix=$HOME/local --disable-dependency-tracking

# Compile (use all CPU cores)
#   macOS: make -j$(sysctl -n hw.ncpu)
#   Linux: make -j$(nproc)
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)

# Install
make install

# Verify
ls ~/local/lib/libonig*
# macOS: libonig.5.dylib, libonig.dylib, libonig.a, etc.
# Linux: libonig.so.5, libonig.so, libonig.a, etc.

cd ..
```

### 7.3 Build jq

```bash
# Download jq source (same URL from the formula)
curl -LO https://github.com/jqlang/jq/releases/download/jq-1.8.1/jq-1.8.1.tar.gz

# Verify checksum
echo "2be64e7129cecb11d5906290eba10af694fb9e3e7f9fc208a311dc33ca837eb0  jq-1.8.1.tar.gz" | shasum -a 256 -c
# Note: on Linux, use 'sha256sum -c' if 'shasum' is not available

# Extract
tar xf jq-1.8.1.tar.gz
cd jq-1.8.1

# Configure — point it to your local prefix AND tell it where oniguruma lives
./configure --prefix=$HOME/local \
            --disable-silent-rules \
            --disable-maintainer-mode \
            CFLAGS="-I$HOME/local/include" \
            LDFLAGS="-L$HOME/local/lib"

# Compile
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)

# Install
make install

cd ..
```

### 7.4 Verify Your Build

```bash
# Run it
~/local/bin/jq --version
# Expected: jq-1.8.1

# Test it
echo '{"name":"homebrew","type":"package manager"}' | ~/local/bin/jq '.name'
# Expected: "homebrew"

# Check what it links against
#   macOS: otool -L ~/local/bin/jq
#   Linux: ldd ~/local/bin/jq
# Should show: oniguruma from ~/local/lib/ and the system C library

# Check the binary type
file ~/local/bin/jq
#   macOS Apple Silicon: Mach-O 64-bit executable arm64
#   macOS Intel:         Mach-O 64-bit executable x86_64
#   Linux x86_64:        ELF 64-bit LSB pie executable, x86-64
#   Linux aarch64:       ELF 64-bit LSB pie executable, ARM aarch64
```

### 7.5 What You Built

```
~/local/
├── bin/
│   ├── jq                    # the jq binary
│   └── onig-config           # oniguruma config tool
├── include/
│   ├── jq.h
│   ├── jv.h
│   └── oniguruma.h
├── lib/
│   ├── libjq.dylib / libjq.so         # shared library (platform-dependent extension)
│   ├── libjq.a                         # static library
│   ├── libonig.dylib / libonig.so      # oniguruma shared library
│   ├── libonig.a                       # oniguruma static library
│   └── pkgconfig/
│       ├── libjq.pc
│       └── oniguruma.pc
└── share/
    └── man/man1/
        └── jq.1
```

This is structurally identical to what Homebrew puts in the Cellar — you just control the prefix.

### 7.6 Cleanup

```bash
# Remove build artifacts when you're done
rm -rf onig-6.9.10 onig-6.9.10.tar.gz jq-1.8.1 jq-1.8.1.tar.gz

# To uninstall your local build entirely:
rm -rf ~/local
```

---

## Chapter 8: Modifying the Source

This is where building from source becomes powerful. You can change anything before compiling.

### 8.1 Example: Add a Custom Built-in Function

After extracting `jq-1.8.1.tar.gz`, before running `make`:

```bash
cd jq-1.8.1

# Look at the source structure
ls src/
# main.c  jv.c  jv.h  compile.c  execute.c  builtin.c  parser.y  lexer.l ...

# View the main entry point
head -30 src/main.c
```

### 8.2 Example: Add Debug Output

```bash
# Edit src/main.c to add a debug banner
# (use any editor — vim, nano, VS Code, etc.)

# For example, add after the includes:
#   fprintf(stderr, "[DEBUG] jq built from source on %s\n", __DATE__);
```

Then rebuild:

```bash
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)
make install

# Test it
echo '{}' | ~/local/bin/jq '.'
# You'll see your debug output on stderr before the normal output
```

### 8.3 Example: Change Optimization Level

```bash
# Rebuild with debug symbols and no optimization (useful for step-through debugging)
make clean
./configure --prefix=$HOME/local \
            CFLAGS="-g -O0 -I$HOME/local/include" \
            LDFLAGS="-L$HOME/local/lib"
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)
make install

# Now you can debug it:
#   macOS: lldb ~/local/bin/jq -- '. | keys' <<< '{"a":1,"b":2}'
#   Linux: gdb --args ~/local/bin/jq '. | keys' <<< '{"a":1,"b":2}'
```

### 8.4 Example: Apply a Patch

If you have a `.patch` file (from a GitHub PR, bug fix, etc.):

```bash
cd jq-1.8.1

# Apply the patch
patch -p1 < /path/to/fix.patch

# Rebuild
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)
make install
```

Homebrew formulas can also include patches inline:

```ruby
# Example from other formulas — jq doesn't need this currently
patch :DATA    # applies a patch embedded at the end of the formula file

# or from a URL
patch do
  url "https://github.com/jqlang/jq/commit/abc123.patch"
  sha256 "..."
end
```

---

## Chapter 9: Bottles vs Source — What's the Difference?

### 9.1 What Is a Bottle?

A bottle is a tarball of a keg that was compiled on Homebrew's CI servers. It's the `make install` output, pre-packaged.

```
Bottle: jq--1.8.1.<platform>.bottle.tar.gz
  Contains: $(brew --cellar)/jq/1.8.1/{bin,lib,include,share,...}
  Downloaded from: ghcr.io/v2/homebrew/core/jq/blobs/sha256:...
```

Each platform gets its own bottle (arm64_sequoia, sonoma, x86_64_linux, etc.).

### 9.2 Comparison

| Aspect               | Bottle (default)             | Source (`--build-from-source`)       |
|-----------------------|------------------------------|--------------------------------------|
| **Speed**             | Seconds (just untar)         | Minutes (compile from scratch)       |
| **Reproducibility**   | Exact binary from CI         | Depends on your local environment    |
| **Customization**     | None — take it or leave it   | Full control over flags and patches  |
| **Debug symbols**     | Stripped (smaller binary)    | Can add `-g` for full debug info     |
| **Compiler**          | CI's compiler version        | Your local compiler version          |
| **Optimization**      | Standard `-O2`               | You choose (`-O0`, `-O3`, `-Os`)     |
| **Dependencies**      | Pre-resolved at CI time      | Resolved against your local state    |
| **INSTALL_RECEIPT**   | `poured_from_bottle: true`   | `poured_from_bottle: false`          |

### 9.3 When to Build from Source

- You need custom compile flags (debug symbols, sanitizers, non-default optimizations)
- You want to apply patches not yet in the official formula
- The bottle doesn't exist for your platform
- You're auditing/studying the build process (like right now)
- You're developing or testing changes to the upstream project
- You need to link against a non-standard dependency version

---

## Chapter 10: Creating Your Own Formula

If you have your own C project and want to package it as a Homebrew formula:

### 10.1 Minimal Formula Template

```ruby
class MyTool < Formula
  desc "A brief description of my tool"
  homepage "https://github.com/yourname/mytool"
  url "https://github.com/yourname/mytool/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "<sha256-of-the-tarball>"
  license "MIT"

  depends_on "some-dependency"

  def install
    system "./configure", "--prefix=#{prefix}"
    system "make", "install"
  end

  test do
    assert_match "mytool v1.0.0", shell_output("#{bin}/mytool --version")
  end
end
```

### 10.2 Non-Autotools Projects

Not every project uses `./configure && make`. Formulas handle many build systems:

**CMake:**
```ruby
def install
  system "cmake", "-S", ".", "-B", "build", *std_cmake_args
  system "cmake", "--build", "build"
  system "cmake", "--install", "build"
end
```

**Meson:**
```ruby
def install
  system "meson", "setup", "build", *std_meson_args
  system "meson", "compile", "-C", "build"
  system "meson", "install", "-C", "build"
end
```

**Go:**
```ruby
def install
  system "go", "build", *std_go_args(ldflags: "-s -w")
end
```

**Rust:**
```ruby
def install
  system "cargo", "install", *std_cargo_args
end
```

### 10.3 Testing Your Formula Locally

```bash
# Create a local tap
brew tap-new myname/mytap

# Create formula (auto-fills url and sha256)
brew create --tap myname/mytap https://github.com/yourname/mytool/archive/v1.0.0.tar.gz

# Edit it
brew edit myname/mytap/mytool

# Install from source
brew install --build-from-source myname/mytap/mytool

# Test it
brew test myname/mytap/mytool

# Audit it (checks formula style and conventions)
brew audit --new myname/mytap/mytool
```

---

## Appendix A: Key Homebrew Commands Reference

### Inspecting Formulas

| Command                         | Purpose                                          |
|---------------------------------|--------------------------------------------------|
| `brew info jq`                  | Show formula metadata, deps, install status      |
| `brew info --json=v2 jq`       | Machine-readable JSON metadata                   |
| `brew cat jq`                   | Print the formula source (Ruby)                  |
| `brew edit jq`                  | Open the formula in your editor                  |
| `brew deps jq`                  | List all dependencies                            |
| `brew deps --tree jq`          | Show dependency tree                             |
| `brew uses --installed oniguruma` | What installed formulas depend on oniguruma    |

### Building & Installing

| Command                                      | Purpose                                   |
|----------------------------------------------|-------------------------------------------|
| `brew install jq`                            | Install (bottle if available, else source)|
| `brew install --build-from-source jq`        | Force compile from source                 |
| `brew install --HEAD jq`                     | Build from Git HEAD (latest commit)       |
| `brew reinstall --build-from-source jq`      | Recompile an already-installed formula    |

### Keg Management

| Command                   | Purpose                                          |
|---------------------------|--------------------------------------------------|
| `brew link jq`            | Create symlinks from prefix to keg               |
| `brew unlink jq`          | Remove symlinks (keg stays in Cellar)            |
| `brew list jq`            | Show all files in the keg                        |
| `brew --cellar jq`        | Print the rack path for jq                       |
| `brew --prefix jq`        | Print the opt prefix path                        |

### Debugging

| Command                   | Purpose                                          |
|---------------------------|--------------------------------------------------|
| `brew test jq`            | Run the formula's test block                     |
| `brew audit jq`           | Check formula for style/correctness issues       |
| `brew log jq`             | Show Git log for formula changes                 |
| `brew linkage jq`         | Check library linkage of installed keg           |

### Platform Discovery

| Command                   | Purpose                                          |
|---------------------------|--------------------------------------------------|
| `brew --prefix`           | Show Homebrew root path                          |
| `brew --cellar`           | Show Cellar path                                 |
| `brew --cache`            | Show download cache path                         |
| `brew --repository`       | Show Homebrew Git repo path                      |
| `brew config`             | Show full Homebrew + system configuration        |

---

## Appendix B: Glossary

| Term                | Definition                                                                 |
|---------------------|----------------------------------------------------------------------------|
| **Formula**         | Ruby script that describes how to download, compile, and install software  |
| **Bottle**          | Pre-compiled binary tarball of a keg, built on Homebrew CI                 |
| **Keg**             | One installed version of a formula, isolated in its own directory          |
| **Cellar**          | The parent directory containing all kegs (`$(brew --cellar)`)              |
| **Rack**            | All kegs for one formula (`$(brew --cellar)/jq/`)                          |
| **Tap**             | A Git repository containing formula files                                 |
| **Prefix**          | Homebrew's root (`$(brew --prefix)`)                                       |
| **opt prefix**      | Stable symlink to the active keg version (`$(brew --prefix)/opt/jq`)       |
| **keg_only**        | Formula that is not linked into the prefix (avoids shadowing system tools) |
| **Autotools**       | GNU build system: `autoconf` + `automake` + `libtool`                     |
| **configure**       | Shell script that probes the system and generates Makefiles                |
| **std_configure_args** | Homebrew's standard flags (`--prefix`, `--disable-debug`, etc.)        |
| **INSTALL_RECEIPT** | JSON file recording how a keg was built/installed                         |
| **livecheck**       | Formula block that detects new upstream releases                          |
| **head**            | Install from the latest Git commit instead of a release tarball           |
| **.dylib**          | macOS shared library format (Dynamic Library)                             |
| **.so**             | Linux shared library format (Shared Object)                               |
| **Mach-O**          | macOS binary executable format                                            |
| **ELF**             | Linux binary executable format (Executable and Linkable Format)           |

---

## Appendix C: Platform Differences at a Glance

| What | macOS (Apple Silicon) | macOS (Intel) | Linux |
|---|---|---|---|
| Homebrew prefix | `/opt/homebrew` | `/usr/local` | `/home/linuxbrew/.linuxbrew` |
| Architecture | `arm64` | `x86_64` | `x86_64` or `aarch64` |
| Shared lib extension | `.dylib` | `.dylib` | `.so` |
| Binary format | Mach-O arm64 | Mach-O x86_64 | ELF 64-bit |
| Inspect linked libs | `otool -L <binary>` | `otool -L <binary>` | `ldd <binary>` |
| CPU core count | `sysctl -n hw.ncpu` | `sysctl -n hw.ncpu` | `nproc` |
| Compiler prereqs | `xcode-select --install` | `xcode-select --install` | `build-essential` / `gcc` |
| Default compiler | Apple Clang | Apple Clang | GCC |
| Debugger | `lldb` | `lldb` | `gdb` |
| System C library | `libSystem.B.dylib` | `libSystem.B.dylib` | `libc.so.6` (glibc) |
| Checksum tool | `shasum -a 256` | `shasum -a 256` | `sha256sum` |
| Caskroom (GUI apps) | Yes | Yes | No (Linux has native package managers) |
| `uses_from_macos` | Uses system copy | Uses system copy | Installs from source |

---

*Guide based on Homebrew formula sources from homebrew-core.*
*Formula: jq 1.8.1 with dependency oniguruma 6.9.10.*
*Portable across macOS (Apple Silicon & Intel) and Linux.*
