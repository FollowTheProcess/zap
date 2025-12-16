package builtins_test

import (
	"fmt"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/resolver/builtins"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name string // Name of the test case
		fn   string // Name of the function to lookup
		ok   bool   // Expected ok return value from Get
	}{
		{
			name: "empty",
			fn:   "",
			ok:   false,
		},
		{
			name: "missing",
			fn:   "dinglefuncbang",
			ok:   false,
		},
		{
			name: "exists",
			fn:   "uuid",
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib, err := builtins.NewLibrary()
			test.Ok(t, err)

			_, ok := lib.Get(tt.fn)
			test.Equal(t, ok, tt.ok, test.Context("Get(%s): expected %v, got %v", tt.fn, ok, tt.ok))
		})
	}
}

func TestBuiltins(t *testing.T) {
	lib, err := builtins.NewLibrary()
	test.Ok(t, err)

	tests := []struct {
		fn      builtins.Builtin // Function under test
		name    string           // Name of the test case
		wantErr bool             // Whether we want an error
	}{
		{
			name:    "uuid",
			fn:      mustGet(lib, "uuid"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn()
			test.WantErr(t, err, tt.wantErr)
			test.NotEqual(t, got, "")
		})
	}
}

// mustGet looks up a builtin function by name and panics
// if it's not found.
func mustGet(lib builtins.Builtins, name string) builtins.Builtin {
	fn, ok := lib.Get(name)
	if !ok {
		panic(fmt.Sprintf("builtin %s not found", name))
	}

	return fn
}
