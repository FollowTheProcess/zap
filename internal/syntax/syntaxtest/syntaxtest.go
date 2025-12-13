// Package syntaxtest provides syntax level test utilities.
package syntaxtest

import (
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
func NewTestLibrary() TestBuiltins {
	library := map[string]builtins.Builtin{
		"uuid": func() (string, error) { return UUID, nil },
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
