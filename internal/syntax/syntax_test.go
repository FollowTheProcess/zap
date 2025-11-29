package syntax_test

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
	"go.followtheprocess.codes/zap/internal/syntax/resolver"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Clean all snapshots and recreate")
)

func TestPositionString(t *testing.T) {
	tests := []struct {
		name string          // Name of the test case
		want string          // Expected return value
		pos  syntax.Position // Position under test
	}{
		{
			name: "empty",
			pos:  syntax.Position{},
			want: `BadPosition: {Name: "", Line: 0, StartCol: 0, EndCol: 0}`,
		},
		{
			name: "missing name",
			pos:  syntax.Position{Line: 12, StartCol: 2, EndCol: 6},
			want: `BadPosition: {Name: "", Line: 12, StartCol: 2, EndCol: 6}`,
		},
		{
			name: "zero line",
			pos:  syntax.Position{Name: "file.txt", Line: 0, StartCol: 12, EndCol: 19},
			want: `BadPosition: {Name: "file.txt", Line: 0, StartCol: 12, EndCol: 19}`,
		},
		{
			name: "zero start column",
			pos:  syntax.Position{Name: "file.txt", Line: 4, StartCol: 0, EndCol: 19},
			want: `BadPosition: {Name: "file.txt", Line: 4, StartCol: 0, EndCol: 19}`,
		},
		{
			name: "zero end column",
			pos:  syntax.Position{Name: "file.txt", Line: 4, StartCol: 1, EndCol: 0},
			want: `BadPosition: {Name: "file.txt", Line: 4, StartCol: 1, EndCol: 0}`,
		},
		{
			name: "end less than start",
			pos:  syntax.Position{Name: "test.http", Line: 1, StartCol: 6, EndCol: 4},
			want: `BadPosition: {Name: "test.http", Line: 1, StartCol: 6, EndCol: 4}`,
		},
		{
			name: "valid single column",
			pos:  syntax.Position{Name: "demo.http", Line: 1, StartCol: 6, EndCol: 6},
			want: "demo.http:1:6",
		},
		{
			name: "valid column range",
			pos:  syntax.Position{Name: "demo.http", Line: 17, StartCol: 20, EndCol: 26},
			want: "demo.http:17:20-26",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.Equal(t, tt.pos.String(), tt.want)
		})
	}
}

func TestFileString(t *testing.T) {
	tests := []struct {
		name string      // Name of the test case
		file syntax.File // File under test
	}{
		{
			name: "empty",
			file: syntax.File{},
		},
		{
			name: "name only",
			file: syntax.File{
				Name: "FileyMcFileFace",
			},
		},
		{
			name: "name and vars",
			file: syntax.File{
				Name: "SomeVars",
				Vars: map[string]string{
					"base":  "https://url.com/api/v1",
					"hello": "world",
				},
			},
		},
		{
			name: "non default timeouts",
			file: syntax.File{
				Name:              "Timeouts",
				Timeout:           42 * time.Second,
				ConnectionTimeout: 12 * time.Second,
			},
		},
		{
			name: "no redirect",
			file: syntax.File{
				Name:       "NoRedirect",
				NoRedirect: true,
			},
		},
		{
			name: "with simple request",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Comment: "A simple request",
						Name:    "GetItem",
						Method:  http.MethodGet,
						URL:     "https://api.com/v1/items/123",
					},
				},
			},
		},
		{
			name: "global prompt",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Prompts: map[string]syntax.Prompt{
					"colour": {
						Name:        "colour",
						Description: "The colour of something",
					},
				},
				Requests: []syntax.Request{
					{
						Comment: "A simple request",
						Name:    "SimpleRequest",
						Method:  http.MethodGet,
						URL:     "https://api.com/v1/items/123",
					},
				},
			},
		},
		{
			name: "request headers",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Method: http.MethodPost,
						URL:    "https://api.com/v1/items/123",
						Headers: http.Header{
							"Accept":        []string{"application/json"},
							"Content-Type":  []string{"application/json"},
							"Authorization": []string{"Bearer xxxxx"},
						},
					},
				},
			},
		},
		{
			name: "request with timeouts",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Name:              "AnotherRequest",
						Comment:           "This time it's a POST",
						Method:            http.MethodPost,
						URL:               "https://api.com/v1/items/123",
						Timeout:           3 * time.Second,
						ConnectionTimeout: 500 * time.Millisecond,
						NoRedirect:        true,
					},
				},
			},
		},
		{
			name: "request with body file",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Name:     "AnotherRequest",
						Method:   http.MethodPost,
						URL:      "https://api.com/v1/items/123",
						BodyFile: "./body.json",
					},
				},
			},
		},
		{
			name: "request with body file and response ref",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Name:        "AnotherRequest",
						Method:      http.MethodPost,
						URL:         "https://api.com/v1/items/123",
						BodyFile:    "./body.json",
						ResponseRef: "response.json",
					},
				},
			},
		},
		{
			name: "request with body",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Name:    "PostJSON",
						Comment: "This time with JSON body",
						Method:  http.MethodPost,
						URL:     "https://api.com/v1/items/123",
						Body:    []byte(`{"some": "json", "here": "yes"}`),
					},
				},
			},
		},
		{
			name: "request with response file",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Method:       http.MethodPost,
						URL:          "https://api.com/v1/items/123",
						ResponseFile: "./response.json",
					},
				},
			},
		},
		{
			name: "request with response ref",
			file: syntax.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Method:      http.MethodPost,
						URL:         "https://api.com/v1/items/123",
						ResponseRef: "./response.json",
					},
				},
			},
		},
		{
			name: "request with prompts",
			file: syntax.File{
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []syntax.Request{
					{
						Method:       http.MethodPost,
						URL:          "https://api.com/v1/items/{{ id }}",
						ResponseFile: "./response.json",
						Prompts: map[string]syntax.Prompt{
							"id": {
								Name:        "id",
								Description: "The ID of the item",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := snapshot.New(
				t,
				snapshot.Update(*update),
				snapshot.Clean(*clean),
			)
			snap.Snap(tt.file.String())
		})
	}
}

func FuzzPosition(f *testing.F) {
	f.Add("", 0, 0, 0)
	f.Add("name.txt", 1, 1, 2)
	f.Add("valid.http", 12, 17, 19)
	f.Add("invalid.http", 0, -9, 9999)

	f.Fuzz(func(t *testing.T, name string, line, startCol, endCol int) {
		pos := syntax.Position{
			Name:     name,
			Line:     line,
			StartCol: startCol,
			EndCol:   endCol,
		}

		got := pos.String()

		// Property: If IsValid returns false, the string must be this format
		if !pos.IsValid() {
			want := fmt.Sprintf(
				"BadPosition: {Name: %q, Line: %d, StartCol: %d, EndCol: %d}",
				name,
				line,
				startCol,
				endCol,
			)
			test.Equal(t, got, want)

			return
		}

		// Property: If IsValid returned true, Line must be >= 1
		test.True(
			t,
			pos.Line >= 1,
			test.Context("IsValid() = true but pos.Line (%d) was not >= 1", pos.Line),
		)

		// Property: If IsValid returned true, StartCol must be >= 1
		test.True(
			t,
			pos.StartCol >= 1,
			test.Context("IsValid() = true but pos.StartCol (%d) was not >= 1", pos.StartCol),
		)

		// Property: If IsValid returned true, EndCol must be >= 1
		test.True(
			t,
			pos.EndCol >= 1,
			test.Context("IsValid() = true but pos.EndCol (%d) was not >= 1", pos.EndCol),
		)

		// Property: If IsValid returned true, EndCol must also be >= StartCol
		test.True(
			t,
			pos.EndCol >= pos.StartCol,
			test.Context(
				"IsValid() = true but pos.EndCol (%d) was not >= pos.StartCol (%d)",
				pos.EndCol,
				pos.StartCol,
			),
		)

		// Property: If StartCol == EndCol, no range must appear in the string
		if startCol == endCol {
			want := fmt.Sprintf("%s:%d:%d", name, line, startCol)
			test.Equal(t, got, want)

			return
		}

		// Otherwise the position must be a valid position with a column range
		want := fmt.Sprintf("%s:%d:%d-%d", name, line, startCol, endCol)
		test.Equal(t, got, want)
	})
}

// A benchmark for the entire parsing pipeline.
func BenchmarkEntireParse(b *testing.B) {
	file := filepath.Join("testdata", "benchmarks", "full.http")

	src, err := os.ReadFile(file)
	test.Ok(b, err)

	for b.Loop() {
		p, err := parser.New(file, bytes.NewReader(src), testFailHandler(b))
		test.Ok(b, err)

		parsed, err := p.Parse()
		test.Ok(b, err)

		res := resolver.New(file)
		_, err = res.Resolve(parsed)
		test.Ok(b, err)
	}
}

// testFailHandler returns a [syntax.ErrorHandler] that handles syntax errors by failing
// the enclosing test.
func testFailHandler(tb testing.TB) syntax.ErrorHandler {
	tb.Helper()

	return func(pos syntax.Position, msg string) {
		tb.Fatalf("%s: %s", pos, msg)
	}
}
