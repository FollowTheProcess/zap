// Package syntaxtest provides syntax level test utilities.
package syntaxtest

import (
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"path/filepath"

	"go.followtheprocess.codes/zap/internal/syntax/resolver/builtins"
)

// Deterministic return values for test comparison.
const (
	// UUID is the value returned from the '$uuid' test builtin.
	UUID = "d0a43b68-b9a1-4e89-bd21-b06fc59fefb5"
)

// TestBuiltins is a [builtins.Library] containing deterministic mock implementations
// of the zap builtins.
type TestBuiltins struct {
	library map[string]builtins.Builtin
}

// NewTestLibrary returns a new [builtins.Library] with mock functions
// that return deterministic outputs.
//
// It takes a map of env vars to be returned from the $env builtin.
func NewTestLibrary(env map[string]string) TestBuiltins {
	library := map[string]builtins.Builtin{
		"uuid": func(...string) (string, error) { return UUID, nil },
		"env": func(args ...string) (string, error) {
			if len(args) == 0 {
				return "", errors.New("$env requires a variable name, use $env.VAR_NAME")
			}

			val, ok := env[args[0]]
			if !ok {
				return "", fmt.Errorf("$env.%s: environment variable %s is not set", args[0], args[0])
			}

			return val, nil
		},
	}

	return TestBuiltins{library: library}
}

// Get looks up a builtin by name, returning the builtin and a boolean
// indicating its existence.
func (t TestBuiltins) Get(name string) (builtins.Builtin, bool) {
	fn, ok := t.library[name]
	if !ok {
		return nil, false
	}

	return fn, true
}

// Env returns a fake os.Environ, primarily used to test $env builtin behaviour.
func Env() map[string]string {
	return map[string]string{
		"ZAP_TEST_VAR": "test_env_value",
	}
}

// AllFilesWithExtension returns an iterator over all filepaths under
// root with the matching extension, recursively.
//
// A call to AllFilesWithExtension like this:
//
//	for file, err := range AllFilesWithExtension(".", ".go") {
//	    // Loop body
//	}
//
// Is roughly equivalent to the following in bash:
//
//	for file in **/*.go; do { # stuff }; done
func AllFilesWithExtension(root, ext string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				yield("", walkErr)
				return walkErr
			}

			if d.Type().IsRegular() && filepath.Ext(d.Name()) == ext {
				if !yield(path, nil) {
					return fs.SkipAll
				}
			}

			return nil
		})
		// handle the error returned by WalkDir itself
		if err != nil {
			yield("", err)
		}
	}
}
