package format

import (
	"fmt"
	"io"

	"go.followtheprocess.codes/zap/internal/spec"
	"go.yaml.in/yaml/v4"
)

const yamlIndent = 2

// YAMLExporter is an [Exporter] that transforms .http files into YAML documents.
type YAMLExporter struct{}

// Export implements [Exporter] for [YAMLExporter] and exports the given file as
// a complete YAML document.
func (y YAMLExporter) Export(w io.Writer, file spec.File) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(yamlIndent)

	return encoder.Encode(file)
}

// YAMLImporter is an [Importer] that transforms valid YAML representations of .http files
// into a [spec.File].
type YAMLImporter struct{}

// Import implements [Importer] for a [YAMLImporter].
func (y YAMLImporter) Import(r io.Reader) (spec.File, error) {
	var file spec.File

	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	if err := decoder.Decode(&file); err != nil {
		return spec.File{}, fmt.Errorf("could not decode YAML: %w", err)
	}

	return file, nil
}
