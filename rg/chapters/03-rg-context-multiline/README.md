# Chapter 3: Context Lines & Multiline Matching

## Overview

Finding a match is only half the battle. Often you need to see what surrounds it --
the function a TODO lives in, the code leading up to an error return, or the full body
of a struct definition that spans many lines.

This chapter covers context line controls (`-A`, `-B`, `-C`), multiline matching (`-U`),
column truncation, match limiting, and passthru mode. These features transform `rg` from
a line-finding tool into a **code understanding** tool.

Every command in this chapter should be run from the project root directory:

```bash
cd /Users/surendraraika/projects/Random/rg
```

---

## Concepts

### Context lines

| Flag | Meaning | Mnemonic |
|---|---|---|
| `-A N` | Show N lines **A**fter each match | "After" |
| `-B N` | Show N lines **B**efore each match | "Before" |
| `-C N` | Show N lines of **C**ontext (both before and after) | "Context" |

When multiple matches are close together, rg merges their context blocks and uses `--`
as a separator between non-contiguous groups.

### Multiline mode

By default, `rg` matches patterns within a single line. The `-U` (or `--multiline`) flag
enables patterns to span multiple lines, which is essential for matching struct blocks,
multi-line SQL, function bodies, and other constructs that grep simply cannot handle.

---

## Exercises

### Exercise 1 -- After context: see what follows a match

Show function signatures with the first 3 lines of their body.

**grep:**

```bash
grep -rnA 3 "func.*Handler" internal/handlers/
```

**rg:**

```bash
rg -A 3 "func.*Handler" internal/handlers/
```

```text
internal/handlers/user.go
14:type UserHandler struct {
15-	service *service.UserService
16-}
17-
--
19:func NewUserHandler(svc *service.UserService) *UserHandler {
20-	return &UserHandler{service: svc}
21-}
22-
--
25:func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
26-	if r.Method != http.MethodGet {
27-		w.WriteHeader(http.StatusMethodNotAllowed) // 405
28-		fmt.Fprintf(w, `{"error":"method not allowed"}`)
...
```

**Why rg wins:** The output uses file headings (not repeated per line), color-coded line
numbers (`:`for matches, `-` for context), and `--` separators between non-adjacent groups.
grep's output is a flat wall of `filename:linenum-text` that is much harder to scan.

---

### Exercise 2 -- Before context: see what precedes a match

Show 2 lines before every `return.*error` to understand what conditions lead to errors.

**grep:**

```bash
grep -rnB 2 "return.*error" internal/
```

**rg:**

```bash
rg -B 2 "return.*err" internal/
```

```text
internal/models/user.go
63:// Validate performs basic validation on a User.
64:// FIXME: This should use a proper validation library like go-playground/validator
65:func (u *User) Validate() error {
--
67-	if u.Email == "" {
68-		return errors.New("email is required")
--
69-	if !strings.Contains(u.Email, "@") {
70-		return fmt.Errorf("invalid email format: %s", u.Email)
...
```

The 2 lines of before-context show you the condition or comment that explains WHY the
error is returned.

---

### Exercise 3 -- Combined context: full surroundings

Show 2 lines before and after every TODO comment.

**grep:**

```bash
grep -rnC 2 "TODO" internal/service/
```

**rg:**

```bash
rg -C 2 "TODO" internal/service/
```

```text
internal/service/user_service.go
14-type UserService struct {
15-	db     *database.PostgresDB
16:	logger interface{} // TODO: Use proper logger interface from pkg/logger
17-}
18-
--
30-	fmt.Printf("[DEBUG] UserService.GetByID: id=%d\n", id)
31-
32:	// TODO: Implement actual database query
33-	// query := "SELECT id, email, username, first_name, last_name, role, is_active FROM users WHERE id = $1"
34-	_ = ctx
--
95-
96:	// TODO: Hash password before storing
97:	// TODO: Insert into database within a transaction
98-	_ = ctx
99-
...
```

This reveals the surrounding code structure so you can understand each TODO in context
without opening the file.

---

### Exercise 4 -- Comparing context output: grep vs rg

**grep -C 2:**

```bash
grep -rnC 2 "FIXME" configs/
```

```text
configs/dev.yaml-18-  user: "devuser"
configs/dev.yaml:19:  password: "devpass123"  # FIXME: Use env variable instead of hardcoded password
configs/dev.yaml-20-  sslmode: "disable"
configs/dev.yaml-21-  max_open_conns: 25
```

**rg -C 2:**

```bash
rg -C 2 "FIXME" configs/
```

```text
configs/dev.yaml
17-  user: "devuser"
18-  user: "devuser"
19:  password: "devpass123"  # FIXME: Use env variable instead of hardcoded password
20-  sslmode: "disable"
21-  max_open_conns: 25

configs/prod.yaml
15-database:
16:  host: "${DB_HOST}"           # FIXME: Ensure env var is set in deployment
17-  port: 5432
18-  name: "webshop_prod"
```

**Why rg wins:**
- File names appear once as headings, not repeated on every line
- Match lines are distinguished from context lines by the separator (`:` vs `-`)
- Color highlighting makes matches immediately visible
- Groups from different files are clearly separated

---

### Exercise 5 -- Multiline matching: capture entire struct blocks

Match complete struct definitions spanning multiple lines.

**grep:** Cannot do this. grep is fundamentally a line-oriented tool.

**rg with `-U` (multiline):**

```bash
rg -U "type \w+ struct \{[^}]*\}" internal/models/user.go
```

```text
internal/models/user.go
22:type User struct {
23:	ID          int64     `json:"id" db:"id"`
24:	Email       string    `json:"email" db:"email" validate:"required,email"`
25:	Username    string    `json:"userName" db:"username" validate:"required,min=3,max=50"`
26:	Password    string    `json:"-" db:"password_hash"`
27:	FirstName   string    `json:"first_name" db:"first_name" validate:"required"`
28:	LastName    string    `json:"lastName" db:"last_name" validate:"required"`
29:	Role        UserRole  `json:"role" db:"role" validate:"required"`
30:	Phone       string    `json:"phone_number" db:"phone" validate:"omitempty"`
31:	AvatarURL   string    `json:"avatarUrl" db:"avatar_url"`
32:	IsActive    bool      `json:"is_active" db:"is_active"`
33:	LastLoginAt time.Time `json:"lastLoginAt" db:"last_login_at"`
34:	CreatedAt   time.Time `json:"created_at" db:"created_at"`
35:	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
36:}
```

**Find all structs across the project:**

```bash
rg -U "type \w+ struct \{[^}]*\}" -t go
```

This matches from `type X struct {` through the closing `}`, capturing all fields.

**How `-U` works:** In multiline mode, `.` still does NOT match `\n` by default. Use
`[\s\S]` or `(?s)` to match across newlines. The `[^}]*` in the pattern above works
because it matches everything (including newlines in multiline mode) except `}`.

---

### Exercise 6 -- Multiline with dotall: multi-line SQL queries

Match SQL queries that span multiple lines in Go source files.

**rg with `-U` and `--multiline-dotall`:**

```bash
rg -U --multiline-dotall "SELECT[\s\S]*?FROM[\s\S]*?WHERE" internal/service/
```

```text
internal/service/product_service.go
33:		SELECT p.id, p.sku, p.name, p.description, p.price, p.compare_price,
34:		       p.category_id, p.stock_quantity, p.is_published, p.image_url,
35:		       p.tags, p.created_at, p.updated_at
36:		FROM products p
37:		WHERE p.id = $1 AND p.is_published = true
...
```

**Explanation:**

- `-U` enables multiline matching
- `--multiline-dotall` makes `.` match `\n` (equivalent to the `(?s)` flag)
- `[\s\S]*?` is a non-greedy "match anything including newlines" pattern
- The `*?` (non-greedy) is critical -- without it, the pattern would match from the
  first `SELECT` to the LAST `WHERE` in the file

**Find multi-line INSERT statements in SQL files:**

```bash
rg -U --multiline-dotall "INSERT INTO \w+[\s\S]*?;" scripts/
```

---

### Exercise 7 -- Max columns: truncate long lines

Some files have very long lines (e.g., JSON struct tags, SQL inserts). Truncate them.

```bash
rg --max-columns 80 "json:" internal/models/
```

```text
internal/models/user.go
23:	Email       string    `json:"email" db:"email" validate:"required,emai [... omitted end of long line]
24:	Username    string    `json:"userName" db:"username" validate:"required [... omitted end of long line]
...
```

Lines longer than 80 characters are truncated with `[... omitted end of long line]`.

**Without `--max-columns`:** the full line wraps across your terminal, making output
hard to read.

---

### Exercise 8 -- Max columns preview

Show truncated lines but include a preview of what was cut.

```bash
rg --max-columns 80 --max-columns-preview "json:" internal/models/
```

```text
internal/models/user.go
23:	Email       string    `json:"email" db:"email" validate:"required,email"` [... 12 more characters]
...
```

The `--max-columns-preview` flag shows a count of how many characters were hidden,
helping you decide whether to look at the full line.

---

### Exercise 9 -- Max count per file: first match only

Show only the FIRST match in each file.

```bash
rg -m 1 "TODO"
```

```text
cmd/webshop/main.go
17:// TODO: Implement graceful shutdown with os.Signal handling

internal/auth/auth.go
61:// TODO: Implement actual bcrypt hashing - this is a stub

internal/config/config.go
10:// TODO: Add validation for required fields after loading

internal/database/postgres.go
21:	maxRetries   = 5             // TODO: Make configurable
...
```

Only one match per file is shown, regardless of how many exist. This is useful for
quickly identifying which files have TODOs without seeing every single one.

**Show first 3 matches per file:**

```bash
rg -m 3 "TODO"
```

**grep equivalent:**

```bash
# grep has -m but it applies globally, not per-file
grep -rm 1 "TODO" .
```

---

### Exercise 10 -- Passthru mode: highlight matches in full file

Show the entire file, but highlight where the pattern matches.

```bash
rg --passthru "TODO" internal/config/config.go
```

```text
package config

import (
	"fmt"
	"os"
	"strconv"
)

// NOTE: Config fields use yaml tags for future YAML unmarshaling support
// [TODO]: Add validation for required fields after loading  <-- highlighted

// Config holds all application configuration.
type Config struct {
	AppName     string         `yaml:"app_name" json:"appName"`
	...
	JWTSecret:      "super-secret-key-change-me", // [TODO]: Load from env var  <-- highlighted
	...
// [TODO]: Actually implement YAML parsing  <-- highlighted
...
```

(In a real terminal, `TODO` would be highlighted in red/bold on each matching line,
while all other lines are printed normally.)

**Use case:** Code review. You want to read a file top to bottom but have all TODOs
visually called out.

**grep equivalent:** No equivalent. You would need `cat file | grep --color=always -E "TODO|$"`.

---

### Exercise 11 -- Combining multiline with context

Get struct definitions with 2 lines of context before (to see the comment):

```bash
rg -B 2 -U "type \w+ struct \{[^}]*\}" internal/models/user.go
```

```text
internal/models/user.go
20-// User represents a registered user in the webshop.
21-// NOTE: The json tags intentionally mix camelCase and snake_case for rg exercises
22:type User struct {
23:	ID          int64     `json:"id" db:"id"`
...
36:}
--
38-// CreateUserRequest is the payload for creating a new user.
39-// BUG: No password strength validation is enforced here
40:type CreateUserRequest struct {
41:	Email     string   `json:"email" validate:"required,email"`
...
48:}
```

The before-context shows the doc comments that describe each struct.

**Context + multiline + type filter:**

```bash
rg -B 1 -U "type \w+ struct \{[^}]*\}" -t go
```

This finds every struct definition in every Go file, showing the comment above each one.

---

### Exercise 12 -- Practical exercises

**12a. Extract all multi-line SQL queries from the service layer:**

```bash
rg -U --multiline-dotall "query\s*:?=\s*\x60[\s\S]*?\x60" internal/service/
```

Here `\x60` matches the backtick character used for raw strings in Go. This captures
the full multi-line query assigned to a `query` variable.

Alternative using `[\s\S]`:

```bash
rg -U "(?:query\s*:?=\s*)[\x60]" -A 10 internal/service/
```

**12b. Find all struct definitions with their fields, sorted by file:**

```bash
rg -U "type \w+ struct \{[^}]*\}" -t go --sort path
```

**12c. Find all BUG and FIXME annotations with surrounding context:**

```bash
rg -C 3 "BUG:|FIXME:" -t go
```

**12d. Find functions that contain more than one error return:**

Use multiline to match function bodies with at least two returns:

```bash
rg -U --multiline-dotall "func \w+.*\{[\s\S]*?return.*err[\s\S]*?return.*err" -t go -l
```

This lists files (via `-l`) that have functions with multiple error returns.

**12e. Find all handler functions and show their HTTP method check:**

```bash
rg -A 4 "func \(h \*\w+Handler\)" internal/handlers/
```

---

## Cheat Sheet

| Task | Flag | Example |
|---|---|---|
| Lines after match | `-A N` | `rg -A 3 "func"` |
| Lines before match | `-B N` | `rg -B 2 "return err"` |
| Lines before + after | `-C N` | `rg -C 2 "TODO"` |
| Multiline matching | `-U` | `rg -U "struct \{[^}]*\}"` |
| Multiline + dotall | `-U --multiline-dotall` | `rg -U --multiline-dotall "SELECT.*FROM"` |
| Truncate long lines | `--max-columns N` | `rg --max-columns 80 "json:"` |
| Truncation preview | `--max-columns-preview` | `rg --max-columns 80 --max-columns-preview "pat"` |
| First N matches/file | `-m N` | `rg -m 1 "TODO"` |
| Show full file with highlights | `--passthru` | `rg --passthru "TODO" file.go` |
| Multiline + context | `-U -B N` | `rg -B 2 -U "type.*struct"` |
