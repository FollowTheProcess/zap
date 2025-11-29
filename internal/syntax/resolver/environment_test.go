package resolver_test

import (
	"testing"

	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/zap/internal/syntax/resolver"
)

func TestEnvironment(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		env := resolver.NewEnvironment()

		got, err := env.Get("anything")
		test.Err(t, err)
		test.Equal(t, got, "")
	})

	t.Run("full", func(t *testing.T) {
		env := resolver.NewEnvironment()

		test.Ok(t, env.Define("something", "here"))
		test.Ok(t, env.Define("other", "too"))

		// Try and define "something" again in the same scope
		test.Err(t, env.Define("something", "else"))

		something, err := env.Get("something")
		test.Ok(t, err)
		test.Equal(t, something, "here")

		other, err := env.Get("other")
		test.Ok(t, err)
		test.Equal(t, other, "too")
	})

	t.Run("parent", func(t *testing.T) {
		env := resolver.NewEnvironment()

		// Define some globals
		test.Ok(t, env.Define("something", "here"))
		test.Ok(t, env.Define("other", "too"))

		// Create a child scope
		child := env.Child()

		// Define some locals
		test.Ok(t, child.Define("more", "here"))
		test.Ok(t, child.Define("another", "yes"))

		// Override a global with a local
		test.Ok(t, child.Define("something", "child something value"))

		// Use the child to access
		other, err := child.Get("other")
		test.Ok(t, err)
		test.Equal(t, other, "too") // Comes from globals

		more, err := child.Get("more")
		test.Ok(t, err)
		test.Equal(t, more, "here") // Comes from locals

		something, err := child.Get("something")
		test.Ok(t, err)
		test.Equal(t, something, "child something value") // Prefers local scope

		something, err = env.Get("something")
		test.Ok(t, err)
		test.Equal(t, something, "here") // Using the global env again
	})
}
