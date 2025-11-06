package format

import (
	"encoding/json"
	"io"

	"go.followtheprocess.codes/zap/internal/spec"
)

// JSONExporter is an [Exporter] that transforms .http files into JSON documents.
type JSONExporter struct{}

// Export implements [Exporter] for [JSONExporter] and exports the given file
// as a complete JSON document.
func (j JSONExporter) Export(w io.Writer, file spec.File) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	return encoder.Encode(file)
}
