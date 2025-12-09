package parser_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
	"go.uber.org/goleak"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Erase and regenerate snapshots")
)

func TestFuzzFail(t *testing.T) {
	t.Skip("manually skip")

	src := []byte("### Body\nPOST https://api.somewhere.com/items/1\n\n{\n  \"somethi\xfeS\xe7C\xb3\x8f?ng\": \"here\"\n}\n\n<> response.json\n")
	p := parser.New("fuzz", src)

	_, err := p.Parse()
	t.Logf("Diagnostics: %+v\n", p.Diagnostics())
	test.Ok(t, err)
}

func TestParse(t *testing.T) {
	pattern := filepath.Join("testdata", "valid", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			snap := snapshot.New(
				t,
				snapshot.Update(*update),
				snapshot.Clean(*clean),
				snapshot.Color(os.Getenv("CI") == ""),
			)

			src, err := os.ReadFile(file)
			test.Ok(t, err)

			p := parser.New(name, src)

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

			src, ok := archive.Read("src.http")
			test.True(t, ok, test.Context("%s missing src.http", file))

			want, ok := archive.Read("want.txt")
			test.True(t, ok, test.Context("%s missing want.txt", file))

			parser := parser.New(name, []byte(src))

			_, err = parser.Parse()
			test.Err(t, err, test.Context("Parse() failed to return an error given invalid syntax"))

			var diagnostics strings.Builder
			for _, diag := range parser.Diagnostics() {
				diagnostics.WriteString(diag.String())
				diagnostics.WriteByte('\n')
			}

			got := diagnostics.String()

			if *update {
				test.Ok(t, archive.Write("want.txt", got))

				test.Ok(t, txtar.DumpFile(file, archive))

				return
			}

			test.Diff(t, got, want)
		})
	}
}

func BenchmarkParser(b *testing.B) {
	file := filepath.Join("testdata", "valid", "full.http")

	src, err := os.ReadFile(file)
	test.Ok(b, err)

	for b.Loop() {
		p := parser.New(file, src)

		_, err = p.Parse()
		test.Ok(b, err)
	}
}

func FuzzParser(f *testing.F) {
	// Get all the .http source from testdata for the corpus
	validPattern := filepath.Join("testdata", "valid", "*.http")
	validFiles, err := filepath.Glob(validPattern)
	test.Ok(f, err)

	invalidPattern := filepath.Join("testdata", "invalid", "*.http")
	invalidFiles, err := filepath.Glob(invalidPattern)
	test.Ok(f, err)

	files := slices.Concat(validFiles, invalidFiles)

	defer goleak.VerifyNone(f)

	for _, file := range files {
		src, err := os.ReadFile(file)
		test.Ok(f, err)

		f.Add(src)
	}

	// Property: The parser never panics or loops indefinitely, fuzz by default
	// will catch both of these
	f.Fuzz(func(t *testing.T, src []byte) {
		parser := parser.New("fuzz", src)

		_, _ = parser.Parse() //nolint:errcheck // Just checking for panics and infinite loops
	})
}
