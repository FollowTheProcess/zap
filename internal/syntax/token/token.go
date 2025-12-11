// Package token provides the set of lexical tokens for a .http file.
package token

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	// HighestPrecedence is the maximum precedence for precedence based parsing.
	HighestPrecedence = 7

	// LowestPrecedence is the lowest precedence.
	LowestPrecedence = 0
)

// Token is a lexical token in a .http file.
type Token struct {
	Kind  Kind // The kind of token this is
	Start int  // Byte offset from the start of the file to the start of this token
	End   int  // Byte offset from the start of the file to the end of this token
}

// String implement [fmt.Stringer] for a [Token].
func (t Token) String() string {
	return fmt.Sprintf("<Token::%s start=%d, end=%d>", t.Kind, t.Start, t.End)
}

// Is reports whether the token is any of the provided [Kind]s.
func (t Token) Is(kinds ...Kind) bool {
	return slices.Contains(kinds, t.Kind)
}

// Precedence returns the precedence of the token, or
// [LowestPrecedence].
func (t Token) Precedence() int {
	switch t.Kind {
	case OpenInterp:
		return HighestPrecedence
	default:
		return LowestPrecedence
	}
}

// Keyword reports whether a string refers to a keyword, returning it's [Kind]
// and true if it is. Otherwise [Ident] and false are returned.
func Keyword(text string) (kind Kind, ok bool) {
	switch text {
	case "name":
		return Name, true
	case "prompt":
		return Prompt, true
	case "timeout":
		return Timeout, true
	case "connection-timeout":
		return ConnectionTimeout, true
	case "no-redirect":
		return NoRedirect, true
	default:
		return Ident, false
	}
}

// Method reports whether a string refers to a HTTP method, returning it's
// [Kind] and true if it is. Otherwise [Text] and false are returned.
func Method(text string) (kind Kind, ok bool) {
	switch text {
	case "GET":
		return MethodGet, true
	case "HEAD":
		return MethodHead, true
	case "POST":
		return MethodPost, true
	case "PUT":
		return MethodPut, true
	case "DELETE":
		return MethodDelete, true
	case "CONNECT":
		return MethodConnect, true
	case "PATCH":
		return MethodPatch, true
	case "OPTIONS":
		return MethodOptions, true
	case "TRACE":
		return MethodTrace, true
	default:
		return Text, false
	}
}

// IsMethod reports whether the given kind is a HTTP Method.
func IsMethod(kind Kind) bool {
	return kind >= MethodGet && kind <= MethodTrace
}

const (
	tokenPrefix = "<Token::"
	tokenSuffix = ">"
)

// Parse parses a Token from the canonical String() format:
//
//	<Token::KIND start=123, end=456>
func Parse(s string) (Token, error) {
	if s == "" {
		return Token{}, errors.New("cannot parse token from empty string")
	}

	if !strings.HasPrefix(s, tokenPrefix) || !strings.HasSuffix(s, tokenSuffix) {
		return Token{}, fmt.Errorf("invalid token format: %q", s)
	}

	// Trim prefix/suffix
	inner := strings.TrimSuffix(strings.TrimPrefix(s, tokenPrefix), tokenSuffix)

	// Expected format inside:
	//   KIND start=123, end=456
	//
	// First split off the KIND part (before the first space)
	firstSpace := strings.IndexByte(inner, ' ')
	if firstSpace == -1 {
		return Token{}, fmt.Errorf("invalid token format (missing fields): %q", s)
	}

	kindStr := inner[:firstSpace]
	rest := inner[firstSpace+1:]

	// Parse Kind
	kind, err := ParseKind(kindStr)
	if err != nil {
		return Token{}, fmt.Errorf("invalid kind %q: %w", kindStr, err)
	}

	// Now parse: start=123, end=456
	// A robust way is to use fmt.Sscanf with exact matching:
	var start, end int

	n, err := fmt.Sscanf(rest, "start=%d, end=%d", &start, &end)
	if err != nil || n != 2 {
		return Token{}, fmt.Errorf("invalid start/end fields in %q: %w", s, err)
	}

	if start < 0 {
		return Token{}, fmt.Errorf("invalid start position (%d), cannot be negative", start)
	}

	if end < 0 {
		return Token{}, fmt.Errorf("invalid end position (%d), cannot be negative", end)
	}

	if end < start {
		return Token{}, fmt.Errorf("invalid start/end positions (%d/%d), end must be >= start", start, end)
	}

	return Token{Kind: kind, Start: start, End: end}, nil
}
