// Package resolver implements a resolver for the AST.
//
// The resolution stage evaluates interpolations, parses durations and otherwise makes
// the structure concrete, resulting in a [spec.File].
package resolver

import (
	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
)

// ResolveFile resolves an [ast.File] into a concrete [spec.File].
func ResolveFile(in ast.File) (spec.File, error) {
	result := spec.File{
		Name: in.Name,
	}

	return result, nil
}
