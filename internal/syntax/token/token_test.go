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
