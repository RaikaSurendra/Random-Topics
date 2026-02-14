# Chapter 7: sd Advanced Patterns

## Overview

Chapter 6 covered `sd` basics: simple replacements, preview vs in-place, and the
escaping advantage over `sed`. This chapter goes deeper. You will learn complex regex
replacements, multiline editing, capture group tricks, working with struct tags, and
professional workflows for safe bulk modifications. Every example uses the webshop
Go codebase.

All commands assume you are running from the project root:

```bash
cd /Users/surendraraika/projects/Random/rg
```

**Tool versions used:** sd 1.0.0, rg 15.1.0

---

## Concepts

### 1. Complex Regex Replacements

`sd` supports the full Rust regex syntax (similar to PCRE but without backreference
in the search pattern). You can build expressive patterns:

**Replace equality checks with method calls:**

```bash
# sed equivalent (extended regex, and escaping pain):
sed -E 's/status\s*==\s*"(\w+)"/status.Is(\1)/g' internal/models/order.go

# sd version:
sd 'status\s*==\s*"(\w+)"' 'status.Is($1)' < internal/models/order.go
```

**Replace hardcoded HTTP status codes with constants:**

```bash
# sed:
sed 's/http\.StatusOK) \/\/ 200/http.StatusOK)/g' internal/handlers/user.go

# sd:
sd 'http\.StatusOK\) // 200' 'http.StatusOK)' < internal/handlers/user.go
```

**Replace raw string comparisons with a helper function:**

```bash
sd 'r\.Method != http\.Method(\w+)' '!isMethod(r, http.Method$1)' < internal/handlers/user.go
```

Expected: `if r.Method != http.MethodGet` becomes `if !isMethod(r, http.MethodGet)`.

### 2. Multiline Replacements

`sd` handles multiline patterns natively. The `\n` in your pattern matches actual
newlines in the file. This is where `sed` becomes almost unusable.

**Replace a simple error return with a wrapped error:**

```bash
# sed multiline is extremely painful (requires N;P;D or hold space tricks)
# sd just works:
sd 'if err != nil \{\n\t\treturn err\n\t\}' 'if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}' < internal/models/user.go
```

**Replace a multi-line debug block:**

```bash
sd 'start := time\.Now\(\)\n\tdefer func\(\) \{\n\t\telapsed := time\.Since\(start\)\n\t\tfmt\.Printf\("\[DEBUG\] QueryRow took %v: %s\\n", elapsed, query\)\n\t\}' 'start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		log.Debug("QueryRow duration=%v query=%s", elapsed, query)
	}' < internal/database/postgres.go
```

**Important notes on multiline sd:**
- Use `\n` in the search pattern to match newlines
- In the replacement, you can use actual newlines (literal multi-line string)
- Tab indentation matters -- match the exact whitespace in the search pattern

### 3. Working with Struct Tags

Go struct tags are a common target for bulk edits. The webshop codebase intentionally
mixes `snake_case` and `camelCase` JSON tags for practice.

**Find all snake_case JSON tags:**

```bash
rg 'json:"\w+_\w+"' internal/models/
```

Expected output:
```
internal/models/user.go:    FirstName   string    `json:"first_name" ...
internal/models/user.go:    Phone       string    `json:"phone_number" ...
internal/models/user.go:    IsActive    bool      `json:"is_active" ...
internal/models/user.go:    CreatedAt   time.Time `json:"created_at" ...
...
```

**Convert single-underscore snake_case to camelCase:**

```bash
# Preview: convert json:"first_name" to json:"firstName"
sd 'json:"(\w+)_(\w)"' 'json:"$1\u$2"' < internal/models/user.go
```

Wait -- `sd` does not support `\u` (uppercase transform) like some `sed` versions.
Instead, you handle these with targeted replacements:

```bash
# Target each specific tag individually
sd 'json:"first_name"' 'json:"firstName"' < internal/models/user.go
sd 'json:"phone_number"' 'json:"phoneNumber"' < internal/models/user.go
sd 'json:"is_active"' 'json:"isActive"' < internal/models/user.go
sd 'json:"created_at"' 'json:"createdAt"' < internal/models/user.go
```

For bulk automated case conversion, use a script approach:

```bash
# Find all snake_case tags, then generate sd commands
rg -o 'json:"(\w+)_(\w+)"' internal/models/ --no-filename | sort -u
```

Expected output:
```
json:"compare_price"
json:"created_at"
json:"first_name"
json:"is_active"
json:"is_published"
json:"phone_number"
...
```

### 4. Backreference Tricks

Capture groups can be reordered, duplicated, and combined with literal text:

**Reorder captures -- swap first and last name fields:**

```bash
echo 'Name: John Smith' | sd '(\w+): (\w+) (\w+)' '$1: $3, $2'
```

Expected output:
```
Name: Smith, John
```

**Duplicate matched text -- add a comment with the original value:**

```bash
sd '(defaultPort\s*=\s*)(":8080")' '$1":9090" // was $2' < cmd/webshop/main.go
```

Expected: `defaultPort = ":9090" // was ":8080"`

**Wrap a value with a function call:**

```bash
sd '(Password:\s*)"([^"]+)"' '$1os.Getenv("DB_PASSWORD") // was "$2"' < internal/config/config.go
```

### 5. Lookahead and Lookbehind

`sd` uses the Rust `regex` crate. As of sd 1.0, the default regex engine does **not**
support lookahead or lookbehind assertions. This is a known limitation.

**Workarounds:**

Instead of lookahead, capture the context and include it in the replacement:

```bash
# Instead of: sd '(?<=func )\w+Handler' 'MyHandler'
# Use capture groups:
sd '(func )(\w+)(Handler)' '${1}My${3}' < internal/handlers/user.go
```

Instead of negative lookahead, use `rg` to pre-filter:

```bash
# Find "return err" but NOT "return err}" -- use rg to filter, then sd to replace
rg 'return err$' -t go -l | xargs sd 'return err$' 'return fmt.Errorf("failed: %w", err)'
```

### 6. Replacing in Specific File Types Only

Combine `rg` file filtering with `sd` for type-safe replacements:

```bash
# Only replace in Go files (skip configs, SQL, etc.)
rg -t go -l 'fmt\.Println' | xargs sd 'fmt\.Println' 'log.Println'

# Only replace in model files
rg -g 'internal/models/*.go' -l 'errors\.New' | xargs sd 'errors\.New' 'fmt.Errorf'
```

**sed equivalent (much harder to scope):**
```bash
# sed has no built-in file type filtering -- you need find + exec
find internal/models -name "*.go" -exec sed -i '' 's/errors\.New/fmt.Errorf/g' {} \;
```

### 7. Conditional-Like Replacements Using Multiple Passes

`sd` does not have conditional logic, but you can achieve conditional effects with
multiple targeted passes:

```bash
# Pass 1: Replace fmt.Println("[ERROR]...") with log.Error()
sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < cmd/webshop/main.go

# Pass 2: Replace fmt.Println("[WARN]...") with log.Warn()
sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")' < cmd/webshop/main.go

# Pass 3: Replace fmt.Println("[INFO]...") with log.Info()
sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' < cmd/webshop/main.go

# Pass 4: Replace fmt.Println("[DEBUG]...") with log.Debug()
sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")' < cmd/webshop/main.go
```

**Full pipeline for applying all passes to a file:**

```bash
sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < cmd/webshop/main.go | \
  sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")' | \
  sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' | \
  sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")'
```

This chains `sd` through stdin pipes -- each pass processes the output of the previous.

### 8. Adding and Removing Lines

Replace text with content that includes `\n` to add lines:

**Add a line after every error return:**

```bash
sd '(return fmt\.Errorf\(.+\))' '$1\n\t// TODO: add metrics counter here' < internal/database/postgres.go
```

**Remove a line by replacing it (plus its newline) with nothing:**

```bash
# Remove all lines containing "// HACK:"
sd '\t// HACK:.+\n' '' < internal/config/config.go
```

**Add an import statement:**

```bash
sd '(\t"fmt"\n)' '$1\t"log"\n' < cmd/webshop/main.go
```

### 9. Handling Special Characters in Replacements

The replacement string in `sd` is literal except for `$` (used for capture groups).

**To insert a literal `$` in the replacement:**

```bash
# Double the $ to escape it
echo "price: 100" | sd 'price: (\d+)' 'price: $$$1 USD'
```

Expected output:
```
price: $100 USD
```

**Backslashes in replacement are literal:**

```bash
echo "path/to/file" | sd 'path/to/file' 'C:\Users\docs\file'
```

Expected output:
```
C:\Users\docs\file
```

No escaping needed. With `sed`, you would need `C:\\Users\\docs\\file`.

**Newlines in replacement:**

```bash
echo "one line" | sd 'one line' 'line one\nline two'
```

Expected output:
```
line one
line two
```

### 10. Dry Run Workflow

Since sd 1.0 modifies files in place by default, build a safe dry-run habit:

```bash
# Step 1: Create backup
cp internal/models/user.go internal/models/user.go.bak

# Step 2: Apply replacement
sd 'json:"first_name"' 'json:"firstName"' internal/models/user.go

# Step 3: Review the diff
diff internal/models/user.go.bak internal/models/user.go
```

Expected diff output:
```
27c27
<     FirstName   string    `json:"first_name" db:"first_name" validate:"required"`
---
>     FirstName   string    `json:"firstName" db:"first_name" validate:"required"`
```

```bash
# Step 4: If wrong, restore
cp internal/models/user.go.bak internal/models/user.go

# Step 5: Clean up backup
rm internal/models/user.go.bak
```

For git-tracked projects, `git diff` and `git checkout` replace the manual backup:

```bash
# Apply change
sd 'json:"first_name"' 'json:"firstName"' internal/models/user.go

# Review
git diff internal/models/user.go

# Revert if wrong
git checkout internal/models/user.go
```

### 11. Batch Operations with xargs

The `rg -l` + `xargs sd` pattern is the workhorse for project-wide changes:

```bash
# Replace all occurrences of OldFunc with NewFunc across Go files
rg -l 'OldFunc' -t go | xargs sd 'OldFunc' 'NewFunc'
```

**Handling filenames with spaces (use `-0` / `--null`):**

```bash
rg -l 'pattern' --null | xargs -0 sd 'pattern' 'replacement'
```

**Limiting concurrency with xargs:**

```bash
rg -l 'fmt\.Printf' -t go | xargs -P1 -I{} sd 'fmt\.Printf' 'log.Printf' {}
```

**Dry run for batch -- preview all changes:**

```bash
for f in $(rg -l 'fmt\.Printf' -t go); do
    echo "=== Changes in $f ==="
    diff <(cat "$f") <(sd 'fmt\.Printf' 'log.Printf' < "$f")
done
```

---

## Exercises

### Exercise 1: Complex Status Pattern

Preview replacing all raw string status comparisons with method calls:

```bash
sd 'o\.Status == (\w+)' 'o.Status.Is($1)' < internal/models/order.go
```

Verify the output shows changes in `CanTransitionTo` and related methods.

### Exercise 2: Multiline Error Block

Preview wrapping the bare error return in `Validate()` functions with context:

```bash
sd 'return errors\.New\("(.+)"\)' 'return fmt.Errorf("validation: $1")' < internal/models/user.go
```

Count how many error returns would change:

```bash
rg 'return errors\.New' internal/models/user.go -c
```

### Exercise 3: Convert snake_case JSON Tags in User Model

Preview all five snake_case tag conversions for `user.go`:

```bash
sd 'json:"first_name"' 'json:"firstName"' < internal/models/user.go | \
  sd 'json:"phone_number"' 'json:"phoneNumber"' | \
  sd 'json:"is_active"' 'json:"isActive"' | \
  sd 'json:"created_at"' 'json:"createdAt"'
```

Then verify with rg that no snake_case tags remain in the output:

```bash
sd 'json:"first_name"' 'json:"firstName"' < internal/models/user.go | \
  sd 'json:"phone_number"' 'json:"phoneNumber"' | \
  sd 'json:"is_active"' 'json:"isActive"' | \
  sd 'json:"created_at"' 'json:"createdAt"' | \
  rg 'json:"\w+_\w+"'
```

If there are remaining snake_case tags, add more `sd` passes (e.g., `tax_amount`,
`stock_quantity`, etc.).

### Exercise 4: Backreference Reordering

Reorder the DSN format string parameters:

```bash
echo 'host=%s port=%d user=%s password=%s dbname=%s sslmode=%s' | \
  sd '(host=%s) (port=%d) (user=%s) (password=%s) (dbname=%s) (sslmode=%s)' '$1 $5 $3 $2 $6'
```

Expected: `host=%s dbname=%s user=%s port=%d sslmode=%s`

### Exercise 5: Multiple Pass Logging Conversion

Chain four `sd` passes to convert all bracket-prefixed log statements in main.go:

```bash
sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < cmd/webshop/main.go | \
  sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' | \
  sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")' | \
  sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")'
```

Verify that no `fmt.Println("[` patterns remain in the output:

```bash
sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < cmd/webshop/main.go | \
  sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' | \
  sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")' | \
  sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")' | \
  rg '\[(?:ERROR|INFO|DEBUG|WARN)\]'
```

### Exercise 6: Remove All HACK Comments

Preview stripping every `// HACK:` comment line from `config.go`:

```bash
sd '\s*// HACK:.+' '' < internal/config/config.go
```

Count how many HACK comments exist first:

```bash
rg '// HACK:' internal/config/config.go -c
```

### Exercise 7: Safe In-Place Edit with git

Make a real change using git as your safety net:

```bash
# Check clean state
git status internal/models/product.go

# Apply: shorten DEPRECATED functions
sd '// DEPRECATED: Use (\w+)\(\) instead\. (\w+) will be removed in v2\.0\.' '// Deprecated: use $1 instead.' internal/models/product.go

# Review the diff
git diff internal/models/product.go

# Revert
git checkout internal/models/product.go
```

### Exercise 8: Add Error Context to Bare Returns

Find all bare `return err` statements and wrap them:

```bash
# Step 1: Find them
rg 'return err$' -t go -n

# Step 2: Preview the replacement on one file
sd 'return err$' 'return fmt.Errorf("unexpected error: %w", err)' < internal/database/postgres.go
```

### Exercise 9: Batch Rename Across Models

Preview renaming the `Validate` method to `Check` across all model files:

```bash
for f in internal/models/*.go; do
    echo "=== $f ==="
    sd 'func \((\w+) \*(\w+)\) Validate\(\)' 'func ($1 *$2) Check()' < "$f" | rg 'func.*Check'
done
```

### Exercise 10: Dollar Signs in Replacement

Practice inserting literal dollar signs (common when working with price formatting):

```bash
echo 'Price: 29.99' | sd 'Price: ([\d.]+)' 'Price: $$$1 USD'
```

Expected output: `Price: $29.99 USD`

Now try it on the FormatPrice function:

```bash
sd 'Sprintf\("\$%.2f"' 'Sprintf("USD $%.2f"' < internal/models/product.go
```

### Exercise 11: Build a Replacement Script

Create a series of `sd` commands that would standardize all JSON tags in `order.go`
from snake_case to camelCase. Preview each:

```bash
sd 'json:"tax_amount"' 'json:"taxAmount"' < internal/models/order.go | \
  sd 'json:"total_amount"' 'json:"totalAmount"' | \
  sd 'json:"payment_status"' 'json:"paymentStatus"' | \
  sd 'json:"created_at"' 'json:"createdAt"' | \
  sd 'json:"shipped_at"' 'json:"shippedAt"' | \
  sd 'json:"unit_price"' 'json:"unitPrice"' | \
  sd 'json:"product_id"' 'json:"productId"' | \
  sd 'json:"stock_quantity"' 'json:"stockQuantity"'
```

### Exercise 12: Wrap All Bare Error Returns with fmt.Errorf

This is a realistic refactoring task. The goal: every `return err` in the project
should become `return fmt.Errorf("<function context>: %w", err)`.

Since `sd` cannot know the function name dynamically, use a two-step approach:

```bash
# Step 1: Find all bare returns
rg 'return err$' -t go -n

# Step 2: For each file, apply a generic wrap
sd 'return err$' 'return fmt.Errorf("operation failed: %w", err)' < internal/database/postgres.go

# Step 3: Manually refine the context string per function
sd 'return fmt\.Errorf\("operation failed: %w", err\)' 'return fmt.Errorf("transaction failed: %w", err)' < internal/database/postgres.go
```

In practice, you would review each change and adjust the context string to match
the function's purpose.

---

## Cheat Sheet

| Task | Command |
|------|---------|
| Complex regex replace | `sd 'status\s*==\s*"(\w+)"' 'status.Is($1)' < f.go` |
| Multiline match | `sd 'line1\nline2' 'replacement' < f.go` |
| Chain multiple passes | `sd 'a' 'b' < f \| sd 'c' 'd'` |
| Remove lines matching | `sd '.*HACK.*\n' '' < f.go` |
| Add a line after match | `sd '(match)' '$1\nnew line' < f.go` |
| Literal $ in replacement | `sd 'price' '$$$1' < f.go` |
| Dry run with diff | `cp f f.bak && sd 'a' 'b' f && diff f.bak f` |
| Git-safe workflow | `sd 'a' 'b' f.go && git diff f.go` |
| Batch via xargs | `rg -l 'old' -t go \| xargs sd 'old' 'new'` |
| Type-scoped replace | `rg -t go -l 'pat' \| xargs sd 'pat' 'rep'` |
| Null-delimited xargs | `rg -l 'pat' --null \| xargs -0 sd 'pat' 'rep'` |
| Preview batch changes | `for f in $(rg -l 'x'); do diff <(cat "$f") <(sd 'x' 'y' < "$f"); done` |

---

## Key Takeaways

1. **Multiline replacements are native.** Use `\n` in the search pattern. No hold
   space tricks like `sed`.
2. **No case-transform operators.** `sd` does not support `\u`, `\l`, `\U`, `\L`.
   Use targeted replacements or a script for case conversion.
3. **No lookahead/lookbehind.** Work around this with capture groups that include
   the surrounding context.
4. **Chain `sd` via pipes for multi-pass edits.** Each pass through stdin feeds the
   next.
5. **Use git as your undo.** `git diff` and `git checkout` are safer than manual
   backup files.
6. **Batch operations need `rg -l | xargs sd`.** This is the standard pattern for
   project-wide changes.

---

Next: [Chapter 8 -- Power Combos: rg + sd Together](../08-rg-sd-power-combos/README.md)
