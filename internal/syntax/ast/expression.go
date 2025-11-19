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

// TextLiteral is a literal text expression.
type TextLiteral struct {
	// The text value (unquoted)
	Value string

	// The [token.Text] token.
	Token token.Token
}

// Start returns the first token of the TextLiteral, which is
// obviously just the [token.Text].
func (t TextLiteral) Start() token.Token {
	return t.Token
}

// End returns the last token in the TextLiteral, which is also
// the [token.Text].
func (t TextLiteral) End() token.Token {
	return t.Token
}

// statementNode marks a [TextLiteral] as an [ast.Expression].
func (t TextLiteral) expressionNode() {}
