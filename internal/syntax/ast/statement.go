package ast

import "go.followtheprocess.codes/zap/internal/syntax/token"

// Statement is a statement node.
type Statement interface {
	Node
	statementNode() // Prevents accidental misuse as another node type
}

// A VarStatement is a single variable declaration.
type VarStatement struct {
	Value Expression  `yaml:"value"` // Value is the expression that [Ident] is being assigned to.
	Ident Ident       `yaml:"ident"` // Ident is the [Ident] node representing the assignee.
	At    token.Token `yaml:"at"`    // At is the '@' token declaring the variable.
	Type  Kind        `yaml:"type"`  // Type is the kind of node [KindVarStatement].
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
	Ident       Ident       `yaml:"ident"`       // Ident is the [Ident] node representing the assignee.
	Description TextLiteral `yaml:"description"` // Description is the [Text] node containing the prompt description.
	At          token.Token `yaml:"at"`          // At is the '@' token declaring the prompt.
	Type        Kind        `yaml:"type"`        // Type is the kind of the node, in this case [KindPromptStatement].
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
	Text  string      `yaml:"text"`  // Text is the test contained in the comment.
	Token token.Token `yaml:"token"` // Token is the [token.Comment] beginning the line comment.
	Type  Kind        `yaml:"type"`  // Type is [KindComment].
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
	Token token.Token `yaml:"token"` // Token is the method token e.g. [token.MethodGet].
	Type  Kind        `yaml:"type"`  // Type is [KindMethod].
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

	// ResponseRedirect is the optional response redirect statement.
	ResponseRedirect *ResponseRedirect `yaml:"responseRedirect"`

	// ResponseReference is the optional response reference statement.
	ResponseReference *ResponseReference `yaml:"responseReference"`

	// HTTPVersion is the optional HTTPVersion statement.
	HTTPVersion *HTTPVersion `yaml:"httpVersion"`

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
	Value Expression  `yaml:"value"` // Value is the value expression of the header.
	Key   string      `yaml:"key"`   // Key is the string containing the header key.
	Token token.Token `yaml:"token"` // Token is the [token.Header] representing the header key.
	Type  Kind        `yaml:"type"`  // Type is [KindHeader].
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
	File  Expression  `yaml:"file"`  // File is the filepath to save the response body.
	Token token.Token `yaml:"token"` // Token is the [token.RightAngle] beginning the redirect statement.
	Type  Kind        `yaml:"type"`  // Type is [KindResponseRedirect].
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
	File  Expression  `yaml:"file"`  // File is the filepath to compare the response body to.
	Token token.Token `yaml:"token"` // Token is the [token.ResponseRef] beginning the reference statement.
	Type  Kind        `yaml:"type"`  // Type is [KindResponseReference].
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

// HTTPVersion is a HTTP version statement.
type HTTPVersion struct {
	// Version is the version number, i.e. in the statement
	// 'HTTP/1.2', Version is '1.2'.
	Version string

	// Token is the [token.HTTPVersion] token.
	Token token.Token

	// Type is [KindHTTPVersion].
	Type Kind
}

// Start returns the [token.HTTPVersion].
func (h HTTPVersion) Start() token.Token {
	return h.Token
}

// End also returns the [token.HTTPVersion].
func (h HTTPVersion) End() token.Token {
	return h.Token
}

// Kind returns [KindHTTPVersion].
func (h HTTPVersion) Kind() Kind {
	return h.Type
}

// statementNode marks a [HTTPVersion] as a [Statement].
func (h HTTPVersion) statementNode() {}
