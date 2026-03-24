package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// HostOverride holds the configuration for host header override.
// This is useful for testing with vhosts where you need to connect to an IP address
// but send a different hostname in the Host header.
type HostOverride struct {
	// Host is the hostname to use in the Host header (e.g., "account.pinner.xyz")
	Host string
	// Target is the IP address to connect to (e.g., "127.0.0.1:8080")
	Target string
}

// hostOverrideRoundTripper is a custom http.RoundTripper that overrides the Host header
// and redirects requests to a target IP address. This is useful for testing with vhosts.
type hostOverrideRoundTripper struct {
	transport http.RoundTripper
	host      string
	target    string
}

// RoundTrip implements the http.RoundTripper interface.
// It modifies the request to use the target URL while keeping the original Host header.
func (h *hostOverrideRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a copy of the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	// Override the Host header with the configured host
	reqCopy.Host = h.host

	// Parse the original URL
	originalURL := req.URL

	// Create a new URL with the target address
	// If target doesn't have a scheme, use the original request's scheme
	targetStr := h.target
	if !strings.Contains(targetStr, "://") {
		targetStr = originalURL.Scheme + "://" + targetStr
	}
	targetURL, err := url.Parse(targetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL %q: %w", h.target, err)
	}

	// Replace the URL scheme, host, and port with the target
	reqCopy.URL.Scheme = targetURL.Scheme
	reqCopy.URL.Host = targetURL.Host

	// Keep the original path, query, and fragment
	reqCopy.URL.Path = originalURL.Path
	reqCopy.URL.RawQuery = originalURL.RawQuery
	reqCopy.URL.Fragment = originalURL.Fragment

	// Ensure we don't override the Host header in the request
	// The Host field is already set above, but we need to make sure
	// it's not lost when the request is sent
	if h.host != "" {
		reqCopy.Host = h.host
	}

	// Use the underlying transport to make the request
	return h.transport.RoundTrip(reqCopy)
}

// NewHostOverrideRoundTripper creates a new host override round tripper.
func NewHostOverrideRoundTripper(host, target string) http.RoundTripper {
	return &hostOverrideRoundTripper{
		transport: http.DefaultTransport,
		host:      host,
		target:    target,
	}
}

// NewHostOverrideRoundTripperWithTransport creates a new host override round tripper with a custom transport.
func NewHostOverrideRoundTripperWithTransport(transport http.RoundTripper, host, target string) http.RoundTripper {
	return &hostOverrideRoundTripper{
		transport: transport,
		host:      host,
		target:    target,
	}
}

// JWTClientOption creates a request editor function that adds a JWT bearer token.
// The token is added to the Authorization header as "Bearer {token}".
func JWTClientOption(token string) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

// DisableClientWrapper wraps an HTTP client to disable redirect following.
type DisableClientWrapper struct {
	client      *http.Client
	disablePtr  *bool
}

// NewDisableClientWrapper creates a wrapper that controls redirect behavior.
func NewDisableClientWrapper(disablePtr *bool, transport http.RoundTripper) *DisableClientWrapper {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if *disablePtr {
				return http.ErrUseLastResponse
			}
			return nil
		},
		Transport: transport,
	}
	return &DisableClientWrapper{
		client:     httpClient,
		disablePtr: disablePtr,
	}
}

// Client returns the HTTP client.
func (w *DisableClientWrapper) Client() *http.Client {
	return w.client
}

// DisableRedirects sets the disable flag.
func (w *DisableClientWrapper) DisableRedirects() {
	*w.disablePtr = true
}

// EnableRedirects clears the disable flag.
func (w *DisableClientWrapper) EnableRedirects() {
	*w.disablePtr = false
}

// BuildHTTPClient creates an HTTP client with optional host override and redirect control.
func BuildHTTPClient(disablePtr *bool, hostOverride *HostOverride) *http.Client {
	transport := http.DefaultTransport
	
	if hostOverride != nil {
		transport = NewHostOverrideRoundTripper(hostOverride.Host, hostOverride.Target)
	}

	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if disablePtr != nil && *disablePtr {
				return http.ErrUseLastResponse
			}
			return nil
		},
		Transport: transport,
	}
}

// BuildHTTPClientWithRedirect creates an HTTP client with custom redirect handler.
func BuildHTTPClientWithRedirect(disablePtr bool, transport http.RoundTripper) *http.Client {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if disablePtr {
				return http.ErrUseLastResponse
			}
			return nil
		},
		Transport: transport,
	}
	return client
}
