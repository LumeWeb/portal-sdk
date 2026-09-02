package admin

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

//go:generate go tool oapi-codegen -config ../oai-codegen-admin.yaml ../specs/admin/quota.yaml

// Common error sentinels re-exported from internal/http for consumer use.
var (
	// ErrUnauthorized is returned when authentication fails (e.g., invalid JWT token).
	ErrUnauthorized = internalhttp.ErrUnauthorized

	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = internalhttp.ErrNotFound

	// ErrForbidden is returned when the user lacks permission for the operation.
	ErrForbidden = internalhttp.ErrForbidden

	// ErrBadRequest is returned when the request is invalid or malformed.
	ErrBadRequest = internalhttp.ErrBadRequest

	// ErrConflict is returned when the request conflicts with the current state.
	ErrConflict = internalhttp.ErrConflict

	// ErrInternalServer is returned when the server encounters an unexpected error.
	ErrInternalServer = internalhttp.ErrInternalServer

	// ErrUnavailable is returned when the service is temporarily unavailable.
	ErrUnavailable = internalhttp.ErrUnavailable
)

// DefaultEndpoint is the default API endpoint for the admin service.
const DefaultEndpoint = "localhost:8080"

// AdminAPI is the interface for admin operations.
type AdminAPI interface {
	Quota() *QuotaService
	Billing() *BillingService
	Website() *WebsiteService
	Profiling() *ProfilingService
	PlatformDomains() *PlatformDomainService
	SocialProviders() *SocialProviderService
}

// AdminClient provides access to admin APIs for managing quotas, billing, and users.
type AdminClient struct {
	client          admin.ClientWithResponsesInterface
	quota           *QuotaService
	billing         *BillingService
	website         *WebsiteService
	profiling       *ProfilingService
	platformDomains *PlatformDomainService
	socialProviders *SocialProviderService
	config          *clientConfig
	jwt             string
	apiKey          string
	disableRedirect bool
}

// clientConfig holds the configuration for creating a new AdminClient.
type clientConfig struct {
	endpoint       string
	jwt            string
	apiKey         string
	hostOverride   *internalhttp.HostOverride
	disableRedirect bool
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

// WithDisableFollowRedirect disables automatic HTTP redirects.
// This is useful for testing scenarios where you need to inspect redirect responses.
func WithDisableFollowRedirect() ClientOption {
	return func(c *clientConfig) {
		c.disableRedirect = true
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
		c.hostOverride = &internalhttp.HostOverride{
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
func NewClient(opts ...ClientOption) (*AdminClient, error) {
	cfg := defaultClientConfig()

	// Apply options to configure the client
	for _, opt := range opts {
		opt(cfg)
	}

	// Create the AdminClient first so we can reference it in the CheckRedirect closure
	clientWrapper := &AdminClient{
		jwt:             cfg.jwt,
		disableRedirect: cfg.disableRedirect,
	}

	// Build admin client options
	clientOpts := []admin.ClientOption{}

	// Create HTTP client with redirect control and optional host override using shared utilities
	httpClient := internalhttp.BuildHTTPClient(&clientWrapper.disableRedirect, cfg.hostOverride)

	// Add the HTTP client to client options
	clientOpts = append(clientOpts, admin.WithHTTPClient(httpClient))

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

	c, err := admin.NewClientWithResponses(cfg.endpoint, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}

	quotaService := &QuotaService{
		client: c,
	}

	billingService := &BillingService{
		client: c,
	}

	websiteService := &WebsiteService{
		client: c,
	}

	profilingService := &ProfilingService{
		client: c,
	}

	platformDomainService := &PlatformDomainService{
		client: c,
	}

	socialProviderService := &SocialProviderService{
		client: c,
	}

	clientWrapper.client = c
	clientWrapper.quota = quotaService
	clientWrapper.billing = billingService
	clientWrapper.website = websiteService
	clientWrapper.profiling = profilingService
	clientWrapper.platformDomains = platformDomainService
	clientWrapper.socialProviders = socialProviderService
	clientWrapper.config = cfg
	return clientWrapper, nil
}

// Quota returns the quota service for managing quotas.
func (a *AdminClient) Quota() *QuotaService {
	return a.quota
}

// Billing returns the billing service for managing billing operations.
func (a *AdminClient) Billing() *BillingService {
	return a.billing
}

// Website returns the website service for managing IPFS websites.
func (a *AdminClient) Website() *WebsiteService {
	return a.website
}

// Profiling returns the profiling service for managing Go runtime profiling.
func (a *AdminClient) Profiling() *ProfilingService {
	return a.profiling
}

// PlatformDomains returns the platform domain service for managing platform-owned
// root domains that users can claim subdomains under.
func (a *AdminClient) PlatformDomains() *PlatformDomainService {
	return a.platformDomains
}

// SocialProviders returns the social provider service for managing configured
// social login providers.
func (a *AdminClient) SocialProviders() *SocialProviderService {
	return a.socialProviders
}

// RequestExecutor provides a method to execute requests with the admin client's configuration.
func (a *AdminClient) RequestExecutor(ctx context.Context) admin.ClientWithResponsesInterface {
	return a.client
}
