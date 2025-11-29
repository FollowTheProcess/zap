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
	Name string `yaml:"name"`

	// The [token.Ident] token.
	Token token.Token `yaml:"token"`

	// Type is the kind of ast node, in this case [KindIdent].
	Type Kind `yaml:"type"`
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
	Value string `yaml:"value"`

	// The [token.Text] token.
	Token token.Token `yaml:"token"`

	// Type is [KindTextLiteral].
	Type Kind `yaml:"type"`
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
	Value string `yaml:"value"`

	// The [token.URL] token.
	Token token.Token `yaml:"token"`

	// Type is [KindURL].
	Type Kind `yaml:"type"`
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
	Expr Expression `yaml:"expr"`

	// Open is the opening interpolation token.
	Open token.Token `yaml:"open"`

	// Close is the closing interpolation token.
	Close token.Token `yaml:"close"`

	// Type is [KindInterp].
	Type Kind `yaml:"type"`
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

// InterpolatedExpression is a composite expression comprised of
// an [Interp] sandwiched between two other expressions.
//
// The Left and Right expressions, may also be arbitrarily nested
// InterpolatedExpressions.
//
// This allows us to do precedence based parsing in a manner similar to
// binary expressions where there is a Left, Right and an Op with different
// precedence dependencing on whether it's '+', '*', '/' etc.
//
// In our case, there is only one precedence based operator: an interp, which
// has the highest precedence.
type InterpolatedExpression struct {
	// Left is the expression before the interp, it may itself
	// be any other valid expression, including more nested InterpolatedExpressions.
	Left Expression `yaml:"left"`

	// Right is the expression immediately after the interp, like Left
	// it may also be any valid expression.
	Right Expression `yaml:"right"`

	// Interp is the [Interp] expression in between Left and Right.
	Interp Interp `yaml:"interp"`

	// Type is [KindInterpolatedExpression].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the left expression if there is one,
// else the start of the interp.
func (i InterpolatedExpression) Start() token.Token {
	if i.Left != nil {
		return i.Left.Start()
	}

	return i.Interp.Start()
}

// End returns the last token associated with the right expression if there is one,
// else the end of the interp.
func (i InterpolatedExpression) End() token.Token {
	if i.Right != nil {
		return i.Right.End()
	}

	return i.Interp.End()
}

// Kind returns [KindInterpolatedExpression].
func (i InterpolatedExpression) Kind() Kind {
	return i.Type
}

// expressionNode marks an [InterpolatedExpression] as an [Expression].
func (i InterpolatedExpression) expressionNode() {}

// Body is the http body expression.
type Body struct {
	// Value is the raw body contents.
	Value string `yaml:"value"`

	// Token is the [token.Body] token.
	Token token.Token `yaml:"token"`

	// Type is [KindBody].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the body, which
// is just the [token.Body].
func (b Body) Start() token.Token {
	return b.Token
}

// End returns the last token associated with the body, which is
// also just the [token.Body].
func (b Body) End() token.Token {
	return b.Token
}

// Kind returns [KindBody].
func (b Body) Kind() Kind {
	return b.Type
}

// expressionNode marks a [Body] as an [Expression].
func (b Body) expressionNode() {}

// BodyFile is a http body from a filepath.
type BodyFile struct {
	// Value is the expression of the filepath.
	Value Expression `yaml:"value"`

	// Token is the [token.LeftAngle] token.
	Token token.Token `yaml:"token"`

	// Type is [KindBodyFile].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the BodyFile, which
// is the [token.LeftAngle].
func (b BodyFile) Start() token.Token {
	return b.Token
}

// End returns the last token associated with the BodyFile, which is
// the final token in the filepath expression.
func (b BodyFile) End() token.Token {
	if b.Value != nil {
		return b.Value.End()
	}

	return b.Token
}

// Kind returns [KindBodyFile].
func (b BodyFile) Kind() Kind {
	return b.Type
}

// expressionNode marks a [BodyFile] as an [Expression].
func (b BodyFile) expressionNode() {}
