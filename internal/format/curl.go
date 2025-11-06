package format

import (
	_ "embed"
	"fmt"
	"io"
	"strings"
	"text/template"

	"go.followtheprocess.codes/zap/internal/spec"
)

//go:embed templates/curl.txt.tmpl
var curlTempl string

// curlFunctions are custom template functions available in the curlTemplate.
var curlFunctions = template.FuncMap{
	"trim": strings.TrimSpace,
}

// curlTemplate is the parsed curl command line text/template.
var curlTemplate = template.Must(template.New("curl").Funcs(curlFunctions).Parse(curlTempl))

// CurlExporter is an [Exporter] that transforms .http files into curl shell scripts.
type CurlExporter struct{}

// Export implements [Exporter] for [CurlExporter] and exports the given
// file as one or more curl snippets.
func (c CurlExporter) Export(w io.Writer, file spec.File) error {
	if err := curlTemplate.Execute(w, file); err != nil {
		return fmt.Errorf("could not export file %s to curl: %w", file.Name, err)
	}

	return nil
}
