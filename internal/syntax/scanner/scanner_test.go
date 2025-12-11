package scanner_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
	"go.followtheprocess.codes/zap/internal/syntax/scanner"
	"go.followtheprocess.codes/zap/internal/syntax/token"
	"go.uber.org/goleak"
)

var update = flag.Bool("update", false, "Update snapshots and testdata")

func TestValid(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	cwd, err := os.Getwd()
	test.Ok(t, err)

	parent := filepath.Dir(cwd)

	inputs := filepath.Join(parent, "testdata", "src", "*.http")
	files, err := filepath.Glob(inputs)
	test.Ok(t, err)

	outputDir := filepath.Join(parent, "testdata", "tokens")

	for _, file := range files {
		name := filepath.Base(file)
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		outputFile := filepath.Join(outputDir, stem+".txt")

		t.Run(name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			contents, err := os.ReadFile(file)
			test.Ok(t, err)

			s := scanner.New(name, contents)
			tokens := collect(s)

			var formattedTokens strings.Builder
			for _, tok := range tokens {
				formattedTokens.WriteString(tok.String())
				formattedTokens.WriteByte('\n')
			}

			got := formattedTokens.String()

			t.Logf("Diagnostics: %+v\n", s.Diagnostics())

			if *update {
				err = os.WriteFile(outputFile, []byte(got), 0o644)
				test.Ok(t, err)

				return
			}

			want, err := os.ReadFile(outputFile)
			test.Ok(t, err)

			test.Diff(t, got, string(want))
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

			want, ok := archive.Read("tokens.txt")
			test.True(t, ok, test.Context("%s missing tokens.txt", file))

			errs, ok := archive.Read("errors.txt")
			test.True(t, ok, test.Context("%s missing errors.txt", file))

			scanner := scanner.New(name, []byte(src))

			tokens := collect(scanner)

			var formattedTokens strings.Builder
			for _, tok := range tokens {
				formattedTokens.WriteString(tok.String())
				formattedTokens.WriteByte('\n')
			}

			got := formattedTokens.String()

			var diagnostics strings.Builder
			for _, diag := range scanner.Diagnostics() {
				diagnostics.WriteString(diag.String())
				diagnostics.WriteByte('\n')
			}

			gotErrs := diagnostics.String()

			if *update {
				err := archive.Write("tokens.txt", got)
				test.Ok(t, err)

				err = archive.Write("errors.txt", gotErrs)
				test.Ok(t, err)

				err = txtar.DumpFile(file, archive)
				test.Ok(t, err)

				return
			}

			test.Diff(t, got, want)

			test.Diff(t, got, want)
			test.Diff(t, gotErrs, errs)
		})
	}
}

func FuzzScanner(f *testing.F) {
	cwd, err := os.Getwd()
	test.Ok(f, err)

	parent := filepath.Dir(cwd)

	// Get all the .http source from testdata for the corpus
	inputs := filepath.Join(parent, "testdata", "src", "*.http")
	files, err := filepath.Glob(inputs)
	test.Ok(f, err)

	for _, file := range files {
		src, err := os.ReadFile(file)
		test.Ok(f, err)

		f.Add(src)
	}

	// Property: The scanner never panics or loops indefinitely, fuzz
	// by default will catch both of these
	f.Fuzz(func(t *testing.T, src []byte) {
		scanner := scanner.New("fuzz", src)

		for {
			tok := scanner.Scan()
			if tok.Is(token.EOF, token.Error) {
				break
			}

			// Property: Positions must be positive integers
			test.True(t, tok.Start >= 0, test.Context("token start position (%d) was negative", tok.Start))
			test.True(t, tok.End >= 0, test.Context("token end position (%d) was negative", tok.End))

			// Property: The kind must be one of the known kinds
			test.True(
				t,
				(tok.Kind >= token.EOF) && (tok.Kind <= token.MethodTrace),
				test.Context("token %s was not one of the pre-defined kinds", tok),
			)

			// Property: End must be >= Start
			test.True(t, tok.End >= tok.Start, test.Context("token %s had invalid start and end positions", tok))
		}
	})
}

func BenchmarkScanner(b *testing.B) {
	cwd, err := os.Getwd()
	test.Ok(b, err)

	parent := filepath.Dir(cwd)

	file := filepath.Join(parent, "testdata", "benchmarks", "full.http")
	src, err := os.ReadFile(file)
	test.Ok(b, err)

	for b.Loop() {
		s := scanner.New("bench", src)

		for {
			tok := s.Scan()
			if tok.Is(token.EOF, token.Error) {
				break
			}
		}
	}
}

// collect gathers up the scanned tokens into a slice.
func collect(s *scanner.Scanner) []token.Token {
	var tokens []token.Token

	for {
		tok := s.Scan()

		tokens = append(tokens, tok)
		if tok.Is(token.EOF, token.Error) {
			break
		}
	}

	return tokens
}
