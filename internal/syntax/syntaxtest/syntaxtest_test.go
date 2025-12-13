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
		name    string // Name of the test case
		fn      string // Name of the builtin to lookup
		want    string // Expected return value
		ok      bool   // The expected ok value of the lookup
		wantErr bool   // Whether we want an error
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
			name:    "valid",
			fn:      "uuid",
			want:    syntaxtest.UUID,
			ok:      true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := syntaxtest.NewTestLibrary()

			fn, ok := lib.Get(tt.fn)
			test.Equal(t, ok, tt.ok, test.Context("Get(%s): got %t, expected %t", tt.fn, ok, tt.ok))

			if !ok {
				// Can't call a nil function
				return
			}

			got, err := fn()
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
