package semantic

import "fmt"

// SymbolKind distinguishes variables from functions.
type SymbolKind int

const (
	SymVar  SymbolKind = iota
	SymFunc
)

func (k SymbolKind) String() string {
	switch k {
	case SymVar:
		return "variable"
	case SymFunc:
		return "function"
	default:
		return fmt.Sprintf("SymbolKind(%d)", int(k))
	}
}

// Symbol represents a declared name in the program.
type Symbol struct {
	Name   string
	Kind   SymbolKind
	Arity  int  // number of parameters (functions only)
	Global bool // declared at top level
	Line   int  // declaration line
}

// Scope represents a lexical scope containing symbol bindings.
type Scope struct {
	symbols map[string]*Symbol
	parent  *Scope
}

// NewScope creates a new scope with the given parent (nil for global).
func NewScope(parent *Scope) *Scope {
	return &Scope{
		symbols: make(map[string]*Symbol),
		parent:  parent,
	}
}

// Define adds a symbol to this scope. Returns an error if already defined in this scope.
func (s *Scope) Define(sym *Symbol) error {
	if existing, ok := s.symbols[sym.Name]; ok {
		return fmt.Errorf("'%s' already declared in this scope (previously at line %d)", sym.Name, existing.Line)
	}
	s.symbols[sym.Name] = sym
	return nil
}

// Resolve looks up a name in this scope and all parent scopes.
// Returns the symbol and true if found, nil and false otherwise.
func (s *Scope) Resolve(name string) (*Symbol, bool) {
	if sym, ok := s.symbols[name]; ok {
		return sym, true
	}
	if s.parent != nil {
		return s.parent.Resolve(name)
	}
	return nil, false
}

// ResolveLocal looks up a name in this scope only (not parents).
func (s *Scope) ResolveLocal(name string) (*Symbol, bool) {
	sym, ok := s.symbols[name]
	return sym, ok
}
