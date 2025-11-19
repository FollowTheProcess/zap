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
		return p.parseVarStatement()
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

	if err := p.expect(token.Ident); err != nil {
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

// parseExpression parses an expression.
func (p *Parser) parseExpression() (ast.Expression, error) {
	switch p.current.Kind {
	case token.Text:
		return p.parseTextLiteral()
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
