# Chapter 6: sd Fundamentals -- The Modern sed

## Overview

`sd` is a modern, intuitive replacement for `sed`. Built in Rust, it eliminates the
most painful parts of `sed`: the escaping nightmare, the cryptic syntax, and the
endless slashes. This chapter teaches you `sd` from the ground up, using the webshop
Go codebase for every example. By the end, you will never reach for `sed` again for
simple find-and-replace operations.

All commands assume you are running from the project root:

```bash
cd /Users/surendraraika/projects/Random/rg
```

**Tool versions used:** sd 1.0.0, rg 15.1.0

---

## Concepts

### 1. Why sd? The Escaping Problem

With `sed`, replacing a string that contains dots, slashes, or special characters
turns into an escaping nightmare. Compare:

**sed -- replacing a Go error function call:**
```bash
# sed requires escaping the dots
sed 's/fmt\.Errorf/errors\.Wrap/g' internal/models/user.go
```

**sd -- the same replacement:**
```bash
# sd search uses regex, but this is already far cleaner
sd 'fmt\.Errorf' 'errors.Wrap' < internal/models/user.go
```

The search pattern in `sd` uses regex (so `.` still needs escaping in the search),
but the replacement string is **literal**. No escaping needed on the right side.

### 2. The Real Killer Feature: No Delimiter Escaping

The biggest win is with strings containing slashes. This is where `sed` completely
falls apart:

**sed -- replacing a URL:**
```bash
sed 's/http:\/\/localhost:8080\/api\/v1/http:\/\/localhost:9090\/api\/v2/g' cmd/webshop/main.go
```

**sd -- the same replacement:**
```bash
sd 'http://localhost:8080/api/v1' 'http://localhost:9090/api/v2' < cmd/webshop/main.go
```

With `sd`, slashes are just characters. No `\/` escaping. The command reads exactly
like what you mean.

### 3. Basic Replacement via stdin

The simplest `sd` usage pipes text through stdin:

```bash
echo "hello world" | sd "hello" "goodbye"
```

Expected output:
```
goodbye world
```

Try it with real project output:

```bash
echo 'defaultPort = ":8080"' | sd '8080' '9090'
```

Expected output:
```
defaultPort = ":9090"
```

### 4. Preview Changes Without Modifying Files

To preview what `sd` would do to a file, redirect stdin from the file:

```bash
sd 'fmt\.Println' 'log.Println' < cmd/webshop/main.go
```

This prints the entire file with replacements applied to stdout. The original file
is **untouched**. This is your safety net -- always preview first.

### 5. In-Place File Editing

To actually modify a file, pass it as an argument:

```bash
sd 'old_text' 'new_text' cmd/webshop/main.go
```

This changes the file directly. There is no undo (no `.bak` file by default).

### 6. sd 1.0 Syntax: In-Place Is the Default

This is critical to understand. In sd 1.0:

| Syntax | Behavior |
|--------|----------|
| `sd "old" "new" file.go` | **Modifies file in place** |
| `sd "old" "new" < file.go` | **Preview only** (prints to stdout) |
| `echo "text" \| sd "old" "new"` | **Stdin/stdout** (no file modified) |

Earlier versions of `sd` required `-i` for in-place editing. In sd 1.0, when you
provide a file argument, in-place is the default. The `-i` flag no longer exists.

**Always preview before applying:**

```bash
# Step 1: Preview
sd 'fmt\.Println' 'log.Info' < internal/database/postgres.go

# Step 2: If it looks right, apply
sd 'fmt\.Println' 'log.Info' internal/database/postgres.go
```

### 7. Regex Support

`sd` uses the Rust regex engine for the search pattern. This gives you a powerful,
modern regex dialect:

```bash
# Match function declarations and rename the prefix
echo 'func HandleUser() {}' | sd 'func (\w+)' 'func My$1'
```

Expected output:
```
func MyHandleUser() {}
```

Real example -- find and rename handler constructors:

```bash
sd 'func New(\w+Handler)' 'func Create$1' < internal/handlers/user.go
```

### 8. Capture Groups

Capture groups use `$1`, `$2`, etc. in the replacement. Use `${1}` when followed
by digits or letters that could be ambiguous:

```bash
# Rename FooError to FooErr across the codebase
echo 'InvalidTokenError = errors.New("invalid")' | sd '(\w+)Error' '${1}Err'
```

Expected output:
```
InvalidTokenErr = errors.New("invalid")
```

Real example with the project:

```bash
# Preview: shorten Error suffix to Err in auth package
sd '(\w+)Token' '${1}Tkn' < internal/auth/auth.go
```

### 9. The Escaping Comparison Deep-Dive

This is the core reason `sd` exists. Here is the full comparison:

**Characters you must escape in sed:**

| Character | sed search | sed replacement | sd search | sd replacement |
|-----------|-----------|----------------|-----------|----------------|
| `/` | `\/` | `\/` | `/` | `/` |
| `.` | `\.` | `.` | `\.` | `.` |
| `*` | `\*` | `*` | `\*` | `*` |
| `+` | `\+` | `+` | `+` | `+` |
| `(`, `)` | `\(`, `\)` | `\(`, `\)` | `(`, `)` | `(`, `)` |
| `{`, `}` | `\{`, `\}` | `\{`, `\}` | `{`, `}` | `{`, `}` |
| `$` | `\$` | `\$` or `&` | `$` | `$` (only for groups) |

**sd rule of thumb:**
- Search pattern = regex (escape regex metacharacters like `.`, `*`, `+`, `?`, etc.)
- Replacement string = **literal** (only `$1`, `$2` etc. are special for groups)

**Five real examples where sd is dramatically cleaner:**

**Example 1: Replace a file path**
```bash
# sed
sed 's/configs\/app\.yaml/configs\/config\.yaml/g' cmd/webshop/main.go

# sd
sd 'configs/app\.yaml' 'configs/config.yaml' < cmd/webshop/main.go
```

**Example 2: Replace a DSN connection string**
```bash
# sed
sed 's/host=%s port=%d user=%s password=%s/host=%s port=%d user=%s/g' internal/database/postgres.go

# sd - the replacement is literal, exactly what you type
sd 'host=%s port=%d user=%s password=%s' 'host=%s port=%d user=%s' < internal/database/postgres.go
```

**Example 3: Replace a JSON error response**
```bash
# sed - the curly braces, quotes, and colons make this painful
sed 's/{"error":"unauthorized","message":"missing token"}/{"error":"unauthenticated","code":401}/g' internal/auth/auth.go

# sd - just type what you mean
sd '{"error":"unauthorized","message":"missing token"}' '{"error":"unauthenticated","code":401}' < internal/auth/auth.go
```

**Example 4: Replace an import path**
```bash
# sed
sed 's/github\.com\/learn\/rg-sd-mastery/github\.com\/myorg\/webshop/g' cmd/webshop/main.go

# sd
sd 'github\.com/learn/rg-sd-mastery' 'github.com/myorg/webshop' < cmd/webshop/main.go
```

**Example 5: Replace a regex pattern containing brackets**
```bash
# sed - escaping brackets inside a sed substitution
sed 's/\[ERROR\] Configuration/\[FATAL\] Configuration/g' cmd/webshop/main.go

# sd
sd '\[ERROR\] Configuration' '[FATAL] Configuration' < cmd/webshop/main.go
```

### 10. Multiple Files

Pass multiple file paths to apply the same replacement across all of them:

```bash
sd 'fmt\.Println' 'log.Info' internal/database/postgres.go internal/models/user.go internal/models/product.go
```

This modifies all three files in place.

### 11. Piping with rg for Targeted Replacements

Use `rg -l` to find files containing a pattern, then pipe to `xargs sd`:

```bash
# Find all Go files containing fmt.Println, then replace
rg -l 'fmt\.Println' -t go | xargs sd 'fmt\.Println' 'log.Info'
```

This is the beginning of the `rg + sd` power workflow covered in depth in Chapter 8.

---

## Exercises

Work through these from top to bottom. Each builds on the previous.

### Exercise 1: First Replacement (Preview Only)

Preview replacing the default port constant in `cmd/webshop/main.go`:

```bash
sd '":8080"' '":3000"' < cmd/webshop/main.go
```

Verify the output shows `defaultPort = ":3000"` on the relevant line. Confirm the
file itself is unchanged:

```bash
rg '":8080"' cmd/webshop/main.go
```

Expected: the original `:8080` is still there.

### Exercise 2: stdin Pipe Replacement

Take the output of `rg` and transform it:

```bash
rg 'TODO:' cmd/webshop/main.go | sd 'TODO:' 'ACTION_REQUIRED:'
```

Expected: each TODO line from main.go is printed with `ACTION_REQUIRED:` instead.

### Exercise 3: Preview a Multi-File Change

Preview what a logging change would look like across the project:

```bash
sd '\[ERROR\]' '[ERR]' < cmd/webshop/main.go
sd '\[ERROR\]' '[ERR]' < internal/database/postgres.go
```

Compare the output of each to see which files have `[ERROR]` markers.

### Exercise 4: Regex with Capture Groups

Rename all `OrderStatus` constants to use a shorter prefix. Preview first:

```bash
sd 'OrderStatus(\w+)' 'OrdStat${1}' < internal/models/order.go
```

Verify that `OrderStatusPending` becomes `OrdStatPending`, `OrderStatusConfirmed`
becomes `OrdStatConfirmed`, etc.

### Exercise 5: Replace an Import Path (Preview)

Preview changing the module path across the entire main.go:

```bash
sd 'github\.com/learn/rg-sd-mastery' 'github.com/acme/webshop-api' < cmd/webshop/main.go
```

Count how many import lines changed in the output.

### Exercise 6: In-Place Replacement (With Backup)

Make a real change. First create a backup, then apply, then verify:

```bash
# Backup
cp internal/models/order.go internal/models/order.go.bak

# Apply: fix the tax rate comment
sd 'Hardcoded 8\.5% tax rate' 'Hardcoded 8.5% tax rate - TODO: make configurable' internal/models/order.go

# Verify the change
rg 'TODO: make configurable' internal/models/order.go

# Restore from backup
cp internal/models/order.go.bak internal/models/order.go
rm internal/models/order.go.bak
```

### Exercise 7: Replace JSON Error Messages

Preview replacing all error response strings in the handlers:

```bash
sd '"method not allowed"' '"http method not supported"' < internal/handlers/user.go
```

Count how many replacements occurred by comparing line counts:

```bash
rg '"method not allowed"' internal/handlers/user.go -c
```

### Exercise 8: Escaping Practice -- sed vs sd

Try both approaches for replacing the DSN format string. First with sed:

```bash
sed 's/host=%s port=%d user=%s password=%s dbname=%s sslmode=%s/host=%s port=%d dbname=%s sslmode=%s/g' internal/database/postgres.go
```

Now the same with sd:

```bash
sd 'host=%s port=%d user=%s password=%s dbname=%s sslmode=%s' 'host=%s port=%d dbname=%s sslmode=%s' < internal/database/postgres.go
```

Notice how the sd version reads naturally while the sed version requires mental
parsing of escape sequences.

### Exercise 9: Multiple File Replacement (Preview)

Find all files with `fmt.Println` and preview the replacement for each:

```bash
for f in $(rg -l 'fmt\.Println' -t go); do
    echo "=== $f ==="
    sd 'fmt\.Println' 'log.Println' < "$f" | rg 'log\.Println'
done
```

This shows you exactly which lines would change in each file.

### Exercise 10: Replace Hardcoded Localhost

Preview replacing all hardcoded `localhost` references with a config variable:

```bash
sd '"localhost"' 'cfg.Database.Host' < internal/config/config.go
```

Then try the more complex URL version:

```bash
sd 'http://localhost:3000' 'cfg.CORS.AllowedOrigins[0]' < internal/config/config.go
```

### Exercise 11: Rename a Struct Field Tag Pattern

Preview converting the `json:"first_name"` tag to `json:"firstName"`:

```bash
sd 'json:"first_name"' 'json:"firstName"' < internal/models/user.go
```

Check how many struct tags use snake_case:

```bash
rg 'json:"\w+_\w+"' internal/models/ -c
```

### Exercise 12: Full Workflow -- Find, Preview, Apply, Verify

Replace all `fmt.Printf("[DEBUG]` statements with structured logging across the
entire project. Follow the full safety workflow:

```bash
# Step 1: Find all occurrences
rg 'fmt\.Printf\("\[DEBUG\]' -t go

# Step 2: Count the impact
rg 'fmt\.Printf\("\[DEBUG\]' -t go -c

# Step 3: Preview one file
sd 'fmt\.Printf\("\[DEBUG\] (.+)\\n"' 'log.Debug("$1"' < internal/database/postgres.go

# Step 4: Apply to all files (only if preview looks correct)
# rg -l 'fmt\.Printf\("\[DEBUG\]' -t go | xargs sd 'fmt\.Printf\("\[DEBUG\] (.+)\\n"' 'log.Debug("$1"'

# Step 5: Verify no old pattern remains
# rg 'fmt\.Printf\("\[DEBUG\]' -t go
```

Note: Steps 4 and 5 are commented out. Only uncomment them if you want to actually
modify the project files.

---

## Cheat Sheet

| Task | Command |
|------|---------|
| Basic stdin replacement | `echo "text" \| sd "old" "new"` |
| Preview file changes | `sd "old" "new" < file.go` |
| In-place file edit (sd 1.0) | `sd "old" "new" file.go` |
| Multiple files in-place | `sd "old" "new" file1.go file2.go` |
| Regex capture group | `sd '(\w+)Error' '${1}Err' < file.go` |
| Numbered back-reference | `sd '(foo)(bar)' '$2$1' < file.go` |
| Literal dots in search | `sd 'fmt\.Println' 'log.Println' < file.go` |
| Slashes (no escaping!) | `sd 'path/to/old' 'path/to/new' < file.go` |
| Pipe from rg | `rg -l "pattern" \| xargs sd "pattern" "replacement"` |
| Preview with diff | `cp f.go f.bak && sd "old" "new" f.go && diff f.bak f.go` |

---

## Key Takeaways

1. **sd search = regex, sd replacement = literal.** This is the single most important
   thing to remember.
2. **In sd 1.0, file arguments mean in-place editing.** Use `< file` for preview.
3. **No delimiter escaping.** Slashes, colons, dots in the replacement are just
   characters.
4. **Always preview before applying.** Use `sd "old" "new" < file` first, then
   `sd "old" "new" file` to commit.
5. **Combine with rg for surgical precision.** `rg -l` finds the files, `sd` does
   the replacement. This workflow is covered in depth in Chapter 8.

---

Next: [Chapter 7 -- sd Advanced Patterns](../07-sd-advanced/README.md)
