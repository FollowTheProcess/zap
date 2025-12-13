// Package builtins provides the implementation of various builtin identifiers and functions
// to the resolver.
package builtins

import (
	"fmt"

	"github.com/google/uuid"
)

// Builtin is an implementation of a zap builtin.
type Builtin func() (string, error)

// Library is a library of builtins.
type Library interface {
	// Get looks up a builtin from the library by name, returning it (or nil)
	// and a boolean indicating its existence.
	Get(name string) (Builtin, bool)
}

// Builtins is a [Library] containing the builtin implementations.
type Builtins struct {
	library map[string]Builtin
}

// NewLibrary returns the zap builtins library.
func NewLibrary() Builtins {
	library := map[string]Builtin{
		"uuid": builtinUUID,
	}

	return Builtins{
		library: library,
	}
}

// Get looks up a builtin by name, returning the builtin and a boolean
// indicating its existence.
func (b Builtins) Get(name string) (Builtin, bool) {
	fn, ok := b.library[name]
	if !ok {
		return nil, false
	}

	return fn, true
}

// builtinUUID is the implementation of the '$uuid' builtin.
func builtinUUID() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate a new uuid: %w", err)
	}

	return uid.String(), nil
}
