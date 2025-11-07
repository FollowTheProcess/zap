// Package format provides mechanisms for format conversions into and from .http files.
//
// Notably, the package provides the [Importer] and [Exporter] interfaces for doing this
// in a format-agnostic way.
//
// It also provides the built in importers and exporters such as JSON, YAML, Curl, Postman etc.
package format

import (
	"io"

	"go.followtheprocess.codes/zap/internal/spec"
)

// TODO(@FollowTheProcess): The body is being formatted as raw bytes, which users probably don't want
// lets implement MarshalText or something that is smarter than that

// Exporter is the interface defining a mechanism for exporting a .http file
// into an external format.
type Exporter interface {
	// Export exports the [spec.File] into an external format, written to w.
	Export(w io.Writer, file spec.File) error
}

// Importer is the interface defining a mechanism for importing external formats
// into .http files.
type Importer interface {
	// Import imports the data from the external format into a [spec.File].
	Import(r io.Reader) (spec.File, error)
}
