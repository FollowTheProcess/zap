package zap

import (
	"net"
	"net/http"
	"time"

	"go.followtheprocess.codes/zap/internal/spec"
)

// HTTP config.
const (
	// DefaultOverallTimeout is the default amount of time allowed for the entire
	// execution. Typically only used when executing multiple requests as a collection.
	DefaultOverallTimeout = 1 * time.Minute

	// DefaultTimeout is the default amount of time allowed for the entire request/response
	// cycle for a single request.
	DefaultTimeout = 30 * time.Second

	// DefaultConnectionTimeout is the default amount of time allowed for the HTTP connection/TLS handshake
	// for a single request.
	DefaultConnectionTimeout = 10 * time.Second

	maxIdleConns          = 100
	idleConnTimeout       = 90 * time.Second
	expectContinueTimeout = 1 * time.Second
)

// NewHTTPClient returns a new HTTP client configured with default values.
func NewHTTPClient(file spec.File) http.Client {
	return http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: file.Timeout,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   file.ConnectionTimeout,
			ExpectContinueTimeout: expectContinueTimeout,
			ForceAttemptHTTP2:     true,
			MaxIdleConnsPerHost:   http.DefaultMaxIdleConnsPerHost,
		},
		Timeout: file.Timeout,
	}
}

// Response is a compact version of a [http.Response] with only the data we need
// to display a HTTP response to a user.
type Response struct {
	Header        http.Header   // Response headers
	Status        string        // E.g. "200 OK"
	Proto         string        // e.g. "HTTP/1.2"
	Body          spec.Body     // The read body
	StatusCode    int           // HTTP status code
	ContentLength int           // len(Body)
	Duration      time.Duration // Duration of the request/response round trip
}
