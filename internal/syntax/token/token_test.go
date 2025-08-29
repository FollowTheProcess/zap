package token_test

import (
	"fmt"
	"testing"
	"testing/quick"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/token"
)

func TestString(t *testing.T) {
	// All we really care about is the format, let's let quick handle it!
	f := func(tok token.Token) bool {
		return tok.String() == fmt.Sprintf(
			"<Token::%s start=%d, end=%d>",
			tok.Kind.String(),
			tok.Start,
			tok.End,
		)
	}

	err := quick.Check(f, nil)
	if err != nil {
		t.Fatal(err)
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
