package zap_test

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"go.followtheprocess.codes/zap/internal/zap"
)

var update = flag.Bool("update", false, "Update testscript snapshots")

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"run": func() {
			app := zap.New(false, "test", os.Stdout, os.Stderr)
			options := zap.RunOptions{
				Timeout:           zap.DefaultTimeout,
				ConnectionTimeout: zap.DefaultConnectionTimeout,
				OverallTimeout:    zap.DefaultOverallTimeout,
			}

			err := app.Run(context.Background(), os.Args[1], nil, simpleErrorHandler(os.Stderr), options)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1) //nolint:revive // redundant-test-main-exit, this is testscript main
			}
		},
	})
}

func TestRun(t *testing.T) {
	// Just always returns a 200
	successHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header()["Date"] = nil
		fmt.Fprint(w, `{"stuff": "here"}`)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /success", successHandler)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	testscript.Run(t, testscript.Params{
		Dir:                 filepath.Join("testdata", "run"),
		UpdateScripts:       *update,
		RequireExplicitExec: true,
		RequireUniqueNames:  true,
		Setup: func(e *testscript.Env) error {
			e.Setenv("ZAP_TEST_URL", server.URL)
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"replace": Replace,
			"expand":  Expand,
		},
	})
}

// Replace is a testscript command that replaces text in a file by way of a regex
// pattern match, useful for replacing non-deterministic output like UUIDs and durations
// with placeholders to facilitate deterministic comparison in tests.
//
// Usage:
//
//	replace <file> <regex> <replacement>
//
// It cannot be negated, regex must be valid, and the file must be present in the
// txtar archive, including "stdout" and "stderr".
func Replace(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! filter")
	}

	if len(args) != 3 {
		ts.Fatalf("Usage: filter <file> <regex> <replacement>")
	}

	file := ts.MkAbs(args[0])
	ts.Logf("filter file: %s", file)

	stdout := ts.ReadFile("stdout")
	regex := args[1]
	replacement := args[2]

	re, err := regexp.Compile(regex)
	ts.Check(err)

	replaced := re.ReplaceAllString(stdout, replacement)

	_, err = ts.Stdout().Write([]byte(replaced))
	ts.Check(err)
}

// Expand expands environment variables in the given files and saves the new contents in place.
//
// Usage:
//
//	expand <files(s)...>
//
// It cannot be negated and works on "stdout" and "stderr".
func Expand(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! expand")
	}

	if len(args) < 1 {
		ts.Fatalf("usage: expand <file(s)...>")
	}

	for _, file := range args {
		file = ts.MkAbs(file)
		str := ts.ReadFile(file)

		str = os.Expand(str, func(key string) string {
			return ts.Getenv(key)
		})

		err := os.WriteFile(file, []byte(str), 0o644)
		ts.Check(err)
	}
}
