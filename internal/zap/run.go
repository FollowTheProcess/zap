package zap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"go.followtheprocess.codes/hue"
	"go.followtheprocess.codes/log"
	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/resolver"
)

// Styles.
const (
	// headerKeyStyle is the style used for printing header keys
	// like Content-Type when we show the response on the command line.
	headerKeyStyle = hue.Cyan

	// dimmed is the style used for printing informational content like
	// response duration or request name.
	dimmed = hue.BrightBlack

	// success is the style used to render successful HTTP response status lines.
	success = hue.Green | hue.Bold

	// failure is the style used to render failed HTTP response status lines.
	failure = hue.Red | hue.Bold

	// sepWidth is the width in characters of the horizontal line separator
	// between HTTP responses.
	sepWidth = 80
)

const (
	defaultFilePermissions = 0o644 // Default permissions for writing files, same as unix touch
	defaultDirPermissions  = 0o755 // Default permissions for creating directories, same as unix mkdir
)

// RunOptions are the options passed to the run subcommand.
type RunOptions struct {
	// File is the path of the http file.
	File string

	// Output is the output format in which to display the HTTP response.
	//
	// Allowed values: 'stdout', 'json', 'yaml'.
	Output string

	// Requests are the names of specific requests to be run.
	//
	// Empty or nil means run all requests in the file.
	// Mutually exclusive with Filter and Pattern.
	Requests []string

	// Timeout is the overall per-request timeout.
	Timeout time.Duration

	// ConnectionTimeout is the per-request connection timeout.
	ConnectionTimeout time.Duration

	// OverallTimeout is the overall timeout, used when running multiple requests.
	OverallTimeout time.Duration

	// NoRedirect, if true, disables following http redirects.
	NoRedirect bool

	// Debug enables debug logging.
	Debug bool

	// Verbose shows additional details about the request, by default
	// only the status and the body are shown.
	Verbose bool
}

// Validate reports whether the RunOptions is valid, returning an error
// if it's not.
//
// nil means the options are valid.
func (r RunOptions) Validate() error {
	switch output := r.Output; output {
	case "stdout", "json", "yaml":
		// Nothing, these are fine
	default:
		return fmt.Errorf("invalid option for --output %q, allowed values are 'stdout', 'json', 'yaml'", output)
	}

	switch {
	case r.Timeout == 0:
		return errors.New("timeout cannot be 0")
	case r.ConnectionTimeout == 0:
		return errors.New("connection-timeout cannot be 0")
	case r.OverallTimeout == 0:
		return errors.New("overall-timeout cannot be 0")
	case r.ConnectionTimeout >= r.OverallTimeout:
		return fmt.Errorf(
			"connection-timeout (%s) cannot be larger than overall-timeout (%s)",
			r.ConnectionTimeout,
			r.OverallTimeout,
		)
	case r.ConnectionTimeout >= r.Timeout:
		return fmt.Errorf("connection-timeout (%s) cannot be larger than timeout (%s)", r.ConnectionTimeout, r.Timeout)
	case r.Timeout >= r.OverallTimeout:
		return fmt.Errorf("timeout (%s) cannot be larger than overall-timeout (%s)", r.Timeout, r.OverallTimeout)
	default:
		return nil
	}
}

// Run implements the run subcommand.
func (z Zap) Run(
	ctx context.Context,
	handler syntax.ErrorHandler,
	options RunOptions,
) error {
	if err := options.Validate(); err != nil {
		return err
	}

	logger := z.logger.Prefixed("run")

	ctx, cancel := context.WithTimeout(ctx, options.OverallTimeout)
	defer cancel()

	if len(options.Requests) == 0 {
		logger.Debug("Executing all requests in file", slog.String("file", options.File))
	} else {
		logger.Debug("Executing specific request(s) in file", slog.String("file", options.File), slog.Any("requests", options.Requests))
	}

	logger.Debug("Run configuration", slog.String("options", fmt.Sprintf("%+v", options)))

	start := time.Now()

	httpFile, err := z.parseFile(options.File, handler)
	if err != nil {
		return err
	}

	logger.Debug(
		"Parsed file successfully",
		slog.String("file", options.File),
		slog.Duration("took", time.Since(start)),
	)

	client := NewHTTPClient(httpFile)

	httpFile, err = z.evaluateGlobalPrompts(logger, httpFile)
	if err != nil {
		return fmt.Errorf("could not evaluate global prompts: %w", err)
	}

	var toExecute []spec.Request

	if len(options.Requests) == 0 {
		// No filter, so execute all the requests
		toExecute = httpFile.Requests
	} else {
		// Only execute the ones asked for (if they exist)
		for _, actualRequest := range httpFile.Requests {
			if slices.Contains(options.Requests, actualRequest.Name) {
				toExecute = append(toExecute, actualRequest)
			}
		}
	}

	if len(toExecute) == 0 {
		return fmt.Errorf("no matching requests for names %v in %s", options.Requests, options.File)
	}

	logger.Debug("Filtered requests to execute", slog.Int("count", len(toExecute)))

	toExecute, err = z.evaluateRequestPrompts(logger, toExecute, httpFile.Prompts)
	if err != nil {
		return fmt.Errorf("could not evaluate request prompts: %w", err)
	}

	for _, request := range toExecute {
		logger.Debug(
			"Executing request",
			slog.String("request", request.Name),
			slog.String("method", request.Method),
			slog.String("url", request.URL),
		)

		response, err := z.doRequest(ctx, logger, client, request)
		if err != nil {
			return err
		}

		base := filepath.Dir(options.File)

		if request.ResponseFile != "" {
			err := z.writeResponseFile(logger, base, request.ResponseFile, response.Body)
			if err != nil {
				return err
			}
		}

		z.showResponse(options.File, request, response, options.Verbose)
	}

	return nil
}

// doRequest executes a single HTTP request.
func (z Zap) doRequest(
	ctx context.Context,
	logger *log.Logger,
	client http.Client,
	request spec.Request,
) (Response, error) {
	timeout := DefaultTimeout
	if request.Timeout != 0 {
		timeout = request.Timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if request.NoRedirect {
		logger.Debug("No-Redirect was set on request", slog.String("request", request.Name))

		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequestWithContext(ctx, request.Method, request.URL, strings.NewReader(request.Body))
	if err != nil {
		return Response{}, fmt.Errorf("HTTP request %q is invalid: %w", request.Name, err)
	}

	req.Header = request.Headers
	req.Header.Add("User-Agent", "go.followtheprocess.codes/zap "+z.version)

	start := time.Now()

	res, err := client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("HTTP response error: %w", err)
	}

	if res == nil {
		return Response{}, errors.New("nil *http.Response from client.Do")
	}
	defer res.Body.Close()

	duration := time.Since(start)

	logger.Debug(
		"Received HTTP response from URL",
		slog.String("url", request.URL),
		slog.Int("status", res.StatusCode),
		slog.String("content-type", res.Header.Get("Content-Type")),
		slog.Duration("duration", duration),
	)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Response{}, fmt.Errorf("could not read HTTP response body: %w", err)
	}

	response := Response{
		Status:        res.Status,
		StatusCode:    res.StatusCode,
		Proto:         res.Proto,
		Header:        res.Header,
		Body:          body,
		ContentLength: len(body),
		Duration:      duration,
	}

	return response, nil
}

// showResponse prints the response in a user friendly way to z.stdout.
func (z Zap) showResponse(file string, request spec.Request, response Response, verbose bool) {
	fmt.Fprintln(z.stdout)

	fmt.Fprintf(z.stdout, "%s: %s\n", hue.Bold.Text(file), dimmed.Text(request.Name))

	fmt.Fprintln(z.stdout, strings.Repeat("─", sepWidth)+"\n")

	if response.StatusCode >= http.StatusBadRequest {
		fmt.Fprintf(
			z.stdout,
			"%s %s (%s)\n",
			hue.Bold.Text(response.Proto),
			failure.Text(response.Status),
			dimmed.Text(response.Duration.String()),
		)
	} else {
		fmt.Fprintf(z.stdout, "%s %s (%s)\n", hue.Bold.Text(response.Proto), success.Text(response.Status), dimmed.Text(response.Duration.String()))
	}

	fmt.Fprintln(z.stdout) // Line space

	// Only print the headers in verbose mode
	if verbose {
		for _, key := range slices.Sorted(maps.Keys(response.Header)) {
			fmt.Fprintf(z.stdout, "%s: %s\n", headerKeyStyle.Text(key), strings.Join(response.Header.Values(key), ", "))
		}

		fmt.Fprintln(z.stdout) // Line space
	}

	fmt.Fprintln(z.stdout, string(response.Body))
}

// evaluateGlobalPrompts asks the user to provide values for prompts defined in the top level
// of the parsed file and replaces the zap::prompt::<id> placeholders inserted by the parser
// with the user-provided values.
func (z Zap) evaluateGlobalPrompts(logger *log.Logger, file spec.File) (spec.File, error) {
	logger.Debug("Evaluating global prompts")

	for id, prompt := range file.Prompts {
		var value string

		err := huh.NewInput().
			Title(prompt.Name).
			Description(prompt.Description).
			Value(&value).
			WithTheme(huh.ThemeCatppuccin()).
			Run()
		if err != nil {
			return spec.File{}, fmt.Errorf("failed to prompt user for %s: %w", prompt.Name, err)
		}

		file.Prompts[id] = spec.Prompt{
			Name:        prompt.Name,
			Description: prompt.Description,
			Value:       value, // The now answered value
		}
	}

	// Go through everywhere that could hold a zap::prompt::global::<id> placeholder and replace it
	// this is just the global vars in this case
	for id, prompt := range file.Prompts {
		replacer := strings.NewReplacer(resolver.PromptPlaceholderGlobal+id, prompt.Value)

		// Global variables
		for name, variable := range file.Vars {
			replaced := replacer.Replace(variable)
			logger.Debug(
				"Replacing prompted global variable",
				slog.String("name", name),
				slog.String("from", variable),
				slog.String("to", replaced),
			)
			file.Vars[name] = replaced
		}
	}

	return file, nil
}

// evaluateRequestPrompts asks the user to provide values for prompts defined in the particular requests
// and replaces the zap::prompt::<id> placeholders inserted by the parser with the user-provided values.
//
// This method differs from it's global counterpart in that request variables could reference either
// prompts defined locally or those in the global scope. To this end, the global prompts are passed in
// as an argument to this method.
func (z Zap) evaluateRequestPrompts(
	logger *log.Logger,
	requests []spec.Request,
	globals map[string]spec.Prompt,
) ([]spec.Request, error) {
	evaluated := make([]spec.Request, 0, len(requests))

	for _, request := range requests {
		logger.Debug("Evaluating request prompts", slog.String("request", request.Name))

		allPrompts := make(map[string]spec.Prompt, len(globals)+len(request.Prompts))
		maps.Copy(allPrompts, globals) // Copy in the global prompts

		for id, prompt := range request.Prompts {
			var value string

			err := huh.NewInput().
				Title(fmt.Sprintf("(%s) %s", request.Name, id)).
				Description(prompt.Description).
				Value(&value).
				WithTheme(huh.ThemeCatppuccin()).
				Run()
			if err != nil {
				return nil, fmt.Errorf("failed to prompt user for %s: %w", prompt.Name, err)
			}

			allPrompts[id] = spec.Prompt{
				Name:        prompt.Name,
				Description: prompt.Description,
				Value:       value, // The now answered value
			}
		}

		// Go through everywhere that could hold a zap::prompt::<id> placeholder and replace it
		for id, prompt := range allPrompts {
			replacer := strings.NewReplacer(
				resolver.PromptPlaceholderLocal+id, prompt.Value,
				resolver.PromptPlaceholderGlobal+id, prompt.Value,
			)

			// Request variables
			for name, variable := range request.Vars {
				replaced := replacer.Replace(variable)
				logger.Debug(
					"Replacing prompted local variable",
					slog.String("name", name),
					slog.String("from", variable),
					slog.String("to", replaced),
				)
				request.Vars[name] = replaced
			}

			// The request URL
			replaced := replacer.Replace(request.URL)
			logger.Debug(
				"Replacing prompted request URL",
				slog.String("from", request.URL),
				slog.String("to", replaced),
			)
			request.URL = replacer.Replace(request.URL)

			// Headers
			for name, header := range request.Headers {
				replaced := make([]string, 0, len(header))
				for _, part := range header {
					newPart := replacer.Replace(part)
					logger.Debug(
						"Replacing prompted request Header",
						slog.String("from", part),
						slog.String("to", newPart),
					)
					replaced = append(replaced, newPart)
				}

				request.Headers[name] = replaced
			}
		}

		evaluated = append(evaluated, request)
	}

	return evaluated, nil
}

// evaluateAllPrompts evaluates global and all request prompts in the file, this is primarily used
// when exporting entire files into 3rd party formats as all variables need to be resolved.
func (z Zap) evaluateAllPrompts(logger *log.Logger, file spec.File) (spec.File, error) {
	file, err := z.evaluateGlobalPrompts(logger, file)
	if err != nil {
		return spec.File{}, err
	}

	// Evaluate all prompts for all requests
	requests, err := z.evaluateRequestPrompts(logger, file.Requests, file.Prompts)
	if err != nil {
		return spec.File{}, err
	}

	file.Requests = requests

	return file, nil
}

// writeResponseFile writes the response body to the file specified in the request.
func (z Zap) writeResponseFile(logger *log.Logger, base, file string, body []byte) error {
	path := filepath.Join(base, file)
	logger.Debug("Writing response body to file", slog.String("file", path))

	// The path might have arbitrary directories, so we should create
	// whatever needs creating
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, defaultDirPermissions); err != nil {
		return fmt.Errorf("could not create response file directory: %w", err)
	}

	// For writing to files, append a trailing newline if one is not already present
	body = fixNL(body)

	if err := os.WriteFile(path, body, defaultFilePermissions); err != nil {
		return fmt.Errorf("could not create response file: %w", err)
	}

	return nil
}

// If data is empty or ends in \n, fixNL returns data.
// Otherwise fixNL returns a new slice consisting of data with a final \n added.
func fixNL(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] == '\n' {
		return data
	}

	d := make([]byte, len(data)+1)
	copy(d, data)
	d[len(data)] = '\n'

	return d
}
