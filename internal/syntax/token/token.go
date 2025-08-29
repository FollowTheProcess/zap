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
	Comment                       // Comment
	Separator                     // Separator
	At                            // At
	Ident                         // Ident
	Eq                            // Eq
	Text                          // Text
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
