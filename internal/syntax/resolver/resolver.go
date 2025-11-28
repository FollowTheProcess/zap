// Package resolver implements a resolver for the AST.
//
// The resolution stage evaluates interpolations, parses durations and otherwise makes
// the structure concrete, resulting in a [spec.File].
package resolver

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

// ErrResolve is a generic resolving error, details on the error are provided through
// a [Diagnostic].
var ErrResolve = errors.New("resolve error")

// Resolver is the ast resolver for http files.
//
// It transforms an [ast.File] into a concrete [spec.File], evaluating interpolations,
// parsing URLs and durations, and otherwise validating and checking the parse tree
// along the way.
type Resolver struct {
	name        string       // The name of the file being resolved.
	diagnostics []Diagnostic // Diagnostics collected during resolving.
	hadErrors   bool         // Whether we encountered resolver errors.
}

// New returns a new [Resolver].
func New(name string) *Resolver {
	return &Resolver{
		name: name,
	}
}

// Resolve resolves an [ast.File] into a concrete [spec.File].
//
// In the presence of an error, Resolve will return [ErrResolve], for more detailed
// inspection of resolution errors, call [Resolver.Diagnostics].
func (r *Resolver) Resolve(in ast.File) (spec.File, error) {
	file := spec.File{
		Name:     in.Name,
		Vars:     make(map[string]string),
		Prompts:  make(map[string]spec.Prompt),
		Requests: []spec.Request{},
	}

	var errs []error

	for _, statement := range in.Statements {
		err := r.resolveFileStatement(&file, statement)
		if err != nil {
			// If we can't resolve this one, try carrying on. This ensures we provide
			// multiple diagnostics for the user rather than one at a time
			errs = append(errs, err)
			continue
		}
	}

	// We've had diagnostics reported during resolving so just bubble up a top level error
	if r.hadErrors {
		return spec.File{}, fmt.Errorf("%w: %w", ErrResolve, errors.Join(errs...))
	}

	return file, nil
}

// Diagnostics returns the diagnostics gathered during resolving.
func (r *Resolver) Diagnostics() []Diagnostic {
	return r.diagnostics
}

// error reports a resolve error with a fixed message.
func (r *Resolver) error(node ast.Node, msg string) {
	r.hadErrors = true

	diag := Diagnostic{
		File:      r.name,
		Msg:       msg,
		Highlight: Span{Start: node.Start().Start, End: node.End().End},
	}

	r.diagnostics = append(r.diagnostics, diag)
}

// errorf calls error with a formatted message.
func (r *Resolver) errorf(node ast.Node, format string, a ...any) {
	r.error(node, fmt.Sprintf(format, a...))
}

// resolveStatement resolves a generic [ast.Statement], adding the fields to
// the file as it goes.
//
// The file passed in is mutated only in the happy path, if err != nil, the file is
// left untouched.
func (r *Resolver) resolveFileStatement(file *spec.File, statement ast.Statement) error {
	switch stmt := statement.(type) {
	case ast.VarStatement:
		err := r.resolveGlobalVarStatement(file, stmt)
		if err != nil {
			return err
		}
	case ast.PromptStatement:
		err := r.resolveGlobalPromptStatement(file, stmt)
		if err != nil {
			return err
		}
	case ast.Request:
		request, err := r.resolveRequestStatement(stmt)
		if err != nil {
			return err
		}

		// If it doesn't have a name set, give it a numerical name based
		// on it's position in the file e.g. "#1", "#2" etc.
		if request.Name == "" {
			request.Name = fmt.Sprintf("#%d", len(file.Requests)+1)
		}

		file.Requests = append(file.Requests, request)

	default:
		return fmt.Errorf("unexpected global statement: %T", stmt)
	}

	return nil
}

// resolveGlobalVarStatement resolves a variable declaration in the global scope, storing it in the file
// passed to it.
//
// The file is only mutated in the happy path.
func (r *Resolver) resolveGlobalVarStatement(file *spec.File, statement ast.VarStatement) error {
	key := statement.Ident.Name

	kind, isKeyword := token.Keyword(key)
	if isKeyword && kind == token.NoRedirect {
		// @no-redirect has no value expression, simply setting it is enough
		file.NoRedirect = true
		return nil
	}

	value, err := r.resolveExpression(statement.Value)
	if err != nil {
		r.errorf(statement, "failed to resolve value expression for key %s: %v", key, err)
		return err
	}

	if !isKeyword {
		// Normal var
		if file.Vars == nil {
			file.Vars = make(map[string]string)
		}

		file.Vars[key] = value

		return nil
	}

	// Otherwise, handle the specific keyword by setting the right field
	switch kind {
	case token.Name:
		file.Name = value
	case token.Timeout:
		duration, err := time.ParseDuration(value)
		if err != nil {
			r.errorf(statement.Value, "invalid timeout value: %v", err)
			return err
		}

		file.Timeout = duration
	case token.ConnectionTimeout:
		duration, err := time.ParseDuration(value)
		if err != nil {
			r.errorf(statement.Value, "invalid connection-timeout value: %v", err)
			return err
		}

		file.ConnectionTimeout = duration
	default:
		return fmt.Errorf("unhandled keyword: %s", kind)
	}

	return nil
}

// resolveGlobalPromptStatement resolves a top level file @prompt statement and
// adds it to the file, returning the new file containing the prompt.
func (r *Resolver) resolveGlobalPromptStatement(file *spec.File, statement ast.PromptStatement) error {
	name := statement.Ident.Name

	prompt := spec.Prompt{
		Name:        name,
		Description: statement.Description.Value,
	}

	if _, exists := file.Prompts[name]; exists {
		r.errorf(statement, "prompt %s already declared", name)
		return fmt.Errorf("prompt %s already declared", name)
	}

	// Shouldn't need this because file is declared top level with all this
	// initialised but let's not panic if we can help it
	if file.Prompts == nil {
		file.Prompts = make(map[string]spec.Prompt)
	}

	file.Prompts[statement.Ident.Name] = prompt

	return nil
}

// resolveRequestStatement resolves an [ast.Request] into a [spec.Request].
func (r *Resolver) resolveRequestStatement(in ast.Request) (spec.Request, error) {
	rawURL, err := r.resolveExpression(in.URL)
	if err != nil {
		r.errorf(in.URL, "failed to resolve URL expression: %v", err)
		return spec.Request{}, err
	}

	// TODO(@FollowTheProcess): Should the spec.Request store the URL as *url.URL?
	//
	// This is probably one to change once parser v2 has been swapped in

	// Validate the URL here
	_, err = url.ParseRequestURI(rawURL)
	if err != nil {
		r.errorf(in.URL, "invalid URL %s: %v", rawURL, err)
		return spec.Request{}, err
	}

	method, err := r.resolveHTTPMethod(in.Method)
	if err != nil {
		return spec.Request{}, err
	}

	request := spec.Request{
		Method: method,
		URL:    rawURL,
	}

	var errs []error

	for _, varStatement := range in.Vars {
		err = r.resolveRequestVarStatement(&request, varStatement)
		if err != nil {
			// So we can report as many diagnostics in one pass as possible
			errs = append(errs, err)
			continue
		}
	}

	for _, promptStatement := range in.Prompts {
		err = r.resolveRequestPromptStatement(&request, promptStatement)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	err = errors.Join(errs...)
	if err != nil {
		return spec.Request{}, err
	}

	return request, nil
}

// resolveExpression resolves an [ast.Expression].
func (r *Resolver) resolveExpression(expression ast.Expression) (string, error) {
	switch expr := expression.(type) {
	case ast.TextLiteral:
		return expr.Value, nil
	case ast.URL:
		return expr.Value, nil
	default:
		return "", fmt.Errorf("unhandled ast expression: %T", expr)
	}
}

// resolveHTTPMethod resolves an [ast.Method].
func (r *Resolver) resolveHTTPMethod(method ast.Method) (string, error) {
	switch method.Token.Kind {
	case token.MethodGet:
		return http.MethodGet, nil
	case token.MethodHead:
		return http.MethodHead, nil
	case token.MethodPost:
		return http.MethodPost, nil
	case token.MethodPut:
		return http.MethodPut, nil
	case token.MethodDelete:
		return http.MethodDelete, nil
	case token.MethodConnect:
		return http.MethodConnect, nil
	case token.MethodPatch:
		return http.MethodPatch, nil
	case token.MethodOptions:
		return http.MethodOptions, nil
	case token.MethodTrace:
		return http.MethodTrace, nil
	default:
		r.error(method, "invalid HTTP method")
		return "", errors.New("invalid HTTP method")
	}
}

// resolveRequestVarStatement resolves a variable declaration in the request scope,
// storing it in the request, mutating it in place.
//
// The request is only mutated in the happy path.
func (r *Resolver) resolveRequestVarStatement(request *spec.Request, statement ast.VarStatement) error {
	key := statement.Ident.Name

	kind, isKeyword := token.Keyword(key)
	if isKeyword && kind == token.NoRedirect {
		// @no-redirect has no value expression, simply setting it is enough
		request.NoRedirect = true
		return nil
	}

	value, err := r.resolveExpression(statement.Value)
	if err != nil {
		r.errorf(statement, "failed to resolve value expression for key %s: %v", key, err)
		return err
	}

	if !isKeyword {
		// Normal var
		if request.Vars == nil {
			request.Vars = make(map[string]string)
		}

		request.Vars[key] = value

		return nil
	}

	// Otherwise, handle the specific keyword by setting the right field
	switch kind {
	case token.Name:
		request.Name = value
	case token.Timeout:
		duration, err := time.ParseDuration(value)
		if err != nil {
			r.errorf(statement.Value, "invalid timeout value: %v", err)
			return err
		}

		request.Timeout = duration
	case token.ConnectionTimeout:
		duration, err := time.ParseDuration(value)
		if err != nil {
			r.errorf(statement.Value, "invalid connection-timeout value: %v", err)
			return err
		}

		request.ConnectionTimeout = duration
	default:
		return fmt.Errorf("unhandled keyword: %s", kind)
	}

	return nil
}

// resolveRequestPromptStatement resolves a request level @prompt statement and
// adds it to the request, returning the new request containing the prompt.
func (r *Resolver) resolveRequestPromptStatement(request *spec.Request, statement ast.PromptStatement) error {
	name := statement.Ident.Name

	prompt := spec.Prompt{
		Name:        name,
		Description: statement.Description.Value,
	}

	if _, exists := request.Prompts[name]; exists {
		r.errorf(statement, "prompt %s already declared", name)
		return fmt.Errorf("prompt %s already declared", name)
	}

	// Shouldn't need this because request is declared top level with all this
	// initialised but let's not panic if we can help it
	if request.Prompts == nil {
		request.Prompts = make(map[string]spec.Prompt)
	}

	request.Prompts[statement.Ident.Name] = prompt

	return nil
}
