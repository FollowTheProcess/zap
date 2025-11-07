package format

import (
	"io"

	"github.com/BurntSushi/toml"
	"go.followtheprocess.codes/zap/internal/spec"
)

// TOMLExporter is an [Exporter] that transforms .http files into TOML documents.
type TOMLExporter struct{}

// Export implements [Exporter] for [TOMLExporter] and exports the given file
// as a complete TOML document.
func (t TOMLExporter) Export(w io.Writer, file spec.File) error {
	encoder := toml.NewEncoder(w)
	encoder.Indent = ""

	return encoder.Encode(file)
}
