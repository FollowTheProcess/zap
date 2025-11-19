// Package ast defines an abstract syntax tree for the .http grammar.
package ast

import (
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

// Node is the interface for ast nodes.
type Node interface {
	// Start returns the first token associated with the node.
	Start() token.Token

	// End returns the last token associated with the node.
	End() token.Token

	// Kind returns the kind of node this is.
	Kind() Kind
}

// File is an ast [Node] representing a single .http file.
type File struct {
	// Name is the name of the file.
	Name string

	// Statements is the list of ast statements in the file.
	Statements []Statement

	// Type is the type of the node, in this case [KindFile].
	Type Kind
}

// Start returns the first token in a file.
//
// If the file is empty, [token.EOF] is returned.
func (f File) Start() token.Token {
	if len(f.Statements) == 0 {
		return token.Token{Kind: token.EOF}
	}

	return f.Statements[0].Start()
}

// End returns the final token in the file.
func (f File) End() token.Token {
	if len(f.Statements) == 0 {
		return token.Token{Kind: token.EOF}
	}

	return f.Statements[len(f.Statements)-1].End()
}

// Kind returns [KindFile].
func (f File) Kind() Kind {
	return f.Type
}
