package token_test

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

var (
	// Everything else has these, this allows passing -update or -clean to go test ./...
	// and not getting a flag not defined error.
	_ = flag.Bool("update", false, "Update snapshots")
	_ = flag.Bool("clean", false, "Clean all snapshots and recreate")
)

func FuzzTokenString(f *testing.F) {
	// Generate some random integers as seeds
	for range 100 {
		f.Add(rand.Int(), rand.Int(), rand.Int())
	}

	f.Fuzz(func(t *testing.T, kind, start, end int) {
		tok := token.Token{
			Kind:  token.Kind(kind),
			Start: start,
			End:   end,
		}

		got := tok.String()

		// It should always look like this, regardless of the numbers
		want := fmt.Sprintf("<Token::%s start=%d, end=%d>", token.Kind(kind), start, end)

		test.Equal(t, got, want)
	})
}

func TestKeyword(t *testing.T) {
	tests := []struct {
		text string     // Text input
		want token.Kind // Expected token Kind return
		ok   bool       // Expected ok return
	}{
		{text: "name", want: token.Name, ok: true},
		{text: "timeout", want: token.Timeout, ok: true},
		{text: "connection-timeout", want: token.ConnectionTimeout, ok: true},
		{text: "no-redirect", want: token.NoRedirect, ok: true},
		{text: "something-else", want: token.Ident, ok: false},
		{text: "base", want: token.Ident, ok: false},
		{text: "myVar", want: token.Ident, ok: false},
		{text: "lots of random crap", want: token.Ident, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := token.Keyword(tt.text)
			test.Equal(t, ok, tt.ok)
			test.Equal(t, got, tt.want)
		})
	}
}

func TestMethod(t *testing.T) {
	tests := []struct {
		text string     // Text input
		want token.Kind // Expected token Kind return
		ok   bool       // Expected ok return
	}{
		{text: "GET", want: token.MethodGet, ok: true},
		{text: "HEAD", want: token.MethodHead, ok: true},
		{text: "POST", want: token.MethodPost, ok: true},
		{text: "PUT", want: token.MethodPut, ok: true},
		{text: "DELETE", want: token.MethodDelete, ok: true},
		{text: "CONNECT", want: token.MethodConnect, ok: true},
		{text: "PATCH", want: token.MethodPatch, ok: true},
		{text: "OPTIONS", want: token.MethodOptions, ok: true},
		{text: "TRACE", want: token.MethodTrace, ok: true},
		{text: "word", want: token.Text, ok: false},
		{text: "patch", want: token.Text, ok: false},
		{text: "get", want: token.Text, ok: false},
		{text: "post", want: token.Text, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := token.Method(tt.text)
			test.Equal(t, ok, tt.ok)
			test.Equal(t, got, tt.want)
		})
	}
}
