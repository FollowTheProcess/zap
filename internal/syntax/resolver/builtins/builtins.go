// Package builtins provides the implementation of various builtin identifiers and functions
// to the resolver.
package builtins

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

// Builtin is an implementation of a zap builtin.
type Builtin func(args ...string) (string, error)

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
func NewLibrary() (Builtins, error) {
	library := map[string]Builtin{
		"uuid": builtinUUID,
		"env":  builtinEnv,
	}

	return Builtins{
		library: library,
	}, nil
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
func builtinUUID(args ...string) (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate a new uuid: %w", err)
	}

	return uid.String(), nil
}

// builtinEnv is the implementation of the '$env' builtin, the
// ident in the selector expression is the env var to retrieve.
//
// For example:
//
//	$env.USER_ID
//
// Maps to:
//
//	os.Getenv("USER_ID")
func builtinEnv(args ...string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf(
			"env: usage error, expected a single argument (name of the env var), got %d: %v",
			len(args),
			args,
		)
	}

	// TODO(@FollowTheProcess): I think we should move the os.Environ inside the resolver
	//
	// Expose the environment struct and have it contain os.Environ, then each builtin
	// can actually return a function that takes in the environment.
	//
	// Then we can mock out the builtins library as well as the environment and have
	// a complete sandbox for testing

	key := args[0]

	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", fmt.Errorf("env: variable %q not set", key)
	}

	return val, nil
}
