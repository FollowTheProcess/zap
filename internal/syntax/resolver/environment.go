package resolver

import "fmt"

// environment is a scoped environment for the resolver.
type environment struct {
	values map[string]string
	parent *environment
}

// newEnvironment creates a new, empty [environment] with no parent.
func newEnvironment() *environment {
	return &environment{
		values: make(map[string]string),
		parent: nil,
	}
}

// define defines a new variable in the innermost scope.
func (e *environment) define(key, value string) error {
	if _, exists := e.values[key]; exists {
		return fmt.Errorf("variable %s already defined", key)
	}

	e.values[key] = value

	return nil
}

// get walks up the scope to find a variable by name, if it reaches the outermost
// scope without finding it, it returns an error.
func (e *environment) get(key string) (string, error) {
	if value, ok := e.values[key]; ok {
		return value, nil
	}

	if e.parent != nil {
		return e.parent.get(key)
	}

	return "", fmt.Errorf("use of undeclared variable %s", key)
}

// child creates a new empty [environment] using the calling one as a parent.
func (e *environment) child() *environment {
	return &environment{
		values: make(map[string]string),
		parent: e,
	}
}
