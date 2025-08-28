// Package scanner implement a lexical scanner for .http files, reading the raw source text
// and emitting a stream of tokens.
//
// Unlike parsing a general purpose programming language, the .http file syntax is very context
// dependent and there's not a lot of symbols or structure to distinguish these contexts from
// one another e.g. a programming language may have braces, brackets etc. where as in .http files
// it's a bit more implicit.
//
// For instance a HTTP header just looks like regular text, there are no quotes to distinguish
// it as a string.
//
// Because of this, the scanner implemented here is context dependent and will only allow certain
// tokens in certain contexts. This adds a bit of complexity but reduces it in the parser, as with
// everything, this is a trade off.
//
// In the [spec] it treats certain whitespace as significant e.g. a GET <url> must be followed by a single
// newline '\n' or '\r\n'. In this implementation whitespace is entirely ignored. This produces a more
// robust implementation as it can handle discrepancies in formatting.
//
// [spec]: https://github.com/JetBrains/http-request-in-editor-spec
package scanner

import (
	"fmt"
	"iter"
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

// scanFn represents the state of the scanner as a function that does the work
// associated with the current state, then returns the next state.
type scanFn func(*Scanner) scanFn

// Scanner is the http file scanner.
type Scanner struct {
	handler           syntax.ErrorHandler // The installed error handler, to be called in response to scanning errors
	tokens            chan token.Token    // Channel on which to emit scanned tokens
	name              string              // Name of the file
	src               []byte              // Raw source text
	start             int                 // The start position of the current token
	pos               int                 // Current scanner position in src (bytes, 0 indexed)
	line              int                 // Current line number, 1 indexed
	currentLineOffset int                 // Offset at which the current line started
}

// New returns a new [Scanner] and kicks off the state machine in a goroutine.
func New(name string, src []byte, handler syntax.ErrorHandler) *Scanner {
	s := &Scanner{
		handler: handler,
		tokens:  make(chan token.Token, bufferSize),
		name:    name,
		src:     src,
		line:    1,
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

// All returns an iterator over the tokens in the file, stopping at EOF or Error.
//
// The final token will still be yielded.
func (s *Scanner) All() iter.Seq[token.Token] {
	return func(yield func(token.Token) bool) {
		for {
			tok, ok := <-s.tokens
			if !ok || !yield(tok) {
				return
			}
		}
	}
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

// takeWhile consumes characters so long as the predicate returns true, stopping at the
// first one that returns false such that after it returns, [Scanner.next] returns the first 'false' rune.
//
//nolint:unused // We will need this
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
	// So that even if there is no handler installed, we still know something
	// went wrong
	s.emit(token.Error)

	if s.handler == nil {
		// Nothing more to do
		return
	}

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

	s.handler(position, msg)
}

// errorf calls error with a formatted message.
func (s *Scanner) errorf(format string, a ...any) {
	s.error(fmt.Sprintf(format, a...))
}

// scanStart is the initial state of the scanner.
//
// At the start (or top) state, the only things that can be encountered in a valid .http
// file are:
//
//   - '#' for comments and request separators
//   - '/' for comments
//   - '@' for global variables
//
// Everything else must only appear in certain contexts e.g. HTTP methods may *only* appear
// immediately after a separator. HTTP versions may *only* appear after a URL etc.
//
// Whitespace is ignored.
func scanStart(s *Scanner) scanFn {
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
	default:
		s.errorf("unrecognised character: %q", char)
		return nil
	}
}

// scanHash scans a '#' character, either as the beginning token for a comment
// or as the first character of a '###' request separator.
func scanHash(s *Scanner) scanFn {
	if s.peek() == '#' {
		// It's a request separator
		return scanSeparator
	}

	return scanComment
}

// scanSlash scans a '/' character, either as the beginning of a '//' style comment
// or as part of some other text we don't care about.
func scanSlash(s *Scanner) scanFn {
	if s.peek() != '/' {
		// Ignore
		s.next()
		return scanStart
	}

	s.next() // Consume the second '/'

	return scanComment
}

// scanComment scans a line comment started by either a '#' or '//'.
//
// The comment opening character(s) have already been consumed.
func scanComment(s *Scanner) scanFn {
	// It must be a comment, skip any leading whitespace between the marker
	// and the comment text
	s.skip(isLineSpace)

	// Requests may have '{//|#} @ident [=] <text>' to set request-scoped
	// variables
	if s.peek() == '@' {
		panic("TODO: Handle request variables")
		// s.next() // Consume the '@'
		// return scanAt
	}

	// Absorb everything until the end of the line or eof
	s.takeUntil('\n', eof)

	s.emit(token.Comment)

	return scanStart
}

// scanSeparator scans the literal '###' used as a request separator.
func scanSeparator(s *Scanner) scanFn {
	// Absorb no more than 3 '#'
	count := 0

	const sepLength = 3 // len("###")

	for s.peek() == '#' {
		count++

		s.next()

		if count == sepLength {
			break
		}
	}

	s.emit(token.Separator)

	// If there is text on the same line as the separator it is a request comment
	s.skip(isLineSpace)

	if s.peek() != '\n' && s.peek() != eof {
		return scanComment
	}

	return scanStart
}

// isLineSpace reports whether r is a non line terminating whitespace character,
// imagine [unicode.IsSpace] but without '\n' or '\r'.
func isLineSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
