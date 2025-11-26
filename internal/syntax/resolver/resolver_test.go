package resolver_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser/v2"
	"go.followtheprocess.codes/zap/internal/syntax/resolver"
	"go.uber.org/goleak"
)

var update = flag.Bool("update", false, "Update testdata")

func TestResolver(t *testing.T) {
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
			test.True(t, ok, test.Context("%s missing src.http", file))

			want, ok := archive.Read("want.json")
			test.True(t, ok, test.Context("%s missing want.json", file))

			p, err := parser.New(name, strings.NewReader(src), testFailHandler(t))
			test.Ok(t, err)

			parsed, err := p.Parse()
			test.Ok(t, err, test.Context("unexpected parser error"))

			resolved, err := resolver.ResolveFile(parsed)
			test.Ok(t, err, test.Context("unexpected resolver error"))

			resolvedJSON, err := json.MarshalIndent(resolved, "", "  ")
			test.Ok(t, err)

			resolvedJSON = append(resolvedJSON, '\n') // MarshalIndent doesn't do a final newline

			got := string(resolvedJSON)

			if *update {
				err := archive.Write("want.json", got)
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
