package format

import (
	"fmt"
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

// TOMLImporter is an [Importer] that transforms valid TOML documents representing
// a .http file into a [spec.File].
type TOMLImporter struct{}

// Import implements [Importer] for [TOMLImporter].
func (t TOMLImporter) Import(r io.Reader) (spec.File, error) {
	var file spec.File

	decoder := toml.NewDecoder(r)

	_, err := decoder.Decode(&file)
	if err != nil {
		return spec.File{}, fmt.Errorf("could not decode TOML: %w", err)
	}

	return file, nil
}
