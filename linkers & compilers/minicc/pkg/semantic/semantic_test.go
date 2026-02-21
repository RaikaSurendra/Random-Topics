package semantic

import (
	"minicc/pkg/lexer"
	"minicc/pkg/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: lex → parse → analyze, return errors
func analyze(t *testing.T, source string) []string {
	t.Helper()
	tokens, lexErrs := lexer.New(source).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	prog, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	return New().Analyze(prog)
}

// helper: expect no errors
func mustAnalyze(t *testing.T, source string) {
	t.Helper()
	errs := analyze(t, source)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}
}

// helper: expect errors containing a substring
func expectError(t *testing.T, source string, substr string) {
	t.Helper()
	errs := analyze(t, source)
	if len(errs) == 0 {
		t.Fatalf("expected error containing %q, got none", substr)
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, substr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error containing %q, got: %v", substr, errs)
	}
}

// --- Valid programs ---

func TestValidEmptyMain(t *testing.T) {
	mustAnalyze(t, `int main() { return 0; }`)
}

func TestValidVarDecl(t *testing.T) {
	mustAnalyze(t, `int main() { int x = 5; return x; }`)
}

func TestValidGlobalVar(t *testing.T) {
	mustAnalyze(t, `int g = 1; int main() { return g; }`)
}

func TestValidFunctionCall(t *testing.T) {
	mustAnalyze(t, `int add(int a, int b) { return a + b; } int main() { return add(1, 2); }`)
}

func TestValidMutualRecursion(t *testing.T) {
	// Two-pass: both functions registered before bodies analyzed
	mustAnalyze(t, `
		int isEven(int n) {
			if (n == 0) { return 1; }
			return isOdd(n - 1);
		}
		int isOdd(int n) {
			if (n == 0) { return 0; }
			return isEven(n - 1);
		}
		int main() { return isEven(4); }
	`)
}

func TestValidNestedScopes(t *testing.T) {
	mustAnalyze(t, `
		int main() {
			int x = 1;
			{
				int x = 2;
				x = x + 1;
			}
			return x;
		}
	`)
}

func TestValidShadowParam(t *testing.T) {
	mustAnalyze(t, `
		int foo(int x) {
			int x = 7;
			return x;
		}
		int main() { return foo(1); }
	`)
}

func TestValidGlobalMutation(t *testing.T) {
	mustAnalyze(t, `
		int counter = 0;
		int inc(int n) { counter = counter + n; return counter; }
		int main() { inc(5); return counter; }
	`)
}

func TestValidManyArgs(t *testing.T) {
	mustAnalyze(t, `
		int sum6(int a, int b, int c, int d, int e, int f) {
			return a + b + c + d + e + f;
		}
		int main() { return sum6(1, 2, 3, 4, 5, 6); }
	`)
}

func TestValidIfElseReturns(t *testing.T) {
	mustAnalyze(t, `
		int main() {
			if (1) { return 1; } else { return 0; }
		}
	`)
}

// --- Error: undeclared variable ---

func TestUndeclaredVar(t *testing.T) {
	expectError(t, `int main() { return x; }`, "undeclared variable 'x'")
}

// --- Error: duplicate declaration in same scope ---

func TestDuplicateVarSameScope(t *testing.T) {
	expectError(t, `int main() { int x = 1; int x = 2; return x; }`, "already declared")
}

// --- Error: duplicate function ---

func TestDuplicateFunction(t *testing.T) {
	expectError(t, `
		int f() { return 0; }
		int f() { return 1; }
		int main() { return f(); }
	`, "already declared")
}

// --- Error: duplicate global var ---

func TestDuplicateGlobalVar(t *testing.T) {
	expectError(t, `
		int g = 1;
		int g = 2;
		int main() { return g; }
	`, "already declared")
}

// --- Error: missing main ---

func TestMissingMain(t *testing.T) {
	expectError(t, `int foo() { return 0; }`, "must contain a 'main' function")
}

// --- Error: main with parameters ---

func TestMainWithParams(t *testing.T) {
	expectError(t, `int main(int x) { return x; }`, "'main' must have no parameters")
}

// --- Error: wrong arity ---

func TestWrongArity(t *testing.T) {
	expectError(t, `
		int add(int a, int b) { return a + b; }
		int main() { return add(1, 2, 3); }
	`, "expects 2 arguments, got 3")
}

func TestWrongArityTooFew(t *testing.T) {
	expectError(t, `
		int add(int a, int b) { return a + b; }
		int main() { return add(1); }
	`, "expects 2 arguments, got 1")
}

// --- Error: calling undeclared function ---

func TestUndeclaredFunc(t *testing.T) {
	expectError(t, `int main() { return foo(1); }`, "undeclared function 'foo'")
}

// --- Error: calling a variable as function ---

func TestCallVariable(t *testing.T) {
	expectError(t, `int main() { int x = 5; return x(1); }`, "not a function")
}

// --- Error: using function as variable ---

func TestUseFuncAsVar(t *testing.T) {
	expectError(t, `
		int f() { return 0; }
		int main() { int x = f; return x; }
	`, "is a function, not a variable")
}

// --- Error: assigning to function ---

func TestAssignToFunc(t *testing.T) {
	expectError(t, `
		int f() { return 0; }
		int main() { f = 5; return 0; }
	`, "cannot assign to function")
}

// --- Error: missing return ---

func TestMissingReturn(t *testing.T) {
	expectError(t, `int main() { int x = 5; }`, "may not return")
}

func TestMissingReturnIfOnly(t *testing.T) {
	// if without else: not guaranteed to return
	expectError(t, `int main() { if (1) { return 0; } }`, "may not return")
}

// --- Scope edge cases ---

func TestVarOutOfScope(t *testing.T) {
	expectError(t, `
		int main() {
			{ int x = 5; }
			return x;
		}
	`, "undeclared variable 'x'")
}

func TestDuplicateParam(t *testing.T) {
	expectError(t, `
		int f(int a, int a) { return a; }
		int main() { return f(1, 2); }
	`, "already declared")
}

// --- Integration: all valid test programs pass semantic analysis ---

func TestValidProgramsSemantic(t *testing.T) {
	validDir := filepath.Join("..", "..", "testdata", "valid")
	entries, err := os.ReadDir(validDir)
	if err != nil {
		t.Fatalf("cannot read testdata/valid: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".mc") {
			continue
		}
		count++
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(validDir, entry.Name()))
			if err != nil {
				t.Fatalf("cannot read %s: %v", entry.Name(), err)
			}
			tokens, lexErrs := lexer.New(string(data)).Tokenize()
			if len(lexErrs) > 0 {
				t.Fatalf("lexer errors: %v", lexErrs)
			}
			prog, parseErrs := parser.New(tokens).Parse()
			if len(parseErrs) > 0 {
				t.Fatalf("parse errors: %v", parseErrs)
			}
			semErrs := New().Analyze(prog)
			if len(semErrs) > 0 {
				t.Errorf("semantic errors in %s: %v", entry.Name(), semErrs)
			}
		})
	}
	if count != 20 {
		t.Errorf("expected 20 valid test programs, found %d", count)
	}
}

// --- Integration: invalid programs that should produce semantic errors ---

func TestInvalidUndeclaredVar(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "02_undeclared_var.mc"))
	if err != nil {
		t.Fatal(err)
	}
	errs := analyze(t, string(data))
	if len(errs) == 0 {
		t.Error("expected semantic errors for undeclared var")
	}
}

func TestInvalidDuplicateDecl(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "03_duplicate_decl.mc"))
	if err != nil {
		t.Fatal(err)
	}
	errs := analyze(t, string(data))
	if len(errs) == 0 {
		t.Error("expected semantic errors for duplicate decl")
	}
}

func TestInvalidMissingMain(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "04_missing_main.mc"))
	if err != nil {
		t.Fatal(err)
	}
	errs := analyze(t, string(data))
	if len(errs) == 0 {
		t.Error("expected semantic errors for missing main")
	}
}

func TestInvalidWrongArity(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "05_wrong_arity.mc"))
	if err != nil {
		t.Fatal(err)
	}
	errs := analyze(t, string(data))
	if len(errs) == 0 {
		t.Error("expected semantic errors for wrong arity")
	}
}

func TestInvalidUndeclaredFunc(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "09_undeclared_func.mc"))
	if err != nil {
		t.Fatal(err)
	}
	errs := analyze(t, string(data))
	if len(errs) == 0 {
		t.Error("expected semantic errors for undeclared func")
	}
}
