package spec_test

import (
	"flag"
	"net/http"
	"testing"
	"time"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/zap/internal/spec"
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
						Body:   `{"some": "json", "here": "yes"}`,
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
