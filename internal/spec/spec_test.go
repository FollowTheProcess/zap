package spec_test

import (
	"flag"
	"net/http"
	"testing"
	"time"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Clean all snapshots and recreate")
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name string    // Name of the test case
		file spec.File // File under test
	}{
		{
			name: "empty",
			file: spec.File{},
		},
		{
			name: "name only",
			file: spec.File{
				Name: "FileyMcFileFace",
			},
		},
		{
			name: "name and vars",
			file: spec.File{
				Name: "SomeVars",
				Vars: map[string]string{
					"base":  "https://url.com/api/v1",
					"hello": "world",
				},
			},
		},
		{
			name: "non default timeouts",
			file: spec.File{
				Name:              "Timeouts",
				Timeout:           42 * time.Second,
				ConnectionTimeout: 12 * time.Second,
			},
		},
		{
			name: "no redirect",
			file: spec.File{
				Name:       "NoRedirect",
				NoRedirect: true,
			},
		},
		{
			name: "global prompts",
			file: spec.File{
				Name: "PromptMe",
				Prompts: map[string]spec.Prompt{
					"value": {
						Name:        "value",
						Description: "Give me a value!",
					},
				},
			},
		},
		{
			name: "with simple request",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:    "GetItem",
						Comment: "A simple request",
						Method:  http.MethodGet,
						URL:     "https://api.com/v1/items/123",
					},
				},
			},
		},
		{
			name: "request with variables",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name: "GetItem",
						Vars: map[string]string{
							"test": "yes",
						},
						Comment: "A simple request",
						Method:  http.MethodGet,
						URL:     "https://api.com/v1/items/123",
					},
				},
			},
		},
		{
			name: "with http version",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:        "GetItem",
						Comment:     "A simple request",
						Method:      http.MethodGet,
						HTTPVersion: "HTTP/1.2",
						URL:         "https://api.com/v1/items/123",
					},
				},
			},
		},
		{
			name: "request headers",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:   "Another Request",
						Method: http.MethodPost,
						URL:    "https://api.com/v1/items/123",
						Headers: map[string]string{
							"Accept":        "application/json",
							"Content-Type":  "application/json",
							"Authorization": "Bearer xxxxx",
						},
					},
				},
			},
		},
		{
			name: "request with timeouts",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:              "Another Request",
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
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:     "Another Request",
						Method:   http.MethodPost,
						URL:      "https://api.com/v1/items/123",
						BodyFile: "./body.json",
					},
				},
			},
		},
		{
			name: "request with body",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:   "Another Request",
						Method: http.MethodPost,
						URL:    "https://api.com/v1/items/123",
						Body:   []byte(`{"some": "json", "here": "yes"}`),
					},
				},
			},
		},
		{
			name: "request with response file",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:         "Another Request",
						Method:       http.MethodPost,
						URL:          "https://api.com/v1/items/123",
						ResponseFile: "./response.json",
					},
				},
			},
		},
		{
			name: "request with response ref",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name:        "Another Request",
						Method:      http.MethodPost,
						URL:         "https://api.com/v1/items/123",
						ResponseRef: "./response.200.json",
					},
				},
			},
		},
		{
			name: "request with prompt",
			file: spec.File{
				Name: "Requests",
				Vars: map[string]string{
					"base": "https://api.com/v1",
				},
				Requests: []spec.Request{
					{
						Name: "Another",
						Prompts: map[string]spec.Prompt{
							"value": {
								Name:        "value",
								Description: "Give me a value!",
							},
							"no-description": {
								Name: "guess",
							},
						},
						Method:       http.MethodPost,
						URL:          "https://api.com/v1/items/123",
						ResponseFile: "./response.json",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := snapshot.New(t, snapshot.Update(*update), snapshot.Clean(*clean))
			snap.Snap(tt.file.String())
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string      // Name of the test case
		in      syntax.File // syntax.File being resolved
		want    spec.File   // Expected spec.File out
		wantErr bool        // Whether we wanted an error
	}{
		{
			name:    "empty",
			in:      syntax.File{},
			want:    spec.File{},
			wantErr: false,
		},
		{
			name: "basic",
			in: syntax.File{
				Name: "test.http",
				Requests: []syntax.Request{
					{
						Headers: map[string]string{
							"Accept": "application/json",
						},
						Prompts: map[string]syntax.Prompt{
							"value": {
								Name:        "value",
								Description: "Give me a value",
							},
						},
						Name:   "blah",
						Method: http.MethodGet,
						URL:    "https://doesntexit.com",
					},
				},
			},
			want: spec.File{
				Name: "test.http",
				Requests: []spec.Request{
					{
						Headers: map[string]string{
							"Accept": "application/json",
						},
						Prompts: map[string]spec.Prompt{
							"value": {
								Name:        "value",
								Description: "Give me a value",
							},
						},
						Name:   "blah",
						Method: http.MethodGet,
						URL:    "https://doesntexit.com",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spec.ResolveFile(tt.in)
			test.WantErr(t, err, tt.wantErr)

			test.Diff(t, got.String(), tt.want.String())
		})
	}
}
