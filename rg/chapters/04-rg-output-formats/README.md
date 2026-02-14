# Chapter 4: Output Formats & Scripting

## Overview

When you use `rg` interactively, the default pretty-printed output is excellent. But when
you integrate `rg` into scripts, CI pipelines, editor plugins, or analysis tools, you need
machine-readable output. This chapter covers every output format `rg` offers: JSON, files-only,
count modes, only-matching, replacement extraction, statistics, vimgrep format, and more.

Every command in this chapter should be run from the project root directory:

```bash
cd /Users/surendraraika/projects/Random/rg
```

---

## Concepts

### Output mode spectrum

From most human-friendly to most machine-friendly:

| Mode | Flag | Output |
|---|---|---|
| Pretty (default) | (none) | Grouped by file, colored, line numbers |
| No heading | `--no-heading` | Flat `file:line:text` per match |
| Count per file | `-c` | `file:count` |
| Count all matches | `--count-matches` | `file:count` (counts every match, not lines) |
| Files with matches | `-l` | One filename per line |
| Files without matches | `--files-without-match` | One filename per line |
| Only matching text | `-o` | Just the matched portion |
| Vimgrep | `--vimgrep` | `file:line:col:text` |
| JSON | `--json` | One JSON object per line |

---

## Exercises

### Exercise 1 -- Default pretty output vs --no-heading

**Default (interactive, human-friendly):**

```bash
rg "TODO" internal/config/
```

```text
internal/config/config.go
10:// TODO: Add validation for required fields after loading
70:	JWTSecret:      "super-secret-key-change-me", // TODO: Load from env var
83:// TODO: Actually implement YAML parsing - currently only uses env vars and defaults
```

Matches are grouped under filename headings with colorized line numbers.

**With `--no-heading` (for piping into other tools):**

```bash
rg --no-heading "TODO" internal/config/
```

```text
internal/config/config.go:10:// TODO: Add validation for required fields after loading
internal/config/config.go:70:	JWTSecret:      "super-secret-key-change-me", // TODO: Load from env var
internal/config/config.go:83:// TODO: Actually implement YAML parsing - currently only uses env vars and defaults
```

Each line is self-contained (`file:line:text`), which is easier to parse with `cut`,
`awk`, or `sed`.

**Note:** When you pipe rg output (e.g., `rg TODO | wc -l`), rg automatically switches
to `--no-heading` and disables color. This auto-detection is smart and usually does the
right thing.

---

### Exercise 2 -- JSON output

Get machine-readable structured output.

```bash
rg --json "TODO" internal/config/config.go
```

Each line is a separate JSON object. There are four message types:

**`begin` -- marks the start of results for a file:**

```json
{"type":"begin","data":{"path":{"text":"internal/config/config.go"}}}
```

**`match` -- one per matching line:**

```json
{
  "type": "match",
  "data": {
    "path": {"text": "internal/config/config.go"},
    "lines": {"text": "// TODO: Add validation for required fields after loading\n"},
    "line_number": 10,
    "absolute_offset": 186,
    "submatches": [
      {"match": {"text": "TODO"}, "start": 3, "end": 7}
    ]
  }
}
```

**`end` -- marks the end of results for a file:**

```json
{"type":"end","data":{"path":{"text":"internal/config/config.go"},"stats":{"matched_lines":3,"matches":3}}}
```

**`summary` -- final statistics (only with `--stats` or as the last JSON line):**

```json
{"type":"summary","data":{"stats":{"elapsed":{"secs":0,"nanos":2500000},"searches":1,"matched_lines":3,"matches":3}}}
```

---

### Exercise 3 -- JSON + jq: extract structured data

Parse `rg --json` output with `jq` for reporting and analysis.

**Extract just the matching text from each match:**

```bash
rg --json "TODO" | jq -r 'select(.type=="match") | .data.lines.text' | head -10
```

```text
// TODO: Implement graceful shutdown with os.Signal handling
// TODO: Add configuration file path as CLI flag
// TODO: Move route registration to a separate function
	defaultPort        = ":8080"          // TODO: Read from config or env var
...
```

**Extract file:line pairs for a quick summary:**

```bash
rg --json "TODO" | jq -r 'select(.type=="match") | "\(.data.path.text):\(.data.line_number)"'
```

```text
cmd/webshop/main.go:17
cmd/webshop/main.go:18
cmd/webshop/main.go:19
cmd/webshop/main.go:23
...
```

**Count TODOs per file using jq:**

```bash
rg --json "TODO" | jq -r 'select(.type=="end") | "\(.data.path.text): \(.data.stats.matches)"'
```

```text
cmd/webshop/main.go: 7
internal/config/config.go: 3
internal/auth/auth.go: 2
internal/handlers/user.go: 4
...
```

---

### Exercise 4 -- Files with matches (files-only mode)

Print only the filenames that contain matches.

**grep:**

```bash
grep -rl "TODO" .
```

**rg:**

```bash
rg -l "TODO"
```

```text
cmd/webshop/main.go
configs/dev.yaml
configs/prod.yaml
configs/test.yaml
internal/auth/auth.go
internal/config/config.go
internal/database/postgres.go
internal/handlers/order.go
internal/handlers/order_handler.go
internal/handlers/product.go
internal/handlers/product_handler.go
internal/handlers/user.go
internal/handlers/user_handler.go
internal/models/order.go
internal/models/user.go
internal/service/order_service.go
internal/service/product_service.go
internal/service/user_service.go
pkg/errors/errors.go
pkg/logger/logger.go
pkg/middleware/middleware.go
scripts/seed.sql
test/integration/api_test.go
```

This is useful for piping into other tools: `rg -l "TODO" | xargs wc -l`.

---

### Exercise 5 -- Files WITHOUT matches

Find files that do NOT contain TODO comments (the clean files).

```bash
rg --files-without-match "TODO" -t go
```

```text
internal/models/product.go
```

This inverts the file-level match: only files with zero matches are listed. Very useful
for finding files that are "clean" of certain patterns, or for finding config files
missing a required field.

**Practical example -- find YAML configs that lack a `timeout` setting:**

```bash
rg --files-without-match "timeout" configs/
```

---

### Exercise 6 -- Count modes: -c vs --count-matches

These two count flags have an important difference.

**`-c` (count lines with matches):**

```bash
rg -c "error" internal/handlers/user.go
```

```text
internal/handlers/user.go:8
```

If a line has "error" twice, `-c` counts it as 1 line.

**`--count-matches` (count every individual match):**

```bash
rg --count-matches "error" internal/handlers/user.go
```

```text
internal/handlers/user.go:8
```

(In this case the counts happen to be the same. But consider:)

```bash
# Count lines with a status code reference
rg -c "http\.Status" internal/handlers/user.go

# Count every individual status code reference (a line may have two)
rg --count-matches "http\.Status" internal/handlers/user.go
```

The difference matters when lines contain multiple matches.

---

### Exercise 7 -- Only matching: extract the match itself

Print only the text that matched the pattern, not the full line.

```bash
rg -o "func \w+" -t go | head -20
```

```text
cmd/webshop/main.go:30:func main
internal/auth/auth.go:63:func HashPassword
internal/auth/auth.go:78:func CheckPassword
internal/auth/auth.go:85:func GenerateToken
internal/auth/auth.go:95:func ValidatePassword
internal/auth/auth.go:131:func GenerateSessionToken
internal/config/config.go:52:func DefaultConfig
internal/config/config.go:84:func Load
...
```

Only `func main`, `func HashPassword`, etc. are printed -- not the full signature.

**Use case:** Build a quick index of all function names in the project.

---

### Exercise 8 -- Capture groups with -r (replace mode for extraction)

Combine `-o` with `-r` to extract specific capture groups.

**Extract only function names (not the `func` keyword):**

```bash
rg -o "func (\w+)" -r '$1' -t go
```

```text
cmd/webshop/main.go:30:main
internal/auth/auth.go:63:HashPassword
internal/auth/auth.go:78:CheckPassword
internal/auth/auth.go:85:GenerateToken
internal/auth/auth.go:95:ValidatePassword
...
```

**Extract JSON field names from struct tags:**

```bash
rg -o 'json:"(\w+)"' -r '$1' internal/models/user.go
```

```text
internal/models/user.go:23:id
internal/models/user.go:24:email
internal/models/user.go:25:userName
internal/models/user.go:26:(hyphen, skipped)
internal/models/user.go:27:first_name
internal/models/user.go:28:lastName
internal/models/user.go:29:role
internal/models/user.go:30:phone_number
...
```

This extracts just the API field names from struct tags.

**Extract error message strings:**

```bash
rg -o 'errors\.New\("([^"]+)"\)' -r '$1' -t go
```

```text
internal/models/user.go:66:email is required
internal/models/user.go:75:username must not exceed 50 characters
internal/models/user.go:78:first name and last name are required
internal/models/user.go:81:role is required
internal/models/product.go:57:SKU is required
...
```

---

### Exercise 9 -- Statistics

Get a summary of your search across the entire project.

```bash
rg --stats "TODO"
```

At the end of the normal output, rg prints:

```text
...
81 matches
81 matched lines
27 files contained matches
42 files searched
0 bytes printed
0.003 seconds spent searching
```

This is invaluable for understanding the scale of technical debt (TODO/FIXME/HACK counts)
or the scope of a refactoring task.

**Stats for annotations by type:**

```bash
rg --stats "FIXME" 2>&1 | tail -6
rg --stats "BUG" 2>&1 | tail -6
rg --stats "HACK" 2>&1 | tail -6
rg --stats "DEPRECATED" 2>&1 | tail -6
```

---

### Exercise 10 -- Null separator for safe piping

When filenames contain spaces, newlines, or special characters, use null-byte separators.

```bash
rg -0 -l "TODO" | xargs -0 wc -l
```

`-0` makes `rg` separate filenames with `\0` (null byte) instead of `\n`. Combined with
`xargs -0`, this safely handles any filename.

**grep equivalent:**

```bash
grep -rlZ "TODO" . | xargs -0 wc -l
```

(grep uses `-Z` for the same purpose.)

---

### Exercise 11 -- List files that WOULD be searched

The `--files` flag lists all files rg would search, without actually searching them.
This is extremely useful for debugging filter combinations.

**List all files rg would search:**

```bash
rg --files
```

**List only Go files:**

```bash
rg --files -t go
```

```text
cmd/webshop/main.go
internal/auth/auth.go
internal/config/config.go
internal/database/postgres.go
internal/handlers/order.go
internal/handlers/order_handler.go
internal/handlers/product.go
internal/handlers/product_handler.go
internal/handlers/user.go
internal/handlers/user_handler.go
internal/models/order.go
internal/models/product.go
internal/models/user.go
internal/service/order_service.go
internal/service/product_service.go
internal/service/user_service.go
pkg/errors/errors.go
pkg/logger/logger.go
pkg/middleware/middleware.go
test/integration/api_test.go
```

**Debug complex filters:**

```bash
rg --files -t go -g "!*_test.go" -g "!internal/handlers/*"
```

```text
cmd/webshop/main.go
internal/auth/auth.go
internal/config/config.go
internal/database/postgres.go
internal/models/order.go
internal/models/product.go
internal/models/user.go
internal/service/order_service.go
internal/service/product_service.go
internal/service/user_service.go
pkg/errors/errors.go
pkg/logger/logger.go
pkg/middleware/middleware.go
```

Use `--files` before running a complex search to verify your filters are correct.

---

### Exercise 12 -- Vimgrep format

Produce output that text editors (Vim, Emacs, VS Code) can parse for jump-to-location.

```bash
rg --vimgrep "TODO" | head -10
```

```text
cmd/webshop/main.go:17:4:// TODO: Implement graceful shutdown with os.Signal handling
cmd/webshop/main.go:18:4:// TODO: Add configuration file path as CLI flag
cmd/webshop/main.go:19:4:// TODO: Move route registration to a separate function
cmd/webshop/main.go:23:37:	defaultPort        = ":8080"          // TODO: Read from config or env var
...
```

Format: `file:line:column:text`

The column number is the byte offset of the match within the line. This allows editors to
jump directly to the match position, not just the line.

**In Vim:**

```vim
:set grepprg=rg\ --vimgrep
:grep TODO
:copen
```

---

### Exercise 13 -- Column number

Add byte-offset column numbers to standard output.

```bash
rg --column "TODO" | head -5
```

```text
cmd/webshop/main.go:17:4:// TODO: Implement graceful shutdown with os.Signal handling
cmd/webshop/main.go:18:4:// TODO: Add configuration file path as CLI flag
cmd/webshop/main.go:19:4:// TODO: Move route registration to a separate function
cmd/webshop/main.go:23:37:	defaultPort        = ":8080"          // TODO: Read from config or env var
cmd/webshop/main.go:34:3:	// TODO: Load config from YAML file with fallback to env vars
```

This is similar to `--vimgrep` but preserves the normal output format with an added column.

---

### Exercise 14 -- Sorted output

Control the order of results.

**Sort by file path (alphabetical):**

```bash
rg --sort path "TODO" | head -20
```

**Sort by last modified time:**

```bash
rg --sort modified "TODO" | head -20
```

**Reverse sort (most recently modified first):**

```bash
rg --sortr modified "TODO" | head -20
```

**Available sort keys:** `path`, `modified`, `accessed`, `created`, `none`

**Note:** Sorting disables parallelism, so it is slower on large codebases. Use it when
you need deterministic output (e.g., in tests or diffs) or when you want to prioritize
recently changed files.

---

### Exercise 15 -- Build a TODO report with rg --json + jq

Create a structured TODO/FIXME/HACK report for your team.

**One-liner that produces a Markdown-formatted report:**

```bash
rg --json "TODO|FIXME|HACK|BUG" -t go | jq -r '
  select(.type=="match") |
  "- **\(.data.path.text):\(.data.line_number)** -- \(.data.lines.text | gsub("^\\s+//\\s*"; "") | gsub("\\n$"; ""))"
' | sort
```

```text
- **cmd/webshop/main.go:17** -- TODO: Implement graceful shutdown with os.Signal handling
- **cmd/webshop/main.go:18** -- TODO: Add configuration file path as CLI flag
- **cmd/webshop/main.go:19** -- TODO: Move route registration to a separate function
- **internal/auth/auth.go:61** -- TODO: Implement actual bcrypt hashing - this is a stub
- **internal/auth/auth.go:62** -- FIXME: Current implementation is NOT secure for production use
...
```

**Count annotations by type across the project:**

```bash
rg --json "TODO|FIXME|HACK|BUG|DEPRECATED" -t go | jq -r '
  select(.type=="match") |
  .data.lines.text' | grep -oE "(TODO|FIXME|HACK|BUG|DEPRECATED)" | sort | uniq -c | sort -rn
```

```text
     42 TODO
     18 FIXME
     12 HACK
     10 BUG
      8 DEPRECATED
```

---

## Cheat Sheet

| Task | Flag | Example |
|---|---|---|
| Pretty output (default) | (none) | `rg "pat"` |
| Flat output for piping | `--no-heading` | `rg --no-heading "pat"` |
| JSON (machine-readable) | `--json` | `rg --json "pat"` |
| Files with matches only | `-l` | `rg -l "pat"` |
| Files without matches | `--files-without-match` | `rg --files-without-match "pat"` |
| Count lines per file | `-c` | `rg -c "pat"` |
| Count all matches | `--count-matches` | `rg --count-matches "pat"` |
| Only matching text | `-o` | `rg -o "func \w+"` |
| Extract capture group | `-o -r '$1'` | `rg -o "func (\w+)" -r '$1'` |
| Search statistics | `--stats` | `rg --stats "pat"` |
| Null-separated filenames | `-0 -l` | `rg -0 -l "pat" \| xargs -0 cmd` |
| List searchable files | `--files` | `rg --files -t go` |
| Vimgrep format | `--vimgrep` | `rg --vimgrep "pat"` |
| Column numbers | `--column` | `rg --column "pat"` |
| Sort by path | `--sort path` | `rg --sort path "pat"` |
| Sort by modified time | `--sort modified` | `rg --sort modified "pat"` |
