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
	"bytes"
	"fmt"
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
func (s *Scanner) rest() []byte { //nolint: unused // We will use this soon
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
	return bytes.HasPrefix(s.src[s.pos:], []byte(prefix))
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
	case '@':
		return scanAt
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
		s.next() // Consume the '@'
		return scanAt
	}

	// Absorb everything until the end of the line or eof
	s.takeUntil('\n', eof)

	s.emit(token.Comment)

	return scanStart
}

// scanSeparator scans the literal '###' used as a request separator.
func scanSeparator(s *Scanner) scanFn {
	// TODO(@FollowTheProcess): This logic is repeated in a few places

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

// scanAt scans an '@' character, used to declare a variable either globally
// scoped at the top level `@name = thing` or request scoped `# @name = thing`.
func scanAt(s *Scanner) scanFn {
	s.emit(token.At)

	if isAlpha(s.peek()) {
		// The name of the variable e.g. @ident [=] <value>
		return scanIdent
	}

	return scanStart
}

// scanIdent scans an ident, that is; a continuous sequence of
// characters that are valid as an identifier.
func scanIdent(s *Scanner) scanFn {
	s.takeWhile(isIdent)

	// Is it a keyword? If so token.Keyword will return it's
	// proper token type, else [token.Ident].
	// Either way we need to emit it and then check for an optional '='
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)
	s.emit(kind)
	s.skip(isLineSpace)

	switch {
	case kind == token.Prompt:
		// Prompts are handled in a special way as you may have e.g.
		// @prompt username <Arbitrary description on a single line>
		return scanPrompt
	case s.peek() == '=':
		// @var = <value>
		return scanEq
	case isAlphaNumeric(s.peek()):
		// @var <value>
		// Note: value could be a timeout, hence alpha numeric
		return scanText
	default:
		return scanStart
	}
}

// scanPrompt scans a variable prompt e.g. @prompt username [description].
//
// The '@' and the 'prompt' have already been consumed and their tokens emitted.
func scanPrompt(s *Scanner) scanFn {
	// The first thing up should be the name of the variable we're prompting
	// for, which follows standard ident rules
	s.takeWhile(isIdent)
	s.emit(token.Ident)

	// Arbitrary space is allowed so long as it's on the same line
	s.skip(isLineSpace)

	// If the next thing looks like regular text, this is the optional
	// description for the prompt
	if isAlphaNumeric(s.peek()) {
		s.takeUntil('\n', eof)
		s.emit(token.Text)
	}

	return scanStart
}

// scanEq scans a '=' character, as used in a variable declaration.
func scanEq(s *Scanner) scanFn {
	s.next()
	s.emit(token.Eq)

	s.skip(isLineSpace)

	if isAlphaNumeric(s.peek()) {
		return scanText
	}

	if s.restHasPrefix("{{") {
		return scanOpenInterp
	}

	return scanStart
}

// scanOpenInterp scans an opening '{{' token.
func scanOpenInterp(s *Scanner) scanFn {
	// Absorb no more than 2 '{'
	count := 0

	const n = 2 // len("{{")

	for s.peek() == '{' {
		count++

		s.next()

		if count == n {
			break
		}
	}

	s.emit(token.OpenInterp)

	// TODO(@FollowTheProcess): More can go here but for now let's assume
	// it's always an ident
	s.skip(isLineSpace)

	if isAlpha(s.peek()) {
		// We don't actually want to move to the next state yet
		// after the ident, just scan it and remember where we should
		// go next
		scanIdent(s)
	}

	if !s.restHasPrefix("}}") {
		s.error("unterminated interpolation")
		return nil
	}

	return scanCloseInterp
}

// scanCloseInterp scans a closing '}}' token.
//
// The '}}' is known to be the next 2 characters in the input by
// the time this is called.
func scanCloseInterp(s *Scanner) scanFn {
	// Absorb no more than 2 '}'
	count := 0

	const n = 2 // len("}}")

	for s.peek() == '}' {
		count++

		s.next()

		if count == n {
			break
		}
	}

	s.emit(token.CloseInterp)

	return scanStart
}

// scanText scans a series of continuous text characters (no whitespace).
func scanText(s *Scanner) scanFn {
	s.takeWhile(isText)

	s.emit(token.Text)

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
