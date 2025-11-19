package ast

import "go.followtheprocess.codes/zap/internal/syntax/token"

// Statement is a statement node.
type Statement interface {
	Node
	statementNode() // Prevents accidental misuse as another node type
}

// A VarStatement is a single variable declaration.
type VarStatement struct {
	// Value is the expression that [Ident] is being assigned to.
	Value Expression

	// Ident is the [Ident] node representing the identifier
	// the expression Value is being assigned to.
	Ident Ident

	// At is the '@' token declaring the variable.
	At token.Token

	// Type is the kind of node.
	Type Kind
}

// Start returns the first token in a VarStatement, which is
// the opening '@'.
func (v VarStatement) Start() token.Token {
	return v.At
}

// End returns the final token in a VarStatement which is
// the final token in the value expression.
func (v VarStatement) End() token.Token {
	return v.Value.End()
}

// Kind returns [KindVarStatement].
func (v VarStatement) Kind() Kind {
	return v.Type
}

// statementNode marks a [VarStatement] as an [ast.Statement].
func (v VarStatement) statementNode() {}
