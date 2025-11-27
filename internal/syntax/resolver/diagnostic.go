package resolver

// TODO(@FollowTheProcess): Line would be good here (start line) but we can't get that
// without reading src as all the tokens store are offsets, if tokens had line info we
// could include that in Span

// Diagnostic is a resolver error containing details about the
// source position and context.
type Diagnostic struct {
	File      string `json:"file"`      // The file we're parsing.
	Msg       string `json:"msg"`       // Descriptive message explaining the error.
	Highlight Span   `json:"highlight"` // Span of source to highlight as the issue.
}

// Span is a span of source code.
type Span struct {
	Start int `json:"start"` // Byte offset of the start of the span.
	End   int `json:"end"`   // Byte offset of the end of the span.
}
