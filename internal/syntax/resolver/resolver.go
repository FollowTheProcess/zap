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

// ResolveFile resolves an [ast.File] into a concrete [spec.File].
func ResolveFile(in ast.File) (spec.File, error) {
	result := spec.File{
		Name:     in.Name,
		Vars:     make(map[string]string),
		Prompts:  make(map[string]spec.Prompt),
		Requests: []spec.Request{},
	}

	// TODO(@FollowTheProcess): We should do some really nice error reporting here as we have
	// all the position info from the ast, we should probably do similar to what the parser does
	// and have a struct that takes in an error handler.

	// TODO(@FollowTheProcess): Rework the syntax ErrorHandler, it should take a range of
	// the entire node that's wrong, with a highlight range pointing to the specific
	// part along with line info

	for _, statement := range in.Statements {
		switch stmt := statement.(type) {
		case ast.VarStatement:
			key, value, err := resolveVarStatement(stmt)
			if err != nil {
				return spec.File{}, errors.New("failed to resolve variable declaration")
			}

			result.Vars[key] = value

		default:
			return spec.File{}, fmt.Errorf("unhandled ast statement: %T", stmt)
		}
	}

	return result, nil
}

// resolveVarStatement resolves a variable declaration.
func resolveVarStatement(statement ast.VarStatement) (key, value string, err error) {
	key = statement.Ident.Name

	value, err = resolveExpression(statement.Value)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve value expression for %s: %w", key, err)
	}

	return key, value, nil
}

// resolveExpression resolves an [ast.Expression].
func resolveExpression(expression ast.Expression) (string, error) {
	switch expr := expression.(type) {
	case ast.TextLiteral:
		return expr.Value, nil
	default:
		return "", fmt.Errorf("unhandled ast expression: %T", expr)
	}
}
