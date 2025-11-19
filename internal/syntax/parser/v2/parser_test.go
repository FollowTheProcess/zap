//nolint:revive // package-directory-mismatch is only temporary
package parser_test

import (
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/ast"
	"go.followtheprocess.codes/zap/internal/syntax/parser/v2"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

func TestParseVarStatement(t *testing.T) {
	tests := []struct {
		name string           // Name of the test case
		src  string           // Input source code
		want ast.VarStatement // Expected ast node
	}{
		{
			name: "simple",
			src:  "@test = something",
			want: ast.VarStatement{
				Value: ast.TextLiteral{
					Token: token.Token{Kind: token.Text, Start: 8, End: 17},
					Value: "something",
				},
				Ident: ast.Ident{
					Name:  "test",
					Token: token.Token{Kind: token.Ident, Start: 1, End: 5},
				},
				At: token.Token{Kind: token.At, Start: 0, End: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parser.New(tt.name, strings.NewReader(tt.src), testFailHandler(t))
			test.Ok(t, err)

			file, err := p.Parse()
			test.Ok(t, err)

			test.Equal(t, len(file.Statements), 1, test.Context("wrong number of statements"))

			got, ok := file.Statements[0].(ast.VarStatement)
			test.True(t, ok, test.Context("could not cast to a VarStatement, got type %T", file.Statements[0]))

			test.Equal(t, got.Start(), tt.want.Start())
			test.Equal(t, got.End(), tt.want.End())
			test.Equal(t, got.At, tt.want.At)
			test.Equal(t, got.Ident, tt.want.Ident)

			gotValue, ok := got.Value.(ast.TextLiteral)
			test.True(t, ok, test.Context("could not cast got to a TextLiteral, got type %T", got.Value))

			wantValue, ok := tt.want.Value.(ast.TextLiteral)
			test.True(t, ok, test.Context("could not cast want to a TextLiteral, got type %T", got.Value))

			test.Equal(t, gotValue, wantValue)
		})
	}
}

// testFailHandler returns a [syntax.ErrorHandler] that handles scanning errors by failing
// the enclosing test.
func testFailHandler(tb testing.TB) syntax.ErrorHandler {
	tb.Helper()

	return func(pos syntax.Position, msg string) {
		tb.Fatalf("%s: %s", pos, msg)
	}
}
