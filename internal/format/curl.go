package format

import (
	_ "embed"
	"io"
	"strings"
	"text/template"

	"go.followtheprocess.codes/zap/internal/spec"
)

// TODO(@FollowTheProcess): Curl importer might be tricky?
//
// We'll need to parse the curl command as a shell script basically, could use
// mvdan/sh for that

// TODO(@FollowTheProcess): It dumps the body as raw text but doesn't correctly escape it to be valid
//
// Multi-line JSON needs to be minified. Probably what we should do is if it's up to a certain limit
// size, minify it and pass it inline. Otherwise save it to a file somewhere and pass it in as data?

//go:embed templates/curl.txt.tmpl
var curlTempl string

// minifier is a [strings.Replacer] that removes all whitespace from a string.
//
//nolint:gochecknoglobals // Also has to be here
var minifier = strings.NewReplacer(
	"\t", "",
	"\n", "",
	"\v", "",
	"\f", "",
	"\r", "",
	" ", "",
)

// curlFunctions are custom template functions available in the curlTemplate.
//
//nolint:gochecknoglobals // This has to be here
var curlFunctions = template.FuncMap{
	"minify": minifier.Replace,
}

// curlTemplate is the parsed curl command line text/template.
//
//nolint:gochecknoglobals // Having the template as a global means it's parsed only once
var curlTemplate = template.Must(template.New("curl").Funcs(curlFunctions).Parse(curlTempl))

// CurlExporter is an [Exporter] that transforms .http files into curl shell scripts.
type CurlExporter struct{}

// Export implements [Exporter] for [CurlExporter] and exports the given
// file as one or more curl snippets.
func (c CurlExporter) Export(w io.Writer, file spec.File) error {
	return curlTemplate.Execute(w, file)
}
