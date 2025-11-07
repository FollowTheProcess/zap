// Package parser implements the .http file parser.
package parser

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.followtheprocess.codes/zap/internal/syntax"
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
	// .http files are small, it's okay to read the whole thing
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

// Parse parses the file to completion returning a [syntax.File] and any parsing errors.
//
// The returned error will simply signify whether or not there were parse errors,
// the installed error handler passed to [New] will have the full detail and should
// be preferred.
func (p *Parser) Parse() (syntax.File, error) {
	file := syntax.File{
		Name: p.name,
	}

	// Parse any global at the top of the file
	file = p.parseGlobals(file)

	for !p.current.Is(token.EOF) {
		if p.current.Is(token.Error) {
			// An error from the scanner
			return syntax.File{}, ErrParse
		}

		request, err := p.parseRequest(file)
		if err != nil {
			// If we couldn't parse that request for whatever reason, let's try and
			// recover the parser state by skipping through tokens until we see the next
			// request and try again. This means the parser is somewhat resilient to localised
			// syntax errors
			p.synchronise()

			// Next up should be '###' or EOF, either way we're back in sync
			continue
		}

		// If it's name is missing, name it after it's position in the file (1 indexed)
		if request.Name == "" {
			request.Name = fmt.Sprintf("#%d", 1+len(file.Requests))
		}

		file.Requests = append(file.Requests, request)

		p.advance()
	}

	if p.hadErrors {
		return syntax.File{}, ErrParse
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
func (p *Parser) expect(kinds ...token.Kind) {
	if p.next.Is(token.Error) {
		// Nobody expects an error!
		// But seriously, this means the scanner has emitted an error and has already
		// passed it to the error handler
		return
	}

	switch len(kinds) {
	case 0:
		return
	case 1:
		if !p.next.Is(kinds[0]) {
			p.errorf("expected %s, got %s", kinds[0], p.next.Kind)
			return
		}
	default:
		if !p.next.Is(kinds...) {
			p.errorf("expected one of %v, got %s", kinds, p.next.Kind)
			return
		}
	}

	p.advance()
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

// parseGlobals parses a run of variable declarations at the top of the file, returning
// the modified [syntax.File].
//
// If p.current is anything other than '@', the input file is returned as is.
func (p *Parser) parseGlobals(file syntax.File) syntax.File {
	if !p.current.Is(token.At) {
		return file
	}

	for p.current.Is(token.At) {
		switch p.next.Kind {
		case token.Timeout:
			file.Timeout = p.parseDuration()
		case token.ConnectionTimeout:
			file.ConnectionTimeout = p.parseDuration()
		case token.NoRedirect:
			p.advance() // Advance because there's no value, @no-redirect is enough

			file.NoRedirect = true
		case token.Name:
			file.Name = p.parseName()
		case token.Prompt:
			prompt := p.parsePrompt()

			if file.Prompts == nil {
				file.Prompts = make(map[string]syntax.Prompt)
			}

			file.Prompts[prompt.Name] = prompt
		case token.Ident:
			key, value := p.parseVar(file, syntax.Request{})

			if file.Vars == nil {
				file.Vars = make(map[string]string)
			}

			file.Vars[key] = value
		default:
			p.expect(
				token.Timeout,
				token.ConnectionTimeout,
				token.NoRedirect,
				token.Name,
				token.Prompt,
				token.Ident,
			)
		}

		p.advance()
	}

	return file
}

// parseRequest parses a single request in a http file.
func (p *Parser) parseRequest(file syntax.File) (syntax.Request, error) {
	if !p.current.Is(token.Separator) {
		p.errorf("expected %s, got %s", token.Separator, p.current.Kind)
		return syntax.Request{}, ErrParse
	}

	request := syntax.Request{
		// Vars and Prompts are lazily initialised as it's not obvious that all requests will have those.
		//
		// Headers on the other hand every request will have so we initialise the map here always.
		Headers: make(http.Header),
	}

	// Does it have a comment as in "### [comment]"
	if p.next.Is(token.Comment) {
		p.advance()
		request.Comment = p.text()
	}

	p.advance()
	request = p.parseRequestVars(file, request)

	if !token.IsMethod(p.current.Kind) {
		p.errorf(
			"request separators must be followed by either a comment or a HTTP method, got %s: %q",
			p.current.Kind,
			p.text(),
		)

		return syntax.Request{}, ErrParse
	}

	request.Method = p.text()

	request = p.parseRequestURL(file, request)

	if p.next.Is(token.HTTPVersion) {
		p.advance()
		request.HTTPVersion = p.text()
	}

	request = p.parseRequestHeaders(file, request)

	request = p.parseRequestBody(file, request)

	// Might be a '< ./body.json'
	if p.next.Is(token.LeftAngle) {
		p.advance()
		p.expect(token.Text)
		request.BodyFile = p.text()
	}

	// We could now also have a response redirect
	// e.g '> ./response.json'
	if p.next.Is(token.RightAngle) {
		p.advance()
		p.expect(token.Text)
		request.ResponseFile = p.text()
	}

	// Or finally, a response ref '<> response.json'
	if p.next.Is(token.ResponseRef) {
		p.advance()
		p.expect(token.Text)
		request.ResponseRef = p.text()
	}

	return request, nil
}

// parseDuration parses a duration declaration e.g. in a global or request variable.
func (p *Parser) parseDuration() time.Duration {
	p.advance()
	// Can either be @timeout = 20s or @timeout 20s
	if p.next.Is(token.Eq) {
		p.advance()
	}

	p.expect(token.Text)

	duration, err := time.ParseDuration(p.text())
	if err != nil {
		p.errorf("bad timeout value: %v", err)
	}

	return duration
}

// parseName parses a name declaration e.g. in a global or request variable.
func (p *Parser) parseName() string {
	p.advance()
	// Can either be @name = MyName or @name MyName
	if p.next.Is(token.Eq) {
		p.advance()
	}

	p.expect(token.Text)

	return p.text()
}

// parsePrompt parses a prompt declaration e.g. in a global or request variable.
func (p *Parser) parsePrompt() syntax.Prompt {
	p.advance()

	p.expect(token.Ident)
	name := p.text()

	p.expect(token.Text)
	description := p.text()

	prompt := syntax.Prompt{
		Name:        name,
		Description: description,
	}

	return prompt
}

// parseVar parses a generic '@ident = <value>' in either global or request scope.
func (p *Parser) parseVar(file syntax.File, request syntax.Request) (key, value string) {
	p.advance()
	key = p.text()
	// Can either be @ident = value or @ident value
	if p.next.Is(token.Eq) {
		p.advance()
	}

	// Can be one of:
	// 1) Text/URL and have no interpolation inside it - easy
	// 2) Start as Text/URL but have one or more interpolation blocks with or without additional Text/URL afterwards
	// 3) Start as OpenInterp but have one or more instances of Text/URL afterwards, or maybe even more interpolations
	//
	// So we actually need to loop continuously until we see a non Text/URL/Interp appending to a string
	// as we go
	builder := &strings.Builder{}

	var isURL bool

	for p.next.Is(token.Text, token.URL, token.OpenInterp) {
		switch kind := p.next.Kind; kind {
		case token.Text:
			p.advance()
			builder.WriteString(p.text())
		case token.URL:
			isURL = true

			p.advance()
			builder.WriteString(p.text())
		case token.OpenInterp:
			p.parseInterp(builder, file, request)
		default:
			continue
		}
	}

	result := builder.String()

	// If it's a URL, let's make a best effort at validating it
	if isURL {
		_, err := url.ParseRequestURI(result)
		if err != nil {
			p.errorf("invalid URL: %v", err)
		}
	}

	return key, result
}

// parseRequestVars parses a run of variable declarations in a request. Returning
// the modified [syntax.Request].
//
// If p.current is anything other than '@', the request is returned as is.
func (p *Parser) parseRequestVars(file syntax.File, request syntax.Request) syntax.Request {
	if !p.current.Is(token.At) {
		return request
	}

	for p.current.Is(token.At) {
		switch p.next.Kind {
		case token.Timeout:
			request.Timeout = p.parseDuration()
		case token.ConnectionTimeout:
			request.ConnectionTimeout = p.parseDuration()
		case token.NoRedirect:
			p.advance()

			request.NoRedirect = true
		case token.Name:
			request.Name = p.parseName()
		case token.Prompt:
			prompt := p.parsePrompt()

			if request.Prompts == nil {
				request.Prompts = make(map[string]syntax.Prompt)
			}

			request.Prompts[prompt.Name] = prompt
		case token.Ident:
			key, value := p.parseVar(file, request)

			if request.Vars == nil {
				request.Vars = make(map[string]string)
			}

			request.Vars[key] = value
		default:
			// Always make progress
			p.advance()
		}

		p.advance()
	}

	return request
}

// parseRequestURL parses a URL following a HTTP method in a single request, returning
// the modified [syntax.Request].
//
// Interpolation is evaluated and replaced on the fly.
func (p *Parser) parseRequestURL(file syntax.File, request syntax.Request) syntax.Request {
	// Can be one of:
	// 1) Text/URL and have no interpolation inside it - easy
	// 2) Start as Text/URL but have one or more interpolation blocks with or without additional Text/URL afterwards
	// 3) Start as OpenInterp but have one or more instances of Text/URL afterwards, or maybe even more interpolations
	//
	// So we actually need to loop continuously until we see a non Text/URL/Interp appending to a string
	// as we go
	builder := &strings.Builder{}

	for p.next.Is(token.URL, token.Text, token.OpenInterp) {
		switch kind := p.next.Kind; kind {
		case token.URL, token.Text:
			p.advance()
			builder.WriteString(p.text())
		case token.OpenInterp:
			p.parseInterp(builder, file, request)
		default:
			// Always make progress
			p.advance()
		}
	}

	result := builder.String()

	_, err := url.ParseRequestURI(result)
	if err != nil {
		p.errorf("invalid URL: %v", err)
		return syntax.Request{}
	}

	request.URL = result

	return request
}

// parseInterp parses and evaluates a templated interpolation.
//
// Variables that exist in either global (file) or local (request) scope are substituted
// at parse time. Prompts are replaced with a placeholder of `zap::prompt::<ident>` as we cannot know their value
// until the file or request is run and the user provides an input.
//
// It takes a *strings.Builder as an input and write its evaluated results to that builder. The caller
// is responsible for creating the builder and using its results.
func (p *Parser) parseInterp(builder *strings.Builder, file syntax.File, request syntax.Request) {
	p.advance()
	// TODO(@FollowTheProcess): Same comment about expecting more than Ident
	p.expect(token.Ident)
	ident := p.text()
	p.expect(token.CloseInterp)

	localVal, isLocal := request.Vars[ident]
	globalVal, isGlobal := file.Vars[ident]

	allPrompts := make(map[string]syntax.Prompt, len(request.Prompts)+len(file.Prompts))
	maps.Copy(allPrompts, request.Prompts)
	maps.Copy(allPrompts, file.Prompts)

	_, isPrompt := allPrompts[ident]

	switch {
	case isLocal:
		builder.WriteString(localVal)
	case isGlobal:
		builder.WriteString(globalVal)
	case isPrompt:
		// We resolve these later when we ask the user, they cannot be
		// resolved at parse time
		builder.WriteString("zap::prompt::" + ident)
	default:
		p.errorf("use of undefined variable %q", ident)
	}
}

// parseRequestHeaders parses a run of request headers, returning the modified
// request.
//
// Interpolation is evaluated and replaced on the fly.
func (p *Parser) parseRequestHeaders(file syntax.File, request syntax.Request) syntax.Request {
	for p.next.Is(token.Header) {
		p.advance()
		key := p.text()
		p.expect(token.Colon)

		builder := &strings.Builder{}

		for p.next.Is(token.Text, token.OpenInterp) {
			switch kind := p.next.Kind; kind {
			case token.Text:
				p.advance()
				builder.WriteString(p.text())
			case token.OpenInterp:
				p.parseInterp(builder, file, request)
			default:
				// Always make progress
				p.advance()
			}
		}

		request.Headers.Add(key, builder.String())
		builder.Reset() // Reset for the next (outer) loop
	}

	return request
}

// parseRequestBody parses a request body, returning the modified request.
//
// Interpolation is evaluated and replaced o the fly.
func (p *Parser) parseRequestBody(file syntax.File, request syntax.Request) syntax.Request {
	builder := &strings.Builder{}

	for p.next.Is(token.Body, token.OpenInterp) {
		switch kind := p.next.Kind; kind {
		case token.Body:
			p.advance()
			builder.Write(p.bytes())
		case token.OpenInterp:
			p.advance()
			p.expect(token.Ident)
			ident := p.text()
			p.expect(token.CloseInterp)

			localVal, isLocal := request.Vars[ident]
			globalVal, isGlobal := file.Vars[ident]

			allPrompts := make(map[string]syntax.Prompt, len(request.Prompts)+len(file.Prompts))
			maps.Copy(allPrompts, request.Prompts)
			maps.Copy(allPrompts, file.Prompts)

			_, isPrompt := allPrompts[ident]

			switch {
			case isLocal:
				builder.WriteString(localVal)
			case isGlobal:
				builder.WriteString(globalVal)
			case isPrompt:
				// We resolve these later when we ask the user, they cannot be
				// resolved at parse time
				builder.WriteString("zap::prompt::" + ident)
			default:
				p.errorf("use of undefined variable %q", ident)
			}

		default:
			// Always make progress
			p.advance()
		}
	}

	request.Body = []byte(builder.String())

	return request
}
