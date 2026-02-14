# rg & sd Mastery: From Intermediate to Advanced

A hands-on learning project to master **ripgrep** (`rg`) and **sd** through progressive exercises against a realistic Go web application codebase.

---

## Prerequisites

| Tool    | Minimum Version | Check Command       |
|---------|-----------------|---------------------|
| Go      | 1.24+           | `go version`        |
| ripgrep | 15.0+           | `rg --version`      |
| sd      | 1.0+            | `sd --version`      |
| jq      | 1.7+ (optional) | `jq --version`      |

## Installation

```bash
# macOS (Homebrew)
brew install ripgrep sd jq

# Verify
rg --version
sd --version
```

### Important: Shell Alias Conflicts

Some tools (including Claude Code) alias `rg` to other commands. If `rg` does not behave as expected, check for aliases:

```bash
type rg          # shows what rg resolves to
which rg         # shows the binary path
alias rg         # shows alias if one exists
```

To fix:

```bash
unalias rg                  # remove the alias for the current session
/opt/homebrew/bin/rg        # call ripgrep directly by path
```

You can add `unalias rg 2>/dev/null` to your `.zshrc` to prevent conflicts permanently.

---

## Project Structure

```
rg/
├── cmd/
│   └── webshop/
│       └── main.go              # Application entry point
├── internal/
│   ├── auth/
│   │   └── auth.go              # JWT authentication logic
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── database/
│   │   └── postgres.go          # Database connection and queries
│   ├── handlers/
│   │   ├── user.go              # User route handlers
│   │   └── user_handler.go      # Additional user handler logic
│   ├── models/
│   │   ├── user.go              # User model and types
│   │   ├── product.go           # Product model and types
│   │   └── order.go             # Order model and types
│   └── service/                 # Business logic layer (future)
├── pkg/
│   ├── errors/                  # Custom error types
│   ├── logger/                  # Structured logging
│   └── middleware/              # HTTP middleware
├── configs/
│   ├── dev.yaml                 # Development configuration
│   ├── prod.yaml                # Production configuration
│   └── test.yaml                # Test configuration
├── scripts/
│   ├── migrate.sql              # Database schema migrations
│   └── seed.sql                 # Seed data for development
├── test/
│   └── integration/             # Integration tests
├── chapters/                    # Learning chapters (see below)
│   ├── 01-rg-pattern-matching/
│   ├── 02-rg-file-filtering/
│   ├── ...
│   └── 09-real-world-refactoring/
├── .rgignore                    # ripgrep ignore rules
├── .gitignore                   # Git ignore rules
├── Makefile                     # Build, test, and rg/sd helper targets
├── go.mod                       # Go module definition
└── README.md                    # This file
```

---

## Chapters

| #  | Directory                        | Topic                                           |
|----|----------------------------------|--------------------------------------------------|
| 01 | `chapters/01-rg-pattern-matching` | Regex patterns, literal search, case sensitivity |
| 02 | `chapters/02-rg-file-filtering`   | File types, globs, include/exclude filters       |
| 03 | `chapters/03-rg-context-multiline`| Context lines (-A/-B/-C), multiline matching     |
| 04 | `chapters/04-rg-output-formats`   | JSON output, stats, count, files-with-matches    |
| 05 | `chapters/05-rg-replace-mode`     | rg --replace for in-line substitutions           |
| 06 | `chapters/06-sd-fundamentals`     | Basic find-and-replace with sd, vs sed syntax    |
| 07 | `chapters/07-sd-advanced`         | Capture groups, multiline, file-scoped edits     |
| 08 | `chapters/08-rg-sd-power-combos`  | Piping rg into sd for surgical codebase edits    |
| 09 | `chapters/09-real-world-refactoring` | Full refactoring scenarios using rg + sd      |

Each chapter contains a `README.md` with exercises to run against this codebase.

---

## Quick Comparison

### grep vs rg

| Feature              | grep                          | rg (ripgrep)                      |
|----------------------|-------------------------------|-----------------------------------|
| Speed                | Slower on large repos         | Extremely fast (parallelized)     |
| Respects .gitignore  | No                            | Yes, by default                   |
| Regex flavor         | POSIX BRE/ERE                 | Rust regex (fast, Unicode-aware)  |
| Binary file handling | Searches by default           | Skips by default                  |
| Recursive search     | Requires `-r` flag            | Recursive by default              |
| Color output         | `--color=auto`                | Color by default in terminal      |

### sed vs sd

| Feature              | sed                           | sd                                |
|----------------------|-------------------------------|-----------------------------------|
| Regex syntax         | POSIX BRE (or `-E` for ERE)   | Rust regex (PCRE-like)            |
| String literals      | Needs heavy escaping          | Use `-F` for fixed strings        |
| In-place editing     | `-i ''` (macOS) or `-i` (GNU) | `-i` works the same everywhere    |
| Readability          | Dense, cryptic syntax         | Clean, intuitive syntax           |
| Multiline            | Awkward with `N` command       | Supports `(?s)` flag naturally    |
| Capture groups       | `\1`, `\2`                    | `$1`, `$2`                        |

---

## How to Use This Project

1. Clone the repository and `cd` into it:
   ```bash
   cd /path/to/rg
   ```

2. Explore available Makefile targets:
   ```bash
   make help
   ```

3. Try a quick search to verify rg works:
   ```bash
   rg 'TODO' .
   rg --type go 'func ' .
   ```

4. Follow the chapters in order, starting with `chapters/01-rg-pattern-matching/`.

5. Each chapter builds on the previous one. Exercises are designed to run against the Go source files, SQL scripts, and YAML configs in this project.

---

## License

This is a personal learning project. Use it however you like.
