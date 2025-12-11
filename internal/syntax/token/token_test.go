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

func TestPrecedence(t *testing.T) {
	tests := []struct {
		kind token.Kind // Token kind under test
		want int        // Expected precedence
	}{
		// Currently the only one with any precedence
		{kind: token.OpenInterp, want: token.HighestPrecedence},
		// Some others
		{kind: token.Text, want: token.LowestPrecedence},
		{kind: token.Separator, want: token.LowestPrecedence},
		{kind: token.Ident, want: token.LowestPrecedence},
		{kind: token.Colon, want: token.LowestPrecedence},
		{kind: token.Comment, want: token.LowestPrecedence},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			tok := token.Token{Kind: tt.kind} // Position info doesn't matter
			test.Equal(t, tok.Precedence(), tt.want)
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    token.Token
		wantErr bool
	}{
		{
			name:  "valid Ident",
			input: "<Token::Ident start=10, end=20>",
			want:  token.Token{Kind: token.Ident, Start: 10, End: 20},
		},
		{
			name:  "valid EOF",
			input: "<Token::EOF start=0, end=0>",
			want:  token.Token{Kind: token.EOF, Start: 0, End: 0},
		},
		{
			name:  "valid Header",
			input: "<Token::Header start=123, end=456>",
			want:  token.Token{Kind: token.Header, Start: 123, End: 456},
		},
		{
			name:    "missing prefix",
			input:   "Token::Ident start=10, end=20>",
			wantErr: true,
		},
		{
			name:    "missing suffix",
			input:   "<Token::Ident start=10, end=20",
			wantErr: true,
		},
		{
			name:    "missing space before fields",
			input:   "<Token::Identstart=10, end=20>",
			wantErr: true,
		},
		{
			name:    "invalid kind",
			input:   "<Token::NotARealKind start=10, end=20>",
			wantErr: true,
		},
		{
			name:    "missing start field",
			input:   "<Token::Ident end=20>",
			wantErr: true,
		},
		{
			name:    "missing end field",
			input:   "<Token::Ident start=10>",
			wantErr: true,
		},
		{
			name:    "invalid numeric format",
			input:   "<Token::Ident start=ten, end=20>",
			wantErr: true,
		},
		{
			name:    "extra trailing characters",
			input:   "<Token::Ident start=10, end=20>garbage",
			wantErr: true,
		},
		{
			name:    "extra internal whitespace not allowed",
			input:   "<Token::Ident   start=10, end=20>",
			wantErr: true,
		},
		{
			name:    "comma missing",
			input:   "<Token::Ident start=10 end=20>",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := token.Parse(tt.input)
			test.WantErr(t, err, tt.wantErr)

			if err == nil {
				test.Equal(t, got, tt.want)
			}
		})
	}
}

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

func FuzzParse(f *testing.F) {
	f.Add("<Token::Ident start=10, end=20>")
	f.Add("<Token::EOF start=0, end=0>")
	f.Add("<Token::Header start=123, end=456>")
	f.Add("<Token::MethodPost start=1, end=999>")
	f.Add("<Token::EOF start=0, end=0>")
	f.Add("<Token::At start=-1, end=20>")               // invalid numeric
	f.Add("<Token::MethodGet start=10, end=-1>")        // invalid numeric
	f.Add("<Token::Name start=, end=>")                 // broken fields
	f.Add("Token::ConnectionTimeout start=10, end=20>") // missing prefix
	f.Add("<Token::Ident start=10, end=20")             // missing suffix

	f.Fuzz(func(t *testing.T, input string) {
		tok, err := token.Parse(input)
		if err != nil {
			// Parser should reject malformed input safely.
			return
		}

		// If no error, ensure round-trip integrity:
		// String() -> ParseToken() must be reversible.
		roundTrip := tok.String()
		gotBack, err := token.Parse(roundTrip)
		test.Ok(t, err, test.Context("Round trip failed"))
		test.Equal(t, gotBack, tok)

		// Indexes should be positive
		test.False(t, tok.Start < 0, test.Context("start (%d) less than zero: %#v", tok.Start, tok))
		test.False(t, tok.End < 0, test.Context("end (%d) less than zero: %#v", tok.End, tok))

		// End should always be >= Start
		test.True(t, tok.End >= tok.Start, test.Context("End (%d) should always be >= Start (%d)", tok.End, tok.Start))

		// Kind must be in the valid range
		test.True(
			t,
			tok.Kind >= token.EOF && tok.Kind <= token.MethodTrace,
			test.Context("invalid Kind parsed: %#v", tok),
		)
	})
}
