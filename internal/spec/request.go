package spec

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// Request is a single HTTP request from a .http file as parsed.
//
// It is *nearly* concrete but may have e.g. variable interpolation to perform, URLs
// may not be valid etc. This is simply a structured version of the as-parsed raw text.
type Request struct {
	// Request scoped variables
	Vars map[string]string `json:"vars,omitempty" toml:"vars,omitempty" yaml:"vars,omitempty"`

	// Request headers, may have variable interpolation in the values but not the keys
	Headers map[string]string `json:"headers,omitempty" toml:"headers,omitempty" yaml:"headers,omitempty"`

	// Request scoped prompts, the user will be asked to provide values for each of these
	// whenever this particular request is invoked.
	//
	// The provided values will then be stored in Vars for future use e.g. as interpolation
	// in the request body.
	Prompts map[string]Prompt `json:"prompts,omitempty" toml:"prompts,omitempty" yaml:"prompts,omitempty"`

	// Optional name, if empty request should be named after it's index e.g. "#1"
	Name string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty"`

	// Optional request comment
	Comment string `json:"comment,omitempty" toml:"comment,omitempty" yaml:"comment,omitempty"`

	// The HTTP method
	Method string `json:"method,omitempty" toml:"method,omitempty" yaml:"method,omitempty"`

	// The complete URL, may have variable interpolation and/or not be a valid URL
	URL string `json:"url,omitempty" toml:"url,omitempty" yaml:"url,omitempty"`

	// Version of the HTTP protocol to use e.g. "1.2"
	HTTPVersion string `json:"httpVersion,omitempty" toml:"httpVersion,omitempty" yaml:"httpVersion,omitempty"`

	// If the body is to be populated by reading a local file, this is the path
	// to that local file (relative to the .http file)
	BodyFile string `json:"bodyFile,omitempty" toml:"bodyFile,omitempty" yaml:"bodyFile,omitempty"`

	// If a response redirect was provided, this is the path to the local file into
	// which to write the response (relative to the .http file)
	ResponseFile string `json:"responseFile,omitempty" toml:"responseFile,omitempty" yaml:"responseFile,omitempty"`

	// If a response reference was provided, this is the path to the local file
	// with which to compare the current response (relative to the .http file)
	ResponseRef string `json:"responseRef,omitempty" toml:"responseRef,omitempty" yaml:"responseRef,omitempty"`

	// Request body, if provided inline. Again, may have variable interpolation still to perform
	Body []byte `json:"body,omitempty" toml:"body,omitempty" yaml:"body,omitempty"`

	// Request scoped timeout, overrides global if set
	Timeout time.Duration `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Request scoped connection timeout, overrides global if set
	ConnectionTimeout time.Duration `json:"connectionTimeout,omitempty" toml:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`

	// Disable following redirects for this request, overrides global if set
	NoRedirect bool `json:"noRedirect,omitempty" toml:"noRedirect,omitempty" yaml:"noRedirect,omitempty"`
}

// String implements [fmt.Stringer] for a [Request] and formats
// the request to be a syntactically valid http request within
// a .http file.
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
