// Package scanner implements a lexical scanner for .http files, reading the raw
// source text and emitting a stream of tokens to be consumed by the parser.
//
// The scanner is a concurrent, state-function based scanner similar to that described by Rob Pike
// in his talk [Lexical Scanning in Go], based on the implementation of text/template in the Go
// standard library.
//
// The scanner proceeds one utf-8 rune at a time until a particular token is recognised,
// the token is then "emitted" over a channel where it may be consumed by a client e.g. the parser.
//
// The state of the scanner is maintained between token emits unlike a more conventional
// switch-based scanner that must determine it's current state from scratch in every loop. This makes
// it particularly useful for scanning .http files as it is not a context free grammar, a header
// looks the same as a HTTP method (it's just raw text), or even a header value to a naive scanner.
//
// A similar approach is used in [BurntSushi/toml].
//
// [Lexical Scanning in Go]: https://go.dev/talks/2011/lex.slide#1
// [BurntSushi/toml]: https://github.com/BurntSushi/toml/blob/master/lex.go
package scanner

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"unicode"
	"unicode/utf8"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

const (
	eof        = rune(-1) // eof signifies we have reached the end of the input.
	bufferSize = 32       // benchmarks suggest this is the optimum token channel buffer size
)

// TODO(@FollowTheProcess): Drop URL in favour of just Text, parser treats them the same

// TODO(@FollowTheProcess): Likewise Body

// scanFn represents the state of the scanner as a function that does the work
// associated with the current state, then returns the next state.
type scanFn func(*Scanner) scanFn

// Scanner is the http file scanner.
type Scanner struct {
	tokens            chan token.Token    // Channel on which to emit scanned tokens
	state             scanFn              // The scanner's current state
	name              string              // Name of the file
	diagnostics       []syntax.Diagnostic // Diagnostics gathered during scanning
	src               []byte              // Raw source text
	start             int                 // The start position of the current token
	pos               int                 // Current scanner position in src (bytes, 0 indexed)
	line              int                 // Current line number, 1 indexed
	currentLineOffset int                 // Offset at which the current line started
}

// New returns a new [Scanner] and kicks off the state machine in a goroutine.
func New(name string, src []byte) *Scanner {
	s := &Scanner{
		tokens: make(chan token.Token, bufferSize),
		name:   name,
		src:    src,
		state:  scanStart,
		line:   1,
	}

	return s
}

// Scan scans the input and returns the next token.
func (s *Scanner) Scan() token.Token {
	for {
		select {
		case tok := <-s.tokens:
			return tok
		default:
			// Move to the next state
			s.state = s.state(s)
		}
	}
}

// Diagnostics returns the list of diagnostics gathered during scanning.
func (s *Scanner) Diagnostics() []syntax.Diagnostic {
	// Create a copy so caller can't mutate the original diagnostics slice
	diagCopy := make([]syntax.Diagnostic, 0, len(s.diagnostics))
	diagCopy = append(diagCopy, s.diagnostics...)

	return diagCopy
}

// atEOF reports whether the scanner has reached the end of the input.
func (s *Scanner) atEOF() bool {
	return s.pos >= len(s.src)
}

// next returns the next utf8 rune in the input, or [eof], and advances the scanner
// over that rune such that successive calls to [Scanner.next] iterate through
// src one rune at a time.
func (s *Scanner) next() rune {
	if s.atEOF() {
		return eof
	}

	char, width := utf8.DecodeRune(s.src[s.pos:])
	if char == utf8.RuneError && width == 1 {
		s.errorf("invalid utf8 character: %U", char)
		return utf8.RuneError
	}

	s.pos += width

	if char == '\n' {
		s.line++
		s.currentLineOffset = s.pos
	}

	return char
}

// peek returns the next utf8 rune in the input, or [eof], but does not
// advance the scanner.
//
// Successive calls to peek simply return the same rune again and again.
func (s *Scanner) peek() rune {
	if s.atEOF() {
		return eof
	}

	char, width := utf8.DecodeRune(s.src[s.pos:])
	if char == utf8.RuneError && width == 1 {
		s.errorf("invalid utf8 character: %U", char)
		return utf8.RuneError
	}

	return char
}

// rest returns the rest of the input from the current scanner position,
// or nil if the scanner is at EOF.
func (s *Scanner) rest() []byte {
	if s.atEOF() {
		return nil
	}

	return s.src[s.pos:]
}

// skip ignores any characters for which the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the
// first 'false' char.
//
// The scanner start position is brought up to the current position before returning, effectively
// ignoring everything it's travelled over in the meantime.
func (s *Scanner) skip(predicate func(r rune) bool) {
	for predicate(s.peek()) {
		s.next()
	}

	s.start = s.pos
}

// restHasPrefix reports whether the remainder of the input begins with the
// provided run of characters.
func (s *Scanner) restHasPrefix(prefix string) bool {
	return bytes.HasPrefix(s.rest(), []byte(prefix))
}

// takeWhile consumes characters so long as the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the first 'false' rune.
func (s *Scanner) takeWhile(predicate func(r rune) bool) {
	for predicate(s.peek()) {
		s.next()
	}
}

// takeUntil consumes characters until it hits any of the specified runes.
//
// It stops before it consumes the first specified rune such that after it returns,
// the next call to [Scanner.next] returns the offending rune.
//
//	s.takeUntil('\n', '\t') // Consume runes until you hit a newline or a tab
func (s *Scanner) takeUntil(runes ...rune) {
	for {
		next := s.peek()
		if slices.Contains(runes, next) || next == utf8.RuneError {
			return
		}
		// Otherwise, advance the scanner
		s.next()
	}
}

// takeExact consumes exactly the provided text if it is the very next thing
// the scanner encounters.
//
// If the next characters in src do not match, this is a no-op.
func (s *Scanner) takeExact(match string) bool {
	if !s.restHasPrefix(match) {
		return false
	}

	for _, char := range match {
		if s.peek() == char {
			s.next()
		}
	}

	return true
}

// emit passes a token over the tokens channel, using the scanner's internal
// state to populate position information.
func (s *Scanner) emit(kind token.Kind) {
	s.tokens <- token.Token{
		Kind:  kind,
		Start: s.start,
		End:   s.pos,
	}

	s.start = s.pos
}

// error calculates the position information and calls the installed error handler
// with the information, emitting an error token in the process.
func (s *Scanner) error(msg string) scanFn {
	// Column is the number of bytes between the last newline and the current position
	// +1 because columns are 1 indexed
	startCol := 1 + s.start - s.currentLineOffset
	endCol := 1 + s.pos - s.currentLineOffset

	// If they only differ by 1, they are just pointing at
	// one character so just point at the start of this char
	if math.Abs(float64(startCol-endCol)) == 1 {
		endCol = startCol
	}

	s.emit(token.Error)

	position := syntax.Position{
		Name:     s.name,
		Offset:   s.pos,
		Line:     s.line,
		StartCol: startCol,
		EndCol:   endCol,
	}

	diag := syntax.Diagnostic{
		Position: position,
		Msg:      msg,
	}

	// TODO(@FollowTheProcess): If we're terminating the scan, do we need diagnostics to be a list?

	s.diagnostics = append(s.diagnostics, diag)

	// Terminate the scan
	return nil
}

// errorf calls error with a formatted message.
func (s *Scanner) errorf(format string, a ...any) scanFn {
	return s.error(fmt.Sprintf(format, a...))
}

// scanStart is the initial state of the scanner.
//
// The only things that are valid at the top level are:
//
//   - '#' Beginning a comment or as the first char in a request separator
//   - '/' Beginning a comment
//   - '@' Declaring a global variable
//
// The scanner progresses from these initial states with each state
// in charge of where it goes next.
func scanStart(s *Scanner) scanFn {
	s.skip(unicode.IsSpace)

	switch char := s.next(); char {
	case '#':
		return scanHash
	case '/':
		return scanSlash
	case '@':
		return scanAt
	case eof:
		s.emit(token.EOF)
		return nil
	default:
		return s.errorf("unrecognised character: %q", char)
	}
}

// scanHash scans a literal '#', either as the start of a
// comment or if it's immediately followed by another 2 '##' then
// as the start of a request separator.
//
// It assumes the opening '#' has already been consumed.
func scanHash(s *Scanner) scanFn {
	// The opening '#' has been consumed so we're looking for 2 more
	// to make a separator
	if s.restHasPrefix("##") {
		return scanSeparator
	}

	return scanComment
}

// scanSlash scans a '/' which only has any significance if
// it is followed by another one to form the start of a slash
// comment.
func scanSlash(s *Scanner) scanFn {
	if s.peek() != '/' {
		// Ignore
		return scanStart
	}

	s.next() // Consume the second '/' we now know is there

	return scanComment
}

// scanAt scans a '@' literal.
//
// It assumes the '@' has already been consumed.
func scanAt(s *Scanner) scanFn {
	s.emit(token.At)

	if isAlpha(s.peek()) {
		return scanIdent
	}

	return scanStart
}

// scanIdent scans an ident, that is; a continuous sequence
// of valid ident characters.
func scanIdent(s *Scanner) scanFn {
	s.takeWhile(isIdent)

	// Is it a keyword? If so token.Keyword will return it's
	// proper token type, else [token.Ident].
	// Either way we need to emit it and then check for an optional '='
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)
	s.emit(kind)

	s.skip(isLineSpace)

	peek := s.peek()

	switch {
	case peek == '=':
		// @var = <value>
		return scanEq
	case isAlphaNumeric(peek):
		// @var <value>
		// Value could be a timeout, hence alpha-numeric
		return scanText
	default:
		return scanStart
	}
}

// scanComment scans a line comment started with either a '#' or '//'.
//
// The comment opening character(s) have already been consumed.
func scanComment(s *Scanner) scanFn {
	s.skip(isLineSpace)

	// Requests may have '{//|#} @ident [=] <text>' to set request-scoped
	// variables
	if s.peek() == '@' {
		return scanAt
	}

	// Absorb the whole line as the comment
	s.takeUntil('\n', eof)

	s.emit(token.Comment)

	return scanStart
}

// scanSeparator scans the literal '###' used as a request separator.
//
// It assumes the first '#' has already been consumed and that the '##' is next.
func scanSeparator(s *Scanner) scanFn {
	s.takeExact("##")
	s.emit(token.Separator)

	// If there is text on the same line as the separator it is a request comment
	s.skip(isLineSpace)

	if isText(s.peek()) {
		return scanComment
	}

	return scanRequest
}

// scanEq scans a '=' character, as used in a variable declaration.
func scanEq(s *Scanner) scanFn {
	s.next()
	s.emit(token.Eq)

	s.skip(isLineSpace)

	// TODO(@FollowTheProcess): More things are allowed after a '='
	return scanText
}

// scanText scans a series of continuous text characters (no whitespace).
func scanText(s *Scanner) scanFn {
	s.takeWhile(isText)
	s.emit(token.Text)

	return scanStart
}

// scanRequest scans inside a request definition, it assumes
// the opening separator '###' and any trailing comment have
// already been consumed and emitted.
func scanRequest(s *Scanner) scanFn {
	s.skip(unicode.IsSpace)

	// TODO(@FollowTheProcess): More things here obviously
	//
	// Like request variables, prompts etc.

	return scanMethod
}

// scanMethod scans a HTTP method.
func scanMethod(s *Scanner) scanFn {
	s.takeWhile(isAlpha)

	text := string(s.src[s.start:s.pos])

	kind, isMethod := token.Method(text)
	if !isMethod {
		if len(text) != 0 {
			return s.errorf("expected HTTP method, got %q", text)
		}
	}

	if s.atEOF() {
		return s.error("unexpected EOF, expected HTTP method")
	}

	s.emit(kind)

	s.skip(isLineSpace)

	// Now URL
	return scanURL
}

// scanURL scans a URL.
func scanURL(s *Scanner) scanFn {
	// TODO(@FollowTheProcess): Interpolation
	s.takeWhile(isText)
	s.emit(token.Text)

	// Is there a HTTP version?
	s.skip(isLineSpace)

	if s.restHasPrefix("HTTP/") {
		return scanHTTPVersion
	}

	return scanStart
}

// scanHTTPVersion scans a 'HTTP/<version>' statement.
//
// The 'HTTP/' is known to exist.
func scanHTTPVersion(s *Scanner) scanFn {
	s.takeExact("HTTP/")

	if peek := s.peek(); !isDigit(peek) {
		return s.errorf("bad HTTP version, expected digit got %q", peek)
	}

	s.takeWhile(isDigit)

	if s.takeExact(".") {
		// It's e.g. 1.2
		s.takeWhile(isDigit)
	}

	s.emit(token.HTTPVersion)

	// TODO(@FollowTheProcess): Headers next
	return scanStart
}

// isAlpha reports whether r is an alpha character.
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isIdent reports whether r is a valid identifier character.
func isIdent(r rune) bool {
	return isAlpha(r) || isDigit(r) || r == '_' || r == '-'
}

// isDigit reports whether r is a valid ASCII digit.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isAlphaNumeric reports whether r is an alpha-numeric character.
func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

// isText reports whether r is valid in a continuous string of text.
func isText(r rune) bool {
	return !unicode.IsSpace(r) && r != eof
}

// isLineSpace reports whether r is a non line terminating whitespace character,
// imagine [unicode.IsSpace] but without '\n' or '\r'.
func isLineSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
