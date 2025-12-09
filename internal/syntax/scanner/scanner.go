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
	"sync"
	"unicode"
	"unicode/utf8"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

const (
	eof        = rune(-1) // eof signifies we have reached the end of the input.
	bufferSize = 32       // benchmarks suggest this is the optimum token channel buffer size.
	stackSize  = 10       // size of the state stack, should be plenty to avoid a re-allocation
)

// stateFn represents the state of the scanner as a function that does the work
// associated with the current state, then returns the next state.
type stateFn func(*Scanner) stateFn

// Scanner is the http file scanner.
type Scanner struct {
	tokens      chan token.Token    // Channel on which to emit scanned tokens.
	name        string              // Name of the file
	diagnostics []syntax.Diagnostic // Diagnostics gathered during scanning
	src         []byte              // Raw source text

	// A stack of state functions used to maintain context.
	//
	// The idea is to reuse parts of the state machine in various places. For
	// example, interpolations can appear in multiple contexts, and how do we
	// know which state to return to when we're done with the '}}'.
	stack []stateFn

	start             int          // The start position of the current token
	pos               int          // Current scanner position in src (bytes, 0 indexed)
	line              int          // Current line number (1 indexed)
	currentLineOffset int          // Offset at which the current line started, used for column calculation
	mu                sync.RWMutex // Guards diagnostics
}

// New returns a new [Scanner].
func New(name string, src []byte) *Scanner {
	s := &Scanner{
		tokens: make(chan token.Token, bufferSize),
		stack:  make([]stateFn, 0, stackSize),
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

// statePush pushes a stateFn onto the stack so the scanner can
// "remember" where it just came from.
func (s *Scanner) statePush(state stateFn) {
	s.stack = append(s.stack, state)
}

// statePop pops a stateFn off the stack so the scanner can return
// to where it just came from in certain contexts.
func (s *Scanner) statePop() stateFn {
	size := len(s.stack)

	if size == 0 {
		// TODO(@FollowTheProcess): Could we be safer and return scanStart here?
		// Or do an error and return nil, this is helpful for tests and fuzz though
		// as it will be very obvious if we've done something wrong
		panic("pop from empty state stack")
	}

	last := s.stack[size-1]
	s.stack = s.stack[:size-1]

	return last
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
	if r == utf8.RuneError || r == 0 {
		s.errorf("invalid utf8 character at position %d: %q", s.pos, s.src[s.pos])
		// Prevent cascading errors by "consuming" all remaining input
		s.pos = len(s.src)

		return utf8.RuneError, 0
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
func (s *Scanner) error(msg string) stateFn {
	s.emit(token.Error)

	// Column is the number of bytes between the last newline and the current position
	// +1 because columns are 1 indexed
	startCol := s.start - s.currentLineOffset
	endCol := s.pos - s.currentLineOffset

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

	s.mu.Lock()
	defer s.mu.Unlock()

	s.diagnostics = append(s.diagnostics, diag)

	return nil
}

// errorf calls error with a formatted message.
func (s *Scanner) errorf(format string, a ...any) stateFn {
	return s.error(fmt.Sprintf(format, a...))
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

	s.statePush(scanStart) // Remember where we came from

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
		return s.errorf("unexpected character: %q", char)
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
		return s.error("invalid use of '/', two '//' mark a comment start, got '/'")
	}

	return scanComment
}

// scanAt scans a literal '@' as in a variable or prompt declaration.
//
// It assumes the '@' has already been consumed.
func scanAt(s *Scanner) stateFn {
	s.emit(token.At)

	if isAlpha(s.peek()) {
		return scanGlobalVariable
	}

	return s.statePop()
}

// scanComment scans a line comment started by either a '#' or '//'.
//
// It assumed he comment opening character(s) have already been consumed.
func scanComment(s *Scanner) stateFn {
	s.skip(isLineSpace)

	s.takeUntil('\n', eof)
	s.emit(token.Comment)

	return s.statePop()
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

// scanGlobalVariable scans a global variable declaration.
func scanGlobalVariable(s *Scanner) stateFn {
	s.takeWhile(isIdent)

	// Is it a keyword?
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)

	if s.pos > s.start {
		s.emit(kind)
	}

	s.skip(isLineSpace)

	if kind == token.Prompt {
		return scanPrompt
	}

	if s.take("=") {
		s.emit(token.Eq)
		s.skip(isLineSpace)
	}

	if s.restHasPrefix("{{") {
		s.statePush(scanGlobalVariable)
		return scanOpenInterp
	}

	if isText(s.peek()) {
		return scanTextLine
	}

	return s.statePop()
}

// scanOpenInterp scans an opening '{{' marking the beginning
// of an interpolation.
func scanOpenInterp(s *Scanner) stateFn {
	s.takeExact("{{")
	s.emit(token.OpenInterp)

	return scanInsideInterp
}

// scanInsideInterp scans the inside of an interpolation.
func scanInsideInterp(s *Scanner) stateFn {
	s.skip(isLineSpace)

	// TODO(@FollowTheProcess): Handle more than just idents
	//
	// That's the whole reason I'm rewriting the scanner, to make this easier
	if isAlpha(s.peek()) {
		s.takeWhile(isIdent)
		s.emit(token.Ident)
	}

	s.skip(isLineSpace)

	if !s.restHasPrefix("}}") {
		return s.error("unterminated interpolation")
	}

	return scanCloseInterp
}

// scanCloseInterp scans a closing '}}' marking the end of an interpolation.
func scanCloseInterp(s *Scanner) stateFn {
	s.takeExact("}}")
	s.emit(token.CloseInterp)

	// Go back to whatever state we were in before entering the interp
	return s.statePop()
}

// scanPrompt scans a prompt statement.
//
// It assumes the '@prompt' has already been consumed.
func scanPrompt(s *Scanner) stateFn {
	if isIdent(s.peek()) {
		s.takeWhile(isIdent)
		s.emit(token.Ident)
	}

	s.skip(isLineSpace)

	if isAlpha(s.peek()) {
		s.takeUntil('\n', eof)
		s.emit(token.Text)
	}

	return s.statePop()
}

// scanTextLine scans a continuous string of text, so long as it's on the same line.
func scanTextLine(s *Scanner) stateFn {
	s.takeWhile(isText)

	if s.pos > s.start {
		s.emit(token.Text)
	}

	if s.restHasPrefix("{{") {
		s.statePush(scanTextLine)
		return scanOpenInterp
	}

	return s.statePop()
}

// scanText scans until it hits an open interp.
func scanText(s *Scanner) stateFn {
	for {
		if s.restHasPrefix("{{") {
			if s.pos > s.start {
				s.emit(token.Body)
			}

			s.statePush(scanText)

			return scanOpenInterp
		}

		next := s.peek()
		if next == '#' || next == '>' || next == '<' || next == eof || next == utf8.RuneError {
			break
		}

		s.next()
	}

	// If we absorbed any text, emit it.
	//
	// This could in theory be empty because the entire body could have just been an interp, which
	// seems incredibly unlikely but possible so lets handle it
	if s.pos > s.start {
		s.emit(token.Body)
	}

	return s.statePop()
}

// scanRequest scans inside a HTTP request definition.
//
// The opening '###' and any request comment has already been consumed.
//
// The only thing allowed top level in a request is '#', '//' and
// a request method.
func scanRequest(s *Scanner) stateFn {
	s.skip(unicode.IsSpace)

	s.statePush(scanRequest) // Remember we were scanning a request

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
			return scanMethod
		}

		return s.errorf("unexpected character: %q", char)
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
		return s.error("invalid use of '/', two '//' mark a comment start, got '/'")
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

	return s.statePop()
}

// scanRequestVariable scans a request variable declaration.
//
// It assumes the opening '@' has already been consumed.
func scanRequestVariable(s *Scanner) stateFn {
	s.takeWhile(isIdent)

	// Is it a keyword?
	text := string(s.src[s.start:s.pos])
	kind, _ := token.Keyword(text)

	if s.pos > s.start {
		s.emit(kind)
	}

	s.skip(isLineSpace)

	if kind == token.Prompt {
		return scanPrompt
	}

	if s.take("=") {
		s.emit(token.Eq)
		s.skip(isLineSpace)
	}

	if s.restHasPrefix("{{") {
		s.statePush(scanRequestVariable)
		return scanOpenInterp
	}

	if isText(s.peek()) {
		return scanTextLine
	}

	return s.statePop()
}

// scanMethod scans a HTTP method.
func scanMethod(s *Scanner) stateFn {
	s.takeWhile(isUpperAlpha)

	text := string(s.src[s.start:s.pos])

	kind, isMethod := token.Method(text)
	if !isMethod {
		return s.errorf("expected HTTP method, got %q", text)
	}

	s.emit(kind)
	s.skip(isLineSpace)

	if (!s.restHasPrefix("http://") && !s.restHasPrefix("https://")) && !s.restHasPrefix("{{") {
		return s.errorf("expected URL or interpolation, got %q", s.peek())
	}

	return scanURL
}

// scanURL scans a request URL.
func scanURL(s *Scanner) stateFn {
	s.takeWhile(isURL)

	if s.pos > s.start {
		s.emit(token.Text)
	}

	if s.restHasPrefix("{{") {
		s.statePush(scanURL)
		return scanOpenInterp
	}

	s.skip(isLineSpace)

	// Is there a HTTP version?
	if s.restHasPrefix("HTTP/") {
		return scanHTTPVersion
	}

	s.skip(unicode.IsSpace)

	if isIdent(s.peek()) {
		return scanHeader
	}

	s.skip(unicode.IsSpace)

	if s.atEOF() || s.restHasPrefix("###") {
		return scanStart
	}

	next := s.peek()

	if next == '#' || s.restHasPrefix("//") || next == eof || next == utf8.RuneError {
		return scanRequest
	}

	return scanBody
}

// scanHTTPVersion scans a literal 'HTTP/<version>' declaration.
func scanHTTPVersion(s *Scanner) stateFn {
	s.takeExact("HTTP/")

	s.takeWhile(isDigit)

	if s.take(".") {
		// It's e.g 1.2
		s.takeWhile(isDigit)
	}

	s.emit(token.HTTPVersion)

	s.skip(unicode.IsSpace)

	if isIdent(s.peek()) {
		return scanHeader
	}

	if s.atEOF() || s.restHasPrefix("###") {
		return scanStart
	}

	return scanBody
}

// scanHeaderValue scans a header value, including any interpolation.
func scanHeaderValue(s *Scanner) stateFn {
	// Handle interpolation somewhere inside the header value
	// e.g. Authorization: Bearer {{ token }}
	for {
		if s.restHasPrefix("{{") {
			// Emit what we have captured up to this point (if there is anything) as Text and then
			// switch to scanning the interpolation
			if s.pos > s.start {
				// We have absorbed stuff, emit it
				s.emit(token.Text)
			}

			s.statePush(scanHeaderValue)

			return scanOpenInterp
		}

		// Scan any text on the same line
		next := s.peek()
		if next == '\n' || next == eof || next == utf8.RuneError {
			break
		}

		s.next()
	}

	// If we absorbed any text, emit it.
	//
	// This could be empty because the entire header value could have just been an interp
	// e.g. X-Api-Key: {{ key }}
	if s.pos > s.start {
		s.emit(token.Text)
	}

	s.skip(unicode.IsSpace)

	// If there are more headers, go there
	if isAlpha(s.peek()) {
		return scanHeader
	}

	return scanBody
}

// scanHeader scans a request header.
func scanHeader(s *Scanner) stateFn {
	s.skip(unicode.IsSpace)

	if !isAlpha(s.peek()) {
		return scanBody
	}

	if s.atEOF() || s.restHasPrefix("###") {
		return scanStart
	}

	s.takeWhile(isIdent)

	if s.pos > s.start {
		s.emit(token.Header)
	}

	if s.peek() != ':' {
		return s.errorf("invalid header, expected ':', got %q", s.peek())
	}

	s.take(":")
	s.emit(token.Colon)
	s.skip(isLineSpace)

	// Handle interpolation somewhere inside the header value
	// e.g. Authorization: Bearer {{ token }}
	s.statePush(scanHeader)

	return scanHeaderValue
}

// scanBody scans a HTTP request body.
func scanBody(s *Scanner) stateFn {
	if s.restHasPrefix("###") {
		return scanStart
	}

	// Reading the request body from a file
	if s.take("<") {
		return scanLeftAngle
	}

	// Redirecting the response without specifying a body
	// e.g. in a GET request
	if s.take(">") {
		return scanRightAngle
	}

	s.skip(unicode.IsSpace)

	if s.take("#") {
		return scanRequestComment
	}

	if s.peek() != eof {
		s.statePush(scanBody)
		return scanText
	}

	// Are we doing the response reference pattern e.g. '<> response.json'
	if s.take("<") {
		return scanLeftAngle
	}

	// Are we redirecting the response *after* a body has been specified
	// e.g. in a POST request, there may be a body *and* a redirect
	if s.take(">") {
		return scanRightAngle
	}

	return scanStart
}

// scanLeftAngle scans a '<' in the context of reading a request
// body from the filepath specified next.
//
// It assumes the '<' has already been consumed.
func scanLeftAngle(s *Scanner) stateFn {
	if !s.take(">") {
		if s.pos > s.start {
			s.emit(token.LeftAngle)
		}
	} else {
		// It's a response reference '<>'
		s.emit(token.ResponseRef)
	}

	s.skip(isLineSpace)

	if s.restHasPrefix("{{") {
		s.statePush(scanLeftAngle)
		return scanOpenInterp
	}

	if isFilePath(s.peek()) {
		s.takeWhile(isText)
		s.emit(token.Text)
	}

	s.skip(unicode.IsSpace)

	// Is the next thing a response ref '<>', which would be the case
	// if a request has a body file *and* a response ref
	if s.take("<") {
		return scanLeftAngle
	}

	// Are we redirecting the response *after* a body has been specified by a file
	if s.take(">") {
		return scanRightAngle
	}

	// Back to start because this marks the end of the request
	return scanStart
}

// scanRightAngle scans a '>' in the context of redirecting a response body
// to a filepath specified next.
//
// It assumes the '>' has already been consumed.
func scanRightAngle(s *Scanner) stateFn {
	if s.pos > s.start {
		s.emit(token.RightAngle)
	}

	s.skip(isLineSpace)

	if s.restHasPrefix("{{") {
		s.statePush(scanRightAngle)
		return scanOpenInterp
	}

	if isFilePath(s.peek()) {
		s.takeWhile(isText)
		s.emit(token.Text)
	}

	// Back to start because this marks the end of the request
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
	return !unicode.IsSpace(r) && r != eof && r != utf8.RuneError && r != '{'
}

// isURL reports whether r is valid in a URL.
func isURL(r rune) bool {
	if r == eof || r == utf8.RuneError {
		return false
	}

	return isAlphaNumeric(r) || strings.ContainsRune("$-_.+!*'(),:/?#[]@&;=", r)
}

// isFilePath reports whether r could be a valid first character in a filepath.
func isFilePath(r rune) bool {
	return isIdent(r) || r == '.' || r == '/' || r == '\\'
}
