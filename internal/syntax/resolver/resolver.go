// Package resolver implements a resolver for the AST.
//
// The resolution stage evaluates interpolations, parses durations and otherwise makes
// the structure concrete, resulting in a [spec.File].
package resolver

import (
	"errors"
	"fmt"

	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
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
	result := spec.File{
		Name:     in.Name,
		Vars:     make(map[string]string),
		Prompts:  make(map[string]spec.Prompt),
		Requests: []spec.Request{},
	}

	for _, statement := range in.Statements {
		switch stmt := statement.(type) {
		case ast.VarStatement:
			key, value, err := r.resolveVarStatement(stmt)
			if err != nil {
				return spec.File{}, ErrResolve
			}

			result.Vars[key] = value

		default:
			return spec.File{}, fmt.Errorf("unhandled ast statement: %T", stmt)
		}
	}

	if r.hadErrors {
		return spec.File{}, ErrResolve
	}

	return result, nil
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

// resolveVarStatement resolves a variable declaration.
func (r *Resolver) resolveVarStatement(statement ast.VarStatement) (key, value string, err error) {
	key = statement.Ident.Name

	value, err = r.resolveExpression(statement.Value)
	if err != nil {
		r.errorf(statement, "failed to resolve value expression for key %s: %v", key, err)
		return "", "", ErrResolve
	}

	return key, value, nil
}

// resolveExpression resolves an [ast.Expression].
func (r *Resolver) resolveExpression(expression ast.Expression) (string, error) {
	switch expr := expression.(type) {
	case ast.TextLiteral:
		return expr.Value, nil
	default:
		return "", fmt.Errorf("unhandled ast expression: %T", expr)
	}
}
