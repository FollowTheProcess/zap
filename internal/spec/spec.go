// Package spec provides the Request and File types, the concrete,
// canonical data structures describing a .http file and a HTTP request contained
// inside a file.
//
// Unlike the representations in the syntax package, the data structures here are
// complete and concrete, i.e. all variable interpolation has been performed
// and urls have been resolved.
//
// spec also provides mechanisms for translating HTTP requests and files into
// other formats such as curl snippets, postman collections etc.
package spec

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// File represents a single .http file.
//
// Interpolation has been performed during resolution so this is a concrete representation
// ready to use.
//
// The only exception are global prompts which do not have values yet. During resolution,
// any prompts have had their idents declared in Vars with the value of "zap::prompt::local::<ident>",
// to be replaced when the user is prompted for the values.
type File struct {
	// Name of the file (or @name in global scope if given)
	Name string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty"`

	// Global variables
	Vars map[string]string `json:"vars,omitempty" toml:"vars,omitempty" yaml:"vars,omitempty"`

	// Global prompts, the user will be asked to provide values for each of these each time the
	// file is parsed.
	Prompts map[string]Prompt `json:"prompts,omitempty" toml:"prompts,omitempty" yaml:"prompts,omitempty"`

	// The HTTP requests described in the file.
	Requests []Request `json:"requests,omitempty" toml:"requests,omitempty" yaml:"requests,omitempty"`

	// Global timeout for all requests.
	Timeout time.Duration `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Global connection timeout for all requests.
	ConnectionTimeout time.Duration `json:"connectionTimeout,omitempty" toml:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`

	// Disable following redirects globally across all requests.
	NoRedirect bool `json:"noRedirect,omitempty" toml:"noRedirect,omitempty" yaml:"noRedirect,omitempty"`
}

// String implements [fmt.Stringer] for a [File] and renders
// the file as a canonical .http file format.
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

// ContainsRequest reports whether a request with the given name is present
// in the file.
func (f File) ContainsRequest(name string) bool {
	for _, request := range f.Requests {
		if request.Name == name {
			return true
		}
	}

	return false
}
