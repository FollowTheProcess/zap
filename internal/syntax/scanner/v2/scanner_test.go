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
	"go.followtheprocess.codes/zap/internal/syntax/scanner/v2"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

var update = flag.Bool("update", false, "Update snapshots and testdata")

func TestFuzzFail(t *testing.T) {
	t.Skip("unskip to investigate/debug fuzz fails")

	src := []byte("###\nGEThttps://2#\nHEADhttps:// ###\x88\f\xadU\x00\x95&\xb1\xa3\x00\x7f\xe3z\x8f\xcar\x99\xd1P\xf0y\xfa\x90\xbe\x94\nPOSThttps{{{{{#+#PUThttps://\n##\n ")
	s := scanner.New("fuzz", src)
	got := collect(s)
	want := []token.Token{{}}

	test.EqualFunc(t, got, want, slices.Equal)
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
			archive, err := txtar.ParseFile(file)
			test.Ok(t, err)

			src, ok := archive.Read("src.http")
			test.True(t, ok, test.Context("%s missing src.http", file))

			want, ok := archive.Read("tokens.txt")
			test.True(t, ok, test.Context("%s missing tokens.txt", file))

			scanner := scanner.New(name, []byte(src))

			tokens := collect(scanner)

			var formattedTokens strings.Builder
			for _, tok := range tokens {
				formattedTokens.WriteString(tok.String())
				formattedTokens.WriteByte('\n')
			}

			got := formattedTokens.String()

			t.Logf("Diagnostics: %+v\n", scanner.Diagnostics())

			if *update {
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

func TestInvalid(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	pattern := filepath.Join("testdata", "invalid", "*.txtar")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
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
	// Get all the .http source from testdata for the corpus
	pattern := filepath.Join("testdata", "valid", "*.txtar")
	files, err := filepath.Glob(pattern)
	test.Ok(f, err)

	for _, file := range files {
		archive, err := txtar.ParseFile(file)
		test.Ok(f, err)

		if archive == nil {
			f.Fatal("txtar.ParseFile returned nil archive")
		}

		src, ok := archive.Read("src.http")
		test.True(f, ok, test.Context("%s missing src.http", file))

		f.Add(src)
	}

	// Property: The scanner never panics or loops indefinitely, fuzz
	// by default will catch both of these
	f.Fuzz(func(t *testing.T, src string) {
		scanner := scanner.New("fuzz", []byte(src))

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
	file := filepath.Join("testdata", "valid", "full.txtar")
	archive, err := txtar.ParseFile(file)
	test.Ok(b, err)

	if archive == nil {
		b.Fatal("txtar.ParseFile returned nil archive")
	}

	src, ok := archive.Read("src.http")
	test.True(b, ok, test.Context("%s missing src.http", file))

	for b.Loop() {
		s := scanner.New("bench", []byte(src))

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
