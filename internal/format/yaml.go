package format

import (
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
