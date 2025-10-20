package zap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"go.followtheprocess.codes/hue"
	"go.followtheprocess.codes/log"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
)

// Styles.
const (
	// headerKeyStyle is the style used for printing header keys
	// like Content-Type when we show the response on the command line.
	headerKeyStyle = hue.Cyan

	// dimmed is the style used for printing informational content like
	// response duration or request name.
	dimmed = hue.BrightBlack | hue.Italic

	// success is the style used to render successful HTTP response status lines.
	success = hue.Green | hue.Bold

	// failure is the style used to render failed HTTP response status lines.
	failure = hue.Red | hue.Bold

	// sepWidth is the width in characters of the horizontal line separator
	// between HTTP responses.
	sepWidth = 80
)

// RunOptions are the options passed to the run subcommand.
type RunOptions struct {
	// Output is the name of a file in which to save the response, if empty,
	// the response is printed to stdout.
	Output string

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
}

// Run implements the run subcommand.
func (z Zap) Run(ctx context.Context, file string, requests []string, handler syntax.ErrorHandler, options RunOptions) error {
	logger := z.logger.Prefixed("run")

	ctx, cancel := context.WithTimeout(ctx, DefaultOverallTimeout)
	defer cancel()

	if len(requests) == 0 {
		logger.Debug("Executing all requests in file", slog.String("file", file))
	} else {
		logger.Debug("Executing specific request(s) in file", slog.String("file", file), slog.Any("requests", requests))
	}

	// TODO(@FollowTheProcess): Is it worth making the options all implement slog.LogValuer?
	logger.Debug("Run configuration", slog.String("options", fmt.Sprintf("%+v", options)))

	start := time.Now()

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	p, err := parser.New(file, f, handler)
	if err != nil {
		return fmt.Errorf("could not initialise the parser: %w", err)
	}

	parsed, err := p.Parse()
	if err != nil {
		return err
	}

	logger.Debug("Parsed file successfully", slog.String("file", file), slog.Duration("took", time.Since(start)))

	// TODO(@FollowTheProcess): Resolve the timeouts and other stuff
	// If it's in the file, use that. If not fall back to defaults

	client := NewHTTPClient()

	// TODO(@FollowTheProcess): Evaluate prompts here and fill in the values

	var toExecute []syntax.Request

	if len(requests) == 0 {
		// No filter, so execute all the requests
		toExecute = parsed.Requests
	} else {
		// Only execute the ones asked for (if they exist)
		for _, actualRequest := range parsed.Requests {
			if slices.Contains(requests, actualRequest.Name) {
				toExecute = append(toExecute, actualRequest)
			}
		}
	}

	if len(toExecute) == 0 {
		return fmt.Errorf("no matching requests for names %v in %s", requests, file)
	}

	logger.Debug("Filtered requests to execute", slog.Int("count", len(toExecute)))

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

		z.showResponse(file, request, response)
	}

	return nil
}

// Response is a compact version of a [http.Response] with only the data we need
// to display a HTTP response to a user.
type Response struct {
	Header        http.Header   // Response headers
	Status        string        // E.g. "200 OK"
	Proto         string        // e.g. "HTTP/1.2"
	Body          []byte        // The read body
	StatusCode    int           // HTTP status code
	ContentLength int           // len(Body)
	Duration      time.Duration // Duration of the request/response round trip
}

// doRequest executes a single HTTP request.
func (z Zap) doRequest(ctx context.Context, logger *log.Logger, client http.Client, request syntax.Request) (Response, error) {
	// TODO(@FollowTheProcess): Use request timeout once it's been resolved
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	if request.NoRedirect {
		logger.Debug("No-Redirect was set on request", slog.String("request", request.Name))

		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequestWithContext(ctx, request.Method, request.URL, bytes.NewReader(request.Body))
	if err != nil {
		return Response{}, fmt.Errorf("HTTP request %q is invalid: %w", request.Name, err)
	}

	for key, value := range request.Headers {
		req.Header.Add(key, value)
	}

	req.Header.Add("User-Agent", "go.followtheprocess.codes/zap "+z.version)

	start := time.Now()

	res, err := client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("HTTP response error: %w", err)
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

	// TODO(@FollowTheProcess): Do we really want to ReadAll?
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
func (z Zap) showResponse(file string, request syntax.Request, response Response) {
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

	for _, key := range slices.Sorted(maps.Keys(response.Header)) {
		fmt.Fprintf(z.stdout, "%s: %s\n", headerKeyStyle.Text(key), response.Header.Get(key))
	}

	fmt.Fprintln(z.stdout) // Line space

	fmt.Fprintln(z.stdout, string(response.Body))
}
