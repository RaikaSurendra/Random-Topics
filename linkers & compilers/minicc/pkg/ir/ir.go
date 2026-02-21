package ir

import "fmt"

// OpCode represents a three-address code operation.
type OpCode int

const (
	OpAdd           OpCode = iota // dst = left + right
	OpSub                         // dst = left - right
	OpMul                         // dst = left * right
	OpDiv                         // dst = left / right
	OpMod                         // dst = left % right
	OpNeg                         // dst = -left
	OpNot                         // dst = !left
	OpEq                          // dst = (left == right)
	OpNeq                         // dst = (left != right)
	OpLt                          // dst = (left < right)
	OpGt                          // dst = (left > right)
	OpLte                         // dst = (left <= right)
	OpGte                         // dst = (left >= right)
	OpAnd                         // dst = left && right (short-circuit)
	OpOr                          // dst = left || right (short-circuit)
	OpCopy                        // dst = left
	OpLoadImm                     // dst = immediate value
	OpCall                        // dst = call FuncName(args...)
	OpParam                       // push parameter (left)
	OpReturn                      // return left
	OpJump                        // unconditional jump to label
	OpJumpIfZero                  // if left == 0, jump to label
	OpJumpIfNotZero               // if left != 0, jump to label
	OpLabel                       // label definition
	OpLoadGlobal                  // dst = global variable
	OpStoreGlobal                 // global variable = left
)

var opNames = [...]string{
	OpAdd: "ADD", OpSub: "SUB", OpMul: "MUL", OpDiv: "DIV", OpMod: "MOD",
	OpNeg: "NEG", OpNot: "NOT",
	OpEq: "EQ", OpNeq: "NEQ", OpLt: "LT", OpGt: "GT", OpLte: "LTE", OpGte: "GTE",
	OpAnd: "AND", OpOr: "OR",
	OpCopy: "COPY", OpLoadImm: "LOADI",
	OpCall: "CALL", OpParam: "PARAM", OpReturn: "RET",
	OpJump: "JMP", OpJumpIfZero: "JZ", OpJumpIfNotZero: "JNZ",
	OpLabel: "LABEL",
	OpLoadGlobal: "LOADG", OpStoreGlobal: "STOREG",
}

func (op OpCode) String() string {
	if int(op) < len(opNames) {
		return opNames[op]
	}
	return fmt.Sprintf("OpCode(%d)", int(op))
}

// OperandKind classifies an IR operand.
type OperandKind int

const (
	OpndNone  OperandKind = iota // unused operand
	OpndTemp                     // compiler-generated temporary (t0, t1, ...)
	OpndVar                      // named variable
	OpndImm                      // immediate integer
	OpndLabel                    // label reference
	OpndFunc                     // function name
)

// Operand is a single operand in a three-address instruction.
type Operand struct {
	Kind  OperandKind
	Name  string // variable name, label, or function name
	Value int    // immediate value or temp number
}

func (o Operand) String() string {
	switch o.Kind {
	case OpndNone:
		return "_"
	case OpndTemp:
		return fmt.Sprintf("t%d", o.Value)
	case OpndVar:
		return o.Name
	case OpndImm:
		return fmt.Sprintf("%d", o.Value)
	case OpndLabel:
		return o.Name
	case OpndFunc:
		return o.Name
	default:
		return "?"
	}
}

// Temp creates a temporary operand.
func Temp(n int) Operand { return Operand{Kind: OpndTemp, Value: n} }

// Var creates a variable operand.
func Var(name string) Operand { return Operand{Kind: OpndVar, Name: name} }

// Imm creates an immediate operand.
func Imm(val int) Operand { return Operand{Kind: OpndImm, Value: val} }

// Label creates a label operand.
func Label(name string) Operand { return Operand{Kind: OpndLabel, Name: name} }

// Func creates a function name operand.
func Func(name string) Operand { return Operand{Kind: OpndFunc, Name: name} }

// None is an unused operand.
func None() Operand { return Operand{Kind: OpndNone} }

// Instruction is a single three-address code instruction.
type Instruction struct {
	Op       OpCode
	Dst      Operand
	Left     Operand
	Right    Operand
	ArgCount int // for OpCall: number of preceding OpParam instructions
}

func (i Instruction) String() string {
	switch i.Op {
	case OpLabel:
		return fmt.Sprintf("%s:", i.Dst)
	case OpJump:
		return fmt.Sprintf("  JMP %s", i.Dst)
	case OpJumpIfZero:
		return fmt.Sprintf("  JZ %s, %s", i.Left, i.Dst)
	case OpJumpIfNotZero:
		return fmt.Sprintf("  JNZ %s, %s", i.Left, i.Dst)
	case OpReturn:
		return fmt.Sprintf("  RET %s", i.Left)
	case OpParam:
		return fmt.Sprintf("  PARAM %s", i.Left)
	case OpCall:
		return fmt.Sprintf("  %s = CALL %s, %d", i.Dst, i.Left, i.ArgCount)
	case OpLoadImm:
		return fmt.Sprintf("  %s = %d", i.Dst, i.Left.Value)
	case OpCopy:
		return fmt.Sprintf("  %s = %s", i.Dst, i.Left)
	case OpNeg, OpNot:
		return fmt.Sprintf("  %s = %s %s", i.Dst, i.Op, i.Left)
	case OpLoadGlobal:
		return fmt.Sprintf("  %s = LOADG %s", i.Dst, i.Left)
	case OpStoreGlobal:
		return fmt.Sprintf("  STOREG %s, %s", i.Dst, i.Left)
	default:
		return fmt.Sprintf("  %s = %s %s, %s", i.Dst, i.Op, i.Left, i.Right)
	}
}

// Function represents a compiled function's IR.
type Function struct {
	Name         string
	Params       []string
	Instructions []Instruction
	LocalCount   int // number of local variables (excluding params)
	TempCount    int // number of temporaries used
}

// IRProgram is the top-level IR output.
type IRProgram struct {
	Globals   []string
	Functions []*Function
}
