package meta

//go:generate go tool oapi-codegen -config ../oai-codegen-meta.yaml ../specs/meta/meta.yaml

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"go.lumeweb.com/portal-sdk/internal/http"
	internalmeta "go.lumeweb.com/portal-sdk/internal/meta"
)

// Common error sentinels re-exported from internal/http for consumer use.
var (
	// ErrUnauthorized is returned when authentication fails (e.g., invalid JWT token).
	ErrUnauthorized = http.ErrUnauthorized

	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = http.ErrNotFound

	// ErrForbidden is returned when the user lacks permission for the operation.
	ErrForbidden = http.ErrForbidden

	// ErrBadRequest is returned when the request is invalid or malformed.
	ErrBadRequest = http.ErrBadRequest

	// ErrConflict is returned when the request conflicts with the current state.
	ErrConflict = http.ErrConflict

	// ErrInternalServer is returned when the server encounters an unexpected error.
	ErrInternalServer = http.ErrInternalServer

	// ErrUnavailable is returned when the service is temporarily unavailable.
	ErrUnavailable = http.ErrUnavailable
)

// DefaultEndpoint is the default API endpoint for the meta service.
const DefaultEndpoint = "localhost:8080"

// MetaAPI is the interface for meta (content metadata) operations.
type MetaAPI interface {
	CID() *CIDService
	Stats() *StatsService
}

// MetaClient provides access to the Meta API for querying content metadata,
// DAG structures, Sia object mappings, and aggregate statistics.
type MetaClient struct {
	client  internalmeta.ClientWithResponsesInterface
	cid     *CIDService
	stats   *StatsService
	config  *clientConfig
	jwt     string
	apiKey  string
	disableRedirect bool
}

// clientConfig holds the configuration for creating a new MetaClient.
type clientConfig struct {
	endpoint       string
	jwt            string
	apiKey         string
	hostOverride   *http.HostOverride
	disableRedirect bool
}

// ClientOption defines a configuration option for MetaClient.
type ClientOption func(*clientConfig)

// WithEndpoint sets the API endpoint for the meta client.
func WithEndpoint(endpoint string) ClientOption {
	return func(c *clientConfig) {
		c.endpoint = endpoint
	}
}

// WithJWT sets the JWT token for the meta client.
func WithJWT(token string) ClientOption {
	return func(c *clientConfig) {
		c.jwt = token
	}
}

// WithAPIKey sets the API key for the meta client.
func WithAPIKey(apiKey string) ClientOption {
	return func(c *clientConfig) {
		c.apiKey = apiKey
	}
}

// WithDisableFollowRedirect disables automatic HTTP redirects.
func WithDisableFollowRedirect() ClientOption {
	return func(c *clientConfig) {
		c.disableRedirect = true
	}
}

// WithHostOverride sets up host header override for testing with vhosts.
//
// Parameters:
//   - host: The hostname to use in the Host header (e.g., "meta.pinner.xyz")
//   - target: The IP address:port to connect to (e.g., "127.0.0.1:8080")
func WithHostOverride(host, target string) ClientOption {
	return func(c *clientConfig) {
		c.hostOverride = &http.HostOverride{
			Host:   host,
			Target: target,
		}
	}
}

// defaultClientConfig returns a clientConfig with sensible defaults.
func defaultClientConfig() *clientConfig {
	return &clientConfig{
		endpoint: DefaultEndpoint,
	}
}

// NewClient creates a new MetaClient.
// Uses DefaultEndpoint if no endpoint is provided via WithEndpoint.
func NewClient(opts ...ClientOption) (*MetaClient, error) {
	cfg := defaultClientConfig()

	for _, opt := range opts {
		opt(cfg)
	}

	clientWrapper := &MetaClient{
		jwt:             cfg.jwt,
		disableRedirect: cfg.disableRedirect,
	}

	clientOpts := []internalmeta.ClientOption{}

	httpClient := http.BuildHTTPClient(&clientWrapper.disableRedirect, cfg.hostOverride)

	clientOpts = append(clientOpts, internalmeta.WithHTTPClient(httpClient))

	if cfg.jwt != "" {
		clientOpts = append(clientOpts, internalmeta.WithRequestEditorFn(func(ctx context.Context, req *stdhttp.Request) error {
			req.Header.Set("Authorization", "Bearer "+cfg.jwt)
			return nil
		}))
	}

	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, internalmeta.WithRequestEditorFn(func(ctx context.Context, req *stdhttp.Request) error {
			req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
			return nil
		}))
	}

	c, err := internalmeta.NewClientWithResponses(cfg.endpoint, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create meta client: %w", err)
	}

	cidService := &CIDService{client: c}
	statsService := &StatsService{client: c}

	clientWrapper.client = c
	clientWrapper.cid = cidService
	clientWrapper.stats = statsService
	clientWrapper.config = cfg
	return clientWrapper, nil
}

// CID returns the CID service for querying content metadata.
func (m *MetaClient) CID() *CIDService {
	return m.cid
}

// Stats returns the stats service for querying aggregate and protocol statistics.
func (m *MetaClient) Stats() *StatsService {
	return m.stats
}

// RequestExecutor provides a method to execute requests with the meta client's configuration.
func (m *MetaClient) RequestExecutor(ctx context.Context) internalmeta.ClientWithResponsesInterface {
	return m.client
}
