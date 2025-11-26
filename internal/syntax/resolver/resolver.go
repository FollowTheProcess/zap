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

// ErrResolve is a generic resolving error, details on the error are passed
// to the resolver's [resolver.ErrorHandler] at the moment it occurs.
var ErrResolve = errors.New("resolve error")

// ErrorHandler is a resolve error handler, it is called with an [ast.Node] and a message
// at the moment the resolver raises an error.
type ErrorHandler func(node ast.Node, msg string)

// Resolver is the ast resolver for http files.
//
// It transforms an [ast.File] into a concrete [spec.File], evaluating interpolations,
// parsing URLs and durations, and otherwise validating and checking the parse tree
// along the way.
type Resolver struct {
	handler   ErrorHandler // The installed error handler.
	name      string       // The name of the file being resolved.
	hadErrors bool         // Whether we encountered resolver errors, allows multiple to be reported.
}

// New returns a new [Resolver].
func New(name string, handler ErrorHandler) *Resolver {
	return &Resolver{
		handler: handler,
		name:    name,
	}
}

// Resolve resolves an [ast.File] into a concrete [spec.File].
func (r *Resolver) Resolve(in ast.File) (spec.File, error) {
	result := spec.File{
		Name:     in.Name,
		Vars:     make(map[string]string),
		Prompts:  make(map[string]spec.Prompt),
		Requests: []spec.Request{},
	}

	// TODO(@FollowTheProcess): We should do some really nice error reporting here as we have
	// all the position info from the ast, we should probably do similar to what the parser does
	// and have a struct that takes in an error handler.

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

// error triggers a resolve error with a fixed message.
func (r *Resolver) error(node ast.Node, msg string) {
	r.hadErrors = true

	if r.handler == nil {
		return
	}

	r.handler(node, msg)
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
