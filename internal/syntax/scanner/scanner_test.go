package scanner_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/scanner"
	"go.followtheprocess.codes/zap/internal/syntax/token"
	"go.uber.org/goleak"
)

var update = flag.Bool("update", false, "Update snapshots and testdata")

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
			name: "variable with interp",
			src:  "@var = {{ base }}",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Ident, Start: 1, End: 4},
				{Kind: token.Eq, Start: 5, End: 6},
				{Kind: token.OpenInterp, Start: 7, End: 9},
				{Kind: token.Ident, Start: 10, End: 14},
				{Kind: token.CloseInterp, Start: 15, End: 17},
				{Kind: token.EOF, Start: 17, End: 17},
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
			name: "variable no equals interp",
			src:  "@var {{ test }}",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Ident, Start: 1, End: 4},
				{Kind: token.OpenInterp, Start: 5, End: 7},
				{Kind: token.Ident, Start: 8, End: 12},
				{Kind: token.CloseInterp, Start: 13, End: 15},
				{Kind: token.EOF, Start: 15, End: 15},
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
			name: "name equals interp",
			src:  "@name = {{ something }}",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Name, Start: 1, End: 5},
				{Kind: token.Eq, Start: 6, End: 7},
				{Kind: token.OpenInterp, Start: 8, End: 10},
				{Kind: token.Ident, Start: 11, End: 20},
				{Kind: token.CloseInterp, Start: 21, End: 23},
				{Kind: token.EOF, Start: 23, End: 23},
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
			name: "name no equals interp",
			src:  "@name {{ something }}",
			want: []token.Token{
				{Kind: token.At, Start: 0, End: 1},
				{Kind: token.Name, Start: 1, End: 5},
				{Kind: token.OpenInterp, Start: 6, End: 8},
				{Kind: token.Ident, Start: 9, End: 18},
				{Kind: token.CloseInterp, Start: 19, End: 21},
				{Kind: token.EOF, Start: 21, End: 21},
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
			name: "hash request variable no equals",
			src:  "# @var test",
			want: []token.Token{
				{Kind: token.At, Start: 2, End: 3},
				{Kind: token.Ident, Start: 3, End: 6},
				{Kind: token.Text, Start: 7, End: 11},
				{Kind: token.EOF, Start: 11, End: 11},
			},
		},
		{
			name: "hash request variable interp",
			src:  "# @var = {{ test }}",
			want: []token.Token{
				{Kind: token.At, Start: 2, End: 3},
				{Kind: token.Ident, Start: 3, End: 6},
				{Kind: token.Eq, Start: 7, End: 8},
				{Kind: token.OpenInterp, Start: 9, End: 11},
				{Kind: token.Ident, Start: 12, End: 16},
				{Kind: token.CloseInterp, Start: 17, End: 19},
				{Kind: token.EOF, Start: 19, End: 19},
			},
		},
		{
			name: "hash request variable interp no equals",
			src:  "# @var {{ test }}",
			want: []token.Token{
				{Kind: token.At, Start: 2, End: 3},
				{Kind: token.Ident, Start: 3, End: 6},
				{Kind: token.OpenInterp, Start: 7, End: 9},
				{Kind: token.Ident, Start: 10, End: 14},
				{Kind: token.CloseInterp, Start: 15, End: 17},
				{Kind: token.EOF, Start: 17, End: 17},
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
			name: "slash request variable interp",
			src:  "// @var = {{ test }}",
			want: []token.Token{
				{Kind: token.At, Start: 3, End: 4},
				{Kind: token.Ident, Start: 4, End: 7},
				{Kind: token.Eq, Start: 8, End: 9},
				{Kind: token.OpenInterp, Start: 10, End: 12},
				{Kind: token.Ident, Start: 13, End: 17},
				{Kind: token.CloseInterp, Start: 18, End: 20},
				{Kind: token.EOF, Start: 20, End: 20},
			},
		},
		{
			name: "slash request variable interp no equals",
			src:  "// @var {{ test }}",
			want: []token.Token{
				{Kind: token.At, Start: 3, End: 4},
				{Kind: token.Ident, Start: 4, End: 7},
				{Kind: token.OpenInterp, Start: 8, End: 10},
				{Kind: token.Ident, Start: 11, End: 15},
				{Kind: token.CloseInterp, Start: 16, End: 18},
				{Kind: token.EOF, Start: 18, End: 18},
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

			var tokens []token.Token

			for {
				tok := scanner.Scan()

				tokens = append(tokens, tok)
				if tok.Is(token.EOF, token.Error) {
					break
				}
			}

			test.EqualFunc(t, tokens, tt.want, slices.Equal, test.Context("token stream mismatch"))
		})
	}
}

func TestValid(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	pattern := filepath.Join("testdata", "valid", "*.txtar")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			archive, err := txtar.ParseFile(file)
			test.Ok(t, err)

			src, ok := archive.Read("src.http")
			test.True(t, ok, test.Context("archive missing src.http"))

			want, ok := archive.Read("tokens.txt")
			test.True(t, ok, test.Context("archive missing tokens.txt"))

			scanner := scanner.New(name, []byte(src), testFailHandler(t))

			var tokens []token.Token

			for {
				tok := scanner.Scan()

				tokens = append(tokens, tok)
				if tok.Is(token.EOF, token.Error) {
					break
				}
			}

			var formattedTokens strings.Builder
			for _, tok := range tokens {
				formattedTokens.WriteString(tok.String())
				formattedTokens.WriteByte('\n')
			}

			got := formattedTokens.String()

			if *update {
				// Update the expected with what's actually been seen
				err := archive.Write("tokens.txt", got)
				test.Ok(t, err)

				err = txtar.DumpFile(file, archive)
				test.Ok(t, err)

				return
			}

			test.Diff(t, got, want)
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
