package ir

import (
	"minicc/pkg/lexer"
	"minicc/pkg/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: lex → parse → build IR
func buildIR(t *testing.T, source string) *IRProgram {
	t.Helper()
	tokens, lexErrs := lexer.New(source).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	prog, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	return NewBuilder().Build(prog)
}

// helper: find a function by name
func findFunc(t *testing.T, prog *IRProgram, name string) *Function {
	t.Helper()
	for _, fn := range prog.Functions {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("function %q not found", name)
	return nil
}

// helper: check that a specific opcode exists in the function
func hasOp(fn *Function, op OpCode) bool {
	for _, instr := range fn.Instructions {
		if instr.Op == op {
			return true
		}
	}
	return false
}

// helper: count occurrences of an opcode
func countOp(fn *Function, op OpCode) int {
	n := 0
	for _, instr := range fn.Instructions {
		if instr.Op == op {
			n++
		}
	}
	return n
}

func TestEmptyMain(t *testing.T) {
	ir := buildIR(t, `int main() { return 0; }`)
	if len(ir.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(ir.Functions))
	}
	fn := ir.Functions[0]
	if fn.Name != "main" {
		t.Errorf("expected 'main', got %q", fn.Name)
	}
	if !hasOp(fn, OpLoadImm) {
		t.Error("expected OpLoadImm for literal 0")
	}
	if !hasOp(fn, OpReturn) {
		t.Error("expected OpReturn")
	}
}

func TestArithmetic(t *testing.T) {
	ir := buildIR(t, `int main() { return 2 + 3 * 4; }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpMul) {
		t.Error("expected OpMul")
	}
	if !hasOp(fn, OpAdd) {
		t.Error("expected OpAdd")
	}
}

func TestAllArithOps(t *testing.T) {
	ir := buildIR(t, `int main() { return 10 + 3 - 7 * 2 / 1 % 5; }`)
	fn := findFunc(t, ir, "main")
	for _, op := range []OpCode{OpAdd, OpSub, OpMul, OpDiv, OpMod} {
		if !hasOp(fn, op) {
			t.Errorf("expected %s", op)
		}
	}
}

func TestUnaryMinus(t *testing.T) {
	ir := buildIR(t, `int main() { return -42; }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpNeg) {
		t.Error("expected OpNeg")
	}
}

func TestUnaryNot(t *testing.T) {
	ir := buildIR(t, `int main() { return !0; }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpNot) {
		t.Error("expected OpNot")
	}
}

func TestComparisons(t *testing.T) {
	ir := buildIR(t, `int main() { int a = 1 < 2; int b = 1 > 2; int c = 1 <= 2; int d = 1 >= 2; int e = 1 == 2; int f = 1 != 2; return a; }`)
	fn := findFunc(t, ir, "main")
	for _, op := range []OpCode{OpLt, OpGt, OpLte, OpGte, OpEq, OpNeq} {
		if !hasOp(fn, op) {
			t.Errorf("expected %s", op)
		}
	}
}

func TestVarDeclAndAssign(t *testing.T) {
	ir := buildIR(t, `int main() { int x = 5; x = x + 1; return x; }`)
	fn := findFunc(t, ir, "main")
	if fn.LocalCount != 1 {
		t.Errorf("expected 1 local, got %d", fn.LocalCount)
	}
	// Should have OpCopy for both init and assignment
	if countOp(fn, OpCopy) < 2 {
		t.Errorf("expected at least 2 OpCopy, got %d", countOp(fn, OpCopy))
	}
}

func TestIfElse(t *testing.T) {
	ir := buildIR(t, `int main() { if (1) { return 1; } else { return 0; } }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpJumpIfZero) {
		t.Error("expected OpJumpIfZero for if condition")
	}
	if !hasOp(fn, OpJump) {
		t.Error("expected OpJump for else skip")
	}
	if countOp(fn, OpLabel) < 2 {
		t.Error("expected at least 2 labels for if/else")
	}
}

func TestWhile(t *testing.T) {
	ir := buildIR(t, `int main() { int i = 0; while (i < 10) { i = i + 1; } return i; }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpJumpIfZero) {
		t.Error("expected OpJumpIfZero for while condition")
	}
	if countOp(fn, OpJump) < 1 {
		t.Error("expected OpJump for loop back")
	}
	if countOp(fn, OpLabel) < 2 {
		t.Error("expected at least 2 labels for while loop")
	}
}

func TestFunctionCallNoArgs(t *testing.T) {
	ir := buildIR(t, `int f() { return 42; } int main() { return f(); }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpCall) {
		t.Error("expected OpCall")
	}
	// No params for zero-arg call
	if hasOp(fn, OpParam) {
		t.Error("unexpected OpParam for zero-arg call")
	}
}

func TestFunctionCallWithArgs(t *testing.T) {
	ir := buildIR(t, `int add(int a, int b) { return a + b; } int main() { return add(3, 4); }`)
	fn := findFunc(t, ir, "main")
	if countOp(fn, OpParam) != 2 {
		t.Errorf("expected 2 OpParam, got %d", countOp(fn, OpParam))
	}
	if !hasOp(fn, OpCall) {
		t.Error("expected OpCall")
	}
}

func TestFunctionParams(t *testing.T) {
	ir := buildIR(t, `int add(int a, int b) { return a + b; } int main() { return add(1, 2); }`)
	fn := findFunc(t, ir, "add")
	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0] != "a" || fn.Params[1] != "b" {
		t.Errorf("expected params [a, b], got %v", fn.Params)
	}
}

func TestGlobalVariable(t *testing.T) {
	ir := buildIR(t, `int g = 1; int main() { return g; }`)
	if len(ir.Globals) != 1 || ir.Globals[0] != "g" {
		t.Errorf("expected globals [g], got %v", ir.Globals)
	}
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpLoadGlobal) {
		t.Error("expected OpLoadGlobal")
	}
}

func TestGlobalAssignment(t *testing.T) {
	ir := buildIR(t, `int g = 0; int main() { g = 5; return g; }`)
	fn := findFunc(t, ir, "main")
	if !hasOp(fn, OpStoreGlobal) {
		t.Error("expected OpStoreGlobal")
	}
}

func TestGlobalInit(t *testing.T) {
	ir := buildIR(t, `int g = 42; int main() { return g; }`)
	init := findFunc(t, ir, "__init")
	if !hasOp(init, OpStoreGlobal) {
		t.Error("expected __init to store global")
	}
}

func TestNoInitForUninitGlobal(t *testing.T) {
	ir := buildIR(t, `int g; int main() { g = 1; return g; }`)
	// Should NOT have __init function since g has no initializer
	for _, fn := range ir.Functions {
		if fn.Name == "__init" {
			t.Error("unexpected __init for uninitialized global")
		}
	}
}

func TestShortCircuitAnd(t *testing.T) {
	ir := buildIR(t, `int main() { return 1 && 0; }`)
	fn := findFunc(t, ir, "main")
	// Short-circuit && uses JumpIfZero
	if countOp(fn, OpJumpIfZero) < 1 {
		t.Error("expected OpJumpIfZero for short-circuit &&")
	}
}

func TestShortCircuitOr(t *testing.T) {
	ir := buildIR(t, `int main() { return 0 || 1; }`)
	fn := findFunc(t, ir, "main")
	if countOp(fn, OpJumpIfNotZero) < 1 {
		t.Error("expected OpJumpIfNotZero for short-circuit ||")
	}
}

func TestNestedBlocks(t *testing.T) {
	ir := buildIR(t, `int main() { int x = 0; { int y = 1; x = y; } return x; }`)
	fn := findFunc(t, ir, "main")
	if fn.LocalCount != 2 {
		t.Errorf("expected 2 locals, got %d", fn.LocalCount)
	}
}

func TestInstructionStringer(t *testing.T) {
	instr := Instruction{Op: OpAdd, Dst: Temp(0), Left: Temp(1), Right: Temp(2)}
	s := instr.String()
	if !strings.Contains(s, "ADD") {
		t.Errorf("expected 'ADD' in %q", s)
	}
}

func TestOperandStringer(t *testing.T) {
	tests := []struct {
		op     Operand
		expect string
	}{
		{Temp(3), "t3"},
		{Var("x"), "x"},
		{Imm(42), "42"},
		{Label(".L_end"), ".L_end"},
		{Func("foo"), "foo"},
		{None(), "_"},
	}
	for _, tt := range tests {
		if s := tt.op.String(); s != tt.expect {
			t.Errorf("Operand.String(): expected %q, got %q", tt.expect, s)
		}
	}
}

// --- Integration: all valid test programs build IR without panic ---

func TestValidProgramsBuildIR(t *testing.T) {
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
			prog := buildIR(t, string(data))
			if prog == nil {
				t.Error("nil IR program")
				return
			}
			if len(prog.Functions) == 0 {
				t.Error("expected at least 1 function")
			}
			// Every program should have a main function
			found := false
			for _, fn := range prog.Functions {
				if fn.Name == "main" {
					found = true
					if !hasOp(fn, OpReturn) {
						t.Errorf("main function has no return instruction")
					}
				}
			}
			if !found {
				t.Error("no 'main' function in IR")
			}
		})
	}
	if count != 20 {
		t.Errorf("expected 20 valid test programs, found %d", count)
	}
}
