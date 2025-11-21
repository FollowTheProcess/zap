package ast_test

import (
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

func TestNode(t *testing.T) {
	tests := []struct {
		node  ast.Node    // Node under test
		name  string      // Name of the test case
		start token.Token // Expected start token
		end   token.Token // Expected end token
		kind  ast.Kind    // Expected node kind
	}{
		{
			name: "text literal",
			node: ast.TextLiteral{
				Value: "a piece of text",
				Token: token.Token{Kind: token.Text, Start: 0, End: 15},
				Type:  ast.KindTextLiteral,
			},
			start: token.Token{Kind: token.Text, Start: 0, End: 15},
			end:   token.Token{Kind: token.Text, Start: 0, End: 15},
			kind:  ast.KindTextLiteral,
		},
		{
			name: "ident",
			node: ast.Ident{
				Name:  "test",
				Token: token.Token{Kind: token.Ident, Start: 0, End: 4},
				Type:  ast.KindIdent,
			},
			start: token.Token{Kind: token.Ident, Start: 0, End: 4},
			end:   token.Token{Kind: token.Ident, Start: 0, End: 4},
			kind:  ast.KindIdent,
		},
		{
			name: "var statement",
			// @variable = sometext
			node: ast.VarStatement{
				Value: ast.TextLiteral{
					Value: "some text",
					Token: token.Token{Kind: token.Text, Start: 12, End: 8},
					Type:  ast.KindTextLiteral,
				},
				Ident: ast.Ident{
					Name:  "variable",
					Token: token.Token{Kind: token.Ident, Start: 1, End: 9},
					Type:  ast.KindIdent,
				},
				At:   token.Token{Kind: token.At, Start: 0, End: 1},
				Type: ast.KindVarStatement,
			},
			start: token.Token{Kind: token.At, Start: 0, End: 1},
			end:   token.Token{Kind: token.Text, Start: 12, End: 8},
			kind:  ast.KindVarStatement,
		},
		{
			name: "interp",
			// {{ hello }}
			node: ast.Interp{
				Expr: ast.Ident{
					Name:  "hello",
					Token: token.Token{Kind: token.Ident, Start: 3, End: 8},
					Type:  ast.KindIdent,
				},
				Open:  token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
				Close: token.Token{Kind: token.CloseInterp, Start: 9, End: 11},
				Type:  ast.KindInterp,
			},
			start: token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
			end:   token.Token{Kind: token.CloseInterp, Start: 9, End: 11},
			kind:  ast.KindInterp,
		},
		{
			name: "prompt",
			node: ast.PromptStatement{
				Ident: ast.Ident{
					Name:  "id",
					Token: token.Token{Kind: token.Ident, Start: 8, End: 10},
					Type:  ast.KindIdent,
				},
				Description: ast.TextLiteral{
					Value: "User ID",
					Token: token.Token{Kind: token.Text, Start: 11, End: 18},
					Type:  ast.KindTextLiteral,
				},
				At:   token.Token{Kind: token.At, Start: 0, End: 1},
				Type: ast.KindPrompt,
			},
			start: token.Token{Kind: token.At, Start: 0, End: 1},
			end:   token.Token{Kind: token.Text, Start: 11, End: 18},
			kind:  ast.KindPrompt,
		},
		{
			name: "prompt no description",
			node: ast.PromptStatement{
				Ident: ast.Ident{
					Name:  "id",
					Token: token.Token{Kind: token.Ident, Start: 8, End: 10},
					Type:  ast.KindIdent,
				},
				At:   token.Token{Kind: token.At, Start: 0, End: 1},
				Type: ast.KindPrompt,
			},
			start: token.Token{Kind: token.At, Start: 0, End: 1},
			end:   token.Token{Kind: token.Ident, Start: 8, End: 10}, // End returns the ident
			kind:  ast.KindPrompt,
		},
		{
			name: "comment",
			node: ast.Comment{
				Token: token.Token{Kind: token.Comment, Start: 12, End: 26},
				Type:  ast.KindComment,
			},
			start: token.Token{Kind: token.Comment, Start: 12, End: 26},
			end:   token.Token{Kind: token.Comment, Start: 12, End: 26},
			kind:  ast.KindComment,
		},
		{
			name: "file",
			node: ast.File{
				Name: "test.http",
				Statements: []ast.Statement{
					// @variable = sometext
					// @other = moretext
					ast.VarStatement{
						Value: ast.TextLiteral{
							Value: "sometext",
							Token: token.Token{Kind: token.Text, Start: 12, End: 20},
							Type:  ast.KindTextLiteral,
						},
						Ident: ast.Ident{
							Name:  "variable",
							Token: token.Token{Kind: token.Ident, Start: 1, End: 9},
							Type:  ast.KindIdent,
						},
						At:   token.Token{Kind: token.At, Start: 0, End: 1},
						Type: ast.KindVarStatement,
					},
					ast.VarStatement{
						Value: ast.TextLiteral{
							Value: "moretext",
							Token: token.Token{Kind: token.Text, Start: 22, End: 30},
							Type:  ast.KindTextLiteral,
						},
						Ident: ast.Ident{
							Name:  "other",
							Token: token.Token{Kind: token.Ident, Start: 14, End: 19},
							Type:  ast.KindIdent,
						},
						At:   token.Token{Kind: token.At, Start: 13, End: 14},
						Type: ast.KindVarStatement,
					},
				},
				Type: ast.KindFile,
			},
			start: token.Token{Kind: token.At, Start: 0, End: 1},
			end:   token.Token{Kind: token.Text, Start: 22, End: 30},
			kind:  ast.KindFile,
		},
		{
			name:  "empty file",
			node:  ast.File{Type: ast.KindFile},
			start: token.Token{Kind: token.EOF},
			end:   token.Token{Kind: token.EOF},
			kind:  ast.KindFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.Equal(t, tt.node.Start(), tt.start, test.Context("wrong start token"))
			test.Equal(t, tt.node.End(), tt.end, test.Context("wrong end token"))
			test.Equal(t, tt.node.Kind(), tt.kind, test.Context("wrong node kind"))
		})
	}
}
