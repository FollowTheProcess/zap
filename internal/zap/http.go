package zap

import (
	"net"
	"net/http"
	"time"
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
func NewHTTPClient() http.Client {
	return http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   DefaultConnectionTimeout,
				KeepAlive: DefaultTimeout,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   DefaultConnectionTimeout,
			ExpectContinueTimeout: expectContinueTimeout,
			ForceAttemptHTTP2:     true,
			MaxIdleConnsPerHost:   http.DefaultMaxIdleConnsPerHost,
		},
		Timeout: DefaultTimeout,
	}
}
