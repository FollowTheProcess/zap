package parser_test

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
	"go.followtheprocess.codes/zap/internal/syntax/parser"
	"go.uber.org/goleak"
)

var update = flag.Bool("update", false, "Update snapshots and testdata")

// TestValid is the primary parser test for valid syntax. It reads src http text from
// a txtar archive in testdata/valid, parses it to completion, serialises that parsed result
// to JSON then generates a pretty diff if it doesn't match.
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
			test.True(t, ok, test.Context("archive %s missing src.http", name))

			want, ok := archive.Read("want.json")
			test.True(t, ok, test.Context("archive %s missing want.json", name))

			parser, err := parser.New(name, strings.NewReader(src), testFailHandler(t))
			test.Ok(t, err)

			got, err := parser.Parse()
			test.Ok(t, err, test.Context("unexpected parse error"))

			gotJSON, err := json.MarshalIndent(got, "", "  ")
			test.Ok(t, err, test.Context("could not marshal JSON"))

			gotJSON = append(gotJSON, '\n') // MarshalIndent doesn't do newlines at the end

			if *update {
				err := archive.Write("want.json", string(gotJSON))
				test.Ok(t, err)

				err = txtar.DumpFile(file, archive)
				test.Ok(t, err)

				return
			}

			test.Diff(t, string(gotJSON), want)
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
