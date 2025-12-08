// Package parser implements the a .http file parser.
//
// The parser parses a stream of tokens from the scanner into ast nodes, if
// a parse error occurs, partial nodes may be returned rather than the idiomatic
// Go norm of <zero value>, error. This is intentional both to aid error reporting and
// to increase the fault tolerance of the parser for use in e.g. language servers that
// commonly parse incomplete or incorrect code and require a best effort partial AST
// to function in these scenarios.
//
// Once parsed, the abstract syntax tree is resolved which is where variable interpolation,
// and more thorough validation happen.
package parser

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
	"go.followtheprocess.codes/zap/internal/syntax/scanner/v2"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

// ErrParse is a generic parsing error, details on the error are passed
// to the parser's [syntax.ErrorHandler] at the moment it occurs.
var ErrParse = errors.New("parse error")

// Parser is the http file parser.
type Parser struct {
	diagnostics []syntax.Diagnostic // Diagnostics gathered during parsing
	scanner     *scanner.Scanner    // Scanner to produce tokens
	name        string              // Name of the file being parsed
	src         []byte              // Raw source text
	current     token.Token         // Current token under inspection
	next        token.Token         // Next token in the stream
	hadErrors   bool                // Whether we encountered parse errors
}

// New initialises and returns a new [Parser] that parses src.
func New(name string, src []byte) *Parser {
	p := &Parser{
		scanner: scanner.New(name, src),
		name:    name,
		src:     src,
	}

	// Read 2 tokens so current and next are set
	p.advance()
	p.advance()

	return p
}

// Parse parses the file to completion returning an [ast.File] and any parsing errors.
//
// The returned error will simply signify whether or not there were parse errors,
// the installed error handler passed to [New] will have the full detail and should
// be preferred.
func (p *Parser) Parse() (ast.File, error) {
	if p == nil {
		return ast.File{}, errors.New("Parse called on nil parser")
	}

	file := ast.File{
		Name:       p.name,
		Statements: make([]ast.Statement, 0),
		Type:       ast.KindFile,
	}

	for !p.current.Is(token.EOF) {
		if p.current.Is(token.Error) {
			p.error("Error token from scanner")
			p.synchronise()

			continue
		}

		statement, err := p.parseStatement()
		if err != nil {
			p.synchronise()
			continue
		}

		if statement != nil {
			file.Statements = append(file.Statements, statement)
		}

		p.advance()
	}

	if p.hadErrors {
		return file, ErrParse
	}

	return file, nil
}

// Diagnostics returns any [syntax.Diagnostic] gathered during parsing.
func (p *Parser) Diagnostics() []syntax.Diagnostic {
	combined := slices.Concat(p.scanner.Diagnostics(), p.diagnostics)

	// Sort by file and line number
	slices.SortFunc(combined, func(a, b syntax.Diagnostic) int {
		return syntax.ComparePosition(a.Position, b.Position)
	})

	return combined
}

// advance advances the parser by a single token.
func (p *Parser) advance() {
	p.current = p.next
	p.next = p.scanner.Scan()
}

// expect asserts that the next token is one of the given kinds, emitting a syntax error if not.
//
// The parser is advanced only if the next token is of one of these kinds such that after returning
// p.current will be one of the kinds.
//
// It returns an [ErrParse] is the expectation is violated, nil otherwise.
func (p *Parser) expect(kinds ...token.Kind) error {
	if p.next.Is(token.Error) {
		// Nobody expects an error!
		p.error("Error token from scanner")
		return ErrParse
	}

	switch len(kinds) {
	case 0:
		return nil
	case 1:
		if !p.next.Is(kinds[0]) {
			p.errorf("expected %s, got %s", kinds[0], p.next.Kind)
			return ErrParse
		}
	default:
		if !p.next.Is(kinds...) {
			p.errorf("expected one of %v, got %s", kinds, p.next.Kind)
			return ErrParse
		}
	}

	p.advance()

	return nil
}

// position returns the parser's current position in the input as a [syntax.Position].
//
// The position is calculated based on the start offset of the current token.
func (p *Parser) position() syntax.Position {
	line := 1              // Line counter
	lastNewLineOffset := 0 // The byte offset of the (end of the) last newline seen

	for index, byt := range p.src {
		if index >= p.current.Start {
			break
		}

		if byt == '\n' {
			lastNewLineOffset = index + 1 // +1 to account for len("\n")
			line++
		}
	}

	// If the next token is EOF, we use the end of the current token as the syntax
	// error is likely to be unexpected EOF so we want to point to the end of the
	// current token as in "something should have gone here"
	start := p.current.Start
	if p.next.Is(token.EOF) {
		start = p.current.End
	}

	end := p.current.End

	// The column is therefore the number of bytes between the end of the last newline
	// and the current position, +1 because editors columns start at 1. Applying this
	// correction here means you can click a syntax error in the terminal and be
	// taken to a precise location in an editor which is probably what we want to happen
	startCol := 1 + start - lastNewLineOffset
	endCol := 1 + end - lastNewLineOffset

	return syntax.Position{
		Name:     p.name,
		Offset:   p.current.Start,
		Line:     line,
		StartCol: startCol,
		EndCol:   endCol,
	}
}

// error calculates the current position and appends a syntax diagnostic to
// the parser.
func (p *Parser) error(msg string) {
	p.hadErrors = true

	diag := syntax.Diagnostic{
		Msg:      msg,
		Position: p.position(),
	}

	p.diagnostics = append(p.diagnostics, diag)
}

// errorf calls error with a formatted message.
func (p *Parser) errorf(format string, a ...any) {
	p.error(fmt.Sprintf(format, a...))
}

// text returns the chunk of source text described by the p.current token.
func (p *Parser) text() string {
	return string(p.bytes())
}

// bytes returns the chunk of source text described by the p.current token
// as a byte slice.
func (p *Parser) bytes() []byte {
	return p.src[p.current.Start:p.current.End]
}

// synchronise is called during error recovery, after a parse error we are unsure of
// the local state as the syntax is invalid.
//
// synchronise discards tokens until it sees the next Separator, EOF after which
// point the parser should be back in sync and can continue normally.
func (p *Parser) synchronise() {
	for {
		p.advance()

		if p.current.Is(token.Separator, token.EOF) {
			break
		}
	}
}

// parseStatement parses a statement.
func (p *Parser) parseStatement() (ast.Statement, error) {
	switch p.current.Kind {
	case token.At:
		if p.next.Is(token.Prompt) {
			return p.parsePrompt()
		}

		return p.parseVarStatement()
	case token.Comment:
		return p.parseComment()
	case token.Separator:
		return p.parseRequest()
	default:
		p.errorf("parseStatement: unrecognised token: %s", p.current.Kind)
		return nil, ErrParse
	}
}

// parseVarStatement parses a variable declaration statement.
func (p *Parser) parseVarStatement() (ast.VarStatement, error) {
	if p.next.Is(token.NoRedirect) {
		return p.parseNoRedirect()
	}

	result := ast.VarStatement{
		At:   p.current,
		Type: ast.KindVarStatement,
	}

	// All keywords like @timeout, @name etc. get parsed in here as they
	// are structurally identical, they are all effectively a variable declaration, just their
	// variables are "special". During resolution they get mapped into dedicated fields in the
	// resulting spec.File.
	if err := p.expect(token.Name, token.Timeout, token.ConnectionTimeout, token.Ident); err != nil {
		return result, err
	}

	result.Ident = p.parseIdent()

	// Optional '='
	if p.next.Is(token.Eq) {
		p.advance()
	}

	p.advance()

	value, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return result, err
	}

	result.Value = value

	return result, nil
}

// parseNoRedirect parses a @no-redirect variable declaration, it is a special case
// of [parseVarStatement] that does not require a value expression.
func (p *Parser) parseNoRedirect() (ast.VarStatement, error) {
	result := ast.VarStatement{
		At:   p.current,
		Type: ast.KindVarStatement,
	}

	if err := p.expect(token.NoRedirect); err != nil {
		return result, err
	}

	result.Ident = p.parseIdent()

	return result, nil
}

// parsePrompt parses a prompt.
func (p *Parser) parsePrompt() (ast.PromptStatement, error) {
	result := ast.PromptStatement{
		At:   p.current,
		Type: ast.KindPrompt,
	}

	if err := p.expect(token.Prompt); err != nil {
		return result, err
	}

	if err := p.expect(token.Ident); err != nil {
		return result, err
	}

	result.Ident = p.parseIdent()

	if p.next.Is(token.Text) {
		p.advance()

		result.Description = p.parseTextLiteral()
	}

	return result, nil
}

// parseComment parses a line comment.
//
// Comments are parsed into ast nodes so that comments above requests may
// be used as their "docstring". Similar to how doc comments are attached
// to ast nodes in Go.
func (p *Parser) parseComment() (*ast.Comment, error) {
	result := &ast.Comment{
		Token: p.current,
		Type:  ast.KindComment,
		Text:  p.text(),
	}

	return result, nil
}

// parseRequest parses a single http request.
func (p *Parser) parseRequest() (ast.Request, error) {
	result := ast.Request{
		Sep:  p.current,
		Type: ast.KindRequest,
	}

	// Optional comment
	if p.next.Is(token.Comment) {
		p.advance()

		comment, err := p.parseComment()
		if err != nil {
			return result, err
		}

		result.Comment = comment
	}

	for p.next.Is(token.At) {
		p.advance()

		switch p.next.Kind {
		// All keywords like @timeout, @no-redirect etc. get parsed in here as they
		// are structurally identical, they are all effectively a variable declaration, just their
		// variables are "special". During resolution they get mapped into dedicated fields in the
		// resulting spec.File.
		case token.Name, token.Timeout, token.ConnectionTimeout, token.Ident:
			varStatement, err := p.parseVarStatement()
			if err != nil {
				return result, err
			}

			result.Vars = append(result.Vars, varStatement)
		case token.NoRedirect:
			// @no-redirect is a special case because it does not take a value, simply specifying
			// it is enough to disable redirects
			noRedirect, err := p.parseNoRedirect()
			if err != nil {
				return result, err
			}

			result.Vars = append(result.Vars, noRedirect)
		case token.Prompt:
			prompt, err := p.parsePrompt()
			if err != nil {
				return result, err
			}

			result.Prompts = append(result.Prompts, prompt)
		default:
			// Use expect for the free error message
			if err := p.expect(token.Name,
				token.Timeout,
				token.ConnectionTimeout,
				token.NoRedirect,
				token.Ident,
				token.Prompt,
			); err != nil {
				return result, err
			}
		}
	}

	// Now must be a HTTP method
	err := p.expect(
		token.MethodConnect,
		token.MethodDelete,
		token.MethodGet,
		token.MethodHead,
		token.MethodOptions,
		token.MethodPatch,
		token.MethodPost,
		token.MethodPut,
		token.MethodTrace,
	)
	if err != nil {
		return result, err
	}

	method, err := p.parseMethod()
	if err != nil {
		return result, err
	}

	result.Method = method

	if err = p.expect(token.Text, token.OpenInterp); err != nil {
		return result, err
	}

	url, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return result, err
	}

	result.URL = url

	if p.next.Is(token.HTTPVersion) {
		p.advance()

		httpVersion, err := p.parseHTTPVersion()
		if err != nil {
			return result, err
		}

		result.HTTPVersion = httpVersion
	}

	for p.next.Is(token.Header) {
		p.advance()

		header, err := p.parseHeader()
		if err != nil {
			return result, err
		}

		result.Headers = append(result.Headers, header)
	}

	// Body or < <body file>
	switch p.next.Kind {
	case token.Body:
		p.advance()

		body, err := p.parseExpression(token.LowestPrecedence)
		if err != nil {
			return result, err
		}

		result.Body = body
	case token.LeftAngle:
		p.advance()

		bodyFile, err := p.parseBodyFile()
		if err != nil {
			return result, err
		}

		result.Body = bodyFile
	default:
		// Nothing, not all requests have a body
	}

	if p.next.Is(token.RightAngle) {
		p.advance()

		redirect, err := p.parseResponseRedirect()
		if err != nil {
			return result, err
		}

		result.ResponseRedirect = redirect
	}

	if p.next.Is(token.ResponseRef) {
		p.advance()

		responseRef, err := p.parseResponseReference()
		if err != nil {
			return result, err
		}

		result.ResponseReference = responseRef
	}

	return result, nil
}

// parseMethod parses a http method.
func (p *Parser) parseMethod() (ast.Method, error) {
	result := ast.Method{
		Token: p.current,
		Type:  ast.KindMethod,
	}

	return result, nil
}

// parseExpression parses an expression given a precedence level.
func (p *Parser) parseExpression(precedence int) (ast.Expression, error) {
	// Prefix expressions, when the expression begins with one of these tokens, analogous
	// to prefix expressions like -5, !true or 'text literal'
	var (
		expr ast.Expression
		err  error
	)

	switch p.current.Kind {
	case token.Text:
		expr = p.parseTextLiteral()
	case token.OpenInterp:
		// If the expression begins with an open interp '{{' it's an interpolated
		// expression with no left hand side, hence nil
		expr, err = p.parseInterpolatedExpression(nil)
	case token.Ident:
		expr = p.parseIdent()
	case token.Body:
		expr, err = p.parseBody()
	default:
		p.errorf("parseExpression: unexpected token %s", p.current.Kind)
		return nil, ErrParse
	}

	if err != nil {
		return nil, err
	}

	// Now what happens if the tokens are found in the middle of the expression. Analogous
	// to infix expressions like 5 + 5.
	//
	// Very similar to how programming languages parse binary expressions using operator precedence
	// e.g. 'a + b / c' should be parsed as '(a + (b / c))' as '/' has a higher precedence or binding
	// power than '+'.
	//
	// The equivalent for us is that {{ <ident> }} should have a higher precedence such that:
	//
	// 'https://example.com/{{ version }}/items/1'
	//
	// should be parsed as:
	//
	// '('https://example.com/(<version>)/items/1')'
	//
	// Where the '{{ version }}' is more deeply nested in the ast.
	//
	// This is done here with the [ast.InterpolatedExpression] node which represents an [ast.Interp]
	// sandwiched between two arbitrary expressions. This is analogous to a binary expression node
	// in most programming languages where two expressions are sandwiched between a binary operator.
	//
	// In our case the Interp is the operator and carries the highest precedence.

	for p.next.Is(token.OpenInterp) && precedence < p.next.Precedence() {
		p.advance()

		switch p.current.Kind {
		case token.OpenInterp:
			// It's an interpolated expression where we already know the left hand side
			expr, err = p.parseInterpolatedExpression(expr)
		default:
			p.errorf("parseExpression: unexpected token: %s", p.current.Kind)
		}

		if err != nil {
			return nil, err
		}
	}

	return expr, nil
}

// parseInterpolatedExpression parses a composite interpolation expression.
func (p *Parser) parseInterpolatedExpression(left ast.Expression) (ast.InterpolatedExpression, error) {
	expr := ast.InterpolatedExpression{
		Left: left,
		Type: ast.KindInterpolatedExpression,
	}

	interp, err := p.parseInterp()
	if err != nil {
		return expr, err
	}

	expr.Interp = interp

	precedence := p.current.Precedence()

	if p.next.Is(token.Text, token.OpenInterp, token.Body) && p.shouldParseRHS(left) {
		p.advance()

		right, err := p.parseExpression(precedence)
		if err != nil {
			return expr, err
		}

		expr.Right = right
	}

	return expr, nil
}

// shouldParseRHS reports whether we should parse the right hand side of an expression,
// given the incoming token and the left hand side of that expression.
//
// Without this, the parser will eagerly consume e.g. a Body as the right hand side of
// an interpolated header value.
//
// For example:
//
//	Authorization: Bearer {{ token }}
//
//	{ "body": "here" }
//
// The header Authorization would have an InterpolatedExpression as it's value, the
// left hand side of which would be "Bearer ", the interp in the middle would of course
// be "{{ token }}", but the body would be consumed as the right hand side of this expression
// which is obviously incorrect.
//
// In general, we only parse the right hand side if it's the same type of expression as the left.
func (p *Parser) shouldParseRHS(left ast.Expression) bool {
	if left == nil {
		// No information to tell otherwise so go ahead and
		// parse the right hand side
		return true
	}

	switch left.Kind() {
	case ast.KindTextLiteral, ast.KindURL:
		return p.next.Is(token.Text)
	case ast.KindIdent:
		return p.next.Is(token.Ident)
	case ast.KindBody:
		return p.next.Is(token.Body)
	default:
		return false
	}
}

// parseTextLiteral parses a TextLiteral.
func (p *Parser) parseTextLiteral() ast.TextLiteral {
	text := ast.TextLiteral{
		Value: p.text(),
		Token: p.current,
		Type:  ast.KindTextLiteral,
	}

	return text
}

// parseHeader parses a Header statement.
func (p *Parser) parseHeader() (ast.Header, error) {
	result := ast.Header{
		Token: p.current,
		Type:  ast.KindHeader,
		Key:   p.text(),
	}

	if err := p.expect(token.Colon); err != nil {
		return result, err
	}

	if err := p.expect(token.Text, token.OpenInterp); err != nil {
		return result, err
	}

	value, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return result, err
	}

	result.Value = value

	return result, nil
}

// parseIdent parses an Ident.
func (p *Parser) parseIdent() ast.Ident {
	ident := ast.Ident{
		Name:  strings.TrimSpace(p.text()),
		Token: p.current,
		Type:  ast.KindIdent,
	}

	return ident
}

// parseInterp parses an interpolation expression, i.e.
// '{{' <expr> '}}'.
func (p *Parser) parseInterp() (ast.Interp, error) {
	result := ast.Interp{
		Open: p.current,
		Type: ast.KindInterp,
	}

	// TODO(@FollowTheProcess): For now we'll assume only idents are allowed here
	if err := p.expect(token.Ident); err != nil {
		return result, err
	}

	expr, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return result, err
	}

	result.Expr = expr

	if err := p.expect(token.CloseInterp); err != nil {
		return result, err
	}

	result.Close = p.current

	return result, nil
}

// parseBody parses a body expression.
func (p *Parser) parseBody() (ast.Body, error) {
	body := ast.Body{
		Token: p.current,
		Value: strings.TrimSpace(p.text()),
		Type:  ast.KindBody,
	}

	return body, nil
}

// parseBodyFile parses a body file expression.
func (p *Parser) parseBodyFile() (ast.BodyFile, error) {
	bodyFile := ast.BodyFile{
		Token: p.current,
		Type:  ast.KindBodyFile,
	}

	if err := p.expect(token.Text, token.OpenInterp); err != nil {
		return bodyFile, err
	}

	value, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return bodyFile, err
	}

	bodyFile.Value = value

	return bodyFile, nil
}

// parseResponseRedirect parses a response redirect statement.
func (p *Parser) parseResponseRedirect() (*ast.ResponseRedirect, error) {
	redirect := &ast.ResponseRedirect{
		Token: p.current,
		Type:  ast.KindResponseRedirect,
	}

	if err := p.expect(token.Text, token.OpenInterp); err != nil {
		return redirect, err
	}

	file, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return redirect, err
	}

	redirect.File = file

	return redirect, nil
}

// parseResponseReference parses a response reference statement.
func (p *Parser) parseResponseReference() (*ast.ResponseReference, error) {
	ref := &ast.ResponseReference{
		Token: p.current,
		Type:  ast.KindResponseReference,
	}

	if err := p.expect(token.Text, token.OpenInterp); err != nil {
		return ref, err
	}

	file, err := p.parseExpression(token.LowestPrecedence)
	if err != nil {
		return ref, err
	}

	ref.File = file

	return ref, nil
}

// parseHTTPVersion parses a http version statement.
func (p *Parser) parseHTTPVersion() (*ast.HTTPVersion, error) {
	version := &ast.HTTPVersion{
		Token: p.current,
		Type:  ast.KindHTTPVersion,
	}

	after, ok := strings.CutPrefix(p.text(), "HTTP/")
	if !ok {
		// Should basically never happen because the scanner would catch it
		// but let's be safe.
		p.errorf("bad HTTP version, missing 'HTTP/' prefix: %s", p.text())
		return version, ErrParse
	}

	version.Version = after

	return version, nil
}
