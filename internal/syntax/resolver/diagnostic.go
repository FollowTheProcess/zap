package resolver

// Diagnostic is a resolver error containing details about the
// source position and context.
type Diagnostic struct {
	File      string // The file we're parsing.
	Msg       string // Descriptive message explaining the error.
	Highlight Span   // Span of source to highlight as the issue.
}

// Span is a span of source code.
type Span struct {
	Start int // Byte offset of the start of the span.
	End   int // Byte offset of the end of the span.
}
