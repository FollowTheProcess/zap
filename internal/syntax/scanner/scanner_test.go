package scanner_test

import (
	"slices"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/scanner"
	"go.followtheprocess.codes/zap/internal/syntax/token"
	"go.uber.org/goleak"
)

func TestBasics(t *testing.T) {
	tests := []struct {
		name string        // Name of the test case
		src  string        // Source text to scan
		want []token.Token // Expected token stream
	}{
		{
			name: "empty",
			src:  "",
			want: []token.Token{
				{Kind: token.EOF, Start: 0, End: 0},
			},
		},
		{
			name: "hash comment",
			src:  "# I'm a hash comment",
			want: []token.Token{
				{Kind: token.Comment, Start: 2, End: 20},
				{Kind: token.EOF, Start: 20, End: 20},
			},
		},
		{
			name: "slash comment",
			src:  "// I'm a slash comment",
			want: []token.Token{
				{Kind: token.Comment, Start: 3, End: 22},
				{Kind: token.EOF, Start: 22, End: 22},
			},
		},
		{
			name: "request separator",
			src:  "###",
			want: []token.Token{
				{Kind: token.Separator, Start: 0, End: 3},
				{Kind: token.EOF, Start: 3, End: 3},
			},
		},
		{
			name: "request separator with comment",
			src:  "### My Special Request",
			want: []token.Token{
				{Kind: token.Separator, Start: 0, End: 3},
				{Kind: token.Comment, Start: 4, End: 22},
				{Kind: token.EOF, Start: 22, End: 22},
			},
		},
		{
			name: "at",
			src:  "@",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.EOF, Start: 1, End: 1},
			},
		},
		{
			name: "variable",
			src:  "@var = test",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Ident, Start: 1, End: 4},
				{Kind: token.Eq, Start: 5, End: 6},
				{Kind: token.Text, Start: 7, End: 11},
				{Kind: token.EOF, Start: 11, End: 11},
			},
		},
		{
			name: "variable no equals",
			src:  "@var test",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Ident, Start: 1, End: 4},
				{Kind: token.Text, Start: 5, End: 9},
				{Kind: token.EOF, Start: 9, End: 9},
			},
		},
		{
			name: "name",
			src:  "@name = MyRequest",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Name, Start: 1, End: 5},
				{Kind: token.Eq, Start: 6, End: 7},
				{Kind: token.Text, Start: 8, End: 17},
				{Kind: token.EOF, Start: 17, End: 17},
			},
		},
		{
			name: "name no equals",
			src:  "@name MyRequest",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Name, Start: 1, End: 5},
				{Kind: token.Text, Start: 6, End: 15},
				{Kind: token.EOF, Start: 15, End: 15},
			},
		},
		{
			name: "hash request variable",
			src:  "# @var = test",
			want: []token.Token{
				{Kind: token.At, Start: 2, End: 3},
				{Kind: token.Ident, Start: 3, End: 6},
				{Kind: token.Eq, Start: 7, End: 8},
				{Kind: token.Text, Start: 9, End: 13},
				{Kind: token.EOF, Start: 13, End: 13},
			},
		},
		{
			name: "slash request variable",
			src:  "// @var = test",
			want: []token.Token{
				{Kind: token.At, Start: 3, End: 4},
				{Kind: token.Ident, Start: 4, End: 7},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.Text, Start: 10, End: 14},
				{Kind: token.EOF, Start: 14, End: 14},
			},
		},
		{
			name: "slash request variable",
			src:  "// @var = test",
			want: []token.Token{
				{Kind: token.At, Start: 3, End: 4},
				{Kind: token.Ident, Start: 4, End: 7},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.Text, Start: 10, End: 14},
				{Kind: token.EOF, Start: 14, End: 14},
			},
		},
		{
			name: "at ident equal integer",
			src:  "@something=20",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Ident, Start: 1, End: 10},
				{Kind: token.Eq, Start: 10, End: 11},
				{Kind: token.Text, Start: 11, End: 13},
				{Kind: token.EOF, Start: 13, End: 13},
			},
		},
		{
			name: "at timeout equal duration",
			src:  "@timeout = 20s", // Space because why not
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Timeout, Start: 1, End: 8},
				{Kind: token.Eq, Start: 9, End: 10},
				{Kind: token.Text, Start: 11, End: 14},
				{Kind: token.EOF, Start: 14, End: 14},
			},
		},
		{
			name: "prompt",
			src:  "@prompt username",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Prompt, Start: 1, End: 7},
				{Kind: token.Ident, Start: 8, End: 16},
				{Kind: token.EOF, Start: 16, End: 16},
			},
		},
		{
			name: "prompt with description",
			src:  "@prompt username The name of an authenticated user",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Prompt, Start: 1, End: 7},
				{Kind: token.Ident, Start: 8, End: 16},
				{Kind: token.Text, Start: 17, End: 50},
				{Kind: token.EOF, Start: 50, End: 50},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			src := []byte(tt.src)
			scanner := scanner.New(tt.name, src, testFailHandler(t))

			tokens := slices.Collect(scanner.All())

			test.EqualFunc(t, tokens, tt.want, slices.Equal, test.Context("token stream mismatch"))
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
