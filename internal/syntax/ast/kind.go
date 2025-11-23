package ast

// Kind is the type of an ast Node.
type Kind int

// AST Node kinds.
//
//go:generate stringer -type Kind -linecomment
const (
	KindInvalid                Kind = iota // Invalid
	KindFile                               // File
	KindVarStatement                       // VarStatement
	KindIdent                              // Ident
	KindTextLiteral                        // TextLiteral
	KindURL                                // URL
	KindInterp                             // Interp
	KindPrompt                             // Prompt
	KindRequest                            // Request
	KindComment                            // Comment
	KindMethod                             // Method
	KindHeader                             // Header
	KindInterpolatedExpression             // InterpolatedExpression
)

// MarshalText implements [encoding.TextMarshaler] for [Kind].
func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}
