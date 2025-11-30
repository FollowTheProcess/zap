// Package scanner implements a lexical scanner for .http files, reading the raw source text
// and emitting a stream of tokens.
//
// The v2 version is intended to be much less stateful than the original.
package scanner

import (
	"bytes"
	"fmt"
	"slices"
	"sync"
	"unicode"
	"unicode/utf8"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

const (
	eof        = rune(-1) // eof signifies we have reached the end of the input.
	bufferSize = 32       // benchmarks suggest this is the optimum token channel buffer size
)

// state represents the state of the scanner as a function that does the work
// associated with the current state, then returns the next state.
type state func(*Scanner) state

// Scanner is the http file scanner.
type Scanner struct {
	tokens            chan token.Token    // Channel on which to emit scanned tokens
	name              string              // Name of the file
	diagnostics       []syntax.Diagnostic // Diagnostics gathered during scanning
	src               []byte              // Raw source text
	start             int                 // The start position of the current token
	pos               int                 // Current scanner position in src (bytes, 0 indexed)
	line              int                 // Current line number, 1 indexed
	currentLineOffset int                 // Offset at which the current line started
	mu                sync.RWMutex        // Guards diagnostics
}

// New returns a new [Scanner] and kicks off the state machine in a goroutine.
func New(name string, src []byte) *Scanner {
	s := &Scanner{
		tokens: make(chan token.Token, bufferSize),
		name:   name,
		src:    src,
		line:   1,
	}

	// run terminates when the scanning state machine is finished and all the
	// tokens are drained from s.tokens, so no other synchronisation needed here
	go s.run()

	return s
}

// Scan scans the input and returns the next token.
func (s *Scanner) Scan() token.Token {
	return <-s.tokens
}

// Diagnostics returns the list of diagnostics gathered during scanning.
func (s *Scanner) Diagnostics() []syntax.Diagnostic {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy so caller can't mutate the original diagnostics slice
	diagCopy := make([]syntax.Diagnostic, 0, len(s.diagnostics))
	diagCopy = append(diagCopy, s.diagnostics...)

	return diagCopy
}

// next returns the next utf8 rune in the input, or [eof], and advances the scanner
// over that rune such that successive calls to [Scanner.next] iterate through
// src one rune at a time.
func (s *Scanner) next() rune {
	if s.pos >= len(s.src) {
		return eof
	}

	char, width := utf8.DecodeRune(s.src[s.pos:])
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
	if s.pos >= len(s.src) {
		return eof
	}

	char, _ := utf8.DecodeRune(s.src[s.pos:])

	return char
}

// rest returns the rest of the input from the current scanner position,
// or nil if the scanner is at EOF.
func (s *Scanner) rest() []byte {
	if s.pos >= len(s.src) {
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
		if slices.Contains(runes, next) {
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
func (s *Scanner) takeExact(match string) {
	if !s.restHasPrefix(match) {
		return
	}

	for _, char := range match {
		if s.peek() == char {
			s.next()
		}
	}
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

// run starts the state machine for the scanner, it runs with each [scanFn] returning the next
// state until one returns nil (typically in response to an error or eof), at which point the tokens channel
// is closed as a signal to the receiver that no more tokens will be sent.
func (s *Scanner) run() {
	for state := scanStart; state != nil; {
		state = state(s)
	}

	close(s.tokens)
}

// error calculates the position information and calls the installed error handler
// with the information, emitting an error token in the process.
func (s *Scanner) error(msg string) {
	s.emit(token.Error)

	// Column is the number of bytes between the last newline and the current position
	// +1 because columns are 1 indexed
	startCol := 1 + s.start - s.currentLineOffset
	endCol := 1 + s.pos - s.currentLineOffset

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

	s.mu.Lock()
	defer s.mu.Unlock()

	s.diagnostics = append(s.diagnostics, diag)
}

// errorf calls error with a formatted message.
func (s *Scanner) errorf(format string, a ...any) {
	s.error(fmt.Sprintf(format, a...))
}

// scanStart is the initial state of the scanner.
func scanStart(s *Scanner) state {
	s.skip(unicode.IsSpace)

	switch char := s.next(); char {
	case eof:
		s.emit(token.EOF)
		return nil
	case utf8.RuneError:
		s.errorf("invalid utf8 character: %U", char)
		return nil
	case '#':
		return scanHash
	case '/':
		return scanSlash
	case '@':
		return scanAt
	case '=':
		return scanEq
	case '{':
		return scanOpenInterp
	default:
		if isIdent(char) {
			return scanText
		}

		s.errorf("unrecognised character: %q", char)

		return nil
	}
}

// scanHash scans a '#' literal.
//
// The (possibly first) '#' has already been consumed.
func scanHash(s *Scanner) state {
	if s.restHasPrefix("##") {
		return scanSeparator
	}

	s.skip(isLineSpace)

	if isText(s.peek()) {
		return scanComment
	}

	return scanStart
}

// scanSlash scans a '/' character as the beginning of a '//' style comment.
//
// If the next token is anything other than '/', this is a no-op and state
// is returned back to scanStart unchanged.
func scanSlash(s *Scanner) state {
	if s.peek() != '/' {
		// Ignore
		return scanStart
	}

	s.next() // Consume the second '/'

	s.skip(isLineSpace)

	if isText(s.peek()) {
		return scanComment
	}

	return scanStart
}

// scanSeparator scans the literal '###' used as a request separator.
func scanSeparator(s *Scanner) state {
	// The first '##' has already been consumed in scanStart
	s.takeExact("##")
	s.emit(token.Separator)

	// If there is text on the same line as the separator it is a request comment
	s.skip(isLineSpace)

	if isText(s.peek()) {
		return scanComment
	}

	return scanStart
}

// scanComment scans a line comment started with either '#' or '//'.
//
// The opening character(s) have already been consumed.
func scanComment(s *Scanner) state {
	// Requests may have '{//|#} @ident [=] <text>' to set request-scoped
	// variables
	if s.peek() == '@' {
		return scanStart
	}

	// Absorb everything on the line
	s.takeUntil('\n', eof)
	s.emit(token.Comment)

	return scanStart
}

// scanAt scans a literal '@'.
func scanAt(s *Scanner) state {
	s.emit(token.At)

	if isIdent(s.peek()) {
		return scanIdent
	}

	return scanStart
}

// scanEq scans a literal '='.
func scanEq(s *Scanner) state {
	s.emit(token.Eq)
	return scanStart
}

// scanIdent scans an identifier.
func scanIdent(s *Scanner) state {
	s.takeWhile(isIdent)

	// Is it a keyword? If so token.Keyword will return its proper
	// token type, else [token.Ident].
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)
	s.emit(kind)

	return scanStart
}

// scanText scans a string of text characters (no whitespace).
func scanText(s *Scanner) state {
	s.takeWhile(isText)

	// Is it a HTTP method?
	text := string(s.src[s.start:s.pos])
	method, _ := token.Method(text)
	s.emit(method)

	return scanStart
}

// scanOpenInterp scans a '{' character, as part of an open interp token.
//
// If the next char is not another '{', this is a no-op and the scanning
// state is returned back to scanStart.
func scanOpenInterp(s *Scanner) state {
	if s.peek() != '{' {
		// Ignore
		return scanStart
	}

	s.next() // Consume the second '{'
	s.emit(token.OpenInterp)

	s.skip(isLineSpace)

	return scanInterpInner
}

// scanInterpInner scans the expression inside an interpolation.
func scanInterpInner(s *Scanner) state {
	// TODO(@FollowTheProcess): More things can go here
	if isIdent(s.peek()) {
		scanIdent(s)
	}

	s.skip(isLineSpace)

	if !s.restHasPrefix("}}") {
		s.errorf("unterminated interpolation, expected %q got %q", "}}", s.peek())
		return nil
	}

	s.takeExact("}}")
	s.emit(token.CloseInterp)

	if isText(s.peek()) {
		return scanText
	}

	return scanStart
}

// isLineSpace reports whether r is a non line terminating whitespace character,
// imagine [unicode.IsSpace] but without '\n' or '\r'.
func isLineSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isAlpha reports whether r is an alpha character.
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDigit reports whether r is a valid ASCII digit.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isIdent reports whether r is a valid identifier character.
func isIdent(r rune) bool {
	return isAlpha(r) || isDigit(r) || r == '_' || r == '-'
}

// isText reports whether r is valid in a continuous string of text.
func isText(r rune) bool {
	return !unicode.IsSpace(r) && r != eof && r != '{' && r != '}'
}
