package format_test

import (
	"bytes"
	"flag"
	"net/http"
	"os"
	"testing"
	"time"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/format"
	"go.followtheprocess.codes/zap/internal/spec"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Clean all snapshots and recreate")
)

func TestCurlExporter(t *testing.T) {
	tests := []struct {
		name string    // Name of the test case
		file spec.File // The HTTP file
	}{
		{
			name: "simple",
			file: spec.File{
				Name: "simple",
				Requests: []spec.Request{
					{
						Method: http.MethodGet,
						URL:    "https://api.nowhere.com/v1/items/1234",
					},
				},
			},
		},
		{
			name: "no redirect",
			file: spec.File{
				Name: "no redirect",
				Requests: []spec.Request{
					{
						Method:     http.MethodGet,
						URL:        "https://api.nowhere.com/v1/items/1234",
						NoRedirect: true,
					},
				},
			},
		},
		{
			name: "with headers",
			file: spec.File{
				Name: "with headers",
				Requests: []spec.Request{
					{
						Method: http.MethodGet,
						URL:    "https://jsonplaceholder.typicode.com/todos/1",
						Headers: map[string]string{
							"Content-Type":    "application/json",
							"Accept":          "application/json",
							"User-Agent":      "go.followtheprocess.codes/zap test",
							"X-Custom-Header": "yes",
						},
					},
				},
			},
		},
		{
			name: "with timeouts",
			file: spec.File{
				Name: "with timeouts",
				Requests: []spec.Request{
					{
						Method:            http.MethodDelete,
						URL:               "https://somewhere.org/api",
						ConnectionTimeout: 1 * time.Second,
						Timeout:           15 * time.Second,
					},
				},
			},
		},
		{
			name: "with body",
			file: spec.File{
				Name: "with body",
				Requests: []spec.Request{
					{
						Method: http.MethodPost,
						URL:    "https://somewhere.org/api/items/1",
						Body:   []byte(`{"stuff":"here"}`),
					},
				},
			},
		},
		{
			name: "with body file",
			file: spec.File{
				Name: "with body file",
				Requests: []spec.Request{
					{
						Method:   http.MethodPost,
						URL:      "https://somewhere.org/api/items/1",
						BodyFile: "a/file.txt",
					},
				},
			},
		},
		{
			name: "with response file",
			file: spec.File{
				Name: "with response file",
				Requests: []spec.Request{
					{
						Method:       http.MethodGet,
						URL:          "https://api.elsehwere.new/users/1",
						ResponseFile: "response.200.json",
					},
				},
			},
		},
		{
			name: "with multiple",
			file: spec.File{
				Name: "with multiple",
				Requests: []spec.Request{
					{
						Method: http.MethodGet,
						URL:    "https://api.nowhere.com/v1/items/1234",
					},
					{
						Method:     http.MethodGet,
						URL:        "https://api.nowhere.com/v1/items/1234",
						NoRedirect: true,
					},
					{
						Method:       http.MethodGet,
						URL:          "https://api.elsehwere.new/users/1",
						ResponseFile: "response.200.json",
					},
					{
						Method: http.MethodGet,
						URL:    "https://jsonplaceholder.typicode.com/todos/1",
						Headers: map[string]string{
							"Content-Type":    "application/json",
							"Accept":          "application/json",
							"User-Agent":      "go.followtheprocess.codes/zap test",
							"X-Custom-Header": "yes",
						},
					},
					{
						Method:            http.MethodDelete,
						URL:               "https://somewhere.org/api",
						ConnectionTimeout: 1 * time.Second,
						Timeout:           15 * time.Second,
					},
					{
						Method: http.MethodPost,
						URL:    "https://somewhere.org/api/items/1",
						Body:   []byte(`{"stuff":"here"}`),
					},
					{
						Method:   http.MethodPost,
						URL:      "https://somewhere.org/api/items/1",
						BodyFile: "a/file.txt",
					},
					{
						Method:       http.MethodGet,
						URL:          "https://api.elsehwere.new/users/1",
						ResponseFile: "response.200.json",
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
				snapshot.Color(os.Getenv("CI") == ""),
			)
			exporter := format.CurlExporter{}
			buf := &bytes.Buffer{}
			test.Ok(t, exporter.Export(buf, tt.file))

			snap.Snap(buf.String())
		})
	}
}
