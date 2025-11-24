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
			name:  "empty file",
			node:  ast.File{Type: ast.KindFile},
			start: token.Token{Kind: token.EOF},
			end:   token.Token{Kind: token.EOF},
			kind:  ast.KindFile,
		},
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
			name: "url literal",
			node: ast.URL{
				Value: "https://example.com",
				Token: token.Token{Kind: token.URL, Start: 0, End: 19},
				Type:  ast.KindURL,
			},
			start: token.Token{Kind: token.URL, Start: 0, End: 19},
			end:   token.Token{Kind: token.URL, Start: 0, End: 19},
			kind:  ast.KindURL,
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
			name: "var statement no value",
			// @variable
			node: ast.VarStatement{
				Value: nil,
				Ident: ast.Ident{
					Name:  "variable",
					Token: token.Token{Kind: token.Ident, Start: 1, End: 9},
					Type:  ast.KindIdent,
				},
				At:   token.Token{Kind: token.At, Start: 0, End: 1},
				Type: ast.KindVarStatement,
			},
			start: token.Token{Kind: token.At, Start: 0, End: 1},
			end:   token.Token{Kind: token.Ident, Start: 1, End: 9},
			kind:  ast.KindVarStatement,
		},
		{
			name: "interp only",
			// {{ hello }}
			node: ast.InterpolatedExpression{
				Left:  nil,
				Right: nil,
				Interp: ast.Interp{
					Expr: ast.Ident{
						Name:  "hello",
						Token: token.Token{Kind: token.Ident, Start: 3, End: 8},
						Type:  ast.KindIdent,
					},
					Open:  token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
					Close: token.Token{Kind: token.CloseInterp, Start: 9, End: 11},
					Type:  ast.KindInterp,
				},
				Type: ast.KindInterpolatedExpression,
			},
			start: token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
			end:   token.Token{Kind: token.CloseInterp, Start: 9, End: 11},
			kind:  ast.KindInterpolatedExpression,
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
				Text:  "a comment",
			},
			start: token.Token{Kind: token.Comment, Start: 12, End: 26},
			end:   token.Token{Kind: token.Comment, Start: 12, End: 26},
			kind:  ast.KindComment,
		},
		{
			name: "method",
			node: ast.Method{
				Token: token.Token{Kind: token.MethodGet, Start: 0, End: 3},
				Type:  ast.KindMethod,
			},
			start: token.Token{Kind: token.MethodGet, Start: 0, End: 3},
			end:   token.Token{Kind: token.MethodGet, Start: 0, End: 3},
			kind:  ast.KindMethod,
		},
		{
			name: "header",
			node: ast.Header{
				Value: ast.TextLiteral{
					Value: "application/json",
					Token: token.Token{Kind: token.Text, Start: 14, End: 30},
					Type:  ast.KindTextLiteral,
				},
				Key:   "Content-Type",
				Token: token.Token{Kind: token.Header, Start: 0, End: 12},
				Type:  ast.KindHeader,
			},
			start: token.Token{Kind: token.Header, Start: 0, End: 12},
			end:   token.Token{Kind: token.Text, Start: 14, End: 30},
			kind:  ast.KindHeader,
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
			name: "request",
			node: ast.Request{
				Body: ast.Body{
					Token: token.Token{Kind: token.Body, Start: 30, End: 110},
					Type:  ast.KindBody,
				},
				URL: ast.TextLiteral{
					Value: "https://example.com",
					Token: token.Token{Kind: token.URL, Start: 9, End: 28},
					Type:  ast.KindTextLiteral,
				},
				Method: ast.Method{
					Token: token.Token{Kind: token.MethodGet, Start: 5, End: 8},
					Type:  ast.KindMethod,
				},
				Sep:  token.Token{Kind: token.Separator, Start: 0, End: 3},
				Type: ast.KindRequest,
			},
			start: token.Token{Kind: token.Separator, Start: 0, End: 3},
			end:   token.Token{Kind: token.Body, Start: 30, End: 110},
			kind:  ast.KindRequest,
		},
		{
			name: "request no body",
			node: ast.Request{
				Body: nil,
				URL: ast.TextLiteral{
					Value: "https://example.com",
					Token: token.Token{Kind: token.URL, Start: 9, End: 28},
					Type:  ast.KindTextLiteral,
				},
				Method: ast.Method{
					Token: token.Token{Kind: token.MethodGet, Start: 5, End: 8},
					Type:  ast.KindMethod,
				},
				Sep:  token.Token{Kind: token.Separator, Start: 0, End: 3},
				Type: ast.KindRequest,
			},
			start: token.Token{Kind: token.Separator, Start: 0, End: 3},
			end:   token.Token{Kind: token.URL, Start: 9, End: 28},
			kind:  ast.KindRequest,
		},
		{
			name: "request no body or url",
			node: ast.Request{
				Body: nil,
				URL:  nil,
				Method: ast.Method{
					Token: token.Token{Kind: token.MethodGet, Start: 5, End: 8},
					Type:  ast.KindMethod,
				},
				Sep:  token.Token{Kind: token.Separator, Start: 0, End: 3},
				Type: ast.KindRequest,
			},
			start: token.Token{Kind: token.Separator, Start: 0, End: 3},
			end:   token.Token{Kind: token.MethodGet, Start: 5, End: 8},
			kind:  ast.KindRequest,
		},
		{
			name: "request response redirect",
			node: ast.Request{
				Body: ast.Body{
					Token: token.Token{Kind: token.Body, Start: 30, End: 110},
					Type:  ast.KindBody,
				},
				ResponseRedirect: &ast.ResponseRedirect{
					File: ast.TextLiteral{
						Value: "response.json",
						Token: token.Token{Kind: token.Text, Start: 114, End: 127},
						Type:  ast.KindTextLiteral,
					},
					Token: token.Token{Kind: token.RightAngle, Start: 111, End: 112},
					Type:  ast.KindResponseRedirect,
				},
				URL: ast.TextLiteral{
					Value: "https://example.com",
					Token: token.Token{Kind: token.URL, Start: 9, End: 28},
					Type:  ast.KindTextLiteral,
				},
				Method: ast.Method{
					Token: token.Token{Kind: token.MethodGet, Start: 5, End: 8},
					Type:  ast.KindMethod,
				},
				Sep:  token.Token{Kind: token.Separator, Start: 0, End: 3},
				Type: ast.KindRequest,
			},
			start: token.Token{Kind: token.Separator, Start: 0, End: 3},
			end:   token.Token{Kind: token.Text, Start: 114, End: 127},
			kind:  ast.KindRequest,
		},
		{
			name: "request response reference",
			node: ast.Request{
				Body: ast.Body{
					Token: token.Token{Kind: token.Body, Start: 30, End: 110},
					Type:  ast.KindBody,
				},
				ResponseReference: &ast.ResponseReference{
					File: ast.TextLiteral{
						Value: "response.json",
						Token: token.Token{Kind: token.Text, Start: 115, End: 128},
						Type:  ast.KindTextLiteral,
					},
					Token: token.Token{Kind: token.ResponseRef, Start: 111, End: 113},
					Type:  ast.KindResponseReference,
				},
				URL: ast.TextLiteral{
					Value: "https://example.com",
					Token: token.Token{Kind: token.URL, Start: 9, End: 28},
					Type:  ast.KindTextLiteral,
				},
				Method: ast.Method{
					Token: token.Token{Kind: token.MethodGet, Start: 5, End: 8},
					Type:  ast.KindMethod,
				},
				Sep:  token.Token{Kind: token.Separator, Start: 0, End: 3},
				Type: ast.KindRequest,
			},
			start: token.Token{Kind: token.Separator, Start: 0, End: 3},
			end:   token.Token{Kind: token.Text, Start: 115, End: 128},
			kind:  ast.KindRequest,
		},
		{
			name: "inner interp",
			node: ast.Interp{
				Expr: ast.Ident{
					Name:  "id",
					Token: token.Token{Kind: token.Ident, Start: 3, End: 5},
					Type:  ast.KindIdent,
				},
				Open:  token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
				Close: token.Token{Kind: token.CloseInterp, Start: 6, End: 8},
				Type:  ast.KindInterp,
			},
			start: token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
			end:   token.Token{Kind: token.CloseInterp, Start: 6, End: 8},
			kind:  ast.KindInterp,
		},
		{
			// https://example/com/{{ version }}/items/123
			// |------ left ------|-- interp --|- right -|
			name: "full interp",
			node: ast.InterpolatedExpression{
				Left: ast.URL{
					Value: "https://example.com/",
					Token: token.Token{Kind: token.URL, Start: 0, End: 20},
					Type:  ast.KindURL,
				},
				Interp: ast.Interp{
					Expr: ast.Ident{
						Name:  "version",
						Token: token.Token{Kind: token.Ident, Start: 23, End: 30},
						Type:  ast.KindIdent,
					},
					Open:  token.Token{Kind: token.OpenInterp, Start: 20, End: 22},
					Close: token.Token{Kind: token.CloseInterp, Start: 31, End: 33},
					Type:  ast.KindInterp,
				},
				Right: ast.URL{
					Value: "/items/123",
					Token: token.Token{Kind: token.URL, Start: 33, End: 43},
					Type:  ast.KindURL,
				},
				Type: ast.KindInterpolatedExpression,
			},
			start: token.Token{Kind: token.URL, Start: 0, End: 20},
			end:   token.Token{Kind: token.URL, Start: 33, End: 43},
			kind:  ast.KindInterpolatedExpression,
		},
		{
			// {{ baseURL }}/items/3
			// |- interp --|-right-|
			name: "interp no left",
			node: ast.InterpolatedExpression{
				Left: nil,
				Interp: ast.Interp{
					Expr: ast.Ident{
						Name:  "baseURL",
						Token: token.Token{Kind: token.Ident, Start: 3, End: 10},
						Type:  ast.KindIdent,
					},
					Open:  token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
					Close: token.Token{Kind: token.CloseInterp, Start: 11, End: 13},
					Type:  ast.KindInterp,
				},
				Right: ast.URL{
					Value: "/items/3",
					Token: token.Token{Kind: token.URL, Start: 13, End: 21},
					Type:  ast.KindURL,
				},
				Type: ast.KindInterpolatedExpression,
			},
			start: token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
			end:   token.Token{Kind: token.URL, Start: 13, End: 21},
			kind:  ast.KindInterpolatedExpression,
		},
		{
			// https://example/com/{{ endpoint }}
			// |------ left ------|-- interp ---|
			name: "interp no right",
			node: ast.InterpolatedExpression{
				Left: ast.URL{
					Value: "https://example.com/",
					Token: token.Token{Kind: token.URL, Start: 0, End: 20},
					Type:  ast.KindURL,
				},
				Interp: ast.Interp{
					Expr: ast.Ident{
						Name:  "endpoint",
						Token: token.Token{Kind: token.Ident, Start: 23, End: 31},
						Type:  ast.KindIdent,
					},
					Open:  token.Token{Kind: token.OpenInterp, Start: 20, End: 22},
					Close: token.Token{Kind: token.CloseInterp, Start: 32, End: 34},
					Type:  ast.KindInterp,
				},
				Right: nil,
				Type:  ast.KindInterpolatedExpression,
			},
			start: token.Token{Kind: token.URL, Start: 0, End: 20},
			end:   token.Token{Kind: token.CloseInterp, Start: 32, End: 34},
			kind:  ast.KindInterpolatedExpression,
		},
		{
			name: "interp no left or right",
			// {{ hello }}
			// |- interp-|
			node: ast.InterpolatedExpression{
				Left:  nil,
				Right: nil,
				Interp: ast.Interp{
					Expr: ast.Ident{
						Name:  "hello",
						Token: token.Token{Kind: token.Ident, Start: 3, End: 8},
						Type:  ast.KindIdent,
					},
					Open:  token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
					Close: token.Token{Kind: token.CloseInterp, Start: 9, End: 11},
					Type:  ast.KindInterp,
				},
				Type: ast.KindInterpolatedExpression,
			},
			start: token.Token{Kind: token.OpenInterp, Start: 0, End: 2},
			end:   token.Token{Kind: token.CloseInterp, Start: 9, End: 11},
			kind:  ast.KindInterpolatedExpression,
		},
		{
			name: "body",
			node: ast.Body{
				Token: token.Token{Kind: token.Body, Start: 12, End: 136},
				Type:  ast.KindBody,
			},
			start: token.Token{Kind: token.Body, Start: 12, End: 136},
			end:   token.Token{Kind: token.Body, Start: 12, End: 136},
			kind:  ast.KindBody,
		},
		{
			name: "body file",
			node: ast.BodyFile{
				Token: token.Token{Kind: token.LeftAngle, Start: 31, End: 32},
				Value: ast.TextLiteral{
					Value: "./body.json",
					Token: token.Token{Kind: token.Text, Start: 33, End: 44},
					Type:  ast.KindTextLiteral,
				},
				Type: ast.KindBodyFile,
			},
			start: token.Token{Kind: token.LeftAngle, Start: 31, End: 32},
			end:   token.Token{Kind: token.Text, Start: 33, End: 44},
			kind:  ast.KindBodyFile,
		},
		{
			name: "body file no value",
			node: ast.BodyFile{
				Token: token.Token{Kind: token.LeftAngle, Start: 31, End: 32},
				Value: nil,
				Type:  ast.KindBodyFile,
			},
			start: token.Token{Kind: token.LeftAngle, Start: 31, End: 32},
			end:   token.Token{Kind: token.LeftAngle, Start: 31, End: 32},
			kind:  ast.KindBodyFile,
		},
		{
			name: "response redirect",
			node: ast.ResponseRedirect{
				File: ast.TextLiteral{
					Value: "response.json",
					Token: token.Token{Kind: token.Text, Start: 34, End: 47},
					Type:  ast.KindTextLiteral,
				},
				Type:  ast.KindResponseRedirect,
				Token: token.Token{Kind: token.RightAngle, Start: 31, End: 32},
			},
			start: token.Token{Kind: token.RightAngle, Start: 31, End: 32},
			end:   token.Token{Kind: token.Text, Start: 34, End: 47},
			kind:  ast.KindResponseRedirect,
		},
		{
			name: "response redirect no file",
			node: ast.ResponseRedirect{
				File:  nil,
				Token: token.Token{Kind: token.RightAngle, Start: 31, End: 32},
				Type:  ast.KindResponseRedirect,
			},
			start: token.Token{Kind: token.RightAngle, Start: 31, End: 32},
			end:   token.Token{Kind: token.RightAngle, Start: 31, End: 32},
			kind:  ast.KindResponseRedirect,
		},
		{
			name: "response reference",
			node: ast.ResponseReference{
				File: ast.TextLiteral{
					Value: "response.json",
					Token: token.Token{Kind: token.Text, Start: 34, End: 47},
					Type:  ast.KindTextLiteral,
				},
				Token: token.Token{Kind: token.ResponseRef, Start: 30, End: 32},
				Type:  ast.KindResponseReference,
			},
			start: token.Token{Kind: token.ResponseRef, Start: 30, End: 32},
			end:   token.Token{Kind: token.Text, Start: 34, End: 47},
			kind:  ast.KindResponseReference,
		},
		{
			name: "response reference no file",
			node: ast.ResponseReference{
				File:  nil,
				Token: token.Token{Kind: token.ResponseRef, Start: 30, End: 32},
				Type:  ast.KindResponseReference,
			},
			start: token.Token{Kind: token.ResponseRef, Start: 30, End: 32},
			end:   token.Token{Kind: token.ResponseRef, Start: 30, End: 32},
			kind:  ast.KindResponseReference,
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
