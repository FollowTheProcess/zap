package resolver //nolint:testpackage // environment is intentionally internal.

import (
	"testing"

	"go.followtheprocess.codes/test"
)

func TestEnvironment(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		env := newEnvironment()

		got, err := env.get("anything")
		test.Err(t, err)
		test.Equal(t, got, "")
	})

	t.Run("full", func(t *testing.T) {
		env := newEnvironment()

		test.Ok(t, env.define("something", "here"))
		test.Ok(t, env.define("other", "too"))

		// Try and define "something" again in the same scope
		test.Err(t, env.define("something", "else"))

		something, err := env.get("something")
		test.Ok(t, err)
		test.Equal(t, something, "here")

		other, err := env.get("other")
		test.Ok(t, err)
		test.Equal(t, other, "too")
	})

	t.Run("parent", func(t *testing.T) {
		env := newEnvironment()

		// Define some globals
		test.Ok(t, env.define("something", "here"))
		test.Ok(t, env.define("other", "too"))

		// Create a child scope
		child := env.child()

		// Define some locals
		test.Ok(t, child.define("more", "here"))
		test.Ok(t, child.define("another", "yes"))

		// Override a global with a local
		test.Ok(t, child.define("something", "child something value"))

		// Use the child to access
		other, err := child.get("other")
		test.Ok(t, err)
		test.Equal(t, other, "too") // Comes from globals

		more, err := child.get("more")
		test.Ok(t, err)
		test.Equal(t, more, "here") // Comes from locals

		something, err := child.get("something")
		test.Ok(t, err)
		test.Equal(t, something, "child something value") // Prefers local scope

		something, err = env.get("something")
		test.Ok(t, err)
		test.Equal(t, something, "here") // Using the global env again
	})
}
