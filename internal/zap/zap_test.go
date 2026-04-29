package zap_test

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/zap"
	"go.uber.org/goleak"
)

// TODO(@FollowTheProcess): This is getting unwieldy and is a bit beyond what testscript is designed for
//
// Once I have builtins and selector expressions working, the http files will be able to
// do e.g. '@base = {{ $env.TEST_SERVER_URL }}' and then I can host a local httpbin in
// a testcontainer or even just create a simple inline server for the tests and it'll
// be much nicer! :)

var update = flag.Bool("update", false, "Update testdata files")

func TestRun(t *testing.T) {
	// Loop through all files in testdata/run/*.txtar
	// Each one expects a src.http and an expected.txt
	// Set up the test server and export it's URL as an env var
	// Read the txtar and parse src.http, each src.http entry has $env.ZAP_TEST_URL as base
	// Execute it
	// Diff result against expected.txt
	pattern := filepath.Join("testdata", "run", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			server := NewTestServer(t)
			t.Cleanup(server.Close)

			t.Setenv("ZAP_TEST_URL", server.URL)

			stdin := &bytes.Buffer{}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := zap.New(false, "test", stdin, stdout, stderr)

			options := zap.RunOptions{
				File:              file,
				Output:            "stdout",
				Timeout:           zap.DefaultTimeout,
				ConnectionTimeout: zap.DefaultConnectionTimeout,
				OverallTimeout:    zap.DefaultOverallTimeout,
				NoRedirect:        false,
				Debug:             false,
				Verbose:           false,
			}

			f, err := os.Open(file)
			test.Ok(t, err)
			t.Cleanup(func() { f.Close() })

			err = app.Run(t.Context(), f, options)
			test.Ok(t, err, test.Context("zap run returned an error: %v", stderr.String()))

			snap := snapshot.New(
				t,
				snapshot.Update(*update),
				snapshot.Description(fmt.Sprintf("response from %s", file)),
				snapshot.Filter(`\d+(?:\.\d+)?(?:s|ms|µs)`, "[DURATION]"),
			)

			snap.Snap(stdout.String())
		})
	}
}

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

			err := app.Check(t.Context(), zap.CheckOptions{Path: file})
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

	err = app.Check(t.Context(), zap.CheckOptions{Path: path})
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

			err := app.Check(t.Context(), zap.CheckOptions{Path: file})
			test.Err(t, err)

			test.Equal(t, stdout.String(), "")

			got := stderr.String()

			// TODO(@FollowTheProcess): If we keep this, maybe have a flag to output parse/resolve errors
			// as JSON?

			t.Logf("stderr:\n\n%s\n", got)

			// The actual error format is down to the handler and parse errors are tested
			// extensively in internal/syntax/parser so all we care about here is it's printing
			// something that looks like an error to stderr
			test.True(
				t,
				strings.Contains(got, filepath.Base(file)),
				test.Context("stderr output did not contain %s", filepath.Base(file)),
			)
		})
	}
}

func TestExport(t *testing.T) {
	pattern := filepath.Join("testdata", "export", "*.http")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			app := zap.New(false, "test", os.Stdin, stdout, stderr)

			format := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

			options := zap.ExportOptions{
				File:   file,
				Format: format,
			}

			f, err := os.Open(file)
			test.Ok(t, err)
			t.Cleanup(func() { f.Close() })

			err = app.Export(t.Context(), f, options)
			test.Ok(t, err)

			snap := snapshot.New(t, snapshot.Update(*update))
			snap.Snap(stdout.String())
		})
	}
}

// NewTestServer spins up a new httptest server with a few endpoints defined for use in
// zap integration tests.
//
// All routes return static JSON content with the Content-Type of application/json.
//
// The routes defined are:
//
//   - GET /ok: returns a 200 OK
//   - POST /bad: returns a 400 Bad Request
//
// The caller is responsible for calling server.Close via t.Cleanup.
func NewTestServer(tb testing.TB) *httptest.Server {
	tb.Helper()

	// Just always returns a 200
	successHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header()["Date"] = nil
		fmt.Fprint(w, `{"stuff": "here"}`)
	}

	// Bad!
	badRequestHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header()["Date"] = nil
		w.WriteHeader(http.StatusBadRequest)

		fmt.Fprint(w, `{"bad": "yes"}`)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", successHandler)
	mux.HandleFunc("POST /bad", badRequestHandler)

	return httptest.NewServer(mux)
}
