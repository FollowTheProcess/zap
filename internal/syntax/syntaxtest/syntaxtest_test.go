// That's right... how meta is this.
package syntaxtest_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/syntaxtest"
)

func TestTestLibrary(t *testing.T) {
	tests := []struct {
		env     map[string]string // Environment variables (fake obviously)
		name    string            // Name of the test case
		fn      string            // Name of the builtin to lookup
		want    string            // Expected return value
		args    []string          // Args to the function
		ok      bool              // The expected ok value of the lookup
		wantErr bool              // Whether we want an error
	}{
		{
			name: "empty",
			fn:   "",
			ok:   false,
		},
		{
			name: "missing",
			fn:   "notreal",
			ok:   false,
		},
		{
			name:    "valid uuid",
			fn:      "uuid",
			want:    syntaxtest.UUID,
			ok:      true,
			wantErr: false,
		},
		{
			name:    "valid env",
			fn:      "env",
			args:    []string{"VAR"},
			want:    "A value here",
			ok:      true,
			wantErr: false,
			env: map[string]string{
				"VAR": "A value here",
			},
		},
		{
			name:    "missing env",
			fn:      "env",
			args:    []string{"var"},
			want:    "",
			ok:      true,
			wantErr: true,
			env:     map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := syntaxtest.NewTestLibrary(tt.env)

			fn, ok := lib.Get(tt.fn)
			test.Equal(t, ok, tt.ok, test.Context("Get(%s): got %t, expected %t", tt.fn, ok, tt.ok))

			if !ok {
				// Can't call a nil function
				return
			}

			got, err := fn(tt.args...)
			test.WantErr(t, err, tt.wantErr)
			test.Equal(t, got, tt.want)
		})
	}
}

func TestAllFilesWithExtension(t *testing.T) {
	cwd, err := os.Getwd()
	test.Ok(t, err)

	var results []string

	for file, err := range syntaxtest.AllFilesWithExtension(cwd, ".go") {
		test.Ok(t, err)

		results = append(results, file)
	}

	slices.Sort(results)

	want := []string{
		// Just the two files
		filepath.Join(cwd, "syntaxtest.go"),
		filepath.Join(cwd, "syntaxtest_test.go"),
	}

	test.EqualFunc(t, results, want, slices.Equal)
}
