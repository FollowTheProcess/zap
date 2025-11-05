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

	"go.followtheprocess.codes/zap/internal/syntax"
)

// TODO(@FollowTheProcess): When parsing, we build up the half-baked version of File and Request
// and then when evaluating, we resolve into one of these. So the prompts and variables should be evaluated
// during the resolve phase, not during parsing as they are currently, so a few rhings will need to change
// to support this.
//
// Basically the pattern will be the parser parses a raw file into a syntax.File, then we resolve it
// into a spec.File, and that is the canonical representation. We can transform a spec.File into a
// postman collection, a JSON document, a YAML document, a collection of curl snippets etc. and vice
// versa for importing those formats.

// TODO(@FollowTheProcess): Comprehensive tests for resolving files, can do that when I move the eval
// of templates to the resolve stage

// TODO(@FollowTheProcess): Use [http.Header] as the type for headers... duh, kind of obvious
// when I think about it now

// File represents a single .http file as parsed.
//
// Interpolation has been performed on the fly during parsing so this
// file is concrete with variables replaced.
type File struct {
	// Name of the file (or @name in global scope if given)
	Name string `json:"name,omitempty"`

	// Global variables
	Vars map[string]string `json:"vars,omitempty"`

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

// ResolveFile resolves a [syntax.File], returning a [File].
//
// Currently this is a straight copy but I want to have this resolve
// the templating etc. see the TODO at the top of this file.
func ResolveFile(in syntax.File) (File, error) {
	requests, err := resolveRequests(in.Requests)
	if err != nil {
		return File{}, fmt.Errorf("could not resolve request: %w", err)
	}

	out := File{
		Name:              in.Name,
		Vars:              in.Vars,
		Prompts:           resolvePrompts(in.Prompts),
		Requests:          requests,
		Timeout:           in.Timeout,
		ConnectionTimeout: in.ConnectionTimeout,
		NoRedirect:        in.NoRedirect,
	}

	return out, nil
}

// resolvePrompts resolves the syntax prompts into spec prompts.
func resolvePrompts(in map[string]syntax.Prompt) map[string]Prompt {
	prompts := make(map[string]Prompt, len(in))
	for name, prompt := range in {
		prompts[name] = Prompt{
			Name:        prompt.Name,
			Description: prompt.Description,
			Value:       prompt.Value,
		}
	}

	return prompts
}

// resolveRequests resolves the syntax requests into spec requests.
func resolveRequests(in []syntax.Request) ([]Request, error) {
	requests := make([]Request, 0, len(in))
	for _, request := range in {
		requests = append(requests, Request{
			Vars:              request.Vars,
			Headers:           request.Headers,
			Prompts:           resolvePrompts(request.Prompts),
			Name:              request.Name,
			Comment:           request.Comment,
			Method:            request.Method,
			URL:               request.URL,
			HTTPVersion:       request.HTTPVersion,
			BodyFile:          request.BodyFile,
			ResponseFile:      request.ResponseFile,
			ResponseRef:       request.ResponseRef,
			Body:              request.Body,
			Timeout:           request.Timeout,
			ConnectionTimeout: request.ConnectionTimeout,
			NoRedirect:        request.NoRedirect,
		})
	}

	return requests, nil
}
