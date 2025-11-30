package scanner_test //nolint:revive // Package directory mismatch will fix itself

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
	"go.followtheprocess.codes/zap/internal/syntax/scanner/v2"
	"go.followtheprocess.codes/zap/internal/syntax/token"
	"go.uber.org/goleak"
)

// TODO(@FollowTheProcess): Remove nolint for package directory mismatch when this scanner becomes the default

var update = flag.Bool("update", false, "Update snapshots and testdata")

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

			want, ok := archive.Read("tokens.txt")
			test.True(t, ok, test.Context("%s missing tokens.txt", file))

			scanner := scanner.New(name, []byte(src))

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

			var diagnostics strings.Builder
			for _, diag := range scanner.Diagnostics() {
				diagnostics.WriteString(diag.String())
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
