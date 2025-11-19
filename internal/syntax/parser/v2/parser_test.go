//nolint:revive // package-directory-mismatch is only temporary
package parser_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser/v2"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Erase and regenerate snapshots")
)

func TestParse(t *testing.T) {
	pattern := filepath.Join("testdata", "src", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			snap := snapshot.New(
				t,
				snapshot.Update(*update),
				snapshot.Clean(*clean),
			)

			src, err := os.Open(file)
			test.Ok(t, err)

			defer src.Close()

			p, err := parser.New(name, src, testFailHandler(t))
			test.Ok(t, err)

			parsed, err := p.Parse()
			test.Ok(t, err)

			snap.Snap(parsed)
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
