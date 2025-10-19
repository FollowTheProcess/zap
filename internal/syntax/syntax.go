// Package syntax handles parsing the raw .http file text into meaningful
// data structures and implements the tokeniser and parser as well as some
// language level integration tests.
package syntax

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"go.followtheprocess.codes/hue"
)

// An ErrorHandler may be provided to parts of the parsing pipeline. If a syntax error is encountered and
// a non-nil handler was provided, it is called with the position info and error message.
type ErrorHandler func(pos Position, msg string)

// Position is an arbitrary source file position including file, line
// and column information. It can also express a range of source via StartCol
// and EndCol, this is useful for error reporting.
//
// Positions without filenames are considered invalid, in the case of stdin
// the string "stdin" may be used.
type Position struct {
	Name     string // Filename
	Offset   int    // Byte offset of the position from the start of the file
	Line     int    // Line number (1 indexed)
	StartCol int    // Start column (1 indexed)
	EndCol   int    // End column (1 indexed), EndCol == StartCol when pointing to a single character
}

// IsValid reports whether the [Position] describes a valid source position.
//
// The rules are:
//
//   - At least Name, Line and StartCol must be set (and non zero)
//   - EndCol cannot be 0, it's only allowed values are StartCol or any number greater than StartCol
func (p Position) IsValid() bool {
	if p.Name == "" || p.Line < 1 || p.StartCol < 1 || p.EndCol < 1 ||
		(p.EndCol >= 1 && p.EndCol < p.StartCol) {
		return false
	}

	return true
}

// String returns a string representation of a [Position].
//
// It is formatted such that most text editors/terminals will be able to support clicking on it
// and navigating to the position.
//
// Depending on which fields are set, the string returned will be different:
//
//   - "file:line:start-end": valid position pointing to a range of text on the line
//   - "file:line:start": valid position pointing to a single character on the line (EndCol == StartCol)
//
// At least Name, Line and StartCol must be present for a valid position, and Line and StarCol must be > 0.
// If not, an error string will be returned.
func (p Position) String() string {
	if !p.IsValid() {
		return fmt.Sprintf(
			"BadPosition: {Name: %q, Line: %d, StartCol: %d, EndCol: %d}",
			p.Name,
			p.Line,
			p.StartCol,
			p.EndCol,
		)
	}

	if p.StartCol == p.EndCol {
		// No range, just a single position
		return fmt.Sprintf("%s:%d:%d", p.Name, p.Line, p.StartCol)
	}

	return fmt.Sprintf("%s:%d:%d-%d", p.Name, p.Line, p.StartCol, p.EndCol)
}

// File represents a single .http file as parsed.
//
// Interpolation has been performed on the fly during parsing so this
// file is concrete with variables replaced.
type File struct {
	// Name of the file (or @name in global scope if given)
	Name string `json:"name,omitempty"`

	// Global variables
	Vars map[string]string `json:"vars,omitempty"`

	// TODO(@FollowTheProcess): I think prompts might need to be a map?
	// A map of a prompt's ident to a struct containing it's description and current value
	// which can either be empty (a signal to the CLI to prompt for it) or have it's
	// real value there

	// Global prompts, the user will be asked to provide values for each of these each time the
	// file is parsed.
	//
	// The provided values will then be stored in Vars.
	Prompts map[string]Prompt `json:"prompts,omitempty"`

	// The HTTP requests described in the file
	Requests []Request `json:"requests,omitempty"`

	// Global timeout for all requests
	Timeout time.Duration `json:"timeout,omitempty"`

	// Global connection timeout for all requests
	ConnectionTimeout time.Duration `json:"connectionTimeout,omitempty"`

	// Disable following redirects globally across all requests
	NoRedirect bool `json:"noRedirect,omitempty"`
}

// String implements [fmt.Stringer] for a [File].
func (f File) String() string {
	builder := &strings.Builder{}

	if f.Name != "" {
		fmt.Fprintf(builder, "@name = %s\n\n", f.Name)
	}

	for _, name := range slices.Sorted(maps.Keys(f.Prompts)) {
		builder.WriteString(f.Prompts[name].String())
	}

	for _, key := range slices.Sorted(maps.Keys(f.Vars)) {
		fmt.Fprintf(builder, "@%s = %s\n", key, f.Vars[key])
	}

	// Only show timeouts if they are non-default
	if f.Timeout != 0 {
		fmt.Fprintf(builder, "@timeout = %s\n", f.Timeout)
	}

	if f.ConnectionTimeout != 0 {
		fmt.Fprintf(builder, "@connection-timeout = %s\n", f.ConnectionTimeout)
	}

	// Same with no-redirect
	if f.NoRedirect {
		fmt.Fprintf(builder, "@no-redirect = %v\n", f.NoRedirect)
	}

	// Separate the request start from the globals by a newline
	builder.WriteByte('\n')

	for _, request := range f.Requests {
		builder.WriteString(request.String())
	}

	return builder.String()
}

// Request is a single HTTP request from a .http file as parsed.
//
// It is *nearly* concrete but may have e.g. variable interpolation to perform, URLs
// may not be valid etc. This is simply a structured version of the as-parsed raw text.
type Request struct {
	// Request scoped variables
	Vars map[string]string `json:"vars,omitempty"`

	// Request headers, may have variable interpolation in the values but not the keys
	Headers map[string]string `json:"headers,omitempty"`

	// Request scoped prompts, the user will be asked to provide values for each of these
	// whenever this particular request is invoked.
	//
	// The provided values will then be stored in Vars for future use e.g. as interpolation
	// in the request body.
	Prompts map[string]Prompt `json:"prompts,omitempty"`

	// Optional name, if empty request should be named after it's index e.g. "#1"
	Name string `json:"name,omitempty"`

	// Optional request comment
	Comment string `json:"comment,omitempty"`

	// The HTTP method
	Method string `json:"method,omitempty"`

	// The complete URL, may have variable interpolation and/or not be a valid URL
	URL string `json:"url,omitempty"`

	// Version of the HTTP protocol to use e.g. "1.2"
	HTTPVersion string `json:"httpVersion,omitempty"`

	// If the body is to be populated by reading a local file, this is the path
	// to that local file (relative to the .http file)
	BodyFile string `json:"bodyFile,omitempty"`

	// If a response redirect was provided, this is the path to the local file into
	// which to write the response (relative to the .http file)
	ResponseFile string `json:"responseFile,omitempty"`

	// If a response reference was provided, this is the path to the local file
	// with which to compare the current response (relative to the .http file)
	ResponseRef string `json:"responseRef,omitempty"`

	// Request body, if provided inline. Again, may have variable interpolation still to perform
	Body []byte `json:"body,omitempty"`

	// Request scoped timeout, overrides global if set
	Timeout time.Duration `json:"timeout,omitempty"`

	// Request scoped connection timeout, overrides global if set
	ConnectionTimeout time.Duration `json:"connectionTimeout,omitempty"`

	// Disable following redirects for this request, overrides global if set
	NoRedirect bool `json:"noRedirect,omitempty"`
}

// String implements [fmt.Stringer] for a [Request].
func (r Request) String() string {
	builder := &strings.Builder{}

	if r.Comment != "" {
		fmt.Fprintf(builder, "### %s\n", r.Comment)
	} else {
		builder.WriteString("###\n")
	}

	if r.Name != "" {
		fmt.Fprintf(builder, "# @name = %s\n", r.Name)
	}

	for _, name := range slices.Sorted(maps.Keys(r.Prompts)) {
		builder.WriteString(r.Prompts[name].String())
	}

	for _, key := range slices.Sorted(maps.Keys(r.Vars)) {
		fmt.Fprintf(builder, "# @%s = %s\n", key, r.Vars[key])
	}

	// Only show timeouts if they are non-default
	if r.Timeout != 0 {
		fmt.Fprintf(builder, "# @timeout = %s\n", r.Timeout)
	}

	if r.ConnectionTimeout != 0 {
		fmt.Fprintf(builder, "# @connection-timeout = %s\n", r.ConnectionTimeout)
	}

	// Same with no-redirect
	if r.NoRedirect {
		fmt.Fprintf(builder, "# @no-redirect = %v\n", r.NoRedirect)
	}

	if r.HTTPVersion != "" {
		fmt.Fprintf(builder, "%s %s %s\n", r.Method, r.URL, r.HTTPVersion)
	} else {
		fmt.Fprintf(builder, "%s %s\n", r.Method, r.URL)
	}

	for _, key := range slices.Sorted(maps.Keys(r.Headers)) {
		fmt.Fprintf(builder, "%s: %s\n", key, r.Headers[key])
	}

	// Separate the body section
	if r.Body != nil || r.BodyFile != "" || r.ResponseFile != "" {
		builder.WriteString("\n")
	}

	if r.BodyFile != "" {
		fmt.Fprintf(builder, "< %s\n", r.BodyFile)
	}

	if r.Body != nil {
		fmt.Fprintf(builder, "%s\n", string(r.Body))
	}

	if r.ResponseFile != "" {
		fmt.Fprintf(builder, "> %s\n", r.ResponseFile)
	}

	if r.ResponseRef != "" {
		fmt.Fprintf(builder, "<> %s\n", r.ResponseRef)
	}

	return builder.String()
}

// Prompt represents a variable that requires the user to specify by responding to a prompt.
type Prompt struct {
	// Name of the variable into which to store the user provided value
	Name string `json:"name,omitempty"`

	// Description of the prompt, optional
	Description string `json:"description,omitempty"`

	// Value is the current value for the prompt variable, empty if
	// not yet provided
	Value string `json:"value,omitempty"`
}

// String implements [fmt.Stringer] for a [Prompt].
func (p Prompt) String() string {
	if p.Description != "" {
		return fmt.Sprintf("@prompt %s %s\n", p.Name, p.Description)
	}

	return fmt.Sprintf("@prompt %s\n", p.Name)
}

// PrettyConsoleHandler returns a [ErrorHandler] that formats the syntax error for
// display on the terminal to a user.
func PrettyConsoleHandler(w io.Writer) ErrorHandler {
	return func(pos Position, msg string) {
		// TODO(@FollowTheProcess): This is a bit better but still some improvement I think
		fmt.Fprintf(w, "%s: %s\n\n", pos, msg)

		contents, err := os.ReadFile(pos.Name)
		if err != nil {
			fmt.Fprintf(w, "unable to show src context: %v\n", err)
			return
		}

		lines := bytes.Split(contents, []byte("\n"))

		const contextLines = 3

		startLine := max(pos.Line-contextLines, 0)
		endLine := min(pos.Line+contextLines, len(lines))

		for i, line := range lines {
			i++ // Lines are 1 indexed
			if i >= startLine && i <= endLine {
				// Note: This is U+2502/"Box Drawings Light Vertical" NOT standard vertical pipe '|'
				margin := fmt.Sprintf("%d │ ", i)
				fmt.Fprintf(w, "%s%s\n", margin, line)

				if i == pos.Line {
					hue.Red.Fprintf(
						w,
						"%s%s\n",
						strings.Repeat(" ", len(margin)+pos.StartCol-1),
						strings.Repeat("─", pos.EndCol-pos.StartCol),
					)
				}
			}
		}
	}
}
