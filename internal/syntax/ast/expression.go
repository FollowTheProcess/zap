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

	// Type is the kind of ast node, in this case [KindIdent].
	Type Kind
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

// Kind returns [KindIdent].
func (i Ident) Kind() Kind {
	return i.Type
}

// statementNode marks an [Ident] as an [ast.Expression].
func (i Ident) expressionNode() {}

// TextLiteral is a literal text expression.
type TextLiteral struct {
	// The text value (unquoted)
	Value string

	// The [token.Text] token.
	Token token.Token

	// Type is [KindTextLiteral].
	Type Kind
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

// Kind returns [KindTextLiteral].
func (t TextLiteral) Kind() Kind {
	return t.Type
}

// expressionNode marks a [TextLiteral] as an [ast.Expression].
func (t TextLiteral) expressionNode() {}

// URL is a literal URL expression.
type URL struct {
	// The url value (unquoted)
	Value string

	// The [token.URL] token.
	Token token.Token

	// Type is [KindURL].
	Type Kind
}

// Start returns the first token of the URL, which is
// obviously just the [token.URL].
func (u URL) Start() token.Token {
	return u.Token
}

// End returns the last token in the URL, which is also
// the [token.URL].
func (u URL) End() token.Token {
	return u.Token
}

// Kind returns [KindURL].
func (u URL) Kind() Kind {
	return u.Type
}

// expressionNode marks a [URL] as an [ast.Expression].
func (u URL) expressionNode() {}

// Interp is a text interpolation expression.
type Interp struct {
	// Expr is the expression inside the interpolation.
	Expr Expression

	// Open is the opening interpolation token.
	Open token.Token

	// Close is the closing interpolation token.
	Close token.Token

	// Type is [KindInterp].
	Type Kind
}

// Start returns the first token of the interpolation, i.e
// the opening [token.OpenInterp].
func (i Interp) Start() token.Token {
	return i.Open
}

// End returns the final token of the interpolation, i.e.
// the closing [token.CloseInterp].
func (i Interp) End() token.Token {
	return i.Close
}

// Kind returns [KindInterp].
func (i Interp) Kind() Kind {
	return i.Type
}

// expressionNode marks an [Interp] as an [ast.Expression].
func (i Interp) expressionNode() {}
