package codegen

import (
	"fmt"
	"minicc/pkg/ir"
	"strings"
)

// Platform selects assembly conventions.
type Platform int

const (
	PlatformMacOS Platform = iota
	PlatformLinux
)

// CodeGen translates IR into x86-64 assembly (AT&T syntax).
type CodeGen struct {
	out          strings.Builder
	platform     Platform
	varOffset    map[string]int // variable/temp → stack offset from RBP (negative)
	nextOffset   int            // next available stack offset
	frameSize    int            // total frame size (positive, multiple of 16)
	paramStaging []ir.Operand   // collects PARAM operands for the next CALL
}

// New creates a code generator for the given platform.
func New(platform Platform) *CodeGen {
	return &CodeGen{platform: platform}
}

// Generate emits x86-64 assembly for the entire IR program.
func (g *CodeGen) Generate(prog *ir.IRProgram) string {
	g.out.Reset()

	// Emit global variables in .data
	if len(prog.Globals) > 0 {
		g.line("    .data")
		for _, name := range prog.Globals {
			sym := g.sym(name)
			g.line("    .globl %s", sym)
			g.line("%s:", sym)
			g.line("    .long 0")
		}
		g.line("")
	}

	g.line("    .text")
	for _, fn := range prog.Functions {
		g.genFunc(fn)
	}

	return g.out.String()
}

func (g *CodeGen) genFunc(fn *ir.Function) {
	g.varOffset = make(map[string]int)
	g.nextOffset = 0
	g.paramStaging = nil

	// Allocate stack slots: params first, then scan all instructions
	for _, p := range fn.Params {
		g.alloc(p)
	}
	for _, instr := range fn.Instructions {
		g.allocOp(instr.Dst)
		g.allocOp(instr.Left)
		g.allocOp(instr.Right)
	}

	// Frame size: 16-byte aligned
	g.frameSize = -g.nextOffset
	if g.frameSize%16 != 0 {
		g.frameSize += 16 - (g.frameSize % 16)
	}

	name := g.funcSym(fn.Name)
	g.line("")
	g.line("    .globl %s", name)
	g.line("%s:", name)

	// Prologue
	g.line("    pushq %%rbp")
	g.line("    movq %%rsp, %%rbp")
	if g.frameSize > 0 {
		g.line("    subq $%d, %%rsp", g.frameSize)
	}

	// Spill register params to stack
	regs32 := []string{"%edi", "%esi", "%edx", "%ecx", "%r8d", "%r9d"}
	for i, p := range fn.Params {
		if i >= 6 {
			break
		}
		g.line("    movl %s, %d(%%rbp)", regs32[i], g.varOffset[p])
	}

	// Emit body
	for _, instr := range fn.Instructions {
		g.genInstr(instr)
	}

	// Fallthrough safety
	g.line("    xorl %%eax, %%eax")
	g.epilogue()
}

func (g *CodeGen) genInstr(i ir.Instruction) {
	switch i.Op {
	case ir.OpLoadImm:
		g.line("    movl $%d, %%eax", i.Left.Value)
		g.store(i.Dst)

	case ir.OpCopy:
		g.load(i.Left, "%eax")
		g.store(i.Dst)

	case ir.OpAdd:
		g.load(i.Left, "%eax")
		g.load(i.Right, "%ecx")
		g.line("    addl %%ecx, %%eax")
		g.store(i.Dst)

	case ir.OpSub:
		g.load(i.Left, "%eax")
		g.load(i.Right, "%ecx")
		g.line("    subl %%ecx, %%eax")
		g.store(i.Dst)

	case ir.OpMul:
		g.load(i.Left, "%eax")
		g.load(i.Right, "%ecx")
		g.line("    imull %%ecx, %%eax")
		g.store(i.Dst)

	case ir.OpDiv:
		g.load(i.Left, "%eax")
		g.line("    cdq")
		g.load(i.Right, "%ecx")
		g.line("    idivl %%ecx")
		g.store(i.Dst)

	case ir.OpMod:
		g.load(i.Left, "%eax")
		g.line("    cdq")
		g.load(i.Right, "%ecx")
		g.line("    idivl %%ecx")
		g.line("    movl %%edx, %%eax")
		g.store(i.Dst)

	case ir.OpNeg:
		g.load(i.Left, "%eax")
		g.line("    negl %%eax")
		g.store(i.Dst)

	case ir.OpNot:
		g.load(i.Left, "%eax")
		g.line("    cmpl $0, %%eax")
		g.line("    sete %%al")
		g.line("    movzbl %%al, %%eax")
		g.store(i.Dst)

	case ir.OpEq:
		g.cmp(i, "sete")
	case ir.OpNeq:
		g.cmp(i, "setne")
	case ir.OpLt:
		g.cmp(i, "setl")
	case ir.OpGt:
		g.cmp(i, "setg")
	case ir.OpLte:
		g.cmp(i, "setle")
	case ir.OpGte:
		g.cmp(i, "setge")

	case ir.OpLabel:
		g.line("%s:", i.Dst.Name)

	case ir.OpJump:
		g.line("    jmp %s", i.Dst.Name)

	case ir.OpJumpIfZero:
		g.load(i.Left, "%eax")
		g.line("    cmpl $0, %%eax")
		g.line("    je %s", i.Dst.Name)

	case ir.OpJumpIfNotZero:
		g.load(i.Left, "%eax")
		g.line("    cmpl $0, %%eax")
		g.line("    jne %s", i.Dst.Name)

	case ir.OpReturn:
		g.load(i.Left, "%eax")
		g.epilogue()

	case ir.OpParam:
		// Collect the operand; CALL will load them into arg registers.
		g.paramStaging = append(g.paramStaging, i.Left)

	case ir.OpCall:
		g.call(i)

	case ir.OpLoadGlobal:
		s := g.sym(i.Left.Name)
		g.line("    movl %s(%%rip), %%eax", s)
		g.store(i.Dst)

	case ir.OpStoreGlobal:
		g.load(i.Left, "%eax")
		s := g.sym(i.Dst.Name)
		g.line("    movl %%eax, %s(%%rip)", s)
	}
}

func (g *CodeGen) cmp(i ir.Instruction, setcc string) {
	g.load(i.Left, "%eax")
	g.load(i.Right, "%ecx")
	g.line("    cmpl %%ecx, %%eax")
	g.line("    %s %%al", setcc)
	g.line("    movzbl %%al, %%eax")
	g.store(i.Dst)
}

func (g *CodeGen) call(i ir.Instruction) {
	argRegs := []string{"%edi", "%esi", "%edx", "%ecx", "%r8d", "%r9d"}

	// Load staged params into argument registers.
	// We must be careful: loading a later param might clobber a register
	// used by an earlier param's source. To avoid this, first spill all
	// param values to their stack slots (they're already there if they're
	// vars/temps), then load from stack into registers.
	//
	// For immediate values in paramStaging, store them to a temp first.
	// For var/temp operands, they're already on the stack.
	// So: just load each from stack/imm into the arg register in order.
	// Since we load into different registers, there's no clobber issue
	// (we only clobber the target register which isn't a source for others).
	for idx, op := range g.paramStaging {
		if idx >= 6 {
			break
		}
		g.load(op, argRegs[idx])
	}

	fnSym := g.funcSym(i.Left.Name)
	g.line("    callq %s", fnSym)
	g.store(i.Dst) // result in %eax

	g.paramStaging = nil
}

func (g *CodeGen) epilogue() {
	if g.frameSize > 0 {
		g.line("    addq $%d, %%rsp", g.frameSize)
	}
	g.line("    popq %%rbp")
	g.line("    retq")
}

// --- Stack allocation ---

func (g *CodeGen) alloc(name string) {
	if _, ok := g.varOffset[name]; !ok {
		g.nextOffset -= 8
		g.varOffset[name] = g.nextOffset
	}
}

func (g *CodeGen) allocOp(op ir.Operand) {
	switch op.Kind {
	case ir.OpndVar:
		g.alloc(op.Name)
	case ir.OpndTemp:
		g.alloc(g.tempKey(op))
	}
}

func (g *CodeGen) tempKey(op ir.Operand) string {
	return fmt.Sprintf("__t%d", op.Value)
}

func (g *CodeGen) offset(op ir.Operand) int {
	switch op.Kind {
	case ir.OpndVar:
		return g.varOffset[op.Name]
	case ir.OpndTemp:
		return g.varOffset[g.tempKey(op)]
	}
	return 0
}

// load emits code to load an operand value into a 32-bit register.
func (g *CodeGen) load(op ir.Operand, reg string) {
	switch op.Kind {
	case ir.OpndImm:
		g.line("    movl $%d, %s", op.Value, reg)
	case ir.OpndVar, ir.OpndTemp:
		g.line("    movl %d(%%rbp), %s", g.offset(op), reg)
	}
}

// store emits code to store %eax into the operand's stack slot.
func (g *CodeGen) store(dst ir.Operand) {
	switch dst.Kind {
	case ir.OpndVar, ir.OpndTemp:
		g.line("    movl %%eax, %d(%%rbp)", g.offset(dst))
	}
}

// --- Symbol naming ---

func (g *CodeGen) sym(name string) string {
	if g.platform == PlatformMacOS {
		return "_" + name
	}
	return name
}

func (g *CodeGen) funcSym(name string) string {
	actual := name
	if name == "main" {
		actual = "mc_main"
	}
	return g.sym(actual)
}

func (g *CodeGen) line(format string, args ...any) {
	fmt.Fprintf(&g.out, format+"\n", args...)
}
