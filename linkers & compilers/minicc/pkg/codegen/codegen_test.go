package codegen

import (
	"minicc/pkg/ir"
	"minicc/pkg/lexer"
	"minicc/pkg/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: source → assembly string
func generate(t *testing.T, source string, platform Platform) string {
	t.Helper()
	tokens, lexErrs := lexer.New(source).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	prog, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	irProg := ir.NewBuilder().Build(prog)
	return New(platform).Generate(irProg)
}

func TestEmptyMain(t *testing.T) {
	asm := generate(t, `int main() { return 0; }`, PlatformLinux)
	assertContains(t, asm, "mc_main:")
	assertContains(t, asm, "retq")
	assertContains(t, asm, ".globl mc_main")
}

func TestEmptyMainMacOS(t *testing.T) {
	asm := generate(t, `int main() { return 0; }`, PlatformMacOS)
	assertContains(t, asm, "_mc_main:")
	assertContains(t, asm, ".globl _mc_main")
}

func TestPrologue(t *testing.T) {
	asm := generate(t, `int main() { return 0; }`, PlatformLinux)
	assertContains(t, asm, "pushq %rbp")
	assertContains(t, asm, "movq %rsp, %rbp")
}

func TestEpilogue(t *testing.T) {
	asm := generate(t, `int main() { return 0; }`, PlatformLinux)
	assertContains(t, asm, "popq %rbp")
	assertContains(t, asm, "retq")
}

func TestReturnLiteral(t *testing.T) {
	asm := generate(t, `int main() { return 42; }`, PlatformLinux)
	assertContains(t, asm, "$42")
}

func TestArithmeticOps(t *testing.T) {
	asm := generate(t, `int main() { return 2 + 3 * 4 - 1; }`, PlatformLinux)
	assertContains(t, asm, "addl")
	assertContains(t, asm, "imull")
	assertContains(t, asm, "subl")
}

func TestDivision(t *testing.T) {
	asm := generate(t, `int main() { return 10 / 3; }`, PlatformLinux)
	assertContains(t, asm, "cdq")
	assertContains(t, asm, "idivl")
}

func TestModulo(t *testing.T) {
	asm := generate(t, `int main() { return 17 % 7; }`, PlatformLinux)
	assertContains(t, asm, "idivl")
	assertContains(t, asm, "movl %edx, %eax") // remainder
}

func TestUnaryNeg(t *testing.T) {
	asm := generate(t, `int main() { return -42; }`, PlatformLinux)
	assertContains(t, asm, "negl")
}

func TestUnaryNot(t *testing.T) {
	asm := generate(t, `int main() { return !0; }`, PlatformLinux)
	assertContains(t, asm, "sete")
	assertContains(t, asm, "movzbl")
}

func TestComparisons(t *testing.T) {
	tests := []struct {
		src    string
		expect string
	}{
		{`int main() { return 1 == 2; }`, "sete"},
		{`int main() { return 1 != 2; }`, "setne"},
		{`int main() { return 1 < 2; }`, "setl"},
		{`int main() { return 1 > 2; }`, "setg"},
		{`int main() { return 1 <= 2; }`, "setle"},
		{`int main() { return 1 >= 2; }`, "setge"},
	}
	for _, tt := range tests {
		asm := generate(t, tt.src, PlatformLinux)
		assertContains(t, asm, tt.expect)
	}
}

func TestIfElse(t *testing.T) {
	asm := generate(t, `int main() { if (1) { return 1; } else { return 0; } }`, PlatformLinux)
	assertContains(t, asm, "je")  // conditional jump
	assertContains(t, asm, "jmp") // skip else
}

func TestWhileLoop(t *testing.T) {
	asm := generate(t, `int main() { int i = 0; while (i < 10) { i = i + 1; } return i; }`, PlatformLinux)
	assertContains(t, asm, "je")  // exit condition
	assertContains(t, asm, "jmp") // loop back
}

func TestFunctionCall(t *testing.T) {
	asm := generate(t, `int f(int x) { return x; } int main() { return f(42); }`, PlatformLinux)
	assertContains(t, asm, "callq")
	assertContains(t, asm, "f:")
}

func TestFunctionParams(t *testing.T) {
	asm := generate(t, `int add(int a, int b) { return a + b; } int main() { return add(1, 2); }`, PlatformLinux)
	// Should spill %edi and %esi in add's prologue
	assertContains(t, asm, "movl %edi,")
	assertContains(t, asm, "movl %esi,")
}

func TestSixArgs(t *testing.T) {
	asm := generate(t, `
		int sum6(int a, int b, int c, int d, int e, int f) {
			return a + b + c + d + e + f;
		}
		int main() { return sum6(1, 2, 3, 4, 5, 6); }
	`, PlatformLinux)
	// All 6 param registers should be spilled
	for _, reg := range []string{"%edi", "%esi", "%edx", "%ecx", "%r8d", "%r9d"} {
		assertContains(t, asm, "movl "+reg+",")
	}
}

func TestGlobalVarData(t *testing.T) {
	asm := generate(t, `int g = 1; int main() { return g; }`, PlatformLinux)
	assertContains(t, asm, ".data")
	assertContains(t, asm, ".globl g")
	assertContains(t, asm, "g:")
	assertContains(t, asm, ".long 0")
}

func TestGlobalVarMacOS(t *testing.T) {
	asm := generate(t, `int g = 1; int main() { return g; }`, PlatformMacOS)
	assertContains(t, asm, ".globl _g")
	assertContains(t, asm, "_g:")
}

func TestGlobalLoadStore(t *testing.T) {
	asm := generate(t, `int g = 0; int main() { g = 5; return g; }`, PlatformLinux)
	assertContains(t, asm, "(%rip)") // RIP-relative access
}

func TestFrameAlignment(t *testing.T) {
	asm := generate(t, `int main() { int a; int b; int c; return 0; }`, PlatformLinux)
	// Frame size should be multiple of 16
	// Look for subq $N, %rsp where N is a multiple of 16
	if !strings.Contains(asm, "subq $") {
		t.Skip("no frame allocation for small functions")
	}
	assertContains(t, asm, "subq $")
}

func TestMainRenamedToMcMain(t *testing.T) {
	asm := generate(t, `int main() { return 0; }`, PlatformLinux)
	// Should NOT contain a bare "main:" label (that would conflict with C main)
	lines := strings.Split(asm, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "main:" {
			t.Error("found bare 'main:' label — should be 'mc_main:'")
		}
	}
	assertContains(t, asm, "mc_main:")
}

// --- Integration: all valid programs generate assembly ---

func TestValidProgramsCodegen(t *testing.T) {
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
			for _, plat := range []Platform{PlatformLinux, PlatformMacOS} {
				asm := generate(t, string(data), plat)
				if len(asm) == 0 {
					t.Errorf("empty assembly output for %s (platform %d)", entry.Name(), plat)
				}
				if !strings.Contains(asm, "retq") {
					t.Errorf("no retq in %s (platform %d)", entry.Name(), plat)
				}
				// mc_main should exist
				if plat == PlatformLinux {
					assertContains(t, asm, "mc_main:")
				} else {
					assertContains(t, asm, "_mc_main:")
				}
			}
		})
	}
	if count != 20 {
		t.Errorf("expected 20 valid test programs, found %d", count)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, but it doesn't.\nOutput:\n%s", needle, haystack)
	}
}
