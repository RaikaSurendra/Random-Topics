# Chapter 9: Real-World Refactoring Scenarios

## Overview

This is the capstone chapter. Here you apply everything from Chapters 1-8 in
complete, realistic scenarios that mirror actual codebase maintenance tasks. Each
scenario is a self-contained walkthrough using the webshop Go project, following
the **search, verify, replace, verify** discipline from Chapter 8.

Every scenario starts with an audit (using `rg`), builds a plan, executes the
changes (using `sd`), and verifies the result. Scenarios progress from straightforward
to complex.

All commands assume you are running from the project root:

```bash
cd /Users/surendraraika/projects/Random/rg
```

**Tool versions used:** sd 1.0.0, rg 15.1.0

**Important:** These scenarios preview changes via stdin by default. Commands that
actually modify files are clearly marked. Always revert with `git checkout .` after
experimenting.

---

## Scenario 1: Security Audit

**Goal:** Find all hardcoded secrets, credentials, and security-sensitive patterns.
Replace them with environment variable references.

### Step 1: Find Hardcoded Secrets

```bash
rg -i 'secret|password|token|key' -t go -g '!*_test.go' -n
```

Expected output (partial):
```
internal/config/config.go:28:    Password        string `yaml:"password" json:"password"`
internal/config/config.go:38:    JWTSecret       string `yaml:"jwt_secret" json:"jwtSecret"`
internal/config/config.go:62:            Password:        "webshop_pass123",
internal/config/config.go:70:            JWTSecret:      "super-secret-key-change-me",
internal/auth/auth.go:26:    ExpiresAt int64  `json:"expiresAt"`
cmd/webshop/main.go:... (TokenHeader references)
```

### Step 2: Find TODO/FIXME for Security Issues

```bash
rg 'TODO.*secur|FIXME.*auth|FIXME.*token|BUG.*password|HACK.*token' -i -t go -n
```

Expected output:
```
internal/auth/auth.go:13:// FIXME: Token validation does not check expiry properly
internal/auth/auth.go:48:// HACK: Currently uses a dummy token check ...
internal/auth/auth.go:93:    // HACK: Dummy validation ...
internal/config/config.go:62:            Password:        "webshop_pass123", // BUG: Hardcoded password
```

### Step 3: Count the Severity

```bash
echo "=== Hardcoded passwords ==="
rg -c 'password' -i -t go -g '!*_test.go'

echo ""
echo "=== Hardcoded secrets ==="
rg -c 'secret' -i -t go -g '!*_test.go'

echo ""
echo "=== Security TODOs/FIXMEs ==="
rg -c 'TODO.*secur|FIXME.*auth|FIXME.*token|HACK.*token|BUG.*password' -i -t go
```

### Step 4: Replace Hardcoded JWT Secret

```bash
# Preview
sd 'JWTSecret:\s*"super-secret-key-change-me"' 'JWTSecret: os.Getenv("JWT_SECRET")' < internal/config/config.go

# Preview the password replacement
sd 'Password:\s*"webshop_pass123"' 'Password: os.Getenv("DB_PASSWORD")' < internal/config/config.go
```

### Step 5: Remove the BUG Comment (It Will Be Fixed)

```bash
sd '// BUG: Hardcoded password in source\n' '' < internal/config/config.go | rg -A1 'Password'
```

### Step 6: Verify No Secrets Remain

```bash
# After applying changes, run the same audit
rg '"(webshop_pass123|super-secret-key-change-me)"' -t go
# Should return empty
```

### Security Audit Summary

| Finding | File | Line | Severity |
|---------|------|------|----------|
| Hardcoded DB password | config.go | 62 | Critical |
| Hardcoded JWT secret | config.go | 70 | Critical |
| Dummy token validation | auth.go | 93 | High |
| No expiry check | auth.go | 13 | Medium |
| SSL disabled | config.go | 64 | Medium |

---

## Scenario 2: Logging Standardization

**Goal:** The codebase uses a mix of `fmt.Println("[LEVEL] ...")`, `fmt.Printf("[LEVEL] ...")`,
and proper `log.Info(...)` calls. Standardize everything to use the structured
logger.

### Step 1: Audit Current Logging

```bash
echo "=== fmt.Println log statements ==="
rg 'fmt\.Println\("\[' -t go -n

echo ""
echo "=== fmt.Printf log statements ==="
rg 'fmt\.Printf\("\[' -t go -n

echo ""
echo "=== Proper logger calls ==="
rg 'log\.(Info|Error|Warn|Debug)\(' -t go -n
```

### Step 2: Count by Log Level

```bash
rg -o '\[(ERROR|WARN|INFO|DEBUG)\]' -t go --no-filename | sort | uniq -c | sort -rn
```

Expected output:
```
      8 [DEBUG]
      6 [INFO]
      3 [ERROR]
      2 [WARN]
```

### Step 3: Count by File

```bash
rg -c 'fmt\.Print(ln|f)\("\[' -t go | sort -t: -k2 -n -r
```

Expected output:
```
internal/database/postgres.go:5
cmd/webshop/main.go:4
internal/handlers/user.go:4
internal/models/user.go:1
internal/models/product.go:1
internal/config/config.go:2
```

### Step 4: Replace fmt.Println("[LEVEL]...") Patterns

Preview on `cmd/webshop/main.go`:

```bash
sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < cmd/webshop/main.go | \
  sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' | \
  sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")' | \
  sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")'
```

### Step 5: Replace fmt.Printf("[LEVEL]...") Patterns

These are harder because they have format arguments:

```bash
# Preview: fmt.Printf("[DEBUG] ... %v ...\n", var) -> log.Debug("...", var)
sd 'fmt\.Printf\("\[DEBUG\] (.+)\\n"(.+)\)' 'log.Debug("$1"$2)' < internal/database/postgres.go
```

### Step 6: Handle the Remaining fmt.Println Without Levels

```bash
# Find fmt.Println that DON'T have [LEVEL] prefix
rg 'fmt\.Println\("[^\["]' -t go -n

# These are debug prints that should also use the logger
sd 'fmt\.Println\("([^"]+)"\)' 'log.Debug("$1")' < internal/database/postgres.go
```

### Step 7: Verify

```bash
# After all replacements, no fmt.Print* should remain for logging
rg 'fmt\.Print(ln|f)\("\[' -t go
# Should return empty
```

---

## Scenario 3: Error Handling Upgrade

**Goal:** Upgrade error handling to follow Go best practices. Wrap all bare error
returns with `fmt.Errorf` and context.

### Step 1: Find Bare Error Returns

```bash
rg 'return err$' -t go -n
```

Expected output:
```
internal/models/... (if any)
```

### Step 2: Find errors.New Without Wrapping Context

```bash
rg 'errors\.New\("' -t go -n
```

Expected output:
```
internal/models/user.go:66:        return errors.New("email is required")
internal/models/user.go:75:        return errors.New("username must not exceed 50 characters")
internal/models/user.go:78:        return errors.New("first name and last name are required")
internal/models/user.go:81:        return errors.New("role is required")
internal/models/product.go:57:        return errors.New("SKU is required")
...
```

### Step 3: Find fmt.Errorf Without %w (No Wrapping)

```bash
rg 'fmt\.Errorf\("[^"]*"\)' -t go -n
```

This finds `fmt.Errorf` calls that have no format verbs at all -- potential errors
that should have context.

### Step 4: Count the Impact

```bash
echo "=== Bare 'return err' ==="
rg -c 'return err$' -t go 2>/dev/null || echo "(none found)"

echo ""
echo "=== errors.New calls ==="
rg -c 'errors\.New' -t go

echo ""
echo "=== fmt.Errorf without %w ==="
rg -c 'fmt\.Errorf\("[^"]*"[^,)]*\)' -t go 2>/dev/null || echo "(none found)"
```

### Step 5: Upgrade errors.New to Include Package Context

```bash
# Preview: Add package context prefix to validation errors
sd 'return errors\.New\("(.+)"\)' 'return fmt.Errorf("user validation: $1")' < internal/models/user.go
```

```bash
sd 'return errors\.New\("(.+)"\)' 'return fmt.Errorf("product validation: $1")' < internal/models/product.go
```

```bash
sd 'return errors\.New\("(.+)"\)' 'return fmt.Errorf("order validation: $1")' < internal/models/order.go
```

### Step 6: Upgrade fmt.Errorf to Use %w for Wrapping

Find existing `fmt.Errorf` calls that use `%v` instead of `%w`:

```bash
rg 'fmt\.Errorf\(".*%v"' -t go -n
```

Preview the upgrade:

```bash
sd '%v", err\)' '%w", err)' < internal/config/config.go
```

### Step 7: Verify the Upgrade

```bash
# No more errors.New in model validation
rg 'errors\.New' internal/models/ -c
# Should be zero (or reduced to sentinel errors only)

# All wrapping uses %w
rg 'fmt\.Errorf.*%v.*err' -t go
# Should return empty
```

---

## Scenario 4: API Versioning Migration

**Goal:** The current routes use no version prefix. Add `/api/v1/` prefix and
prepare for a v2 migration.

### Step 1: Find All Route Definitions

```bash
rg 'HandleFunc\("/' -t go -n
```

Expected output:
```
cmd/webshop/main.go:63:    mux.HandleFunc("/users", ...
cmd/webshop/main.go:64:    mux.HandleFunc("/users/create", ...
cmd/webshop/main.go:65:    mux.HandleFunc("/users/get", ...
cmd/webshop/main.go:66:    mux.HandleFunc("/users/update", ...
cmd/webshop/main.go:67:    mux.HandleFunc("/users/delete", ...
cmd/webshop/main.go:70:    mux.HandleFunc("/products", ...
cmd/webshop/main.go:71:    mux.HandleFunc("/products/create", ...
cmd/webshop/main.go:72:    mux.HandleFunc("/products/get", ...
cmd/webshop/main.go:74:    mux.HandleFunc("/orders", ...
cmd/webshop/main.go:75:    mux.HandleFunc("/orders/create", ...
cmd/webshop/main.go:76:    mux.HandleFunc("/orders/status", ...
```

### Step 2: Count Routes

```bash
rg -c 'HandleFunc\("/' cmd/webshop/main.go
```

### Step 3: Preview Adding v1 Prefix

```bash
# Add /api/v1 prefix to all resource routes (but NOT /health)
sd 'HandleFunc\("/(users|products|orders)' 'HandleFunc("/api/v1/$1' < cmd/webshop/main.go
```

Expected: `/users` becomes `/api/v1/users`, etc. The `/health` endpoint stays unchanged.

### Step 4: Also Update the Health Endpoint Comment

```bash
sd 'DEPRECATED: /health endpoint is being replaced by /healthz' 'Legacy: /health maintained for backward compat, prefer /api/v1/healthz' < cmd/webshop/main.go
```

### Step 5: Update Auth Middleware Skip Paths

```bash
# The auth middleware skips /health -- update it for the new paths
rg 'r\.URL\.Path ==' internal/auth/auth.go -n

# Preview adding v1 health check
sd 'r\.URL\.Path == "/health" \|\| r\.URL\.Path == "/healthz"' 'r.URL.Path == "/health" || r.URL.Path == "/healthz" || r.URL.Path == "/api/v1/healthz"' < internal/auth/auth.go
```

### Step 6: Verify All Routes Have Version Prefix

```bash
# After applying, check that no bare resource routes remain
sd 'HandleFunc\("/(users|products|orders)' 'HandleFunc("/api/v1/$1' < cmd/webshop/main.go | \
  rg 'HandleFunc\("/(users|products|orders)'
# Should return nothing (all are now /api/v1/...)

# But /health should still be unversioned
sd 'HandleFunc\("/(users|products|orders)' 'HandleFunc("/api/v1/$1' < cmd/webshop/main.go | \
  rg 'HandleFunc\("/health'
# Should still show the /health route
```

---

## Scenario 5: Database Query Audit

**Goal:** Find problematic database query patterns and replace them with safer
alternatives.

### Step 1: Find All SELECT * Queries

```bash
rg 'SELECT \*' -t go -t sql -n
```

### Step 2: Find String-Concatenated Queries (SQL Injection Risk)

```bash
rg '"\s*SELECT.*"\s*\+' -t go -n
rg 'fmt\.Sprintf\("SELECT' -t go -n
rg 'fmt\.Sprintf\("INSERT' -t go -n
rg 'fmt\.Sprintf\("UPDATE' -t go -n
```

### Step 3: Find Direct Query Construction

```bash
rg 'Exec\(|QueryRow\(' -t go -n
```

Expected output:
```
internal/database/postgres.go:60:func (db *DB) QueryRow(query string, args ...interface{}) error {
internal/database/postgres.go:76:func (db *DB) Exec(query string, args ...interface{}) error {
```

### Step 4: Find DSN String Construction (Credential Exposure)

```bash
rg 'Sprintf.*host=.*password=' -t go -n
```

Expected output:
```
internal/database/postgres.go:23:    dsn := fmt.Sprintf(
internal/database/postgres.go:24:        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
```

### Step 5: Preview Replacing DSN Construction with a Builder

```bash
sd 'dsn := fmt\.Sprintf\(\n\t\t"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",\n\t\tcfg\.Host, cfg\.Port, cfg\.User, cfg\.Password, cfg\.DBName, cfg\.SSLMode,\n\t\)' 'dsn := buildDSN(cfg)' < internal/database/postgres.go
```

### Step 6: Find Debug Logging That Leaks Credentials

```bash
rg 'Printf.*dsn\b|Printf.*password|Println.*password' -t go -i -n
```

Expected:
```
internal/database/postgres.go:29:    fmt.Printf("[DEBUG] Connecting to PostgreSQL: %s\n", dsn)
```

The DSN contains the password. This is a security leak in logs.

```bash
# Preview replacing with a safe version
sd 'fmt\.Printf\("\[DEBUG\] Connecting to PostgreSQL: %s\\n", dsn\)' 'log.Debug("Connecting to PostgreSQL at %s:%d/%s", cfg.Host, cfg.Port, cfg.DBName)' < internal/database/postgres.go
```

---

## Scenario 6: Dependency Injection Refactoring

**Goal:** Find all direct struct initialization (tight coupling) and prepare to
replace with constructor patterns.

### Step 1: Find Direct Struct Initialization

```bash
rg '\w+Handler\{' -t go -n
```

Expected output:
```
internal/handlers/user.go:19:    return &UserHandler{service: svc}
internal/handlers/user_handler.go:22:    return &UserHandler{userService: svc}
```

### Step 2: Find All Constructor Functions

```bash
rg '^func New\w+\(' -t go -n
```

Expected output:
```
internal/handlers/user.go:18:func NewUserHandler(svc *service.UserService) *UserHandler {
internal/handlers/user_handler.go:21:func NewUserHandler(svc *service.UserService) *UserHandler {
internal/database/postgres.go:22:func NewPostgresDB(cfg config.DatabaseConfig) (*DB, error) {
```

### Step 3: Find Where Constructors Are Called with nil

```bash
rg 'New\w+\(nil\)' -t go -n
```

Expected output:
```
cmd/webshop/main.go:55:    userHandler := handlers.NewUserHandler(nil)
cmd/webshop/main.go:56:    productHandler := handlers.NewProductHandler(nil)
cmd/webshop/main.go:57:    orderHandler := handlers.NewOrderHandler(nil)
```

This is the HACK referenced in the comment above -- passing `nil` for services.

### Step 4: Preview Replacing nil with Real Service Initialization

```bash
# Preview: Replace the nil handlers with proper initialization
sd 'userHandler := handlers\.NewUserHandler\(nil\)' 'userSvc := service.NewUserService(db)
	userHandler := handlers.NewUserHandler(userSvc)' < cmd/webshop/main.go
```

### Step 5: Find the HACK Comment and Remove It

```bash
sd '// HACK: Creating handlers directly instead of using dependency injection\n\t' '' < cmd/webshop/main.go
```

### Step 6: Verify the Pattern

```bash
# After changes, no nil constructors should remain
rg 'New\w+\(nil\)' -t go
# Should return empty
```

---

## Scenario 7: Test Coverage Gap Analysis

**Goal:** Identify exported functions that lack corresponding test functions.

### Step 1: Find All Exported Functions

```bash
rg '^func [A-Z]\w+' -t go -g '!*_test.go' --no-filename | sort
```

Expected output:
```
func DefaultConfig() *Config {
func GetClaimsFromContext(ctx context.Context) (*TokenClaims, error) {
func Middleware(next http.Handler, log Logger) http.Handler {
func NewPostgresDB(cfg config.DatabaseConfig) (*DB, error) {
func NewUserHandler(svc *service.UserService) *UserHandler {
func RequireRole(roles ...string) func(http.Handler) http.Handler {
func TaxRate() float64 {
func ValidateToken(tokenStr string) (*TokenClaims, error) {
```

### Step 2: Find All Exported Methods

```bash
rg '^func \(\w+ \*\w+\) [A-Z]\w+' -t go -g '!*_test.go' --no-filename | sort
```

Expected output:
```
func (db *DB) Close() error {
func (db *DB) Exec(query string, args ...interface{}) error {
func (db *DB) Ping() error {
func (db *DB) QueryRow(query string, args ...interface{}) error {
func (db *DB) Transaction(fn func() error) error {
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
func (o *Order) CalculateTotal() {
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
func (o *Order) Validate() error {
func (p *Product) DiscountPercentage() float64 {
func (p *Product) FormatPrice() string {
func (p *Product) InStock() bool {
func (p *Product) IsAvailable() bool {
func (p *Product) IsOnSale() bool {
func (p *Product) Validate() error {
func (u *User) FullName() string {
func (u *User) HasAdminAccess() bool {
func (u *User) IsAdmin() bool {
func (u *User) Validate() error {
```

### Step 3: Find All Test Functions

```bash
rg '^func Test' -t go --no-filename | sort
```

If no test files exist (as in this project), the output is empty -- meaning 0%
coverage of exported functions.

### Step 4: Extract Just the Function Names for Comparison

```bash
# Exported functions (names only)
rg -o '^func (?:\(\w+ \*\w+\) )?([A-Z]\w+)' -t go -g '!*_test.go' --no-filename -r '$1' | sort -u

# Test functions (names only)
rg -o '^func (Test\w+)' -t go --no-filename -r '$1' | sort -u
```

### Step 5: Count the Gap

```bash
echo "Exported functions/methods:"
rg '^func .*[A-Z]\w+' -t go -g '!*_test.go' -c | \
  awk -F: '{sum+=$2} END {print sum}'

echo "Test functions:"
rg '^func Test' -t go -c 2>/dev/null | \
  awk -F: '{sum+=$2} END {print sum}' || echo "0"
```

### Step 6: Generate a Test Coverage Report

```bash
# Build a list of untested functions
echo "=== Untested exported functions ==="
rg -o '^func (?:\(\w+ \*\w+\) )?([A-Z]\w+)' -t go -g '!*_test.go' --no-filename -r '$1' | sort -u | while read func; do
    if ! rg -q "Test.*${func}" -t go 2>/dev/null; then
        echo "  MISSING TEST: $func"
    fi
done
```

---

## Scenario 8: Config Externalization

**Goal:** Find all hardcoded values in the Go codebase that should come from
configuration, and replace them with config references.

### Step 1: Find Hardcoded Network Values

```bash
rg '"(localhost|127\.0\.0\.1|0\.0\.0\.0)"' -t go -n
rg '":\d{4}"' -t go -n
rg 'http://localhost' -t go -n
```

Expected output:
```
internal/config/config.go:59:            Host:            "localhost",
internal/config/config.go:76:            AllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
cmd/webshop/main.go:23:    defaultPort        = ":8080"
```

### Step 2: Find Hardcoded Timeouts and Limits

```bash
rg '\d+\s*\*\s*time\.' -t go -n
rg 'maxHeaderBytes|max.*[Cc]onn|[Ll]imit.*=.*\d' -t go -n
```

Expected output:
```
cmd/webshop/main.go:24:    readTimeout        = 15 * time.Second
cmd/webshop/main.go:25:    writeTimeout       = 15 * time.Second
cmd/webshop/main.go:26:    maxHeaderBytes     = 1 << 20
cmd/webshop/main.go:27:    shutdownGracePeriod = 30 * time.Second
```

### Step 3: Find Hardcoded Business Logic Values

```bash
rg '0\.085|8\.5%' -t go -n    # Tax rate
rg 'limit > 100|limit.*20' -t go -n  # Pagination
rg '\$.*%.2f' -t go -n        # Currency format
```

Expected:
```
internal/models/order.go:112:    return 0.085
internal/handlers/user.go:78:    if limit <= 0 || limit > 100 {
internal/handlers/user.go:79:        limit = 20
internal/models/product.go:103:    return fmt.Sprintf("$%.2f", p.Price)
```

### Step 4: Replace Hardcoded Port with Config

```bash
# Preview
sd 'defaultPort\s*=\s*":8080"' 'defaultPort = fmt.Sprintf(":%d", cfg.Port)' < cmd/webshop/main.go
```

### Step 5: Replace Hardcoded Tax Rate

```bash
# Preview
sd 'return 0\.085' 'return cfg.TaxRate // TODO: load from config' < internal/models/order.go
```

### Step 6: Replace Hardcoded Currency Symbol

```bash
# Preview
sd 'fmt\.Sprintf\("\$%.2f", p\.Price\)' 'fmt.Sprintf("%s%.2f", cfg.CurrencySymbol, p.Price)' < internal/models/product.go
```

### Step 7: Build the Full Externalization Report

```bash
echo "=== Configuration Externalization Audit ==="
echo ""
echo "NETWORK VALUES:"
rg -n '"(localhost|127\.0\.0\.1|:\d{4})"' -t go
echo ""
echo "TIMEOUTS:"
rg -n '\d+\s*\*\s*time\.(Second|Minute)' -t go
echo ""
echo "BUSINESS LOGIC:"
rg -n '0\.085|limit.*=.*20|"\$%.2f"' -t go
echo ""
echo "CREDENTIALS:"
rg -n '"(webshop_pass|super-secret|disable)"' -t go
echo ""
echo "Total hardcoded values to externalize:"
rg -c '"(localhost|127\.0\.0\.1|:\d{4}|webshop_|super-secret)"' -t go | \
  awk -F: '{sum+=$2} END {print "  " sum " occurrences across " NR " files"}'
```

---

## Exercises

These exercises combine all scenarios. Each should be attempted as a complete
workflow.

### Exercise 1: Security Sweep

Run the full security audit from Scenario 1. Then extend it:

```bash
# Find any TODO or FIXME related to security
rg '(TODO|FIXME|BUG|HACK).*\b(auth|secur|cred|token|jwt|ssl|tls|encrypt|hash|salt)\b' -i -t go

# Count total security-related issues
rg -c '(TODO|FIXME|BUG|HACK)' -t go | awk -F: '{sum+=$2} END {print "Total: " sum}'
```

### Exercise 2: Standardize All Logging

Apply the full logging standardization from Scenario 2 as a preview pipeline:

```bash
# Build the complete multi-pass pipeline
for f in $(rg -l 'fmt\.Print' -t go); do
    echo "=== $f ==="
    sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")' < "$f" | \
      sd 'fmt\.Println\("\[WARN\] (.+)"\)' 'log.Warn("$1")' | \
      sd 'fmt\.Println\("\[INFO\] (.+)"\)' 'log.Info("$1")' | \
      sd 'fmt\.Println\("\[DEBUG\] (.+)"\)' 'log.Debug("$1")' | \
      rg 'fmt\.Println\("\[' || echo "  (all bracket logs converted)"
done
```

### Exercise 3: Error Wrapping Upgrade

Find all `errors.New` calls and preview wrapping them with file-specific context:

```bash
# For each model file, preview the upgrade
for f in internal/models/*.go; do
    pkg=$(basename "$f" .go)
    echo "=== $f (context: $pkg) ==="
    sd 'return errors\.New\("(.+)"\)' "return fmt.Errorf(\"${pkg}: \$1\")" < "$f" | rg 'fmt\.Errorf'
done
```

### Exercise 4: Route Versioning

Preview adding `/api/v1/` to all routes, then verify the health endpoint is excluded:

```bash
sd 'HandleFunc\("/(users|products|orders)' 'HandleFunc("/api/v1/$1' < cmd/webshop/main.go | \
  rg 'HandleFunc'
```

Count versioned vs unversioned routes in the output.

### Exercise 5: Full Module Rename

Preview renaming the Go module from `rg-sd-mastery` to `webshop-api`:

```bash
# Step 1: Count all references
rg -c 'rg-sd-mastery' -t go

# Step 2: Preview
for f in $(rg -l 'rg-sd-mastery' -t go); do
    echo "=== $f ==="
    sd 'rg-sd-mastery' 'webshop-api' < "$f" | rg 'webshop-api'
done

# Step 3: Verify no old references in preview
for f in $(rg -l 'rg-sd-mastery' -t go); do
    sd 'rg-sd-mastery' 'webshop-api' < "$f" | rg 'rg-sd-mastery' || true
done
```

### Exercise 6: DEPRECATED Function Cleanup

Find all deprecated functions and preview removing them:

```bash
# Step 1: Find all DEPRECATED markers
rg 'DEPRECATED' -t go -n

# Step 2: Find the deprecated functions
rg -A5 '// DEPRECATED:' -t go

# Step 3: Count them
rg -c 'DEPRECATED' -t go
```

### Exercise 7: JSON Tag Normalization

Build and run the complete JSON tag normalization across all model files:

```bash
# Step 1: Find all snake_case tags
rg -o 'json:"\w+_\w+"' internal/models/ --no-filename | sort -u

# Step 2: Count per file
rg -c 'json:"\w+_\w+"' internal/models/

# Step 3: Preview conversion for order.go (the file with the most tags)
sd 'json:"tax_amount"' 'json:"taxAmount"' < internal/models/order.go | \
  sd 'json:"total_amount"' 'json:"totalAmount"' | \
  sd 'json:"payment_status"' 'json:"paymentStatus"' | \
  sd 'json:"created_at"' 'json:"createdAt"' | \
  sd 'json:"shipped_at"' 'json:"shippedAt"' | \
  sd 'json:"unit_price"' 'json:"unitPrice"' | \
  sd 'json:"product_id"' 'json:"productId"' | \
  rg 'json:"\w+_\w+"'
# Remaining snake_case tags in the output (if any) need more passes
```

### Exercise 8: Cross-Cutting Concern Audit

Use `rg` to build a complete audit of cross-cutting concerns in the codebase:

```bash
echo "=== Logging Calls ==="
rg -c 'fmt\.Print|log\.' -t go | sort -t: -k2 -n -r

echo ""
echo "=== Error Handling ==="
rg -c 'errors\.New|fmt\.Errorf|return err' -t go | sort -t: -k2 -n -r

echo ""
echo "=== Code Comments ==="
rg -c '// (TODO|FIXME|BUG|HACK|NOTE|DEPRECATED):' -t go | sort -t: -k2 -n -r

echo ""
echo "=== Comment Breakdown ==="
rg -o '// (TODO|FIXME|BUG|HACK|NOTE|DEPRECATED):' -t go --no-filename | sort | uniq -c | sort -rn
```

### Exercise 9: The Grand Refactoring Challenge

This is the ultimate exercise. Perform ALL of these changes in sequence as previews.
Track the order of operations:

```bash
# 1. Rename module path
echo "Step 1: Module rename"
rg -c 'rg-sd-mastery' -t go

# 2. Standardize logging
echo "Step 2: Logging audit"
rg -c 'fmt\.Print' -t go

# 3. Externalize secrets
echo "Step 3: Secret audit"
rg -c '"(webshop_pass|super-secret)"' -t go

# 4. Version API routes
echo "Step 4: Route audit"
rg -c 'HandleFunc\("/' cmd/webshop/main.go

# 5. Normalize JSON tags
echo "Step 5: Tag audit"
rg -c 'json:"\w+_\w+"' internal/models/

# 6. Count total changes needed
echo ""
echo "Total refactoring scope:"
echo "  Module references: $(rg -c 'rg-sd-mastery' -t go | awk -F: '{s+=$2}END{print s}')"
echo "  Log statements:    $(rg -c 'fmt\.Print' -t go | awk -F: '{s+=$2}END{print s}')"
echo "  Hardcoded secrets: $(rg -c '"(webshop_pass|super-secret)"' -t go | awk -F: '{s+=$2}END{print s}')"
echo "  Routes to version: $(rg -c 'HandleFunc\("/' cmd/webshop/main.go)"
echo "  Tags to normalize: $(rg -c 'json:"\w+_\w+"' internal/models/ | awk -F: '{s+=$2}END{print s}')"
```

### Exercise 10: Build Your Own Audit Script

Using everything you have learned, build a one-shot audit command that reports
the health of the codebase:

```bash
echo "=========================================="
echo " WEBSHOP CODEBASE HEALTH AUDIT"
echo "=========================================="
echo ""
echo "FILES:"
rg --files -t go | wc -l | xargs echo "  Go files:"
echo ""
echo "CODE QUALITY:"
rg -c '// TODO:' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  TODOs:       %d\n", s}'
rg -c '// FIXME:' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  FIXMEs:      %d\n", s}'
rg -c '// BUG:' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  BUGs:        %d\n", s}'
rg -c '// HACK:' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  HACKs:       %d\n", s}'
rg -c '// DEPRECATED:' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  DEPRECATED:  %d\n", s}'
echo ""
echo "SECURITY:"
rg -c '"(webshop_pass|super-secret)"' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  Hardcoded secrets: %d\n", s}'
rg -c 'sslmode.*disable' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  SSL disabled:      %d\n", s}'
echo ""
echo "LOGGING:"
rg -c 'fmt\.Println\("\[' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  Raw fmt log calls: %d\n", s}'
rg -c 'log\.(Info|Error|Warn|Debug)' -t go 2>/dev/null | awk -F: '{s+=$2}END{printf "  Structured logs:   %d\n", s}'
echo ""
echo "CONSISTENCY:"
rg -c 'json:"\w+_\w+"' internal/models/ 2>/dev/null | awk -F: '{s+=$2}END{printf "  Snake_case tags:   %d\n", s}'
rg -c 'json:"\w+[A-Z]\w+"' internal/models/ 2>/dev/null | awk -F: '{s+=$2}END{printf "  CamelCase tags:    %d\n", s}'
echo ""
echo "=========================================="
```

---

## Cheat Sheet: Complete Command Reference

| Scenario | rg Command | sd Command |
|----------|-----------|-----------|
| Find secrets | `rg -i 'secret\|password' -t go` | -- |
| Replace secret | -- | `sd 'JWTSecret:\s*"[^"]+"' 'JWTSecret: os.Getenv("JWT_SECRET")'` |
| Audit log levels | `rg -o '\[(ERROR\|INFO)\]' \| sort \| uniq -c` | -- |
| Replace log calls | -- | `sd 'fmt\.Println\("\[ERROR\] (.+)"\)' 'log.Error("$1")'` |
| Find bare returns | `rg 'return err$' -t go` | -- |
| Wrap error returns | -- | `sd 'return err$' 'return fmt.Errorf("ctx: %w", err)'` |
| Find routes | `rg 'HandleFunc\("/' -t go` | -- |
| Version routes | -- | `sd 'HandleFunc\("/(users)' 'HandleFunc("/api/v1/$1'` |
| Find hardcoded vals | `rg '"localhost"' -t go` | -- |
| Externalize config | -- | `sd '"localhost"' 'os.Getenv("DB_HOST")'` |
| Find exported funcs | `rg '^func [A-Z]' -t go -g '!*_test.go'` | -- |
| Module rename | `rg -l 'old/module' -t go` | `xargs sd 'old/module' 'new/module'` |
| Tag normalization | `rg 'json:"\w+_\w+"' internal/models/` | `sd 'json:"old_tag"' 'json:"newTag"'` |

---

## Key Takeaways

1. **Every refactoring follows the same pattern:** Audit with `rg`, plan the change,
   preview with `sd` via stdin, apply with `sd` to files, verify with `rg`.
2. **Start with the audit.** Understanding the scope prevents surprises. Use `rg -c`
   and `rg -l` before touching anything.
3. **Scope your changes.** Use `-t go`, `-g '!*_test.go'`, or directory paths.
   Never apply globally without scoping first.
4. **Chain `sd` passes for complex scenarios.** Logging conversion, tag normalization,
   and multi-aspect refactoring all benefit from piped multi-pass `sd`.
5. **Git is mandatory.** Always work in a clean git state. `git diff` to review,
   `git checkout .` to revert. No exceptions.
6. **Build audit scripts.** Reusable audit commands (like Exercise 10) catch regressions
   over time. Run them before and after every refactoring session.
7. **Document your commands.** When you find a useful `rg | sd` pipeline, save it.
   These become your team's refactoring playbook.

---

Previous: [Chapter 8 -- Power Combos: rg + sd Together](../08-rg-sd-power-combos/README.md)
