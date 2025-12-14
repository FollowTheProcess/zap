package ast

import "go.followtheprocess.codes/zap/internal/syntax/token"

// Expression is an expression node.
type Expression interface {
	Node
	expressionNode() // Prevents accidental misuse as another node type
}

// Ident is a named identifier expression.
type Ident struct {
	Name  string      `yaml:"name"`  // Name is the ident's name
	Token token.Token `yaml:"token"` // The [token.Ident] token.
	Type  Kind        `yaml:"type"`  // Type is the kind of ast node, in this case [KindIdent].
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

// Builtin is a built in identifier, prefixed with a '$'.
//
// E.g. 'uuid' is a perfectly fine identifier for any user to use for a variable
// but '$uuid' is a builtin that returns a random UUIDv4.
//
// The Builtin AST node is functionality identical to an [Ident], but must
// be separate to allow differentiating builtins from regular idents.
type Builtin struct {
	Name   string      `yaml:"name"`   // Name is the name of the builtin ident.
	Dollar token.Token `yaml:"dollar"` // Dollar is the opening [token.Dollar].
	Token  token.Token `yaml:"token"`  // The [token.Ident] token.
	Type   Kind        `yaml:"type"`   // Type is the kind of ast node, in this case [KindBuiltin].
}

// Start returns the first token in the Builtin, which is
// the [token.Dollar] opening it.
func (b Builtin) Start() token.Token {
	return b.Dollar
}

// End returns the last token in the Builtin, which is
// the [token.Ident].
func (b Builtin) End() token.Token {
	return b.Token
}

// Kind returns [KindBuiltin].
func (b Builtin) Kind() Kind {
	return b.Type
}

// statementNode marks a [Builtin] as an [ast.Expression].
func (b Builtin) expressionNode() {}

// TextLiteral is a literal text expression.
type TextLiteral struct {
	Value string      `yaml:"value"` // The text value (unquoted)
	Token token.Token `yaml:"token"` // The [token.Text] token.
	Type  Kind        `yaml:"type"`  // Type is [KindTextLiteral].
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

// Interp is a text interpolation expression.
type Interp struct {
	Expr  Expression  `yaml:"expr"`  // Expr is the expression inside the interpolation.
	Open  token.Token `yaml:"open"`  // Open is the opening interpolation token.
	Close token.Token `yaml:"close"` // Close is the closing interpolation token.
	Type  Kind        `yaml:"type"`  // Type is [KindInterp].
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
	Left   Expression `yaml:"left"`   // Left is the expression before the interp
	Right  Expression `yaml:"right"`  // Right is the expression immediately after the interp
	Interp Interp     `yaml:"interp"` // Interp is the [Interp] expression in between Left and Right
	Type   Kind       `yaml:"type"`   // Type is [KindInterpolatedExpression]
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
	Value Expression  `yaml:"value"` // Value is the expression of the filepath.
	Token token.Token `yaml:"token"` // Token is the [token.LeftAngle] token.
	Type  Kind        `yaml:"type"`  // Type is [KindBodyFile].
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

// SelectorExpression represents an expression followed by a selector.
type SelectorExpression struct {
	Expr     Expression // Expr is the expression
	Selector Ident      // Selector is the selector
	Type     Kind       // Type is [KindSelector]
}

// Start returns the first token associated with the SelectorExpression, which
// is the first token in the Expr.
func (s SelectorExpression) Start() token.Token {
	if s.Expr != nil {
		return s.Expr.Start()
	}

	return s.Selector.Start()
}

// End returns the last token associated with the SelectorExpression, which
// is the ident token of the selector.
func (s SelectorExpression) End() token.Token {
	return s.Selector.End()
}

// Kind returns [KindSelector].
func (s SelectorExpression) Kind() Kind {
	return s.Type
}

// expressionNode marks a [SelectorExpression] as an [Expression].
func (s SelectorExpression) expressionNode() {}
