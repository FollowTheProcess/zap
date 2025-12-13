// Package syntaxtest provides syntax level test utilities.
package syntaxtest

import "go.followtheprocess.codes/zap/internal/syntax/resolver/builtins"

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
