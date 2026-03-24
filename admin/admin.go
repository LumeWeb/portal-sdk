package admin

import (
	"context"
	stdhttp "net/http"

	"go.lumeweb.com/portal-sdk/internal/admin"
	"go.lumeweb.com/portal-sdk/internal/http"
)

//go:generate go tool oapi-codegen -config ../oai-codegen-admin.yaml ../specs/admin/quota.yaml

// DefaultEndpoint is the default API endpoint for the admin service.
const DefaultEndpoint = "localhost:8080"

// AdminAPI is the interface for admin operations.
type AdminAPI interface {
	Quota() *QuotaService
}

// AdminClient provides access to admin APIs for managing quotas, billing, and users.
type AdminClient struct {
	client       admin.ClientWithResponsesInterface
	quota        *QuotaService
	config       *clientConfig
	jwt          string
	apiKey       string
}

// clientConfig holds the configuration for creating a new AdminClient.
type clientConfig struct {
	endpoint    string
	jwt         string
	apiKey      string
	hostOverride *http.HostOverride
}

// ClientOption defines a configuration option for AdminClient.
type ClientOption func(*clientConfig)

// WithEndpoint sets the API endpoint for the admin client.
func WithEndpoint(endpoint string) ClientOption {
	return func(c *clientConfig) {
		c.endpoint = endpoint
	}
}

// WithJWT sets the JWT token for the admin client.
func WithJWT(token string) ClientOption {
	return func(c *clientConfig) {
		c.jwt = token
	}
}

// WithAPIKey sets the API key for the admin client.
func WithAPIKey(apiKey string) ClientOption {
	return func(c *clientConfig) {
		c.apiKey = apiKey
	}
}

// WithHostOverride sets up host header override for testing with vhosts.
// This allows connecting to a specific IP address while sending a different hostname in the Host header.
//
// Parameters:
//   - host: The hostname to use in the Host header (e.g., "admin.pinner.xyz")
//   - target: The IP address:port to connect to (e.g., "127.0.0.1:8080")
//
// Example:
//
//	client := admin.NewClient(
//	    admin.WithHostOverride("admin.pinner.xyz", "127.0.0.1:8080"),
//	)
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

// NewClient creates a new AdminClient.
// Uses DefaultEndpoint if no endpoint is provided via WithEndpoint.
func NewClient(opts ...ClientOption) *AdminClient {
	cfg := defaultClientConfig()

	// Apply options to configure the client
	for _, opt := range opts {
		opt(cfg)
	}

	// Build admin client options
	clientOpts := []admin.ClientOption{}

	// Create HTTP client with host override if configured
	var httpClient *stdhttp.Client
	if cfg.hostOverride != nil {
		// Create a simple HTTP client with host override for testing
		// No redirect control needed for admin API initially
		disableRedirect := false
		httpClient = http.BuildHTTPClient(&disableRedirect, cfg.hostOverride)
	}

	// Add the HTTP client if one was created
	if httpClient != nil {
		clientOpts = append(clientOpts, admin.WithHTTPClient(httpClient))
	}

	// Add JWT request editor if provided
	if cfg.jwt != "" {
		clientOpts = append(clientOpts, admin.WithRequestEditorFn(func(ctx context.Context, req *stdhttp.Request) error {
			req.Header.Set("Authorization", "Bearer "+cfg.jwt)
			return nil
		}))
	}

	// Add API key request editor if provided
	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, admin.WithRequestEditorFn(func(ctx context.Context, req *stdhttp.Request) error {
			req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
			return nil
		}))
	}

	c, _ := admin.NewClientWithResponses(cfg.endpoint, clientOpts...)

	quotaService := &QuotaService{
		client: c,
	}

	return &AdminClient{
		client: c,
		quota:  quotaService,
		config: cfg,
	}
}

// Quota returns the quota service for managing quotas.
func (a *AdminClient) Quota() *QuotaService {
	return a.quota
}

// RequestExecutor provides a method to execute requests with the admin client's configuration.
func (a *AdminClient) RequestExecutor(ctx context.Context) admin.ClientWithResponsesInterface {
	return a.client
}
