package traefik_real_ip

import (
	"context"
	"net"
	"net/http"
	"strings"
	"strconv"
)

const (
	xRealIP       = "X-Real-Ip"
	xForwardedFor = "X-Forwarded-For"
)

// Config the plugin configuration.
type Config struct {
	ForwardedForDepth int `json:"forwardedForDepth,omitempty" toml:"forwardedForDepth,omitempty" yaml:"forwardedForDepth,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		ForwardedForDepth: 1, // Default depth if not provided
	}
}

// RealIPOverWriter is a plugin that extracts real IP from X-Forwarded-For header.
type RealIPOverWriter struct {
	next             http.Handler
	name             string
	ForwardedForDepth int
}

// New creates a new RealIPOverWriter plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	ipOverWriter := &RealIPOverWriter{
		next:             next,
		name:             name,
		ForwardedForDepth: config.ForwardedForDepth,
	}

	return ipOverWriter, nil
}

func (r *RealIPOverWriter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	forwardedIPs := strings.Split(req.Header.Get(xForwardedFor), ",")

	// Determine the index to use based on ForwardedForDepth
	index := len(forwardedIPs) - r.ForwardedForDepth
	if index < 0 {
		index = 0
	}

	trimmedIP := strings.TrimSpace(forwardedIPs[index])
	if trimmedIP != "" {
		req.Header.Set(xRealIP, trimmedIP)
	}

	r.next.ServeHTTP(rw, req)
}
