# Chapter 1: Pattern Matching Fundamentals

## Overview

This chapter covers the core of what makes ripgrep (`rg`) a superior search tool: pattern matching.
You will learn literal searches, regular expressions, case sensitivity controls, word boundaries,
fixed strings, inverted matching, multiple patterns, and PCRE2 features -- all compared against
traditional `grep` so you can see exactly what you gain by switching.

Every command in this chapter should be run from the project root directory:

```bash
cd /Users/surendraraika/projects/Random/rg
```

> **Convention used throughout:** each exercise shows the `grep` way first, then the `rg` way,
> followed by a "Why rg wins" note.

---

## Concepts

### Default behavior differences

| Behavior | `grep` | `rg` |
|---|---|---|
| Recursive search | Requires `-r` flag | Automatic |
| `.gitignore` respect | No | Yes (automatic) |
| Colored output | Requires `--color=auto` | Automatic in terminal |
| Binary file skip | No (prints "Binary file matches") | Yes (skips by default) |
| Hidden files | Searched by default with `-r` | Skipped by default |
| Regex engine | POSIX BRE/ERE | Rust regex (fast, Unicode-aware) |

### Regex engine comparison

`rg` uses the Rust `regex` crate by default, which supports most Perl-compatible features
**except** lookahead and lookbehind. For those, pass `--pcre2` (or `-P`) to switch to the
PCRE2 engine, which supports the full range of zero-width assertions.

---

## Exercises

### Exercise 1 -- Basic literal search

Search the codebase for `TODO` comments.

**grep:**

```bash
grep -rn "TODO" .
```

This requires `-r` for recursion and `-n` for line numbers. It will also descend into
`.git/`, `vendor/`, and any binary files.

**rg:**

```bash
rg TODO
```

```text
cmd/webshop/main.go
17:// TODO: Implement graceful shutdown with os.Signal handling
18:// TODO: Add configuration file path as CLI flag
19:// TODO: Move route registration to a separate function
23:	defaultPort        = ":8080"          // TODO: Read from config or env var
34:	// TODO: Load config from YAML file with fallback to env vars
49:	_ = db // TODO: Pass db to service layer
61:	// TODO: Add API versioning prefix (e.g., /api/v1/)

internal/config/config.go
10:// TODO: Add validation for required fields after loading
70:	JWTSecret:      "super-secret-key-change-me", // TODO: Load from env var
83:// TODO: Actually implement YAML parsing - currently only uses env vars and defaults
...
```

**Why rg wins:** No flags needed. Auto-recurses, respects `.gitignore`, prints colorized
output with file headings, and skips binary files. On this project, `grep -r` would also
search inside `.git/` objects unless you add `--exclude-dir=.git`.

---

### Exercise 2 -- Regex patterns: find all function definitions

Search for Go function declarations using a regex.

**grep:**

```bash
grep -rnE "func \w+\(" .
```

Requires `-E` to enable extended regex (otherwise `\w` and `+` are not supported in basic
regex). Also requires `-r` and `-n`.

**rg:**

```bash
rg "func \w+\("
```

```text
cmd/webshop/main.go
30:func main() {

internal/auth/auth.go
48:func Middleware(next http.Handler, log Logger) http.Handler {
85:func ValidateToken(tokenStr string) (*TokenClaims, error) {
101:func RequireRole(roles ...string) func(http.Handler) http.Handler {
117:func GetClaimsFromContext(ctx context.Context) (*TokenClaims, error) {

internal/config/config.go
52:func DefaultConfig() *Config {
84:func Load(path string) (*Config, error) {
...
```

**Why rg wins:** Rust regex is the default engine -- no `-E` flag needed. The output is
grouped by file with line numbers, making it far easier to scan than grep's flat
`file:line:match` format.

---

### Exercise 3 -- Case-insensitive search

Find all occurrences of "error" regardless of case (error, Error, ERROR).

**grep:**

```bash
grep -rni "error" .
```

**rg:**

```bash
rg -i error
```

```text
cmd/webshop/main.go
37:		log.Error("Failed to load configuration: %v", err)
38:		fmt.Println("[ERROR] Configuration load failed, using defaults")
45:		log.Error("Failed to connect to database: %v", err)
46:		fmt.Println("[ERROR] Database connection failed")
...

pkg/errors/errors.go
13:type AppError struct {
17:func (e *AppError) Error() string {
...
```

Both tools produce similar results, but `rg` is significantly faster on large codebases
because it parallelizes across CPU cores and uses memory-mapped I/O.

---

### Exercise 4 -- Smart case (rg exclusive)

Smart case is one of rg's most useful features. When your pattern is all lowercase,
the search is case-insensitive. The moment any character is uppercase, the search
becomes case-sensitive.

**grep:** No equivalent. You must choose `-i` or not.

**rg (all lowercase = case-insensitive):**

```bash
rg -S error
```

This matches `error`, `Error`, `ERROR`, `AppError`, etc.

**rg (has uppercase = case-sensitive):**

```bash
rg -S Error
```

This matches only `Error` (capital E) -- it will NOT match `error` or `ERROR`.

```text
internal/config/config.go
96:		return nil, fmt.Errorf("invalid APP_PORT value %q: %w", port, err)

pkg/errors/errors.go
13:type AppError struct {
17:func (e *AppError) Error() string {
...
```

**Why rg wins:** `-S` (smart case) eliminates the most common grep frustration: having to
decide between `-i` and no flag. When you type the pattern with mixed case, you clearly
intend an exact match; when all lowercase, you usually want to be flexible.

---

### Exercise 5 -- Word boundary matching

Search for the identifier `id` as a whole word, not as part of `valid`, `void`, `provide`, etc.

**grep:**

```bash
grep -rnw "id" .
```

**rg:**

```bash
rg -w id
```

```text
internal/handlers/user.go
32:	idStr := r.URL.Query().Get("id")
34:		w.WriteHeader(http.StatusBadRequest) // 400
...

internal/handlers/product.go
33:	if idStr == "" {
...

internal/models/order.go
47:	ID        int64   `json:"id" db:"id"`
...
```

Without `-w`, searching for `id` would match inside words like `valid`, `provide`,
`productId`, `Middleware`, etc. -- burying the real matches in noise.

**Tip:** `-w` wraps the pattern in `\b...\b` word boundaries internally.

---

### Exercise 6 -- Fixed string search (no regex)

Search for a literal string that contains regex metacharacters.

**grep:**

```bash
grep -rnF "fmt.Errorf(" .
```

`-F` tells grep to treat the pattern as a literal string, so `.` and `(` are not
interpreted as regex metacharacters.

**rg:**

```bash
rg -F "fmt.Errorf("
```

```text
internal/config/config.go
96:		return nil, fmt.Errorf("invalid APP_PORT value %q: %w", port, err)

internal/models/user.go
69:		return fmt.Errorf("invalid email format: %s", u.Email)
72:		return fmt.Errorf("username must be at least 3 characters, got %d", len(u.Username))
88:		return fmt.Errorf("invalid role: %s", u.Role)
...
```

**When to use `-F`:** Whenever your search term contains `.`, `(`, `)`, `[`, `*`, `+`,
`?`, `{`, `|`, `\`, or `$`. Without `-F`, `fmt.Errorf(` would treat `.` as "any character"
and `(` as a group start.

---

### Exercise 7 -- Inverted match

Find all non-empty lines in a specific file (exclude blank lines).

**grep:**

```bash
grep -nv "^$" internal/config/config.go
```

**rg:**

```bash
rg -v "^$" internal/config/config.go
```

```text
1:package config
3:import (
4:	"fmt"
5:	"os"
6:	"strconv"
7:)
9:// NOTE: Config fields use yaml tags for future YAML unmarshaling support
10:// TODO: Add validation for required fields after loading
...
```

Now try finding all lines in YAML config files that are NOT comments:

```bash
rg -v "^\s*#" configs/
```

This shows only the actual configuration values, stripping out comment-only lines.

---

### Exercise 8 -- Multiple patterns

Search for all annotation comments: TODO, FIXME, HACK, BUG, or DEPRECATED.

**grep (alternation):**

```bash
grep -rnE "TODO|FIXME|HACK|BUG|DEPRECATED" .
```

**rg (alternation):**

```bash
rg "TODO|FIXME|HACK|BUG|DEPRECATED"
```

**rg (multiple `-e` flags -- equivalent, sometimes clearer):**

```bash
rg -e TODO -e FIXME -e HACK -e BUG -e DEPRECATED
```

The `-e` form is useful in scripts where you build the pattern list dynamically:

```bash
# Build pattern list from a file
patterns=(-e TODO -e FIXME -e HACK)
rg "${patterns[@]}"
```

**Counting annotations by type across the project:**

```bash
rg -c "TODO"
rg -c "FIXME"
rg -c "HACK"
rg -c "BUG"
rg -c "DEPRECATED"
```

---

### Exercise 9 -- Lookahead and lookbehind (PCRE2)

Extract just the function names (the word after `func `) using a lookbehind.

**grep:** No support for lookbehind in standard grep. GNU grep has `-P` but it is
unavailable on macOS by default.

**rg with PCRE2:**

```bash
rg -P "(?<=func )\w+" -o
```

```text
cmd/webshop/main.go
30:main

internal/auth/auth.go
48:Middleware
85:ValidateToken
101:RequireRole
117:GetClaimsFromContext

internal/config/config.go
52:DefaultConfig
84:Load
...
```

The `-o` flag prints only the matching portion (the function name), not the whole line.
`(?<=func )` is a positive lookbehind that asserts `func ` precedes the match without
including it in the result.

**Lookahead example -- find variable names followed by `error`:**

```bash
rg -P "\w+(?=Error)" -o
```

```text
pkg/errors/errors.go
13:App
17:App
...
```

---

### Exercise 10 -- Default regex engine vs PCRE2

The default Rust regex engine is extremely fast but does not support:
- Lookahead: `(?=...)`, `(?!...)`
- Lookbehind: `(?<=...)`, `(?<!...)`
- Backreferences: `\1`, `\2`
- Atomic groups: `(?>...)`
- Possessive quantifiers: `*+`, `++`

**This works with the default engine (no PCRE2 needed):**

```bash
# Named capture groups work in the default engine
rg "(?P<name>func \w+)" -o
```

**This REQUIRES `-P` (PCRE2):**

```bash
# Lookbehind requires PCRE2
rg -P "(?<=func )\w+(?=\()" -o

# Negative lookbehind: functions NOT starting with "New"
rg -P "(?<=func )(?!New)\w+(?=\()" -o
```

**Performance note:** The default engine uses finite automata and runs in guaranteed
linear time O(n). PCRE2 uses backtracking and can be slower on pathological patterns.
Only use `-P` when you actually need its extra features.

Check if your rg was compiled with PCRE2 support:

```bash
rg --pcre2-version
```

---

### Exercise 11 -- Counting matches

Count how many times "error" appears in each file (case-insensitive).

**grep:**

```bash
grep -rci "error" . | grep -v ":0$"
```

grep's `-c` outputs every file including those with 0 matches, so you must filter.

**rg:**

```bash
rg -c -i error
```

```text
cmd/webshop/main.go:5
internal/auth/auth.go:4
internal/database/postgres.go:3
internal/handlers/user_handler.go:3
internal/handlers/order_handler.go:2
internal/models/user.go:4
internal/models/order.go:2
internal/config/config.go:2
pkg/errors/errors.go:14
pkg/logger/logger.go:3
pkg/middleware/middleware.go:2
test/integration/api_test.go:3
...
```

**Why rg wins:** Only files with matches are shown (no `:0` noise). Combine with `--sort` to
see which files have the most matches:

```bash
rg -c -i error --sort none | sort -t: -k2 -rn | head -10
```

---

### Exercise 12 -- Putting it all together

Try these progressively harder searches:

**12a. Find all hardcoded string literals (quoted strings in Go code):**

```bash
rg '"[^"]*"' -t go -o | head -30
```

**12b. Find all struct definitions with their names:**

```bash
rg "type \w+ struct" -t go
```

```text
internal/models/user.go:22:type User struct {
internal/models/user.go:40:type CreateUserRequest struct {
internal/models/user.go:52:type UpdateUserRequest struct {
internal/models/product.go:22:type Product struct {
internal/models/product.go:41:type CreateProductRequest struct {
internal/models/order.go:25:type Order struct {
internal/models/order.go:46:type OrderItem struct {
internal/config/config.go:13:type Config struct {
internal/config/config.go:24:type DatabaseConfig struct {
...
```

**12c. Find all HTTP status codes used in handler files:**

```bash
rg "http\.Status\w+" -o internal/handlers/
```

```text
internal/handlers/user.go:27:http.StatusMethodNotAllowed
internal/handlers/user.go:34:http.StatusBadRequest
internal/handlers/user.go:58:http.StatusOK
internal/handlers/user.go:88:http.StatusOK
internal/handlers/user.go:118:http.StatusCreated
...
```

**12d. Find all `fmt.Errorf` calls that use `%w` (error wrapping) vs `%v` (no wrapping):**

```bash
# Wrapped errors (correct Go 1.13+ practice)
rg 'fmt\.Errorf.*%w' -t go

# Non-wrapped errors (potential improvement targets)
rg 'fmt\.Errorf.*%[^w]' -t go
```

---

## Cheat Sheet

| Task | `grep` | `rg` |
|---|---|---|
| Recursive search | `grep -r "pattern" .` | `rg "pattern"` |
| With line numbers | `grep -rn "pattern" .` | `rg "pattern"` (default) |
| Extended regex | `grep -rE "pat\w+"` | `rg "pat\w+"` (default) |
| Case insensitive | `grep -ri "pat"` | `rg -i "pat"` |
| Smart case | N/A | `rg -S "pat"` |
| Word boundary | `grep -rw "word"` | `rg -w "word"` |
| Fixed string | `grep -rF "lit("` | `rg -F "lit("` |
| Invert match | `grep -rv "^$"` | `rg -v "^$"` |
| Multiple patterns | `grep -rE "a\|b\|c"` | `rg "a\|b\|c"` or `rg -e a -e b -e c` |
| Lookahead/behind | N/A on macOS | `rg -P "(?<=x)y"` |
| Count matches | `grep -rc "pat" \| grep -v :0` | `rg -c "pat"` |
| Only match | `grep -oE "pat"` | `rg -o "pat"` |
