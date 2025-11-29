package resolver_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
	"go.followtheprocess.codes/zap/internal/syntax/resolver"
	"go.uber.org/goleak"
	"go.yaml.in/yaml/v4"
)

var update = flag.Bool("update", false, "Update testdata")

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
			test.True(t, ok, test.Context("%s missing src.http", file))

			want, ok := archive.Read("want.yaml")
			test.True(t, ok, test.Context("%s missing want.yaml", file))

			p, err := parser.New(name, strings.NewReader(src), testFailHandler(t))
			test.Ok(t, err)

			parsed, err := p.Parse()
			test.Ok(t, err, test.Context("unexpected parser error"))

			res := resolver.New(name)

			resolved, err := res.Resolve(parsed)
			if err != nil {
				t.Logf("%+v\n", res.Diagnostics())
			}

			test.Ok(t, err, test.Context("unexpected resolver error"))

			test.Equal(t, len(res.Diagnostics()), 0)

			buf := &bytes.Buffer{}

			encoder := yaml.NewEncoder(buf)
			encoder.SetIndent(2)

			test.Ok(t, encoder.Encode(resolved))

			got := buf.String()

			if *update {
				err := archive.Write("want.yaml", got)
				test.Ok(t, err)

				err = txtar.DumpFile(file, archive)
				test.Ok(t, err)

				return
			}

			test.Diff(t, got, want)
		})
	}
}

func TestInvalid(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	pattern := filepath.Join("testdata", "invalid", "*.txtar")
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

			want, ok := archive.Read("diagnostics.json")
			test.True(t, ok, test.Context("%s missing diagnostics.json", file))

			p, err := parser.New(name, strings.NewReader(src), testFailHandler(t))
			test.Ok(t, err)

			parsed, err := p.Parse()
			test.Ok(t, err, test.Context("unexpected parser error"))

			res := resolver.New(name)

			_, err = res.Resolve(parsed)
			test.Err(t, err, test.Context("resolved did not return an error but should have"))

			diagnosticJSON, err := json.MarshalIndent(res.Diagnostics(), "", "  ")
			test.Ok(t, err)

			diagnosticJSON = append(diagnosticJSON, '\n') // MarshalIndent doesn't do a final newline

			got := string(diagnosticJSON)

			if *update {
				err := archive.Write("diagnostics.json", got)
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

// TODO(@FollowTheProcess): Benchmark the resolver once it's complete, use the same full.http file
// as the parser benchmark.

func FuzzResolver(f *testing.F) {
	// Get all valid .http source from testdata for the corpus
	validPattern := filepath.Join("testdata", "valid", "*.txtar")
	validFiles, err := filepath.Glob(validPattern)
	test.Ok(f, err)

	// Invalid ones too!
	invalidPattern := filepath.Join("testdata", "invalid", "*.txtar")
	invalidFiles, err := filepath.Glob(invalidPattern)
	test.Ok(f, err)

	files := slices.Concat(validFiles, invalidFiles)

	defer goleak.VerifyNone(f)

	for _, file := range files {
		archive, err := txtar.ParseFile(file)
		test.Ok(f, err)

		src, ok := archive.Read("src.http")
		test.True(f, ok, test.Context("%s missing src.http", file))

		// Add the src to the fuzz corpus
		f.Add(src)
	}

	// This also fuzzes the parser (again) but realistically there's no way around that. It's also
	// not a bad thing as the parser produces partial trees in error cases to aid error reporting
	// so we need to be able to handle these.

	// Property: The resolver never panics or loops indefinitely, fuzz by default will
	// catch both of these
	f.Fuzz(func(t *testing.T, src string) {
		parser, err := parser.New("fuzz", strings.NewReader(src), nil)
		test.Ok(t, err)

		parsed, _ := parser.Parse() //nolint:errcheck // Just checking for panics and infinite loops

		res := resolver.New(parsed.Name)

		resolved, err := res.Resolve(parsed)
		// Property: If there is an error, the file should be the zero spec.File{}
		if err != nil {
			if !reflect.DeepEqual(resolved, spec.File{}) {
				// Marshal it as JSON for readability
				resolvedJSON, err := json.MarshalIndent(resolved, "", "  ")
				test.Ok(t, err)

				t.Fatalf("got a non-zero spec.File{} in err != nil case:\n\n%s\n\n", resolvedJSON)
			}
		}
	})
}
