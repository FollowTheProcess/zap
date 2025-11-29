package resolver

import "fmt"

// Environment is a scoped environment for the resolver.
type Environment struct {
	values map[string]string
	parent *Environment
}

// NewEnvironment creates a new, empty [Environment] with no parent.
func NewEnvironment() *Environment {
	return &Environment{
		values: make(map[string]string),
		parent: nil,
	}
}

// Define defines a new variable in the innermost scope.
func (e *Environment) Define(key, value string) error {
	if _, exists := e.values[key]; exists {
		return fmt.Errorf("variable %s already defined", key)
	}

	if e.values == nil {
		e.values = make(map[string]string)
	}

	e.values[key] = value

	return nil
}

// Get walks up the scope to find a variable by name, if it reaches the outermost
// scope without finding it, it returns an error.
func (e *Environment) Get(key string) (string, error) {
	if value, ok := e.values[key]; ok {
		return value, nil
	}

	if e.parent != nil {
		return e.parent.Get(key)
	}

	return "", fmt.Errorf("use of undeclared variable %s", key)
}

// Child creates a new empty [Environment] using the calling one as a parent.
func (e *Environment) Child() *Environment {
	return &Environment{
		values: make(map[string]string),
		parent: e,
	}
}
