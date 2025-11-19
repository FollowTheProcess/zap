package ast

import "go.followtheprocess.codes/zap/internal/syntax/token"

// Expression is an expression node.
type Expression interface {
	Node
	expressionNode() // Prevents accidental misuse as another node type
}

// Ident is a named identifier expression.
type Ident struct {
	// Name is the ident's name.
	Name string

	// The [token.Ident] token.
	Token token.Token
}

// Start returns the first token in the Ident, which is
// the [token.Ident].
func (i Ident) Start() token.Token {
	return i.Token
}

// End returns the last token in the Ident, which is also
// the [token.Ident].
func (i Ident) End() token.Token {
	return i.Token
}

// statementNode marks an [Ident] as an [ast.Expression].
func (i Ident) expressionNode() {}
