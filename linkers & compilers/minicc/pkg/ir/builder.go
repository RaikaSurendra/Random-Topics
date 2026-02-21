package ir

import (
	"fmt"
	"minicc/pkg/lexer"
	"minicc/pkg/parser"
)

// Builder translates an AST into three-address code IR.
type Builder struct {
	tempCount  int
	labelCount int
	current    *Function
	program    *IRProgram
	globals    map[string]bool // set of global variable names
}

// NewBuilder creates a new IR builder.
func NewBuilder() *Builder {
	return &Builder{
		program: &IRProgram{},
		globals: make(map[string]bool),
	}
}

// Build translates the entire program AST into IR.
func (b *Builder) Build(prog *parser.Program) *IRProgram {
	// First pass: collect global variable names
	for _, decl := range prog.Declarations {
		if vd, ok := decl.(*parser.VarDecl); ok {
			b.program.Globals = append(b.program.Globals, vd.Name)
			b.globals[vd.Name] = true
		}
	}

	// Second pass: build IR for each function
	for _, decl := range prog.Declarations {
		if fn, ok := decl.(*parser.FuncDecl); ok {
			b.buildFunc(fn)
		}
	}

	// Third pass: emit global variable initializers into a synthetic __init function
	// (only if there are initialized globals)
	b.buildGlobalInits(prog)

	return b.program
}

func (b *Builder) buildGlobalInits(prog *parser.Program) {
	var inits []parser.Declaration
	for _, decl := range prog.Declarations {
		if vd, ok := decl.(*parser.VarDecl); ok && vd.Init != nil {
			inits = append(inits, vd)
		}
	}
	if len(inits) == 0 {
		return
	}

	b.tempCount = 0
	b.labelCount = 0
	fn := &Function{Name: "__init"}
	b.current = fn

	for _, decl := range inits {
		vd := decl.(*parser.VarDecl)
		src := b.buildExpr(vd.Init)
		b.emit(Instruction{Op: OpStoreGlobal, Dst: Var(vd.Name), Left: src})
	}
	b.emit(Instruction{Op: OpReturn, Left: Imm(0)})

	fn.TempCount = b.tempCount
	b.program.Functions = append(b.program.Functions, fn)
}

func (b *Builder) buildFunc(fn *parser.FuncDecl) {
	b.tempCount = 0
	b.labelCount = 0
	irFn := &Function{
		Name: fn.Name,
	}
	for _, p := range fn.Params {
		irFn.Params = append(irFn.Params, p.Name)
	}
	b.current = irFn

	b.buildBlock(fn.Body)

	irFn.TempCount = b.tempCount
	b.program.Functions = append(b.program.Functions, irFn)
}

func (b *Builder) buildBlock(block *parser.BlockStmt) {
	for _, item := range block.Items {
		b.buildStmt(item)
	}
}

func (b *Builder) buildStmt(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.VarDecl:
		b.buildVarDecl(s)
	case *parser.ExprStmt:
		b.buildExpr(s.Expr)
	case *parser.ReturnStmt:
		val := b.buildExpr(s.Value)
		b.emit(Instruction{Op: OpReturn, Left: val})
	case *parser.IfStmt:
		b.buildIf(s)
	case *parser.WhileStmt:
		b.buildWhile(s)
	case *parser.BlockStmt:
		b.buildBlock(s)
	}
}

func (b *Builder) buildVarDecl(vd *parser.VarDecl) {
	b.current.LocalCount++
	if vd.Init != nil {
		src := b.buildExpr(vd.Init)
		b.emit(Instruction{Op: OpCopy, Dst: Var(vd.Name), Left: src})
	}
}

func (b *Builder) buildIf(s *parser.IfStmt) {
	cond := b.buildExpr(s.Condition)

	if s.Else != nil {
		elseLabel := b.newLabel("else")
		endLabel := b.newLabel("endif")

		b.emit(Instruction{Op: OpJumpIfZero, Left: cond, Dst: Label(elseLabel)})
		b.buildStmt(s.Then)
		b.emit(Instruction{Op: OpJump, Dst: Label(endLabel)})
		b.emit(Instruction{Op: OpLabel, Dst: Label(elseLabel)})
		b.buildStmt(s.Else)
		b.emit(Instruction{Op: OpLabel, Dst: Label(endLabel)})
	} else {
		endLabel := b.newLabel("endif")
		b.emit(Instruction{Op: OpJumpIfZero, Left: cond, Dst: Label(endLabel)})
		b.buildStmt(s.Then)
		b.emit(Instruction{Op: OpLabel, Dst: Label(endLabel)})
	}
}

func (b *Builder) buildWhile(s *parser.WhileStmt) {
	topLabel := b.newLabel("while")
	endLabel := b.newLabel("endwhile")

	b.emit(Instruction{Op: OpLabel, Dst: Label(topLabel)})
	cond := b.buildExpr(s.Condition)
	b.emit(Instruction{Op: OpJumpIfZero, Left: cond, Dst: Label(endLabel)})
	b.buildStmt(s.Body)
	b.emit(Instruction{Op: OpJump, Dst: Label(topLabel)})
	b.emit(Instruction{Op: OpLabel, Dst: Label(endLabel)})
}

// buildExpr emits IR for an expression, returns the operand holding the result.
func (b *Builder) buildExpr(expr parser.Expression) Operand {
	switch e := expr.(type) {
	case *parser.IntLiteral:
		dst := b.newTemp()
		b.emit(Instruction{Op: OpLoadImm, Dst: dst, Left: Imm(e.Value)})
		return dst

	case *parser.Ident:
		if b.globals[e.Name] {
			dst := b.newTemp()
			b.emit(Instruction{Op: OpLoadGlobal, Dst: dst, Left: Var(e.Name)})
			return dst
		}
		return Var(e.Name)

	case *parser.UnaryExpr:
		return b.buildUnary(e)

	case *parser.BinaryExpr:
		return b.buildBinary(e)

	case *parser.AssignExpr:
		src := b.buildExpr(e.Value)
		if b.globals[e.Name] {
			b.emit(Instruction{Op: OpStoreGlobal, Dst: Var(e.Name), Left: src})
		} else {
			b.emit(Instruction{Op: OpCopy, Dst: Var(e.Name), Left: src})
		}
		return src

	case *parser.CallExpr:
		return b.buildCall(e)

	default:
		return None()
	}
}

func (b *Builder) buildUnary(e *parser.UnaryExpr) Operand {
	operand := b.buildExpr(e.Operand)
	dst := b.newTemp()
	switch e.Op {
	case lexer.Minus:
		b.emit(Instruction{Op: OpNeg, Dst: dst, Left: operand})
	case lexer.Not:
		b.emit(Instruction{Op: OpNot, Dst: dst, Left: operand})
	}
	return dst
}

func (b *Builder) buildBinary(e *parser.BinaryExpr) Operand {
	// Short-circuit && and ||
	if e.Op == lexer.And {
		return b.buildShortCircuitAnd(e)
	}
	if e.Op == lexer.Or {
		return b.buildShortCircuitOr(e)
	}

	left := b.buildExpr(e.Left)
	right := b.buildExpr(e.Right)
	dst := b.newTemp()

	op := tokenToOp(e.Op)
	b.emit(Instruction{Op: op, Dst: dst, Left: left, Right: right})
	return dst
}

func (b *Builder) buildShortCircuitAnd(e *parser.BinaryExpr) Operand {
	result := b.newTemp()
	falseLabel := b.newLabel("and_false")
	endLabel := b.newLabel("and_end")

	left := b.buildExpr(e.Left)
	b.emit(Instruction{Op: OpJumpIfZero, Left: left, Dst: Label(falseLabel)})

	right := b.buildExpr(e.Right)
	b.emit(Instruction{Op: OpJumpIfZero, Left: right, Dst: Label(falseLabel)})

	b.emit(Instruction{Op: OpLoadImm, Dst: result, Left: Imm(1)})
	b.emit(Instruction{Op: OpJump, Dst: Label(endLabel)})

	b.emit(Instruction{Op: OpLabel, Dst: Label(falseLabel)})
	b.emit(Instruction{Op: OpLoadImm, Dst: result, Left: Imm(0)})

	b.emit(Instruction{Op: OpLabel, Dst: Label(endLabel)})
	return result
}

func (b *Builder) buildShortCircuitOr(e *parser.BinaryExpr) Operand {
	result := b.newTemp()
	trueLabel := b.newLabel("or_true")
	endLabel := b.newLabel("or_end")

	left := b.buildExpr(e.Left)
	b.emit(Instruction{Op: OpJumpIfNotZero, Left: left, Dst: Label(trueLabel)})

	right := b.buildExpr(e.Right)
	b.emit(Instruction{Op: OpJumpIfNotZero, Left: right, Dst: Label(trueLabel)})

	b.emit(Instruction{Op: OpLoadImm, Dst: result, Left: Imm(0)})
	b.emit(Instruction{Op: OpJump, Dst: Label(endLabel)})

	b.emit(Instruction{Op: OpLabel, Dst: Label(trueLabel)})
	b.emit(Instruction{Op: OpLoadImm, Dst: result, Left: Imm(1)})

	b.emit(Instruction{Op: OpLabel, Dst: Label(endLabel)})
	return result
}

func (b *Builder) buildCall(e *parser.CallExpr) Operand {
	// Evaluate and emit all arguments as PARAM instructions
	var argOperands []Operand
	for _, arg := range e.Args {
		argOperands = append(argOperands, b.buildExpr(arg))
	}
	for _, ao := range argOperands {
		b.emit(Instruction{Op: OpParam, Left: ao})
	}

	dst := b.newTemp()
	b.emit(Instruction{
		Op:       OpCall,
		Dst:      dst,
		Left:     Func(e.Name),
		ArgCount: len(e.Args),
	})
	return dst
}

// --- Helpers ---

func (b *Builder) newTemp() Operand {
	n := b.tempCount
	b.tempCount++
	return Temp(n)
}

func (b *Builder) newLabel(prefix string) string {
	n := b.labelCount
	b.labelCount++
	return fmt.Sprintf(".L_%s_%d", prefix, n)
}

func (b *Builder) emit(instr Instruction) {
	b.current.Instructions = append(b.current.Instructions, instr)
}

func tokenToOp(tok lexer.TokenType) OpCode {
	switch tok {
	case lexer.Plus:
		return OpAdd
	case lexer.Minus:
		return OpSub
	case lexer.Star:
		return OpMul
	case lexer.Slash:
		return OpDiv
	case lexer.Percent:
		return OpMod
	case lexer.Equal:
		return OpEq
	case lexer.NotEqual:
		return OpNeq
	case lexer.Less:
		return OpLt
	case lexer.Greater:
		return OpGt
	case lexer.LessEq:
		return OpLte
	case lexer.GreaterEq:
		return OpGte
	default:
		return OpAdd // shouldn't happen
	}
}
