# Chapter 2: File & Directory Filtering

## Overview

One of ripgrep's most powerful advantages over `grep` is how it handles **which files to
search**. Traditional `grep -r` searches everything, forcing you to bolt on `find`, `xargs`,
and `--exclude-dir` flags. `rg` has first-class support for file type filtering, glob
patterns, `.gitignore` integration, and depth limits -- all with clean, memorable syntax.

Every command in this chapter should be run from the project root directory:

```bash
cd /Users/surendraraika/projects/Random/rg
```

---

## Concepts

### How rg decides which files to search

When you run `rg PATTERN`, it applies these filters **in order**:

1. Skip hidden files and directories (those starting with `.`)
2. Read `.gitignore` rules and skip matching paths
3. Read `.rgignore` rules (if present) and skip matching paths
4. Read `.ignore` rules (if present) and skip matching paths
5. Apply any `--type`, `--glob`, or `--max-depth` filters you specified
6. Skip binary files (by default)

This means `rg` searches only the files that matter, with zero configuration.

### grep's file filtering -- the painful way

To achieve similar filtering with `grep`, you typically need multi-command pipelines:

```bash
# grep equivalent of "rg -t go TODO":
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs grep -n "TODO"

# grep equivalent of "rg -g '!*_test.go' func":
find . -name "*.go" -not -name "*_test.go" | xargs grep -n "func"
```

---

## Exercises

### Exercise 1 -- Type filtering: search only Go files

Search for `TODO` comments only in Go source files.

**grep:**

```bash
find . -name "*.go" -not -path "./.git/*" | xargs grep -n "TODO"
```

This requires piping `find` into `xargs`, handling spaces in filenames, and manually
excluding `.git`.

**rg:**

```bash
rg -t go "TODO"
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

**Exclude a type** -- search for "host" but skip YAML files:

```bash
rg -T yaml "host"
```

This searches all file types **except** YAML.

---

### Exercise 2 -- Custom type definitions

Define your own file type and search within it.

**rg:**

```bash
rg --type-add 'config:*.yaml' -t config "host"
```

```text
configs/dev.yaml
6:  host: "localhost"

configs/prod.yaml
6:  host: "0.0.0.0"
16:  host: "${DB_HOST}"

configs/test.yaml
5:  host: "127.0.0.1"
```

You can define types that match multiple extensions:

```bash
rg --type-add 'webcode:*.{go,yaml,sql}' -t webcode "TODO"
```

This searches `.go`, `.yaml`, and `.sql` files in a single command.

**Persisting custom types:** Add them to your `~/.ripgreprc` file:

```text
--type-add
config:*.yaml
--type-add
webcode:*.{go,yaml,sql}
```

---

### Exercise 3 -- Glob patterns: include files

Use glob patterns for ad-hoc file filtering.

Search for `func` only in `.go` files:

```bash
rg -g "*.go" "func"
```

Search for `TODO` only in files under `internal/`:

```bash
rg -g "internal/**" "TODO"
```

**Negated globs** -- search all Go files EXCEPT test files:

```bash
rg -g "!*_test.go" "func" -t go
```

```text
cmd/webshop/main.go
30:func main() {

internal/auth/auth.go
63:func HashPassword(password string) (string, error) {
78:func CheckPassword(password, hash string) bool {
85:func GenerateToken() (string, error) {
95:func ValidatePassword(password string) error {
...
```

Notice: `test/integration/api_test.go` results are excluded.

---

### Exercise 4 -- Multiple globs

Combine multiple `-g` flags for precise filtering.

Search for `TODO` in Go files, excluding anything in a `vendor/` directory:

```bash
rg -g "*.go" -g "!vendor/*" "TODO"
```

Search for `password` in config and SQL files only:

```bash
rg -g "*.yaml" -g "*.sql" -i "password"
```

```text
configs/dev.yaml
19:  password: "devpass123"  # FIXME: Use env variable instead of hardcoded password

configs/prod.yaml
20:  password: "${DB_PASSWORD}"   # NOTE: Injected from Vault/Secrets Manager

scripts/seed.sql
11:INSERT INTO users (id, email, username, password_hash, role, created_at, updated_at) VALUES
12:  (1, 'admin@webshop.dev', 'admin', '$2a$10$N9qo8u...', 'admin', ...),
...
```

Search all files except Go and SQL (useful when debugging config issues):

```bash
rg -g "!*.go" -g "!*.sql" "TODO"
```

---

### Exercise 5 -- Automatic .gitignore respect

This project has a `.gitignore` that excludes `vendor/`, `.env`, `*.log`, `bin/`, etc.

**Demonstrate the difference:**

```bash
# rg automatically respects .gitignore
rg "secret"
```

This will search only tracked files. If you had a `.env` file with `DB_SECRET=abc`, rg
would **not** search it because `.env` is in `.gitignore`.

**grep has no idea about .gitignore:**

```bash
grep -rn "secret" .
```

This would search `.env`, `vendor/`, log files, and everything else -- potentially
exposing secrets in your terminal output and polluting results with irrelevant matches.

---

### Exercise 6 -- The .rgignore file

Create an `.rgignore` file to add project-specific exclusions beyond `.gitignore`.

Create the file:

```bash
cat > .rgignore << 'EOF'
# Ignore generated files
*.pb.go
*_generated.go

# Ignore chapter exercise files when searching the main codebase
chapters/

# Ignore seed data
scripts/seed.sql
EOF
```

Now search with the ignore in effect:

```bash
rg "TODO"
```

The `chapters/` directory and `scripts/seed.sql` are now excluded from search results.

**How ignore files are layered:**

| File | Scope | Overrides |
|---|---|---|
| `.gitignore` | Repository-wide | Baseline |
| `.rgignore` | Repository-wide | Adds to .gitignore |
| `.ignore` | Repository-wide | Adds to .gitignore |
| `~/.config/ripgrep/ignore` | Global | All repos |

Each more specific file adds to (does not replace) the parent.

> **Important:** Remove or rename the `.rgignore` after this exercise so it does not
> affect later chapters:
>
> ```bash
> rm .rgignore
> ```

---

### Exercise 7 -- Overriding ignore files

Sometimes you **want** to search ignored files.

**`--no-ignore` bypasses .gitignore and .rgignore:**

```bash
rg --no-ignore "secret"
```

This will also search `.env`, `vendor/`, and any other gitignored paths.

**More granular overrides:**

```bash
# Only bypass .gitignore (still respect .rgignore)
rg --no-ignore-vcs "pattern"

# Only bypass .rgignore and .ignore (still respect .gitignore)
rg --no-ignore-dot "pattern"

# Search everything, including hidden and ignored
rg --no-ignore --hidden "pattern"
```

**grep equivalent:**

```bash
# There is no equivalent -- grep searches everything by default,
# so you get the "no-ignore" behavior whether you want it or not.
```

---

### Exercise 8 -- Searching hidden files

By default, `rg` skips files and directories starting with `.` (hidden on Unix).

**Default behavior (hidden files skipped):**

```bash
rg "settings"
```

This will NOT find matches in `.claude/settings.local.json` or `.gitignore`.

**Include hidden files:**

```bash
rg --hidden "settings"
```

```text
.claude/settings.local.json
... (matches shown)
```

**Search hidden files but still respect .gitignore:**

```bash
rg --hidden "pattern"
```

This is the default when `--hidden` is used -- `.gitignore` is still respected.
To truly search everything:

```bash
rg --hidden --no-ignore "pattern"
```

---

### Exercise 9 -- Binary files

By default, `rg` skips binary files entirely. This is usually what you want.

**Force searching binary files:**

```bash
rg --binary "pattern"
```

**grep comparison:**

```bash
# grep prints unhelpful messages like:
# Binary file ./bin/webshop matches
grep -r "pattern" .
```

`rg` with `--binary` will show the actual matching lines from binary files, replacing
non-printable bytes with a placeholder.

---

### Exercise 10 -- Searching specific directories

Limit your search to a specific subdirectory.

**Search only the handlers package:**

```bash
rg "error" internal/handlers/
```

```text
internal/handlers/user.go
25:// BUG: Missing input validation - id param is not checked for SQL injection
43:		fmt.Fprintf(w, `{"error":"invalid id format"}`)
108:		fmt.Fprintf(w, `{"error":"invalid request body: %s"}`, err.Error())
...

internal/handlers/product.go
43:		fmt.Printf("[ERROR] Invalid product id: %s\n", idStr)
139:		fmt.Printf("[ERROR] Failed to decode CreateProduct: %v\n", err)
...
```

**Search multiple specific directories:**

```bash
rg "TODO" internal/handlers/ internal/service/
```

**Compare with grep:**

```bash
grep -rn "TODO" internal/handlers/ internal/service/
```

Both work, but rg's grouped output with file headings is far more readable.

---

### Exercise 11 -- Limiting search depth

Control how deep rg recurses into subdirectories.

**Search only the top-level directory (depth 1):**

```bash
rg --max-depth 1 "func"
```

This searches only files directly in the project root (like `main.go` if it were there).

**Search two levels deep:**

```bash
rg --max-depth 2 "func"
```

```text
cmd/webshop/main.go
30:func main() {
```

Only `cmd/webshop/` is two levels deep from root. Deeper paths like
`internal/handlers/user.go` (three levels) are excluded.

**Search three levels (includes most of this project):**

```bash
rg --max-depth 3 "type.*struct"
```

**grep equivalent:**

```bash
find . -maxdepth 3 -name "*.go" | xargs grep "type.*struct"
```

---

### Exercise 12 -- Listing supported file types

See every file type that rg knows about:

```bash
rg --type-list
```

```text
agda: *.agda, *.lagda
aidl: *.aidl
amake: *.bp, *.mk
...
go: *.go
...
sql: *.sql
...
yaml: *.yaml, *.yml
...
```

There are over 150 built-in types. Some useful ones for this project:

```bash
rg --type-list | rg "^(go|sql|yaml|json|toml|markdown)"
```

```text
go: *.go
json: *.json, *.sarif
markdown: *.markdown, *.md, *.mdown, *.mkdn
sql: *.sql, *.psql
toml: *.toml
yaml: *.yaml, *.yml
```

You can use these directly: `rg -t go`, `rg -t sql`, `rg -t yaml`, etc.

---

## Exercises -- Putting It Together

**E1. Find all files that contain the word "password" in any file type (Go, YAML, SQL):**

```bash
rg -l -i "password"
```

The `-l` flag prints only filenames, not matching lines.

**E2. Find TODO comments only in service layer code (not handlers, not models):**

```bash
rg -g "internal/service/*.go" "TODO"
```

**E3. List all Go files that would be searched (without actually searching):**

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

**E4. Search for "DEPRECATED" in everything except test files and chapters:**

```bash
rg -g "!*_test.go" -g "!chapters/" "DEPRECATED"
```

---

## Cheat Sheet

| Task | `grep` | `rg` |
|---|---|---|
| Only Go files | `find . -name "*.go" \| xargs grep "pat"` | `rg -t go "pat"` |
| Exclude YAML files | `find . -not -name "*.yaml" \| xargs grep "pat"` | `rg -T yaml "pat"` |
| Custom type | N/A | `rg --type-add 'cfg:*.yaml' -t cfg "pat"` |
| Glob include | `find . -name "*.go" \| xargs grep` | `rg -g "*.go" "pat"` |
| Glob exclude | `find . -not -name "*_test.go" \| xargs grep` | `rg -g "!*_test.go" "pat"` |
| Respect .gitignore | Not supported | Automatic |
| Search hidden files | `grep -r` (always does) | `rg --hidden "pat"` |
| Override ignores | N/A (always searches all) | `rg --no-ignore "pat"` |
| Specific directory | `grep -rn "pat" dir/` | `rg "pat" dir/` |
| Max depth | `find . -maxdepth N \| xargs grep` | `rg --max-depth N "pat"` |
| List known types | N/A | `rg --type-list` |
| List files to search | `find . -name "*.go"` | `rg --files -t go` |
| Files with matches | `grep -rl "pat" .` | `rg -l "pat"` |
