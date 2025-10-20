package parser_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
	"go.uber.org/goleak"
	"golang.org/x/tools/txtar"
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

			test.Equal(t, len(archive.Files), 2, test.Context("%s should contain 2 files, got %d", file, len(archive.Files)))
			test.Equal(
				t,
				archive.Files[0].Name,
				"src.http",
				test.Context("first file should be named 'src.http', got %q", archive.Files[0].Name),
			)
			test.Equal(
				t,
				archive.Files[1].Name,
				"want.json",
				test.Context("second file should be named 'want.json', got %q", archive.Files[1].Name),
			)

			src := archive.Files[0].Data
			want := archive.Files[1].Data

			parser, err := parser.New(name, bytes.NewReader(src), testFailHandler(t))
			test.Ok(t, err)

			got, err := parser.Parse()
			test.Ok(t, err, test.Context("unexpected parse error"))

			gotJSON, err := json.MarshalIndent(got, "", "  ")
			test.Ok(t, err, test.Context("could not marshal JSON"))

			gotJSON = append(gotJSON, '\n') // MarshalIndent doesn't do newlines at the end

			if *update {
				archive.Files[1].Data = gotJSON

				err := os.WriteFile(file, txtar.Format(archive), 0o644)
				test.Ok(t, err)

				return
			}

			test.DiffBytes(t, gotJSON, want)
		})
	}
}

// TestInvalid is the primary test for invalid syntax. It does much the same as TestParseValid
// but instead of failing tests if a syntax error is encounter, it fails if there is not any syntax errors.
//
// Additionally, the errors are compared against a reference.
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

			test.Equal(t, len(archive.Files), 2, test.Context("%s should contain 2 files, got %d", file, len(archive.Files)))
			test.Equal(
				t,
				archive.Files[0].Name,
				"src.http",
				test.Context("first file should be named 'src.http', got %q", archive.Files[0].Name),
			)
			test.Equal(
				t,
				archive.Files[1].Name,
				"want.txt",
				test.Context("second file should be named 'want.txt', got %q", archive.Files[1].Name),
			)

			src := archive.Files[0].Data
			want := archive.Files[1].Data

			collector := &errorCollector{}

			parser, err := parser.New(name, bytes.NewReader(src), collector.handler())
			test.Ok(t, err)

			_, err = parser.Parse()
			test.Err(t, err, test.Context("Parse() failed to return an error given invalid syntax"))

			got := collector.String()

			if *update {
				archive.Files[1].Data = []byte(got)

				err := os.WriteFile(file, txtar.Format(archive), 0o644)
				test.Ok(t, err)

				return
			}

			test.DiffBytes(t, []byte(got), want)
		})
	}
}

func BenchmarkParser(b *testing.B) {
	file := filepath.Join("testdata", "valid", "full.txtar")
	archive, err := txtar.ParseFile(file)
	test.Ok(b, err)

	test.True(b, len(archive.Files) > 1, test.Context("%s should contain at least 1 file, got %d", file, len(archive.Files)))
	test.Equal(b, archive.Files[0].Name, "src.http", test.Context("first file should be named 'src.http', got %q", archive.Files[0].Name))

	src := archive.Files[0].Data

	for b.Loop() {
		p, err := parser.New("bench", bytes.NewReader(src), testFailHandler(b))
		test.Ok(b, err)

		_, err = p.Parse()
		test.Ok(b, err)
	}
}

func FuzzParser(f *testing.F) {
	// Get all the .http source from testdata for the corpus
	pattern := filepath.Join("testdata", "valid", "*.txtar")
	files, err := filepath.Glob(pattern)
	test.Ok(f, err)

	for _, file := range files {
		archive, err := txtar.ParseFile(file)
		test.Ok(f, err)

		test.True(f, len(archive.Files) > 1, test.Context("%s should contain at least 1 file, got %d", file, len(archive.Files)))
		test.Equal(
			f,
			archive.Files[0].Name,
			"src.http",
			test.Context("first file should be named 'src.http', got %q", archive.Files[0].Name),
		)

		src := archive.Files[0].Data

		f.Add(src)
	}

	// Property: The parser never panics or loops indefinitely, fuzz by default
	// will catch both of these
	f.Fuzz(func(t *testing.T, src []byte) {
		// Note: no ErrorHandler installed, because if we let it report errors
		// it would kill the fuzz test straight away e.g. on the first invalid
		// utf-8 char
		parser, err := parser.New("fuzz", bytes.NewReader(src), nil)
		test.Ok(t, err)

		file, err := parser.Parse()

		var zeroFile syntax.File

		// Property: If the parser returned an error, then file must be empty
		if err != nil {
			if !reflect.DeepEqual(file, zeroFile) {
				t.Fatalf("\nnon zero syntax.File returned when err != nil: %#v\n", file)
			}
		}
	})
}

// testFailHandler returns a [syntax.ErrorHandler] that handles scanning errors by failing
// the enclosing test.
func testFailHandler(tb testing.TB) syntax.ErrorHandler {
	tb.Helper()

	return func(pos syntax.Position, msg string) {
		tb.Fatalf("%s: %s", pos, msg)
	}
}

// errorCollector is a helper struct that implements a [syntax.ErrorHandler] which
// simply collects the scanning errors internally to be inspected later.
type errorCollector struct {
	errs []string
	mu   sync.RWMutex
}

func (e *errorCollector) String() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Take a copy so as not to alter the original
	errsCopy := slices.Clone(e.errs)

	var s strings.Builder

	slices.Sort(errsCopy) // Deterministic

	for _, err := range errsCopy {
		s.WriteString(err)
	}

	return s.String()
}

// handler returns the [syntax.ErrorHandler] to be plugged in to the scanning operation.
func (e *errorCollector) handler() syntax.ErrorHandler {
	return func(pos syntax.Position, msg string) {
		// Because the scanner runs in it's own goroutine and also makes use of the
		// handler
		e.mu.Lock()
		defer e.mu.Unlock()

		e.errs = append(e.errs, fmt.Sprintf("%s: %s\n", pos, msg))
	}
}
