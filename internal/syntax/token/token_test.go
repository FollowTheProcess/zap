package token_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/token"
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
