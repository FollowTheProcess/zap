package spec

import (
	"fmt"
)

// Prompt represents a variable that requires the user to specify by responding to a prompt.
type Prompt struct {
	// Name of the variable into which to store the user provided value
	Name string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty"`

	// Description of the prompt, optional
	Description string `json:"description,omitempty" toml:"description,omitempty" yaml:"description,omitempty"`

	// Value is the current value for the prompt variable, empty if
	// not yet provided
	Value string `json:"value,omitempty" toml:"value,omitempty" yaml:"value,omitempty"`
}

// String implements [fmt.Stringer] for a [Prompt].
func (p Prompt) String() string {
	if p.Description != "" {
		return fmt.Sprintf("@prompt %s %s\n", p.Name, p.Description)
	}

	return fmt.Sprintf("@prompt %s\n", p.Name)
}
