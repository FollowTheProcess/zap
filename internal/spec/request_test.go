package spec_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/spec"
)

func TestRequestAsCurl(t *testing.T) {
	tests := []struct {
		name    string       // Name of the test case
		request spec.Request // The HTTP request
	}{
		{
			name: "simple",
			request: spec.Request{
				Method: http.MethodGet,
				URL:    "https://api.nowhere.com/v1/items/1234",
			},
		},
		{
			name: "no redirect",
			request: spec.Request{
				Method:     http.MethodGet,
				URL:        "https://api.nowhere.com/v1/items/1234",
				NoRedirect: true,
			},
		},
		{
			name: "with headers",
			request: spec.Request{
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
		{
			name: "with timeouts",
			request: spec.Request{
				Method:            http.MethodDelete,
				URL:               "https://somewhere.org/api",
				ConnectionTimeout: 1 * time.Second,
				Timeout:           15 * time.Second,
			},
		},
		{
			name: "with body",
			request: spec.Request{
				Method: http.MethodPost,
				URL:    "https://somewhere.org/api/items/1",
				Body:   []byte(`{"stuff":"here"}`),
			},
		},
		{
			name: "with body file",
			request: spec.Request{
				Method:   http.MethodPost,
				URL:      "https://somewhere.org/api/items/1",
				BodyFile: "a/file.txt",
			},
		},
		{
			name: "with response file",
			request: spec.Request{
				Method:       http.MethodGet,
				URL:          "https://api.elsehwere.new/users/1",
				ResponseFile: "response.200.json",
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
			got, err := tt.request.AsCurl()
			test.Ok(t, err)

			snap.Snap(got)
		})
	}
}
