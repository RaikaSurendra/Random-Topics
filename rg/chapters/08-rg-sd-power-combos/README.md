# Chapter 8: Power Combos -- rg + sd Together

## Overview

`rg` finds. `sd` replaces. Together they form the most powerful text refactoring
pipeline available on the command line. This chapter teaches you the disciplined
workflow of **search, verify, replace, verify** that makes project-wide changes safe
and predictable. Every pattern here uses the webshop Go codebase as the target.

This is the key chapter of the series. Everything from Chapters 1-7 converges here.

All commands assume you are running from the project root:

```bash
cd /Users/surendraraika/projects/Random/rg
```

**Tool versions used:** sd 1.0.0, rg 15.1.0

---

## Concepts

### 1. The Fundamental Workflow

Every refactoring follows this four-step discipline:

```bash
# Step 1: FIND -- what matches?
rg "OldPattern" -t go

# Step 2: SCOPE -- which files are affected?
rg -l "OldPattern" -t go

# Step 3: REPLACE -- apply the change
rg -l "OldPattern" -t go | xargs sd "OldPattern" "NewPattern"

# Step 4: VERIFY -- confirm nothing old remains
rg "OldPattern" -t go
```

Step 4 should return zero results. If it returns matches, something went wrong --
your regex did not cover all variations.

**Real example -- rename a constant prefix:**

```bash
# Step 1: Find all OrderStatus references
rg "OrderStatus" -t go

# Expected output:
# internal/models/order.go:    OrderStatusPending    OrderStatus = "pending"
# internal/models/order.go:    OrderStatusConfirmed  OrderStatus = "confirmed"
# internal/models/order.go:    OrderStatusProcessing OrderStatus = "processing"
# ... (many more lines)

# Step 2: Which files?
rg -l "OrderStatus" -t go

# Expected output:
# internal/models/order.go

# Step 3: Preview first
sd 'OrderStatus' 'OrdStatus' < internal/models/order.go | head -25

# Step 4: Verify in preview output
sd 'OrderStatus' 'OrdStatus' < internal/models/order.go | rg 'OrderStatus'
# Should return nothing
```

### 2. The Safe Refactoring Pipeline

For production codebases, add impact analysis before replacing:

```bash
# Step 1: Preview what the change looks like
rg "OldName" -r "NewName" -t go

# Step 2: Count the impact per file
rg -c "OldName" -t go | sort -t: -k2 -n -r

# Expected output:
# internal/models/order.go:15
# cmd/webshop/main.go:3
# internal/handlers/user.go:1

# Step 3: Apply
rg -l "OldName" -t go | xargs sd "OldName" "NewName"

# Step 4: Verify -- zero matches expected
rg "OldName" -t go

# Step 5: Run tests (if applicable)
# go test ./...
```

**The key insight:** `rg -r` previews in stdout (never modifies files). `sd` with
file arguments modifies in place. Use `rg -r` for quick visual checks, then `sd`
for the actual change.

### 3. Type-Scoped Replacement

Replace only in specific file types:

```bash
# Only Go files
rg -t go -l "fmt\.Println" | xargs sd 'fmt\.Println' 'log.Println'

# Only SQL files (if they existed)
rg -t sql -l "SELECT \*" | xargs sd 'SELECT \*' 'SELECT id, name, email'
```

**Why this matters:** Your project might have the same string in Go code, YAML
configs, and SQL scripts. Type scoping ensures you only change what you intend.

```bash
# Example: "localhost" appears in Go code AND config
rg "localhost"

# Expected output:
# internal/config/config.go:            Host:            "localhost",
# internal/config/config.go:            AllowedOrigins: []string{"http://localhost:3000", ...
# cmd/webshop/main.go:    // TODO: ... (possibly)

# Replace only in Go code, not in config files
rg -t go -l "localhost" | xargs sd '"localhost"' 'os.Getenv("DB_HOST")'
```

### 4. Glob-Scoped Replacement

Use glob patterns for finer control than type filters:

```bash
# Replace in all Go files EXCEPT test files
rg -g '!*_test.go' -l 'fmt\.Printf' -t go | xargs sd 'fmt\.Printf' 'log.Printf'

# Replace only in model files
rg -g 'internal/models/*.go' -l 'errors\.New' | xargs sd 'errors\.New' 'fmt.Errorf'

# Replace only in handler files
rg -g 'internal/handlers/*.go' -l 'http\.Error' | xargs sd 'http\.Error' 'respondError'
```

**sed equivalent (much more verbose):**
```bash
find internal/models -name "*.go" ! -name "*_test.go" -exec sed -i '' 's/errors\.New/fmt.Errorf/g' {} \;
```

### 5. Directory-Scoped Replacement

Limit replacements to a specific directory tree:

```bash
# Only replace in the handlers directory
rg -l "fmt\.Fprintf" internal/handlers/ | xargs sd 'fmt\.Fprintf' 'json.NewEncoder'

# Only replace in the models directory
rg -l "errors\.New" internal/models/ | xargs sd 'errors\.New\("(.+)"\)' 'fmt.Errorf("$1")'

# Only replace in cmd/
rg -l 'fmt\.Println' cmd/ | xargs sd 'fmt\.Println' 'log.Println'
```

### 6. Complex Multi-Step Refactoring: Rename a Service

Here is a complete, realistic refactoring. Rename `UserService` to `AccountService`
throughout the codebase.

```bash
# Step 1: Survey the impact
rg "UserService" -t go

# Expected output:
# internal/handlers/user.go:    service *service.UserService
# internal/handlers/user.go:func NewUserHandler(svc *service.UserService) *UserHandler {
# internal/handlers/user_handler.go:    userService *service.UserService
# internal/handlers/user_handler.go:func NewUserHandler(svc *service.UserService) *UserHandler {
# cmd/webshop/main.go:  (referenced via handlers)

# Step 2: Count per file
rg -c "UserService" -t go | sort -t: -k2 -n -r

# Step 3: Preview the rename
rg "UserService" -r "AccountService" -t go

# Step 4: Apply -- rename the type reference
rg -l "UserService" -t go | xargs sd 'UserService' 'AccountService'

# Step 5: Verify
rg "UserService" -t go
# Should return empty

# Step 6: Also rename the handler struct to match
rg -l "UserHandler" -t go | xargs sd 'UserHandler' 'AccountHandler'

# Step 7: Rename the constructor
rg -l "NewUserHandler" -t go | xargs sd 'NewUserHandler' 'NewAccountHandler'

# Step 8: Update variable names
rg -l "userHandler" -t go | xargs sd 'userHandler' 'accountHandler'

# Step 9: Final verification -- no "User" references in handler context
rg "userHandler|UserHandler|UserService" -t go
# Should return empty

# Step 10: Revert everything (this was a demo)
git checkout .
```

### 7. Extract-Transform-Load Pattern

Use `rg -o` to extract unique values, analyze them, then make targeted replacements:

```bash
# Extract all unique log level patterns
rg -o '\[(ERROR|WARN|INFO|DEBUG)\]' --no-filename -t go | sort | uniq -c | sort -rn

# Expected output:
#    8 [DEBUG]
#    5 [INFO]
#    3 [ERROR]
#    2 [WARN]

# Extract all unique JSON tag names
rg -o 'json:"(\w+)"' --no-filename -t go -r '$1' | sort -u

# Expected output:
# avatarUrl
# categoryId
# compare_price
# costPrice
# created_at
# ...

# Now make targeted replacements based on what you found
# Replace the most common pattern first
rg -l '\[DEBUG\]' -t go | xargs sd '\[DEBUG\]' '[DBG]'
```

### 8. Combining rg --json with sd for Surgical Replacements

Use `rg --json` to get structured data about matches, then target specific files:

```bash
# Get JSON output with file and line info
rg --json 'fmt\.Println' -t go | head -20

# Parse to get just the filenames with match counts
rg --json 'fmt\.Println' -t go | \
  rg '"type":"match"' | \
  sd '.*"path":\{"text":"([^"]+)"\}.*' '$1' | \
  sort | uniq -c | sort -rn

# Use this to decide which files to target
# Then apply sd only to high-impact files
sd 'fmt\.Println' 'log.Println' internal/database/postgres.go
sd 'fmt\.Println' 'log.Println' internal/models/user.go
```

### 9. Mass Import Path Updates

One of the most common refactoring tasks: changing the Go module path.

```bash
# Step 1: Find all import references
rg 'github\.com/learn/rg-sd-mastery' -t go

# Expected output:
# cmd/webshop/main.go:    "github.com/learn/rg-sd-mastery/internal/auth"
# cmd/webshop/main.go:    "github.com/learn/rg-sd-mastery/internal/config"
# cmd/webshop/main.go:    "github.com/learn/rg-sd-mastery/internal/database"
# cmd/webshop/main.go:    "github.com/learn/rg-sd-mastery/internal/handlers"
# cmd/webshop/main.go:    "github.com/learn/rg-sd-mastery/pkg/logger"
# cmd/webshop/main.go:    "github.com/learn/rg-sd-mastery/pkg/middleware"
# internal/database/postgres.go:    "github.com/learn/rg-sd-mastery/internal/config"
# internal/handlers/user.go:    "github.com/learn/rg-sd-mastery/internal/models"
# internal/handlers/user.go:    "github.com/learn/rg-sd-mastery/internal/service"
# internal/handlers/user_handler.go:    "github.com/learn/rg-sd-mastery/internal/service"

# Step 2: Count the impact
rg -c 'github\.com/learn/rg-sd-mastery' -t go

# Step 3: Preview
rg 'github\.com/learn/rg-sd-mastery' -r 'github.com/acme-corp/webshop' -t go

# Step 4: Apply
rg -l 'github\.com/learn/rg-sd-mastery' -t go | \
  xargs sd 'github\.com/learn/rg-sd-mastery' 'github.com/acme-corp/webshop'

# Step 5: Verify
rg 'github\.com/learn/rg-sd-mastery' -t go
# Should return empty

# Step 6: Also update go.mod if it exists
# sd 'module github\.com/learn/rg-sd-mastery' 'module github.com/acme-corp/webshop' go.mod

# Step 7: Revert
git checkout .
```

**The sd advantage here:** The import path `github.com/learn/rg-sd-mastery` contains
dots and slashes. With `sed`, every slash needs escaping. With `sd`, you only escape
the dots in the search regex (since `.` is a regex metacharacter).

### 10. Config Migration Across Environments

Update configuration keys across dev, prod, and test configs simultaneously.
(The webshop project has a `configs/` directory structure for this purpose.)

```bash
# If YAML config files existed, you would:
# Step 1: Find all config references to a key
rg 'db_host' configs/

# Step 2: Replace across all config files
rg -l 'db_host' configs/ | xargs sd 'db_host' 'database_host'

# For Go config files, the same pattern works:
# Find all hardcoded config values
rg '"webshop_' -t go

# Expected output:
# internal/config/config.go:            User:            "webshop_user",
# internal/config/config.go:            Password:        "webshop_pass123",
# internal/config/config.go:            DBName:          "webshop_dev",

# Replace the DB name across environments
sd '"webshop_dev"' 'os.Getenv("DB_NAME")' < internal/config/config.go
```

### 11. The "Search, Verify, Replace, Verify" Discipline

Internalize this workflow. Here is a complete example:

**Task:** Replace all `fmt.Fprintf(w, ...)` response writes with proper JSON encoding.

```bash
# SEARCH: What are we dealing with?
rg 'fmt\.Fprintf\(w,' -t go -n

# Expected output:
# internal/handlers/user.go:30:    fmt.Fprintf(w, `{"error":"method not allowed"}`)
# internal/handlers/user.go:34:    fmt.Fprintf(w, `{"error":"missing required parameter: id"}`)
# ... (many lines)

# VERIFY: How many files and matches?
rg -c 'fmt\.Fprintf\(w,' -t go | sort -t: -k2 -n -r

# Expected:
# internal/handlers/user.go:8
# internal/handlers/user_handler.go:6
# cmd/webshop/main.go:1

# REPLACE: Apply to one file first (surgical approach)
sd 'fmt\.Fprintf\(w, `\{"error":"([^"]+)"\}`\)' 'json.NewEncoder(w).Encode(map[string]string{"error": "$1"})' < internal/handlers/user_handler.go

# If the preview looks right, apply in-place:
# sd 'fmt\.Fprintf\(w, ...' '...' internal/handlers/user_handler.go

# VERIFY: Check the file
# rg 'fmt\.Fprintf\(w,' internal/handlers/user_handler.go
# Should return empty for the patterns we replaced
```

**Common mistakes to avoid:**
- Skipping the SEARCH step and applying blindly
- Not counting the impact before replacing
- Applying to all files at once instead of one at a time
- Forgetting to verify after replacement
- Not having a revert plan (git checkout, backups)

---

## Exercises

### Exercise 1: Basic Pipeline Practice

Run the full four-step workflow to find and replace all `BUG:` comments with
`KNOWN_ISSUE:` comments:

```bash
# Step 1: Find
rg 'BUG:' -t go

# Step 2: Scope
rg -l 'BUG:' -t go

# Step 3: Preview one file
sd '// BUG:' '// KNOWN_ISSUE:' < internal/models/order.go

# Step 4: Verify the preview
sd '// BUG:' '// KNOWN_ISSUE:' < internal/models/order.go | rg 'BUG:'
```

How many files contain `BUG:` comments? How many total occurrences?

### Exercise 2: Type-Scoped Refactoring

Replace `errors.New` with `fmt.Errorf` **only in model files**, not in handlers
or auth:

```bash
# Find everywhere
rg 'errors\.New' -t go -c

# Scope to models only
rg -g 'internal/models/*.go' -l 'errors\.New'

# Preview
for f in $(rg -g 'internal/models/*.go' -l 'errors\.New'); do
    echo "=== $f ==="
    sd 'errors\.New\("(.+)"\)' 'fmt.Errorf("$1")' < "$f" | rg 'fmt\.Errorf'
done
```

### Exercise 3: Glob-Scoped -- Skip Test Files

Replace all `FIXME` comments with `TODO` comments, but skip test files:

```bash
# Count FIXMEs everywhere
rg -c 'FIXME' -t go

# Count FIXMEs excluding tests
rg -c 'FIXME' -t go -g '!*_test.go'

# Preview
rg -g '!*_test.go' -l 'FIXME' -t go | while read f; do
    echo "=== $f ==="
    sd '// FIXME:' '// TODO:' < "$f" | rg '// TODO:'
done
```

### Exercise 4: Directory-Scoped Replacement

Replace all `fmt.Printf` debug logging in just the `internal/` directory:

```bash
# Count in internal/ only
rg -c 'fmt\.Printf' internal/

# List affected files
rg -l 'fmt\.Printf' internal/

# Preview on each
rg -l 'fmt\.Printf' internal/ | while read f; do
    echo "=== $f ==="
    diff <(cat "$f") <(sd 'fmt\.Printf\("\[DEBUG\] (.+)\\n"' 'log.Debug("$1"' < "$f") || true
done
```

### Exercise 5: Multi-Step Service Rename

Practice the full rename workflow from Concept 6. Rename `ProductHandler` references:

```bash
# Step 1: Survey
rg "ProductHandler\|productHandler\|NewProductHandler" -t go

# Step 2: Count
rg -c "productHandler" -t go

# Step 3: Preview each rename
rg "NewProductHandler" -r "NewCatalogHandler" -t go
rg "productHandler" -r "catalogHandler" -t go

# Step 4: Do NOT apply (this is practice) -- just verify your commands are correct
```

### Exercise 6: Extract-Analyze-Replace

Extract all unique error messages in the project, then standardize the most common:

```bash
# Extract unique error strings
rg -o '"[^"]*error[^"]*"' -t go --no-filename -i | sort | uniq -c | sort -rn | head -15

# Find the most common error pattern
rg '"method not allowed"' -t go -c

# Preview standardizing it
rg -l '"method not allowed"' -t go | while read f; do
    echo "=== $f ==="
    sd '"method not allowed"' '"HTTP method not supported for this endpoint"' < "$f" | rg 'not supported'
done
```

### Exercise 7: Import Path Migration

Preview a full module path change:

```bash
# Step 1: Count all imports
rg -c 'github\.com/learn/rg-sd-mastery' -t go

# Step 2: Preview the change
rg 'github\.com/learn/rg-sd-mastery' -r 'github.com/mycompany/shop-api' -t go

# Step 3: Verify every occurrence would be replaced
rg 'github\.com/learn/rg-sd-mastery' -t go | wc -l
rg 'github\.com/learn/rg-sd-mastery' -r 'github.com/mycompany/shop-api' -t go | wc -l
```

The two `wc -l` counts should be equal -- every match has a replacement.

### Exercise 8: Config Value Externalization

Find all hardcoded configuration values and preview replacing them with env var
lookups:

```bash
# Find hardcoded values in config
rg '"(localhost|webshop_|super-secret|disable)' internal/config/config.go -n

# Preview replacing the password
sd 'Password:\s*"webshop_pass123"' 'Password: os.Getenv("DB_PASSWORD")' < internal/config/config.go

# Preview replacing the JWT secret
sd 'JWTSecret:\s*"super-secret-key-change-me"' 'JWTSecret: os.Getenv("JWT_SECRET")' < internal/config/config.go
```

### Exercise 9: Multi-File Coordinated Change

Rename the `/health` endpoint to `/healthz` across all files that reference it:

```bash
# Step 1: Find all references
rg '/health' -t go

# Step 2: Scope (should be auth.go and main.go)
rg -l '/health[^z]' -t go

# Step 3: Preview each file
for f in $(rg -l '"/health"' -t go); do
    echo "=== $f ==="
    sd '"/health"' '"/healthz"' < "$f" | rg 'healthz'
done

# Step 4: Also update the DEPRECATED comment
sd 'DEPRECATED: /health endpoint is being replaced by /healthz' 'Backward-compat: /healthz is the canonical health endpoint' < cmd/webshop/main.go | rg 'canonical'
```

### Exercise 10: The Complete Audit-and-Fix Pipeline

Perform a full audit of `fmt.Println` usage and build the replacement plan:

```bash
# Audit: How many, where, what kind?
echo "=== Total count ==="
rg -c 'fmt\.Println' -t go | sort -t: -k2 -n -r

echo ""
echo "=== By log level ==="
rg -o 'fmt\.Println\("\[(\w+)\]' -t go --no-filename -r '$1' | sort | uniq -c | sort -rn

echo ""
echo "=== Files affected ==="
rg -l 'fmt\.Println' -t go

# Build the replacement plan:
# 1. fmt.Println("[ERROR]...") -> log.Error(...)
# 2. fmt.Println("[WARN]...")  -> log.Warn(...)
# 3. fmt.Println("[INFO]...")  -> log.Info(...)
# 4. fmt.Println("[DEBUG]...") -> log.Debug(...)

# Preview the full pipeline on one file
sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < cmd/webshop/main.go | \
  sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")' | \
  sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' | \
  sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")'
```

### Exercise 11: Verify-After-Replace Discipline

Practice the verification step that catches incomplete replacements:

```bash
# Apply a replacement (preview)
sd 'fmt\.Println' 'log.Println' < internal/handlers/user.go > /tmp/user_replaced.go

# Verify: search for ANY remaining fmt.Print* calls
rg 'fmt\.Print' /tmp/user_replaced.go

# If matches remain, your pattern was too narrow
# Broaden it:
sd 'fmt\.Printf?' 'log.Printf' < internal/handlers/user.go > /tmp/user_replaced2.go
rg 'fmt\.Print' /tmp/user_replaced2.go

# Clean up
rm -f /tmp/user_replaced.go /tmp/user_replaced2.go
```

### Exercise 12: Full Refactoring Scenario

**Scenario:** Rename the `User` package path from `models` to `domain`.

This requires coordinated changes across multiple files:

```bash
# Step 1: Find all imports of the models package
rg 'internal/models' -t go

# Step 2: Find all usage of models.* types
rg 'models\.' -t go -c

# Step 3: Preview import path change
rg 'internal/models' -r 'internal/domain' -t go

# Step 4: Preview package qualifier change
rg 'models\.' -r 'domain.' -t go

# Step 5: Would need to also rename the directory:
# mv internal/models internal/domain
# And update package declaration:
# sd 'package models' 'package domain' internal/domain/*.go

# Step 6: Build the full command sequence (DO NOT RUN):
echo "Commands that would be needed:"
echo "1. mv internal/models internal/domain"
echo "2. rg -l 'package models' internal/domain/ | xargs sd 'package models' 'package domain'"
echo "3. rg -l 'internal/models' -t go | xargs sd 'internal/models' 'internal/domain'"
echo "4. rg -l 'models\.' -t go | xargs sd 'models\.' 'domain.'"
echo "5. rg 'models' -t go  # verify no old references remain"
```

---

## Cheat Sheet

| Workflow | Command |
|----------|---------|
| Find matches | `rg "pattern" -t go` |
| List affected files | `rg -l "pattern" -t go` |
| Count per file | `rg -c "pattern" -t go \| sort -t: -k2 -n -r` |
| Preview replacement (rg) | `rg "old" -r "new" -t go` |
| Preview replacement (sd) | `sd "old" "new" < file.go` |
| Apply to one file | `sd "old" "new" file.go` |
| Apply to matching files | `rg -l "old" -t go \| xargs sd "old" "new"` |
| Skip test files | `rg -g '!*_test.go' -l "old" \| xargs sd "old" "new"` |
| Directory scope | `rg -l "old" internal/handlers/ \| xargs sd "old" "new"` |
| Verify after replace | `rg "old" -t go` (expect empty) |
| Extract unique matches | `rg -o "pattern" --no-filename \| sort \| uniq -c \| sort -rn` |
| Multi-step preview | `sd 'a' 'b' < f \| sd 'c' 'd' \| sd 'e' 'f'` |
| Safe batch with diff | `for f in $(rg -l 'x'); do diff <(cat "$f") <(sd 'x' 'y' < "$f"); done` |

---

## Key Takeaways

1. **Always follow the four-step discipline:** Search, Scope, Replace, Verify.
2. **`rg -r` previews.** `sd` with file arguments commits. Never confuse the two.
3. **Scope your replacements.** Use `-t go`, `-g '!*_test.go'`, or directory paths
   to avoid collateral damage.
4. **Start with one file.** Preview on a single file before applying to the entire
   project.
5. **Verify means zero matches.** After replacing, `rg "old_pattern"` should return
   nothing. If it does, your regex missed some variations.
6. **Git is your safety net.** `git diff` to review, `git checkout .` to revert.
   Always ensure clean git state before bulk operations.

---

Next: [Chapter 9 -- Real-World Refactoring Scenarios](../09-real-world-refactoring/README.md)
