package token

import "fmt"

// Kind is the kind of a token.
type Kind int

// Token definitions.
//
//go:generate stringer -type Kind -linecomment
const (
	EOF               Kind = iota // EOF
	Error                         // Error
	Comment                       // Comment
	Separator                     // Separator
	At                            // At
	Ident                         // Ident
	Eq                            // Eq
	Dollar                        // Dollar
	Colon                         // Colon
	LeftAngle                     // LeftAngle
	RightAngle                    // RightAngle
	ResponseRef                   // ResponseRef
	Text                          // Text
	Body                          // Body
	HTTPVersion                   // HTTPVersion
	Header                        // Header
	OpenInterp                    // OpenInterp
	CloseInterp                   // CloseInterp
	Name                          // Name
	Prompt                        // Prompt
	Timeout                       // Timeout
	ConnectionTimeout             // ConnectionTimeout
	NoRedirect                    // NoRedirect
	MethodGet                     // MethodGet
	MethodHead                    // MethodHead
	MethodPost                    // MethodPost
	MethodPut                     // MethodPut
	MethodDelete                  // MethodDelete
	MethodConnect                 // MethodConnect
	MethodPatch                   // MethodPatch
	MethodOptions                 // MethodOptions
	MethodTrace                   // MethodTrace
)

// MarshalText implements [encoding.TextMarshaler] for [Kind].
func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

//nolint:gochecknoglobals // This is okay.
var kindByString map[string]Kind

//nolint:gochecknoinits // Simplest way of doing this.
func init() {
	kindByString = make(map[string]Kind)

	// Populate reverse lookup table using stringer-generated names
	for k := EOF; k <= MethodTrace; k++ {
		kindByString[k.String()] = k
	}
}

// ParseKind parses a [Kind] from it's canonical string representation.
func ParseKind(s string) (Kind, error) {
	k, ok := kindByString[s]
	if !ok {
		return EOF, fmt.Errorf("unknown kind: %q", s)
	}

	return k, nil
}
