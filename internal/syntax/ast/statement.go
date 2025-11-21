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

	// Type is the kind of node [KindVarStatement].
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
	if v.Value != nil {
		return v.Value.End()
	}

	return v.Ident.End()
}

// Kind returns [KindVarStatement].
func (v VarStatement) Kind() Kind {
	return v.Type
}

// statementNode marks a [VarStatement] as an [ast.Statement].
func (v VarStatement) statementNode() {}

// A PromptStatement is a single prompt declaration.
type PromptStatement struct {
	// Ident is the [Ident] node representing the identifier
	// the prompt Value is being assigned to.
	Ident Ident

	// Description is the [Text] node containing the description
	// of the prompt.
	Description TextLiteral

	// At is the '@' token declaring the prompt.
	At token.Token

	// Type is the kind of the node, in this case
	// [KindPromptStatement].
	Type Kind
}

// Start returns the first token in a PromptStatement, which is
// the opening '@'.
func (p PromptStatement) Start() token.Token {
	return p.At
}

// End returns the final token in a PromptStatement which is
// either the [TextLiteral] of the description if it's present
// or the [Ident] if not.
func (p PromptStatement) End() token.Token {
	if p.Description.Value != "" {
		return p.Description.End()
	}

	return p.Ident.End()
}

// Kind returns [KindPromptStatement].
func (p PromptStatement) Kind() Kind {
	return p.Type
}

// statementNode marks a [PromptStatement] as an [ast.Statement].
func (p PromptStatement) statementNode() {}

// Comment represents a single line comment.
type Comment struct {
	// Token is the [token.Comment] beginning the line comment.
	Token token.Token

	// Type is [KindComment].
	Type Kind
}

// Start returns the [token.Comment].
func (c Comment) Start() token.Token {
	return c.Token
}

// End also returns the [token.Comment].
func (c Comment) End() token.Token {
	return c.Token
}

// Kind returns [KindComment].
func (c Comment) Kind() Kind {
	return c.Type
}

// statementNode marks a [Comment] as an [ast.Statement].
func (c Comment) statementNode() {}

// Method represents a HTTP method.
type Method struct {
	// Token is the method token e.g. [token.MethodGet].
	Token token.Token

	// Type is [KindMethod].
	Type Kind
}

// Start returns the method token.
func (m Method) Start() token.Token {
	return m.Token
}

// End also returns the method token.
func (m Method) End() token.Token {
	return m.Token
}

// Kind returns [KindMethod].
func (m Method) Kind() Kind {
	return m.Type
}

// statementNode marks a [Method] as an [ast.Statement].
func (m Method) statementNode() {}

// Request is a single HTTP request.
type Request struct {
	// URL is the expression that when evaluated, returns the URL
	// for the request. May be a [TextLiteral] or an [Interp].
	URL Expression

	// Comment is the optional [Comment] node attached to a request.
	Comment Comment

	// Method is the [Method] node.
	Method Method

	// Sep is the [token.Separator] immediately before the request.
	Sep token.Token

	// Type is [KindRequest].
	Type Kind
}

// Start returns the first token associated with the [Request],
// which is the [token.Separator] immediately before it.
func (r Request) Start() token.Token {
	return r.Sep
}

// End returns the last token associated with the [Request].
func (r Request) End() token.Token {
	// TODO(@FollowTheProcess): There's more that can come after this
	if r.URL != nil {
		return r.URL.End()
	}

	return r.Method.End()
}

// Kind returns [KindRequest].
func (r Request) Kind() Kind {
	return r.Type
}

// statementNode marks a [Request] as an [ast.Statement].
func (r Request) statementNode() {}
