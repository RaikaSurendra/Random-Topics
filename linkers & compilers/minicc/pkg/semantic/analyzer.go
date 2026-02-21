package semantic

import (
	"fmt"
	"minicc/pkg/parser"
)

// Analyzer performs semantic analysis on a parsed AST.
// Uses a two-pass approach:
//   - Pass 1: register all top-level declarations (globals + functions)
//   - Pass 2: analyze function bodies (variable usage, call arity, returns)
type Analyzer struct {
	global *Scope
	scope  *Scope
	errors []string
	inFunc bool // true while analyzing inside a function body
}

// New creates a new semantic analyzer.
func New() *Analyzer {
	global := NewScope(nil)
	return &Analyzer{
		global: global,
		scope:  global,
	}
}

// Analyze runs semantic analysis on the program and returns any errors.
func (a *Analyzer) Analyze(prog *parser.Program) []string {
	// Pass 1: register all top-level names
	for _, decl := range prog.Declarations {
		switch d := decl.(type) {
		case *parser.FuncDecl:
			a.registerFunc(d)
		case *parser.VarDecl:
			a.registerGlobalVar(d)
		}
	}

	// Pass 2: analyze function bodies
	for _, decl := range prog.Declarations {
		if fn, ok := decl.(*parser.FuncDecl); ok {
			a.analyzeFunc(fn)
		}
	}

	// Check that main exists with 0 parameters
	mainSym, found := a.global.Resolve("main")
	if !found || mainSym.Kind != SymFunc {
		a.errorf(0, "program must contain a 'main' function")
	} else if mainSym.Arity != 0 {
		a.errorf(mainSym.Line, "'main' must have no parameters")
	}

	return a.errors
}

// --- Pass 1: register top-level declarations ---

func (a *Analyzer) registerFunc(fn *parser.FuncDecl) {
	sym := &Symbol{
		Name:   fn.Name,
		Kind:   SymFunc,
		Arity:  len(fn.Params),
		Global: true,
		Line:   fn.Line,
	}
	if err := a.global.Define(sym); err != nil {
		a.errorf(fn.Line, "function %s", err)
	}
}

func (a *Analyzer) registerGlobalVar(vd *parser.VarDecl) {
	sym := &Symbol{
		Name:   vd.Name,
		Kind:   SymVar,
		Global: true,
		Line:   vd.Line,
	}
	if err := a.global.Define(sym); err != nil {
		a.errorf(vd.Line, "global variable %s", err)
	}
}

// --- Pass 2: analyze function bodies ---

func (a *Analyzer) analyzeFunc(fn *parser.FuncDecl) {
	// Create function scope (child of global)
	funcScope := NewScope(a.global)
	a.scope = funcScope
	a.inFunc = true

	// Define parameters in function scope
	for _, p := range fn.Params {
		sym := &Symbol{Name: p.Name, Kind: SymVar, Line: p.Line}
		if err := funcScope.Define(sym); err != nil {
			a.errorf(p.Line, "parameter %s", err)
		}
	}

	// Analyze body
	hasReturn := a.analyzeBlock(fn.Body)

	if !hasReturn {
		a.errorf(fn.Line, "function '%s' may not return a value on all paths", fn.Name)
	}

	a.scope = a.global
	a.inFunc = false
}

// analyzeBlock analyzes a block statement and returns true if it definitely returns.
func (a *Analyzer) analyzeBlock(block *parser.BlockStmt) bool {
	prevScope := a.scope
	a.scope = NewScope(prevScope)
	defer func() { a.scope = prevScope }()

	hasReturn := false
	for _, item := range block.Items {
		if a.analyzeStmt(item) {
			hasReturn = true
		}
	}
	return hasReturn
}

// analyzeStmt analyzes a statement and returns true if it definitely returns.
func (a *Analyzer) analyzeStmt(stmt parser.Statement) bool {
	switch s := stmt.(type) {
	case *parser.VarDecl:
		a.analyzeVarDecl(s)
		return false
	case *parser.ExprStmt:
		a.analyzeExpr(s.Expr)
		return false
	case *parser.ReturnStmt:
		a.analyzeExpr(s.Value)
		return true
	case *parser.IfStmt:
		return a.analyzeIfStmt(s)
	case *parser.WhileStmt:
		a.analyzeWhileStmt(s)
		return false // while may not execute
	case *parser.BlockStmt:
		return a.analyzeBlock(s)
	default:
		return false
	}
}

func (a *Analyzer) analyzeVarDecl(vd *parser.VarDecl) {
	if vd.Init != nil {
		a.analyzeExpr(vd.Init)
	}
	sym := &Symbol{Name: vd.Name, Kind: SymVar, Line: vd.Line}
	if err := a.scope.Define(sym); err != nil {
		a.errorf(vd.Line, "%s", err)
	}
}

func (a *Analyzer) analyzeIfStmt(s *parser.IfStmt) bool {
	a.analyzeExpr(s.Condition)
	thenReturns := a.analyzeStmt(s.Then)
	elseReturns := false
	if s.Else != nil {
		elseReturns = a.analyzeStmt(s.Else)
	}
	// Only definitely returns if both branches return
	return thenReturns && elseReturns
}

func (a *Analyzer) analyzeWhileStmt(s *parser.WhileStmt) {
	a.analyzeExpr(s.Condition)
	a.analyzeStmt(s.Body)
}

// --- Expression analysis ---

func (a *Analyzer) analyzeExpr(expr parser.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *parser.IntLiteral:
		// nothing to check
	case *parser.Ident:
		sym, found := a.scope.Resolve(e.Name)
		if !found {
			a.errorf(e.Line, "undeclared variable '%s'", e.Name)
		} else if sym.Kind == SymFunc {
			a.errorf(e.Line, "'%s' is a function, not a variable", e.Name)
		}
	case *parser.BinaryExpr:
		a.analyzeExpr(e.Left)
		a.analyzeExpr(e.Right)
	case *parser.UnaryExpr:
		a.analyzeExpr(e.Operand)
	case *parser.AssignExpr:
		// Check the target exists and is a variable
		sym, found := a.scope.Resolve(e.Name)
		if !found {
			a.errorf(e.Line, "undeclared variable '%s'", e.Name)
		} else if sym.Kind == SymFunc {
			a.errorf(e.Line, "cannot assign to function '%s'", e.Name)
		}
		a.analyzeExpr(e.Value)
	case *parser.CallExpr:
		sym, found := a.scope.Resolve(e.Name)
		if !found {
			a.errorf(e.Line, "undeclared function '%s'", e.Name)
		} else if sym.Kind != SymFunc {
			a.errorf(e.Line, "'%s' is not a function", e.Name)
		} else if len(e.Args) != sym.Arity {
			a.errorf(e.Line, "function '%s' expects %d arguments, got %d",
				e.Name, sym.Arity, len(e.Args))
		}
		for _, arg := range e.Args {
			a.analyzeExpr(arg)
		}
	}
}

func (a *Analyzer) errorf(line int, format string, args ...any) {
	var msg string
	if line > 0 {
		msg = fmt.Sprintf("semantic error at line %d: %s", line, fmt.Sprintf(format, args...))
	} else {
		msg = fmt.Sprintf("semantic error: %s", fmt.Sprintf(format, args...))
	}
	a.errors = append(a.errors, msg)
}
