package token

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
	Colon                         // Colon
	LeftAngle                     // LeftAngle
	RightAngle                    // RightAngle
	ResponseRef                   // ResponseRef
	Text                          // Text
	Body                          // Body
	URL                           // URL
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
