// Package token provides the set of lexical tokens for a .http file.
package token

import (
	"fmt"
	"slices"
)

// Kind is the kind of a token.
type Kind int

//go:generate stringer -type Kind -linecomment
const (
	EOF               Kind = iota // EOF
	Error                         // Error
	Separator                     // Separator
	Comment                       // Comment
	Text                          // Text
	URL                           // URL
	Ident                         // Ident
	At                            // At
	Eq                            // Eq
	Colon                         // Colon
	LeftAngle                     // LeftAngle
	RightAngle                    // RightAngle
	HTTPVersion                   // HTTPVersion
	Header                        // Header
	Body                          // Body
	MethodGet                     // MethodGet
	MethodHead                    // MethodHead
	MethodPost                    // MethodPost
	MethodPut                     // MethodPut
	MethodDelete                  // MethodDelete
	MethodConnect                 // MethodConnect
	MethodPatch                   // MethodPatch
	MethodOptions                 // MethodOptions
	MethodTrace                   // MethodTrace
	Name                          // Name
	Prompt                        // Prompt
	Timeout                       // Timeout
	ConnectionTimeout             // ConnectionTimeout
	NoRedirect                    // NoRedirect
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
