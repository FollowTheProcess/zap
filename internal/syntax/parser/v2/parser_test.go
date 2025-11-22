//nolint:revive // package-directory-mismatch is only temporary
package parser_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/rogpeppe/go-internal/txtar"
	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser/v2"
	"go.uber.org/goleak"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Erase and regenerate snapshots")
)

func TestParse(t *testing.T) {
	pattern := filepath.Join("testdata", "valid", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			snap := snapshot.New(
				t,
				snapshot.Update(*update),
				snapshot.Clean(*clean),
				snapshot.Color(os.Getenv("CI") == ""),
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

// TestInvalid is the primary test for invalid syntax. It does much the same as TestParse
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

			test.Equal(
				t,
				len(archive.Files),
				2,
				test.Context("%s should contain 2 files, got %d", file, len(archive.Files)),
			)
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

// testFailHandler returns a [syntax.ErrorHandler] that handles syntax errors by failing
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
