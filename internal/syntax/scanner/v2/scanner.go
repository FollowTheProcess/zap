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
	tokens             chan token.Token    // Channel on which to emit scanned tokens.
	state              stateFn             // The scanner's current state
	name               string              // Name of the file
	diagnostics        []syntax.Diagnostic // Diagnostics gathered during scanning
	src                []byte              // Raw source text
	start              int                 // The start position of the current token
	pos                int                 // Current scanner position in src (bytes, 0 indexed)
	width              int                 // Width of the last rune scanned, allows backing up
	line               int                 // Current line number (1 indexed)
	currentLineOffset  int                 // Offset at which the current line started, used for column calculation
	previousLineOffset int                 // Offset at which the previous line started, used when backing up over a line
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

// next returns the next utf8 rune in the input or [eof], and advances
// the scanner over that rune such that successive calls to next iterate
// through src one rune at a time.
func (s *Scanner) next() rune {
	if s.atEOF() {
		return eof
	}

	char, width := utf8.DecodeRune(s.src[s.pos:])
	if char == utf8.RuneError && width == 1 {
		s.errorf("invalid utf8 character at position %d: %q", s.pos, s.src[s.pos])
		return utf8.RuneError
	}

	s.width = width
	s.pos += width

	if char == '\n' {
		s.line++
		s.previousLineOffset = s.currentLineOffset
		s.currentLineOffset = s.pos
	}

	return char
}

// backup backs up by one rune, can only be called once per call of next.
func (s *Scanner) backup() {
	s.pos -= s.width
	if s.pos < len(s.src) && s.src[s.pos] == '\n' {
		s.line--
		s.currentLineOffset = s.previousLineOffset
	}
}

// peek returns the next utf8 rune in the input or [eof], but does not
// advance the scanner. Successive calls to peek return the same char
// over and over again.
func (s *Scanner) peek() rune {
	next := s.next()
	s.backup()

	return next
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
	for {
		if predicate(s.next()) {
			continue
		}

		s.backup()
		s.start = s.pos
		return
	}
}

// takeWhile consumes characters so long as the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the first 'false' rune.
func (s *Scanner) takeWhile(predicate func(r rune) bool) {
	for {
		if predicate(s.next()) {
			continue
		}

		s.backup()
		return
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
		next := s.next()
		if slices.Contains(runes, next) {
			s.backup()
			return
		}
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

	s.start = s.pos
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
	default:
		s.errorf("unexpected character: %q", char)
		return nil
	}
}
