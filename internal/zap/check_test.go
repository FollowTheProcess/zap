package zap_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
	"go.uber.org/goleak"
)

func TestCheckValid(t *testing.T) {
	pattern := filepath.Join("testdata", "check", "valid", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := zap.New(false, "test", os.Stdin, stdout, stderr)

			err := app.Check(t.Context(), simpleErrorHandler(stderr), zap.CheckOptions{Path: file})
			test.Ok(t, err)

			test.Diff(t, stdout.String(), fmt.Sprintf("Success: %s is valid\n", file))
			test.Diff(t, stderr.String(), "")
		})
	}
}

func TestCheckValidDir(t *testing.T) {
	path := filepath.Join("testdata", "check", "valid")
	pattern := filepath.Join(path, "*.http")

	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := zap.New(false, "test", os.Stdin, stdout, stderr)

	err = app.Check(t.Context(), simpleErrorHandler(stderr), zap.CheckOptions{Path: path})
	test.Ok(t, err)

	s := &strings.Builder{}

	// Write a success line for every file in the dir
	for _, file := range files {
		fmt.Fprintf(s, "Success: %s is valid\n", file)
	}

	test.Diff(t, stdout.String(), s.String())
	test.Diff(t, stderr.String(), "")
}

func TestCheckInvalid(t *testing.T) {
	pattern := filepath.Join("testdata", "check", "invalid", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := zap.New(false, "test", os.Stdin, stdout, stderr)

			err := app.Check(t.Context(), simpleErrorHandler(stderr), zap.CheckOptions{Path: file})
			test.Err(t, err)

			test.Equal(t, stdout.String(), "")

			// The actual error format is down to the handler and parse errors are tested
			// extensively in internal/syntax/parser so all we care about here is it's printing
			// something that looks like an error to stderr
			test.True(t, strings.Contains(stderr.String(), file))
		})
	}
}

// simpleErrorHandler returns a [syntax.ErrorHandler] that returns a simple, unstyled
// string representation of the error.
func simpleErrorHandler(w io.Writer) syntax.ErrorHandler {
	return func(pos syntax.Position, msg string) {
		fmt.Fprintf(w, "%s: %s\n", pos, msg)
	}
}
