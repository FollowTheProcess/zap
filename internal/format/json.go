package format

import (
	"encoding/json"
	"fmt"
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

// JSONImporter is an [Importer] that transforms JSON representations of
// .http files into the equivalent [spec.File].
type JSONImporter struct{}

// Import implements [Importer] for [JSONImporter] and imports the given
// JSON document into a [spec.File].
func (j JSONImporter) Import(r io.Reader) (spec.File, error) {
	var file spec.File

	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&file); err != nil {
		return spec.File{}, fmt.Errorf("could not decode JSON: %w", err)
	}

	return file, nil
}
