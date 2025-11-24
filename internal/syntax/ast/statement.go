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
	Value Expression `yaml:"value"`

	// Ident is the [Ident] node representing the identifier
	// the expression Value is being assigned to.
	Ident Ident `yaml:"ident"`

	// At is the '@' token declaring the variable.
	At token.Token `yaml:"at"`

	// Type is the kind of node [KindVarStatement].
	Type Kind `yaml:"type"`
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

// statementNode marks a [VarStatement] as an [Statement].
func (v VarStatement) statementNode() {}

// A PromptStatement is a single prompt declaration.
type PromptStatement struct {
	// Ident is the [Ident] node representing the identifier
	// the prompt Value is being assigned to.
	Ident Ident `yaml:"ident"`

	// Description is the [Text] node containing the description
	// of the prompt.
	Description TextLiteral `yaml:"description"`

	// At is the '@' token declaring the prompt.
	At token.Token `yaml:"at"`

	// Type is the kind of the node, in this case
	// [KindPromptStatement].
	Type Kind `yaml:"type"`
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

// statementNode marks a [PromptStatement] as an [Statement].
func (p PromptStatement) statementNode() {}

// Comment represents a single line comment.
type Comment struct {
	// Text is the test contained in the comment.
	Text string `yaml:"text"`

	// Token is the [token.Comment] beginning the line comment.
	Token token.Token `yaml:"token"`

	// Type is [KindComment].
	Type Kind `yaml:"type"`
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

// statementNode marks a [Comment] as an [Statement].
func (c Comment) statementNode() {}

// Method represents a HTTP method.
type Method struct {
	// Token is the method token e.g. [token.MethodGet].
	Token token.Token `yaml:"token"`

	// Type is [KindMethod].
	Type Kind `yaml:"type"`
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

// statementNode marks a [Method] as an [Statement].
func (m Method) statementNode() {}

// Request is a single HTTP request.
type Request struct {
	// URL is the expression that when evaluated, returns the URL
	// for the request. May be a [TextLiteral] or an [Interp].
	URL Expression `yaml:"url"`

	// Body is the body expression.
	Body Expression `yaml:"body"`

	// ResponseRedirect is the optional response redirect statement
	// provided for the request.
	ResponseRedirect *ResponseRedirect `yaml:"responseRedirect"`

	// ResponseReference is the optional response reference statement
	// provided for the request.
	ResponseReference *ResponseReference `yaml:"responseReference"`

	// Comment is the optional [Comment] node attached to a request.
	Comment *Comment `yaml:"comment"`

	// Vars are any [VarStatement] nodes attached to the request defining
	// local variables.
	Vars []VarStatement `yaml:"vars"`

	// Prompts are any [PromptStatement] nodes attached to the request
	// defining local prompted variables.
	Prompts []PromptStatement `yaml:"prompts"`

	// Headers are the [HeaderStatement] nodes attached to the request
	Headers []Header `yaml:"headers"`

	// Method is the [Method] node.
	Method Method `yaml:"method"`

	// Sep is the [token.Separator] immediately before the request.
	Sep token.Token `yaml:"sep"`

	// Type is [KindRequest].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the [Request],
// which is the [token.Separator] immediately before it.
func (r Request) Start() token.Token {
	return r.Sep
}

// End returns the last token associated with the [Request].
func (r Request) End() token.Token {
	if r.ResponseRedirect != nil {
		return r.ResponseRedirect.End()
	}

	if r.ResponseReference != nil {
		return r.ResponseReference.End()
	}

	if r.Body != nil {
		return r.Body.End()
	}

	if r.URL != nil {
		return r.URL.End()
	}

	return r.Method.End()
}

// Kind returns [KindRequest].
func (r Request) Kind() Kind {
	return r.Type
}

// statementNode marks a [Request] as an [Statement].
func (r Request) statementNode() {}

// Header is a HTTP header node.
type Header struct {
	// Value is the value expression of the header.
	Value Expression `yaml:"value"`

	// Key is the string containing the header key.
	Key string `yaml:"key"`

	// Token is the [token.Header] representing the header key.
	Token token.Token `yaml:"token"`

	// Type is [KindHeader].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the header, in this
// case the [token.Header] token.
func (h Header) Start() token.Token {
	return h.Token
}

// End returns the last token associated with the header, which
// is the final token in the Value expression.
func (h Header) End() token.Token {
	return h.Value.End()
}

// Kind returns [KindHeader].
func (h Header) Kind() Kind {
	return h.Type
}

// statementNode marks a [Header] as an [Statement].
func (h Header) statementNode() {}

// ResponseRedirect is a response redirection statement.
type ResponseRedirect struct {
	// File is the filepath to save the response body.
	File Expression `yaml:"file"`

	// Token is the [token.RightAngle] beginning the redirect statement.
	Token token.Token `yaml:"token"`

	// Type is [KindResponseRedirect].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the redirect, which
// is the opening [token.RightAngle].
func (r ResponseRedirect) Start() token.Token {
	return r.Token
}

// End returns the last token associated with the redirect, which
// is the final token in the File expression.
func (r ResponseRedirect) End() token.Token {
	if r.File != nil {
		return r.File.End()
	}

	return r.Token
}

// Kind returns [KindResponseRedirect].
func (r ResponseRedirect) Kind() Kind {
	return r.Type
}

// statementNode marks a [ResponseRedirect] as a [Statement].
func (r ResponseRedirect) statementNode() {}

// ResponseReference is a response reference statement.
type ResponseReference struct {
	// File is the filepath to compare the response body to.
	File Expression `yaml:"file"`

	// Token is the [token.ResponseRef] beginning the reference statement.
	Token token.Token `yaml:"token"`

	// Type is [KindResponseReference].
	Type Kind `yaml:"type"`
}

// Start returns the first token associated with the reference, which
// is the opening [token.ResponseRef].
func (r ResponseReference) Start() token.Token {
	return r.Token
}

// End returns the last token associated with the reference, which
// is the final token in the File expression.
func (r ResponseReference) End() token.Token {
	if r.File != nil {
		return r.File.End()
	}

	return r.Token
}

// Kind returns [KindResponseReference].
func (r ResponseReference) Kind() Kind {
	return r.Type
}

// statementNode marks a [ResponseReference] as a [Statement].
func (r ResponseReference) statementNode() {}
