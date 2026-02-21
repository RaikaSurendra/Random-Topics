package parser

import (
	"minicc/pkg/lexer"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: tokenize then parse, fail on errors
func mustParse(t *testing.T, source string) *Program {
	t.Helper()
	tokens, lexErrs := lexer.New(source).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	prog, parseErrs := New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	return prog
}

// helper: tokenize then parse, expect errors
func parseWithErrors(t *testing.T, source string) (*Program, []string) {
	t.Helper()
	tokens, _ := lexer.New(source).Tokenize()
	return New(tokens).Parse()
}

func TestEmptyMain(t *testing.T) {
	prog := mustParse(t, `int main() { return 0; }`)
	if len(prog.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Declarations))
	}
	fn, ok := prog.Declarations[0].(*FuncDecl)
	if !ok {
		t.Fatal("expected FuncDecl")
	}
	if fn.Name != "main" {
		t.Errorf("expected name 'main', got %q", fn.Name)
	}
	if len(fn.Params) != 0 {
		t.Errorf("expected 0 params, got %d", len(fn.Params))
	}
	if len(fn.Body.Items) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(fn.Body.Items))
	}
	ret, ok := fn.Body.Items[0].(*ReturnStmt)
	if !ok {
		t.Fatal("expected ReturnStmt")
	}
	lit, ok := ret.Value.(*IntLiteral)
	if !ok {
		t.Fatal("expected IntLiteral")
	}
	if lit.Value != 0 {
		t.Errorf("expected 0, got %d", lit.Value)
	}
}

func TestVarDecl(t *testing.T) {
	prog := mustParse(t, `int main() { int x = 5; return x; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	if len(fn.Body.Items) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(fn.Body.Items))
	}
	vd, ok := fn.Body.Items[0].(*VarDecl)
	if !ok {
		t.Fatal("expected VarDecl")
	}
	if vd.Name != "x" {
		t.Errorf("expected 'x', got %q", vd.Name)
	}
	lit := vd.Init.(*IntLiteral)
	if lit.Value != 5 {
		t.Errorf("expected init=5, got %d", lit.Value)
	}
}

func TestVarDeclNoInit(t *testing.T) {
	prog := mustParse(t, `int main() { int x; return 0; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	vd := fn.Body.Items[0].(*VarDecl)
	if vd.Init != nil {
		t.Error("expected nil init for uninitialized var")
	}
}

func TestGlobalVar(t *testing.T) {
	prog := mustParse(t, `int g = 1; int main() { return g; }`)
	if len(prog.Declarations) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(prog.Declarations))
	}
	vd, ok := prog.Declarations[0].(*VarDecl)
	if !ok {
		t.Fatal("expected VarDecl for global")
	}
	if vd.Name != "g" {
		t.Errorf("expected 'g', got %q", vd.Name)
	}
}

func TestFuncWithParams(t *testing.T) {
	prog := mustParse(t, `int add(int a, int b) { return a + b; } int main() { return add(1, 2); }`)
	fn := prog.Declarations[0].(*FuncDecl)
	if fn.Name != "add" {
		t.Errorf("expected 'add', got %q", fn.Name)
	}
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "a" || fn.Params[1].Name != "b" {
		t.Errorf("expected params (a, b), got (%s, %s)", fn.Params[0].Name, fn.Params[1].Name)
	}
}

func TestArithmetic(t *testing.T) {
	prog := mustParse(t, `int main() { return 2 + 3 * 4; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	// Should be: BinaryExpr(+, 2, BinaryExpr(*, 3, 4))
	add, ok := ret.Value.(*BinaryExpr)
	if !ok || add.Op != lexer.Plus {
		t.Fatal("expected top-level +")
	}
	left, ok := add.Left.(*IntLiteral)
	if !ok || left.Value != 2 {
		t.Error("expected left=2")
	}
	mul, ok := add.Right.(*BinaryExpr)
	if !ok || mul.Op != lexer.Star {
		t.Fatal("expected right to be *")
	}
	if mul.Left.(*IntLiteral).Value != 3 || mul.Right.(*IntLiteral).Value != 4 {
		t.Error("expected 3 * 4")
	}
}

func TestPrecedence(t *testing.T) {
	// 1 + 2 * 3 == 7 → BinaryExpr(==, BinaryExpr(+, 1, BinaryExpr(*, 2, 3)), 7)
	prog := mustParse(t, `int main() { return 1 + 2 * 3 == 7; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	eq, ok := ret.Value.(*BinaryExpr)
	if !ok || eq.Op != lexer.Equal {
		t.Fatal("expected top-level ==")
	}
	add, ok := eq.Left.(*BinaryExpr)
	if !ok || add.Op != lexer.Plus {
		t.Fatal("expected left of == to be +")
	}
}

func TestUnaryMinus(t *testing.T) {
	prog := mustParse(t, `int main() { return -42; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	un, ok := ret.Value.(*UnaryExpr)
	if !ok || un.Op != lexer.Minus {
		t.Fatal("expected UnaryExpr(-)")
	}
	lit := un.Operand.(*IntLiteral)
	if lit.Value != 42 {
		t.Errorf("expected 42, got %d", lit.Value)
	}
}

func TestUnaryNot(t *testing.T) {
	prog := mustParse(t, `int main() { return !0; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	un, ok := ret.Value.(*UnaryExpr)
	if !ok || un.Op != lexer.Not {
		t.Fatal("expected UnaryExpr(!)")
	}
}

func TestIfElse(t *testing.T) {
	prog := mustParse(t, `int main() { if (1) { return 1; } else { return 0; } }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ifStmt, ok := fn.Body.Items[0].(*IfStmt)
	if !ok {
		t.Fatal("expected IfStmt")
	}
	if ifStmt.Else == nil {
		t.Error("expected else branch")
	}
}

func TestIfWithoutElse(t *testing.T) {
	prog := mustParse(t, `int main() { if (1) { return 1; } return 0; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ifStmt := fn.Body.Items[0].(*IfStmt)
	if ifStmt.Else != nil {
		t.Error("expected no else branch")
	}
}

func TestWhile(t *testing.T) {
	prog := mustParse(t, `int main() { int i = 0; while (i < 10) { i = i + 1; } return i; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	_, ok := fn.Body.Items[1].(*WhileStmt)
	if !ok {
		t.Fatal("expected WhileStmt")
	}
}

func TestNestedBlocks(t *testing.T) {
	prog := mustParse(t, `int main() { { { int x = 1; } } return 0; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	outer, ok := fn.Body.Items[0].(*BlockStmt)
	if !ok {
		t.Fatal("expected outer BlockStmt")
	}
	inner, ok := outer.Items[0].(*BlockStmt)
	if !ok {
		t.Fatal("expected inner BlockStmt")
	}
	if len(inner.Items) != 1 {
		t.Errorf("expected 1 item in inner block, got %d", len(inner.Items))
	}
}

func TestFunctionCall(t *testing.T) {
	prog := mustParse(t, `int f(int x) { return x; } int main() { return f(42); }`)
	fn := prog.Declarations[1].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	call, ok := ret.Value.(*CallExpr)
	if !ok {
		t.Fatal("expected CallExpr")
	}
	if call.Name != "f" {
		t.Errorf("expected call to 'f', got %q", call.Name)
	}
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}
}

func TestCallNoArgs(t *testing.T) {
	prog := mustParse(t, `int f() { return 0; } int main() { return f(); }`)
	fn := prog.Declarations[1].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	call := ret.Value.(*CallExpr)
	if len(call.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(call.Args))
	}
}

func TestCallMultipleArgs(t *testing.T) {
	prog := mustParse(t, `int f(int a, int b, int c) { return a; } int main() { return f(1, 2, 3); }`)
	fn := prog.Declarations[1].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	call := ret.Value.(*CallExpr)
	if len(call.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(call.Args))
	}
}

func TestAssignment(t *testing.T) {
	prog := mustParse(t, `int main() { int x = 0; x = 5; return x; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	es, ok := fn.Body.Items[1].(*ExprStmt)
	if !ok {
		t.Fatal("expected ExprStmt")
	}
	assign, ok := es.Expr.(*AssignExpr)
	if !ok {
		t.Fatal("expected AssignExpr")
	}
	if assign.Name != "x" {
		t.Errorf("expected 'x', got %q", assign.Name)
	}
}

func TestChainedAssignment(t *testing.T) {
	// a = b = 5 → AssignExpr("a", AssignExpr("b", 5))
	prog := mustParse(t, `int main() { int a; int b; a = b = 5; return a; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	es := fn.Body.Items[2].(*ExprStmt)
	outer, ok := es.Expr.(*AssignExpr)
	if !ok {
		t.Fatal("expected outer AssignExpr")
	}
	if outer.Name != "a" {
		t.Errorf("expected outer name 'a', got %q", outer.Name)
	}
	inner, ok := outer.Value.(*AssignExpr)
	if !ok {
		t.Fatal("expected inner AssignExpr")
	}
	if inner.Name != "b" {
		t.Errorf("expected inner name 'b', got %q", inner.Name)
	}
}

func TestLogicalOperators(t *testing.T) {
	prog := mustParse(t, `int main() { return 1 && 0 || 1; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	// || has lower precedence than &&, so: BinaryExpr(||, BinaryExpr(&&, 1, 0), 1)
	or, ok := ret.Value.(*BinaryExpr)
	if !ok || or.Op != lexer.Or {
		t.Fatal("expected top-level ||")
	}
	and, ok := or.Left.(*BinaryExpr)
	if !ok || and.Op != lexer.And {
		t.Fatal("expected left of || to be &&")
	}
}

func TestGroupedExpression(t *testing.T) {
	prog := mustParse(t, `int main() { return (1 + 2) * 3; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	mul, ok := ret.Value.(*BinaryExpr)
	if !ok || mul.Op != lexer.Star {
		t.Fatal("expected top-level *")
	}
	add, ok := mul.Left.(*BinaryExpr)
	if !ok || add.Op != lexer.Plus {
		t.Fatal("expected grouped + on left")
	}
}

func TestModulo(t *testing.T) {
	prog := mustParse(t, `int main() { return 17 % 7; }`)
	fn := prog.Declarations[0].(*FuncDecl)
	ret := fn.Body.Items[0].(*ReturnStmt)
	bin, ok := ret.Value.(*BinaryExpr)
	if !ok || bin.Op != lexer.Percent {
		t.Fatal("expected % operator")
	}
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		source string
		op     lexer.TokenType
	}{
		{`int main() { return 1 < 2; }`, lexer.Less},
		{`int main() { return 1 > 2; }`, lexer.Greater},
		{`int main() { return 1 <= 2; }`, lexer.LessEq},
		{`int main() { return 1 >= 2; }`, lexer.GreaterEq},
		{`int main() { return 1 == 2; }`, lexer.Equal},
		{`int main() { return 1 != 2; }`, lexer.NotEqual},
	}
	for _, tt := range tests {
		prog := mustParse(t, tt.source)
		fn := prog.Declarations[0].(*FuncDecl)
		ret := fn.Body.Items[0].(*ReturnStmt)
		bin, ok := ret.Value.(*BinaryExpr)
		if !ok || bin.Op != tt.op {
			t.Errorf("source %q: expected op %s, got %v", tt.source, tt.op, ret.Value)
		}
	}
}

func TestExprStmt(t *testing.T) {
	// Bare expression statement (e.g., function call as statement)
	prog := mustParse(t, `int f() { return 0; } int main() { f(); return 0; }`)
	fn := prog.Declarations[1].(*FuncDecl)
	es, ok := fn.Body.Items[0].(*ExprStmt)
	if !ok {
		t.Fatal("expected ExprStmt")
	}
	_, ok = es.Expr.(*CallExpr)
	if !ok {
		t.Fatal("expected CallExpr inside ExprStmt")
	}
}

// --- Error tests ---

func TestMissingSemicolon(t *testing.T) {
	_, errs := parseWithErrors(t, `int main() { return 0 }`)
	if len(errs) == 0 {
		t.Error("expected parse error for missing semicolon")
	}
}

func TestMissingCloseParen(t *testing.T) {
	_, errs := parseWithErrors(t, `int main() { return (1 + 2; }`)
	if len(errs) == 0 {
		t.Error("expected parse error for missing )")
	}
}

func TestUnexpectedToken(t *testing.T) {
	_, errs := parseWithErrors(t, `return 0;`)
	if len(errs) == 0 {
		t.Error("expected parse error for top-level return")
	}
}

// --- Integration: parse all valid test programs ---

func TestValidProgramsParse(t *testing.T) {
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
			prog, parseErrs := New(tokens).Parse()
			if len(parseErrs) > 0 {
				t.Errorf("parse errors in %s: %v", entry.Name(), parseErrs)
			}
			if prog == nil {
				t.Errorf("%s: nil program", entry.Name())
				return
			}
			// Every valid program should have at least one declaration (main)
			if len(prog.Declarations) == 0 {
				t.Errorf("%s: expected at least 1 declaration", entry.Name())
			}
			// At least one declaration should be a FuncDecl named "main"
			hasMain := false
			for _, d := range prog.Declarations {
				if fn, ok := d.(*FuncDecl); ok && fn.Name == "main" {
					hasMain = true
				}
			}
			if !hasMain {
				t.Errorf("%s: no main function found", entry.Name())
			}
		})
	}
	if count != 20 {
		t.Errorf("expected 20 valid test programs, found %d", count)
	}
}
