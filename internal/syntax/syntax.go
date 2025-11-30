// Package syntax handles parsing the raw .http file text into meaningful
// data structures and implements the tokeniser and parser as well as some
// language level integration tests.
package syntax

import (
	"cmp"
	"fmt"
)

// Position is an arbitrary source file position including file, line
// and column information. It can also express a range of source via StartCol
// and EndCol, this is useful for error reporting.
//
// Positions without filenames are considered invalid, in the case of stdin
// the string "stdin" may be used.
type Position struct {
	Name     string `json:"name"`     // Filename
	Offset   int    `json:"offset"`   // Byte offset of the position from the start of the file
	Line     int    `json:"line"`     // Line number (1 indexed)
	StartCol int    `json:"startCol"` // Start column (1 indexed)
	EndCol   int    `json:"endCol"`   // End column (1 indexed), EndCol == StartCol when pointing to a single character
}

// IsValid reports whether the [Position] describes a valid source position.
//
// The rules are:
//
//   - At least Name, Line and StartCol must be set (and non zero)
//   - EndCol cannot be 0, it's only allowed values are StartCol or any number greater than StartCol
func (p Position) IsValid() bool {
	if p.Name == "" || p.Line < 1 || p.StartCol < 1 || p.EndCol < 1 ||
		(p.EndCol >= 1 && p.EndCol < p.StartCol) {
		return false
	}

	return true
}

// String returns a string representation of a [Position].
//
// It is formatted such that most text editors/terminals will be able to support clicking on it
// and navigating to the position.
//
// Depending on which fields are set, the string returned will be different:
//
//   - "file:line:start-end": valid position pointing to a range of text on the line
//   - "file:line:start": valid position pointing to a single character on the line (EndCol == StartCol)
//
// At least Name, Line and StartCol must be present for a valid position, and Line and StarCol must be > 0.
// If not, an error string will be returned.
func (p Position) String() string {
	if !p.IsValid() {
		return fmt.Sprintf(
			"BadPosition: {Name: %q, Line: %d, StartCol: %d, EndCol: %d}",
			p.Name,
			p.Line,
			p.StartCol,
			p.EndCol,
		)
	}

	if p.StartCol == p.EndCol {
		// No range, just a single position
		return fmt.Sprintf("%s:%d:%d", p.Name, p.Line, p.StartCol)
	}

	return fmt.Sprintf("%s:%d:%d-%d", p.Name, p.Line, p.StartCol, p.EndCol)
}

// ComparePosition is like [cmp.Compare] for a [syntax.Position].
//
// If x and y are equal ComparePosition returns 0.
//
// If x and y refer to the same file, it returns [cmp.Compare] of
// the two offsets.
//
// If the positions refer to different files, they are compared alphabetically.
func ComparePosition(x, y Position) int {
	if x == y {
		return 0
	}

	if x.Name == y.Name {
		return cmp.Compare(x.Offset, y.Offset)
	}

	return cmp.Compare(x.Name, y.Name)
}

// Diagnostic is a syntax level diagnostic.
type Diagnostic struct {
	Msg      string   `json:"msg"`      // A descriptive message explaining the error
	Position Position `json:"position"` // The source position the diagnostic points to.
}

// String prints a [Diagnostic].
func (d Diagnostic) String() string {
	return d.Position.String() + ": " + d.Msg + "\n"
}
