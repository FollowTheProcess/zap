// Package scanner implements a lexical scanner for .http files, reading the raw source
// text and emitting a stream of tokens.
//
// The scanner is a concurrent, state-function based scanner similar to that described by
// Rob Pike in his talk [Lexical Scanning in Go], based on the implementation of [text/template].
//
// The scanner proceeds one utf8 rune at a time until a particular token is recognised, the token
// is then emitted over a channel where it may be consumed by the parser. The state of the scanner
// is maintained between token emits unlike a more traditional switch-based lexer.
//
// This is useful for http files as they are not a completely context-free grammar, there are no
// quotes to begin a string for example, the start of a request header can look like a request
// method etc.
//
// A similar approach is taken in [BurntSushi/toml].
//
// [Lexical Scanning in Go]: https://go.dev/talks/2011/lex.slide#1
// [BurntSushi/toml]: https://github.com/BurntSushi/toml/blob/master/lex.go
package scanner

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

const (
	eof        = rune(-1) // eof signifies we have reached the end of the input.
	bufferSize = 32       // benchmarks suggest this is the optimum token channel buffer size.
)

// stateFn represents the state of the scanner as a function that does the work
// associated with the current state, then returns the next state.
type stateFn func(*Scanner) stateFn

// Scanner is the http file scanner.
type Scanner struct {
	tokens            chan token.Token    // Channel on which to emit scanned tokens.
	state             stateFn             // The scanner's current state
	name              string              // Name of the file
	diagnostics       []syntax.Diagnostic // Diagnostics gathered during scanning
	src               []byte              // Raw source text
	start             int                 // The start position of the current token
	pos               int                 // Current scanner position in src (bytes, 0 indexed)
	line              int                 // Current line number (1 indexed)
	currentLineOffset int                 // Offset at which the current line started, used for column calculation
}

// New returns a new [Scanner].
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

			if s.state == nil {
				// No more tokens
				close(s.tokens)
			}
		}
	}
}

// Diagnostics returns the list of diagnostics gathered during scanning.
func (s *Scanner) Diagnostics() []syntax.Diagnostic {
	return s.diagnostics
}

// atEOF reports whether the scanner is at the end of the input.
func (s *Scanner) atEOF() bool {
	return s.pos >= len(s.src)
}

// char returns the next utf8 rune in the input or [eof], along with it's width.
func (s *Scanner) char() (rune, int) {
	if s.atEOF() {
		return eof, 0
	}

	r, width := utf8.DecodeRune(s.src[s.pos:])
	if r == utf8.RuneError && width == 1 {
		s.errorf("invalid utf8 character at position %d: %q", s.pos, s.src[s.pos])
		return utf8.RuneError, width
	}

	return r, width
}

// next returns the next utf8 rune in the input or [eof], and advances
// the scanner over that rune such that successive calls to next iterate
// through src one rune at a time.
func (s *Scanner) next() rune {
	char, width := s.char()

	// Advance the state of the scanner
	s.pos += width

	if char == '\n' {
		s.line++
		s.currentLineOffset = s.pos
	}

	return char
}

// peek returns the next utf8 rune in the input or [eof], but does not
// advance the scanner. Successive calls to peek return the same char
// over and over again.
func (s *Scanner) peek() rune {
	// No advancing the state
	char, _ := s.char()
	return char
}

// discard brings the start position up to current, effectively discarding
// any text the scanner has "collected" up to this point.
func (s *Scanner) discard() {
	s.start = s.pos
}

// rest returns the rest of the input from the current scanner position,
// or nil if the scanner is an EOF.
func (s *Scanner) rest() []byte {
	if s.atEOF() {
		return nil
	}

	return s.src[s.pos:]
}

// restHasPrefix reports whether the remainder of the input begins with the
// provided run of characters.
func (s *Scanner) restHasPrefix(prefix string) bool {
	return bytes.HasPrefix(s.rest(), []byte(prefix))
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

	s.discard()
}

// take consumes the next rune if it's from the valid set, and returns
// whether it was accepted.
func (s *Scanner) take(valid string) bool {
	if strings.ContainsRune(valid, s.peek()) {
		s.next()
		return true
	}

	return false
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
	// Implicitly also break on RuneError
	runes = append(runes, utf8.RuneError)

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

	for range match {
		s.next()
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

	// We've just emitted it, no need to keep it
	s.discard()
}

// error calculates the position information and calls the installed error handler
// with the information, emitting an error token in the process.
func (s *Scanner) error(msg string) {
	// Column is the number of bytes between the last newline and the current position
	// +1 because columns are 1 indexed
	startCol := 1 + s.start - s.currentLineOffset
	endCol := 1 + s.pos - s.currentLineOffset

	s.emit(token.Error)

	// If startCol and endCol only differ by 1, it's pointing
	// at a single character so we don't need a range, just point
	// at the start of the char.
	if math.Abs(float64(startCol-endCol)) <= 1 {
		endCol = startCol
	}

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

	s.diagnostics = append(s.diagnostics, diag)
}

// errorf calls error with a formatted message.
func (s *Scanner) errorf(format string, a ...any) {
	s.error(fmt.Sprintf(format, a...))
}

// scanStart is the initial state of the scanner.
//
// The only things it can encounter at the top level of a valid
// http file are:
//
//   - '#' for comments and request separators
//   - '/' for comments
//   - '@' for global variables
//
// Everything else must only appear in certain contexts.
//
// Whitespace is ignored.
func scanStart(s *Scanner) stateFn {
	s.skip(unicode.IsSpace)

	switch char := s.next(); char {
	case eof:
		s.emit(token.EOF)
		return nil
	case utf8.RuneError:
		// next() already emits an error for this
		return nil
	case '#':
		return scanHash
	case '/':
		return scanSlash
	case '@':
		return scanAt
	default:
		s.errorf("unexpected character: %q", char)
		return nil
	}
}

// scanHash scans a literal '#' either as the opener to a comment
// or the first char in a request separator.
//
// It assumes the '#' has already been consumed.
func scanHash(s *Scanner) stateFn {
	if s.restHasPrefix("##") {
		return scanSeparator
	}

	return scanComment
}

// scanSlash scans a literal '/' as the opener to a slash comment, if
// the next char is not another '/', it is ignored.
//
// It assumes the first '/' has already been consumed.
func scanSlash(s *Scanner) stateFn {
	if !s.take("/") {
		s.error("invalid use of '/', two '//' mark a comment start, got '/'")
		return nil
	}

	return scanComment
}

// scanAt scans a literal '@' as in a variable or prompt declaration.
//
// It assumes the '@' has already been consumed.
func scanAt(s *Scanner) stateFn {
	s.emit(token.At)

	if isAlpha(s.peek()) {
		return scanIdent
	}

	return scanStart
}

// scanComment scans a line comment started by either a '#' or '//'.
//
// It assumed he comment opening character(s) have already been consumed.
func scanComment(s *Scanner) stateFn {
	s.skip(isLineSpace)

	s.takeUntil('\n', eof)
	s.emit(token.Comment)

	return scanStart
}

// scanSeparator scans a '###' request separator.
//
// It assumes the first '#' has already been consumed.
func scanSeparator(s *Scanner) stateFn {
	s.takeExact("##")
	s.emit(token.Separator)

	// Is there a request comment?
	s.skip(isLineSpace)

	if s.peek() != '\n' && s.peek() != eof {
		s.takeUntil('\n', eof)
		s.emit(token.Comment)
	}

	return scanRequest
}

// scanIdent scans a continuous string of characters as an identifier.
func scanIdent(s *Scanner) stateFn {
	s.takeWhile(isIdent)

	// Is it a keyword?
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)
	s.emit(kind)

	s.skip(isLineSpace)

	if s.take("=") {
		s.emit(token.Eq)
		s.skip(isLineSpace)
	}

	if isText(s.peek()) {
		return scanText
	}

	return scanStart
}

// scanText scans a continuous string of text.
func scanText(s *Scanner) stateFn {
	s.takeWhile(isText)
	s.emit(token.Text)

	return scanStart
}

// scanRequest scans inside a HTTP request definition.
//
// The opening '###' and any request comment has already been consumed.
func scanRequest(s *Scanner) stateFn {
	s.skip(unicode.IsSpace)

	switch char := s.next(); char {
	case eof:
		s.emit(token.EOF)
		return nil
	case utf8.RuneError:
		// next() already emits an error for this
		return nil
	case '#':
		return scanRequestHash
	case '/':
		return scanRequestSlash
	default:
		if isUpperAlpha(char) {
			// A request method
			return scanMethod
		}

		s.errorf("unexpected character: %q", char)

		return nil
	}
}

// scanRequestHash scans a a literal '#' in the context of
// a request local variable or line comment.
func scanRequestHash(s *Scanner) stateFn {
	return scanRequestComment
}

// scanRequestSlash scans a literal '/' as the opener to a slash comment, if
// the next char is not another '/', it is ignored.
//
// It assumes the first '/' has already been consumed.
func scanRequestSlash(s *Scanner) stateFn {
	if !s.take("/") {
		s.error("invalid use of '/', two '//' mark a comment start, got '/'")
		return nil
	}

	return scanRequestComment
}

// scanRequestComment scans a line comment inside a request block.
func scanRequestComment(s *Scanner) stateFn {
	s.skip(isLineSpace)

	// Requests may have # @ident = text variables
	if s.take("@") {
		s.emit(token.At)
		return scanRequestVariable
	}

	s.takeUntil('\n', eof)
	s.emit(token.Comment)

	return scanRequest
}

// scanRequestVariable scans a request variable declaration.
//
// It assumes the opening '@' has already been consumed.
func scanRequestVariable(s *Scanner) stateFn {
	s.takeWhile(isIdent)

	// Could be a keyword like timeout etc.
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)
	s.emit(kind)

	s.skip(isLineSpace)

	if s.take("=") {
		s.emit(token.Eq)
		s.skip(isLineSpace)
	}

	if isText(s.peek()) {
		s.takeWhile(isText)
		s.emit(token.Text)
	}

	return scanRequest
}

// scanMethod scans a HTTP method.
func scanMethod(s *Scanner) stateFn {
	s.takeWhile(isUpperAlpha)

	text := string(s.src[s.start:s.pos])

	kind, isMethod := token.Method(text)
	if !isMethod {
		s.errorf("expected HTTP method, got %q", text)
		return nil
	}

	s.emit(kind)
	s.skip(isLineSpace)

	if !s.restHasPrefix("http") && !s.restHasPrefix("{{") {
		s.errorf("expected URL or interpolation, got %q", s.peek())
		return nil
	}

	return scanURL
}

// scanURL scans a request URL.
func scanURL(s *Scanner) stateFn {
	// TODO(@FollowTheProcess): Handle interp
	s.takeWhile(isURL)
	s.emit(token.Text)

	// TODO(@FollowTheProcess): Should move the state to scanHTTPVersion or scanHeaders
	return scanStart
}

// isLineSpace reports whether r is a non line terminating whitespace character,
// imagine [unicode.IsSpace] but without '\n' or '\r'.
func isLineSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isDigit reports whether r is a valid ASCII digit.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isAlpha reports whether r is an alpha character.
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isUpperAlpha reports whether r is an upper case alpha character.
func isUpperAlpha(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// isAlphaNumeric reports whether r is a valid alpha-numeric character.
func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

// isIdent reports whether r is a valid identifier character.
func isIdent(r rune) bool {
	return isAlphaNumeric(r) || r == '_' || r == '-'
}

// isText reports whether r is valid in a continuous string of text.
//
// The only things that are rejected by this are:
//
//   - whitespace
//   - eof
//   - bad utf8 runes
func isText(r rune) bool {
	return !unicode.IsSpace(r) && r != eof && r != utf8.RuneError
}

// isURL reports whether r is valid in a URL.
func isURL(r rune) bool {
	return isAlphaNumeric(r) || strings.ContainsRune("$-_.+!*'(),:/?#[]@&;=", r)
}
