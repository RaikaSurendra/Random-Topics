# Chapter 5: Replace Mode (Preview)

## Overview

Ripgrep includes a `-r` (replace) flag that lets you **preview** what a replacement would
look like without actually modifying any files. This is one of the most misunderstood
features of `rg`: it is a **read-only preview**, not an in-place editor.

This chapter covers basic replacement previews, capture group extraction, named captures,
backreferences, combining replace with other flags, and the critical understanding of why
`rg` deliberately does NOT modify files. The workflow is: **use rg to FIND, use sd (or sed)
to REPLACE** -- a topic covered in later chapters.

Every command in this chapter should be run from the project root directory:

```bash
cd /Users/surendraraika/projects/Random/rg
```

> **Critical rule:** `rg -r` NEVER modifies files. Every command in this chapter produces
> output to stdout only. Your files remain untouched.

---

## Concepts

### Why rg does not modify files

Andrew Gallant (the author of ripgrep) made a deliberate design choice: rg is a **search**
tool, not a search-and-replace tool. The reasons:

1. **Unix philosophy:** Do one thing well. Searching and replacing are different operations.
2. **Safety:** An in-place replace that goes wrong is catastrophic. Preview-only is safe.
3. **Composability:** rg's output feeds into other tools (`sd`, `sed`, `xargs`) naturally.

### The replace workflow

```text
Step 1: rg "old_pattern" -r "new_text"     --> Preview the replacement
Step 2: Verify the preview looks correct
Step 3: sd "old_pattern" "new_text" file    --> Actually apply the replacement
        -- OR --
        rg -l "old_pattern" | xargs sed -i '' 's/old_pattern/new_text/g'
```

---

## Exercises

### Exercise 1 -- Basic replace preview

Preview replacing `TODO` with `DONE` across the project.

```bash
rg "TODO" -r "DONE"
```

```text
cmd/webshop/main.go
17:// DONE: Implement graceful shutdown with os.Signal handling
18:// DONE: Add configuration file path as CLI flag
19:// DONE: Move route registration to a separate function
23:	defaultPort        = ":8080"          // DONE: Read from config or env var
34:	// DONE: Load config from YAML file with fallback to env vars
49:	_ = db // DONE: Pass db to service layer
61:	// DONE: Add API versioning prefix (e.g., /api/v1/)
...
```

The output shows what the lines WOULD look like with `TODO` replaced by `DONE`.

**Verify the files are unchanged:**

```bash
rg "TODO" cmd/webshop/main.go | head -3
```

```text
17:// TODO: Implement graceful shutdown with os.Signal handling
18:// TODO: Add configuration file path as CLI flag
19:// TODO: Move route registration to a separate function
```

The original file is untouched. The replacement was purely in the output.

---

### Exercise 2 -- Understanding the preview-only nature

This is worth emphasizing because it trips up many users coming from `sed -i`.

**rg -r is preview only:**

```bash
# This does NOT change any file:
rg "FIXME" -r "FIXED" internal/config/config.go
```

```text
internal/config/config.go
41:	SessionTimeout  int    `yaml:"session_timeout" json:"sessionTimeout"` // FIXED: not implemented yet
51:// HACK: Hardcoded values should be loaded from a config file in production
59:		Host:            "localhost",       // FIXED: Should be configurable
62:		Password:        "webshop_pass123", // BUG: Hardcoded password in source
```

Notice that only lines containing `FIXME` are shown, with `FIXME` replaced by `FIXED`.
Lines without `FIXME` are not shown (unlike `--passthru`, covered later).

**To actually modify files, you would use `sd` (covered in Chapter 6):**

```bash
# DO NOT RUN THIS YET -- this is the "real" replacement syntax:
sd "FIXME" "FIXED" internal/config/config.go
```

---

### Exercise 3 -- Capture groups: extract and reformat

Use capture groups to restructure matched text.

**Extract function names from declarations:**

```bash
rg "func (\w+)\(" -r 'Function: $1' -t go
```

```text
cmd/webshop/main.go
30:Function: main

internal/auth/auth.go
63:Function: HashPassword
78:Function: CheckPassword
85:Function: GenerateToken
95:Function: ValidatePassword
131:Function: GenerateSessionToken

internal/config/config.go
52:Function: DefaultConfig
84:Function: Load
...
```

`$1` refers to the first capture group `(\w+)`. The entire matching line is printed, but
the matched portion `func X(` is replaced with `Function: X`.

**Reformat method signatures to show receiver and name:**

```bash
rg "func \((\w+) \*(\w+)\) (\w+)\(" -r '$2.$3 (receiver: $1)' -t go
```

```text
internal/models/user.go
64:User.Validate (receiver: u)
94:User.FullName (receiver: u)
99:User.IsAdmin (receiver: u)
104:User.HasAdminAccess (receiver: u)

internal/models/product.go
55:Product.Validate (receiver: p)
76:Product.IsOnSale (receiver: p)
82:Product.DiscountPercentage (receiver: p)
90:Product.InStock (receiver: p)
...
```

Three capture groups:
- `$1` = receiver variable name (e.g., `u`, `p`, `o`)
- `$2` = receiver type (e.g., `User`, `Product`, `Order`)
- `$3` = method name (e.g., `Validate`, `FullName`)

---

### Exercise 4 -- Named capture groups

Use `(?P<name>...)` syntax for self-documenting patterns.

```bash
rg '(?P<type>func \w+)' -r 'Found: $type' -t go -o | head -10
```

```text
cmd/webshop/main.go:30:Found: func main
internal/auth/auth.go:63:Found: func HashPassword
internal/auth/auth.go:78:Found: func CheckPassword
internal/auth/auth.go:85:Found: func GenerateToken
...
```

**Named captures with multiple groups:**

```bash
rg '(?P<keyword>TODO|FIXME|HACK|BUG):\s*(?P<message>.+)' -r '[$keyword] $message' | head -10
```

```text
cmd/webshop/main.go
17:// [TODO] Implement graceful shutdown with os.Signal handling
18:// [TODO] Add configuration file path as CLI flag
19:// [TODO] Move route registration to a separate function
...
```

Named captures make complex patterns readable. The names (`keyword`, `message`) serve
as documentation for what each group captures.

---

### Exercise 5 -- Multiple capture groups: preview bulk renaming

Preview what renaming the `Error` suffix to `Err` would look like across the codebase.

```bash
rg '(\w+)Error' -r '${1}Err' -t go
```

```text
pkg/errors/errors.go
13:type AppErr struct {
17:func (e *AppErr) Err() string {
40:type NotFoundErr struct {
46:func (e *NotFoundErr) Err() string {
50:type ValidationErr struct {
56:func (e *ValidationErr) Err() string {
60:type AuthErr struct {
65:func (e *AuthErr) Err() string {
69:type InternalErr struct {
74:func (e *InternalErr) Err() string {
...
```

Notice `${1}` syntax with braces -- this is necessary when the capture group reference
is immediately followed by text (`Err`). Without braces, `$1Err` would be interpreted
as capture group `$1E` followed by `rr`.

**Preview renaming `StatusOK` to `StatusSuccess`:**

```bash
rg "StatusOK" -r "StatusSuccess" -t go
```

---

### Exercise 6 -- Replace with passthru: see full file context

Combine `-r` with `--passthru` to see the entire file with replacements highlighted.

```bash
rg "TODO" -r "DONE" --passthru internal/config/config.go
```

```text
package config

import (
	"fmt"
	"os"
	"strconv"
)

// NOTE: Config fields use yaml tags for future YAML unmarshaling support
// DONE: Add validation for required fields after loading

// Config holds all application configuration.
type Config struct {
	AppName     string         `yaml:"app_name" json:"appName"`
	Environment string         `yaml:"environment" json:"environment"`
	...
```

Every line of the file is printed. Lines containing `TODO` show the replacement
(`DONE`), while all other lines appear unchanged. In a color terminal, the replaced
text is highlighted.

This is extremely useful for **reviewing** what a replacement would look like in context
before committing to it with `sd`.

---

### Exercise 7 -- Backreferences and $0 (whole match)

`$0` refers to the entire match (all capture groups plus surrounding matched text).

**Wrap all TODO comments in brackets:**

```bash
rg "(TODO|FIXME|HACK|BUG): (.+)" -r '[$1] $2 {annotation}' | head -10
```

```text
cmd/webshop/main.go
17:// [TODO] Implement graceful shutdown with os.Signal handling {annotation}
18:// [TODO] Add configuration file path as CLI flag {annotation}
19:// [TODO] Move route registration to a separate function {annotation}
...
```

**Use $0 to duplicate the match:**

```bash
rg "DEPRECATED" -r '$0 (MARKED FOR REMOVAL)' | head -5
```

```text
internal/models/user.go
17:	RoleGuest    UserRole = "guest" // DEPRECATED (MARKED FOR REMOVAL): Guest role will be removed in v2.0
...
```

`$0` inserts the entire original match, so you can append or prepend text around it.

---

### Exercise 8 -- Combine -o with -r: transform extracted data

Use `-o` (only matching) with `-r` to extract and reformat in one step.

**Preview converting snake_case identifiers to camelCase:**

```bash
rg -o '(\w+)_handler' -r '${1}Handler' internal/handlers/ | head -10
```

This finds patterns like `user_handler`, `product_handler` and previews them as
`userHandler`, `productHandler`.

**Extract and reformat struct field definitions:**

```bash
rg -o '(\w+)\s+(\w+)\s+`json:"(\w+)"' -r 'Field $1 (type: $2, json: $3)' internal/models/user.go
```

```text
internal/models/user.go
23:Field ID (type: int64, json: id)
24:Field Email (type: string, json: email)
25:Field Username (type: string, json: userName)
27:Field FirstName (type: string, json: first_name)
28:Field LastName (type: string, json: lastName)
...
```

This extracts the Go field name, type, and JSON tag name, reformatting them into a
readable report.

---

### Exercise 9 -- Replace with JSON output for programmatic use

Combine `--json` with `-r` to get structured replacement data.

```bash
rg --json "TODO" -r "DONE" internal/config/config.go | jq 'select(.type=="match")'
```

The JSON output includes both the original line and the replacement text in the
`submatches` field, making it possible to build automated refactoring tools.

```json
{
  "type": "match",
  "data": {
    "path": {"text": "internal/config/config.go"},
    "lines": {"text": "// DONE: Add validation for required fields after loading\n"},
    "line_number": 10,
    "submatches": [
      {"match": {"text": "TODO"}, "start": 3, "end": 7}
    ]
  }
}
```

**Build a replacement plan as JSON:**

```bash
rg --json "TODO" -r "DONE" | jq -c '
  select(.type=="match") |
  {
    file: .data.path.text,
    line: .data.line_number,
    original: .data.lines.text,
    replacement: (.data.lines.text | gsub("TODO"; "DONE"))
  }
' | head -5
```

This produces a structured replacement plan that a script could consume.

---

### Exercise 10 -- Limitations and the rg + sd workflow

**What rg -r CANNOT do:**

1. Modify files in-place (by design)
2. Apply different replacements to different matches in one command
3. Handle replacements that depend on surrounding context
4. Perform conditional replacements

**The standard workflow is rg to FIND, sd to REPLACE:**

```bash
# Step 1: Preview with rg
rg "userId" -r "userID" -t go

# Step 2: Verify the preview looks correct

# Step 3: Apply with sd (covered in Chapter 6)
sd "userId" "userID" $(rg -l "userId" -t go)
```

**Alternative using sed (when sd is not available):**

```bash
# Step 1: Preview with rg
rg "userId" -r "userID" -t go

# Step 2: Apply with sed via xargs
rg -l "userId" -t go | xargs sed -i '' 's/userId/userID/g'
```

**Why this two-step approach is better than `sed -i` alone:**

1. **Preview first:** You see exactly what will change before any file is modified
2. **Precise file list:** `rg -l` gives you only the files that need changing
3. **Better regex:** rg's regex engine is more predictable than sed's
4. **Safe:** No accidental modifications to files that should not be touched

---

### Exercise 11 -- Practical exercises

**11a. Preview renaming all `userId` to `userID` (Go convention for acronyms):**

```bash
rg "userId" -r "userID" -t go
```

```text
internal/models/order.go
27:	userID          int64       `json:"userID" db:"user_id" validate:"required"`

internal/handlers/order.go
89:	fmt.Printf("[INFO] Creating order for userID=%d with %d items\n", req.UserID, len(req.Items))
...
```

Count how many files and lines would be affected:

```bash
rg -c "userId" -t go
```

```text
internal/models/order.go:1
internal/handlers/order.go:1
internal/service/user_service.go:1
...
```

**11b. Preview extracting all function signatures into a report format:**

```bash
rg -o "func (\(\w+ \*\w+\) )?\w+\([^)]*\)( \(?[^)]*\)?)?" -t go | head -20
```

Or use a simpler approach with `-r`:

```bash
rg "^func (\w+)\(([^)]*)\)\s*(.*)\{" -r 'FUNC $1($2) -> $3' -t go
```

**11c. Preview converting all `fmt.Println("[WARN]` to `log.Warn(`:**

```bash
rg 'fmt\.Println\("\[WARN\] (.+)"\)' -r 'log.Warn("$1")' -t go
```

```text
internal/auth/auth.go
72:	log.Warn("Using placeholder password hashing - NOT SECURE")
132:	log.Warn("GenerateSessionToken is deprecated, use GenerateToken instead")

internal/models/user.go
105:	log.Warn("HasAdminAccess is deprecated, use IsAdmin instead")
...
```

**11d. Preview wrapping raw error strings with error codes:**

```bash
rg 'errors\.New\("(.+)"\)' -r 'errors.New("ERR001: $1")' -t go | head -10
```

**11e. Preview what the JSON field names would look like if standardized to camelCase:**

```bash
rg -o 'json:"(\w+_\w+)"' -r 'json:"$1" -> needs camelCase' internal/models/
```

This finds all JSON tags using snake_case and flags them for conversion.

---

## Summary: When to Use Each Flag Combination

| Goal | Command |
|---|---|
| Preview simple replacement | `rg "old" -r "new"` |
| Preview with full file context | `rg "old" -r "new" --passthru file` |
| Extract capture group | `rg -o "func (\w+)" -r '$1'` |
| Named capture group | `rg '(?P<fn>func \w+)' -r '$fn'` |
| Wrap match with text | `rg "TODO" -r '[$0]'` |
| JSON replacement plan | `rg --json "old" -r "new" \| jq ...` |
| Actual file modification | `sd "old" "new" file` (Chapter 6) |

---

## Cheat Sheet

| Task | Command |
|---|---|
| Basic preview | `rg "pat" -r "replacement"` |
| Capture group $1 | `rg "(group)" -r '$1'` |
| Named capture | `rg "(?P<n>pat)" -r '$n'` |
| Multiple captures | `rg "(a)(b)" -r '${1}_${2}'` |
| Whole match $0 | `rg "pat" -r '[$0]'` |
| Only-match + replace | `rg -o "(pat)" -r '$1'` |
| Passthru preview | `rg "pat" -r "new" --passthru file` |
| JSON + replace | `rg --json "pat" -r "new"` |
| Find files for sd | `rg -l "pat" -t go` |
| Preview then apply | `rg "pat" -r "new" && sd "pat" "new" $(rg -l "pat")` |
