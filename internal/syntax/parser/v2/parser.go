// Package parser implements the new .http file parser.
//
// The original parser was "dumb" in the sense that it could handle http syntax but as
// I went to implement more complex features like more interpolation expressions, loading
// env vars using {{ env.VAR }}, and referring to previous requests responses via their
// name e.g. {{ requests.<name>.response.body }} I realised I needed to re-think the implementation.
//
// So this parser (which will eventually replace the existing one) parses a http file into
// a proper abstract syntax tree, which will then allow things like expression precedence
// and proper scoped variable resolution.
package parser

import (
	"errors"
	"fmt"
	"io"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
	"go.followtheprocess.codes/zap/internal/syntax/scanner"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

// ErrParse is a generic parsing error, details on the error are passed
// to the parser's [syntax.ErrorHandler] at the moment it occurs.
var ErrParse = errors.New("parse error")

// Parser is the http file parser.
type Parser struct {
	handler   syntax.ErrorHandler // The installed error handler, to be called in response to parse errors
	scanner   *scanner.Scanner    // Scanner to produce tokens
	name      string              // Name of the file being parsed
	src       []byte              // Raw source text
	current   token.Token         // Current token under inspection
	next      token.Token         // Next token in the stream
	hadErrors bool                // Whether we encountered parse errors
}

// New initialises and returns a new [Parser] that reads from r.
func New(name string, r io.Reader, handler syntax.ErrorHandler) (*Parser, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from input: %w", err)
	}

	p := &Parser{
		handler: handler,
		scanner: scanner.New(name, src, handler),
		name:    name,
		src:     src,
	}

	// Read 2 tokens so current and next are set
	p.advance()
	p.advance()

	return p, nil
}

// Parse parses the file to completion returning an [ast.File] and any parsing errors.
//
// The returned error will simply signify whether or not there were parse errors,
// the installed error handler passed to [New] will have the full detail and should
// be preferred.
func (p *Parser) Parse() (ast.File, error) {
	file := ast.File{
		Name:       p.name,
		Statements: make([]ast.Statement, 0),
		Type:       ast.KindFile,
	}

	for !p.current.Is(token.EOF) {
		if p.current.Is(token.Error) {
			// An error from the scanner
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
		// But seriously, this means the scanner has emitted an error and has already
		// passed it to the error handler
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

// error calculates the current position and calls the installed error handler
// with the correct information.
func (p *Parser) error(msg string) {
	p.hadErrors = true

	if p.handler == nil {
		// I guess ignore?
		return
	}

	p.handler(p.position(), msg)
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
	result := ast.VarStatement{
		At:   p.current,
		Type: ast.KindVarStatement,
	}

	// All keywords like @timeout, @no-redirect etc. get parsed in here as they
	// are structurally identical, they are all effectively a variable declaration, just their
	// variables are "special". During resolution they get mapped into dedicated fields in the
	// resulting spec.File.
	if err := p.expect(token.Name, token.Timeout, token.ConnectionTimeout, token.NoRedirect, token.Ident); err != nil {
		return result, err
	}

	result.Ident = ast.Ident{
		Name:  p.text(),
		Token: p.current,
		Type:  ast.KindIdent,
	}

	// Optional '='
	if p.next.Is(token.Eq) {
		p.advance()
	}

	p.advance()

	value, err := p.parseExpression()
	if err != nil {
		return result, err
	}

	result.Value = value

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

	ident, err := p.parseIdent()
	if err != nil {
		return result, err
	}

	result.Ident = ident

	if p.next.Is(token.Text) {
		p.advance()

		text, err := p.parseTextLiteral()
		if err != nil {
			return result, err
		}

		result.Description = text
	}

	return result, nil
}

// parseComment parses a line comment.
//
// Comments are parsed into ast nodes so that comments above requests may
// be used as their "docstring". Similar to how doc comments are attached
// to ast nodes in Go.
func (p *Parser) parseComment() (ast.Comment, error) {
	result := ast.Comment{
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
		case token.Name, token.Timeout, token.ConnectionTimeout, token.NoRedirect, token.Ident:
			varStatement, err := p.parseVarStatement()
			if err != nil {
				return result, err
			}

			result.Vars = append(result.Vars, varStatement)
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

	if err = p.expect(token.URL, token.OpenInterp); err != nil {
		return result, err
	}

	url, err := p.parseExpression()
	if err != nil {
		return result, err
	}

	result.URL = url

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

// parseExpression parses an expression.
func (p *Parser) parseExpression() (ast.Expression, error) {
	// TODO(@FollowTheProcess): We need some precedence in here so that interps get evaluated first
	switch p.current.Kind {
	case token.Text:
		return p.parseTextLiteral()
	case token.OpenInterp:
		return p.parseInterp()
	case token.Ident:
		return p.parseIdent()
	case token.URL:
		return p.parseURL()
	default:
		p.errorf("parseExpression: unexpected token %s", p.current.Kind)
		return nil, ErrParse
	}
}

// parseTextLiteral parses a TextLiteral.
func (p *Parser) parseTextLiteral() (ast.TextLiteral, error) {
	text := ast.TextLiteral{
		Value: p.text(),
		Token: p.current,
		Type:  ast.KindTextLiteral,
	}

	return text, nil
}

// parseURL parses a URL literal.
func (p *Parser) parseURL() (ast.URL, error) {
	result := ast.URL{
		Value: p.text(),
		Token: p.current,
		Type:  ast.KindURL,
	}

	return result, nil
}

// parseIdent parses an Ident.
func (p *Parser) parseIdent() (ast.Ident, error) {
	ident := ast.Ident{
		Name:  p.text(),
		Token: p.current,
		Type:  ast.KindIdent,
	}

	return ident, nil
}

// parseInterp parses an interpolation expression, i.e.
// '{{' <expr> '}}'.
func (p *Parser) parseInterp() (ast.Interp, error) {
	result := ast.Interp{
		Open: p.current,
		Type: ast.KindInterp,
	}

	// TODO(@FollowTheProcess): Just like the other parser, for now we'll assume only idents are allowed here
	if err := p.expect(token.Ident); err != nil {
		return result, err
	}

	expr, err := p.parseExpression()
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
