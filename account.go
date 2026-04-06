package account

//go:generate go tool oapi-codegen -config oai-codegen.yaml specs/account.yaml

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.lumeweb.com/portal-sdk/internal/client"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
	"go.lumeweb.com/queryutil"
	"go.lumeweb.com/queryutil/filter/serializer"
)

// OperationStatus represents the status of an account operation.
type OperationStatus string

const (
	OperationStatusPending   OperationStatus = "pending"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusCompleted OperationStatus = "completed"
	OperationStatusFailed    OperationStatus = "failed"
	OperationStatusError     OperationStatus = "error"
)

// DefaultSettledStates are the default operation statuses considered "settled" (finished).
var DefaultSettledStates = []OperationStatus{
	OperationStatusCompleted,
	OperationStatusFailed,
	OperationStatusError,
}

// RateLimiterFunc is a function that checks whether an operation is allowed based on available quota.
// The function takes a context and the requested size, and returns true if allowed, false if not,
// or an error if the quota check cannot be performed.
type RateLimiterFunc func(ctx context.Context, size int64) (bool, error)

// Named error types for error comparison
var (
	// ErrOperationTimeout is returned when WaitForOperation times out waiting for an operation to settle.
	ErrOperationTimeout = errors.New("operation timed out")

	// ErrUnauthorized is returned when authentication fails (e.g., invalid JWT token).
	ErrUnauthorized = errors.New("unauthorized")

	// InsufficientQuotaError is returned when a requested operation would exceed available quota.
	InsufficientQuotaError = errors.New("insufficient quota")
)

// Operation identifiers for error message mapping.
const (
	// AuthTokenCookie is the name of the cookie that contains the JWT token.
	AuthTokenCookie = "auth_token"

	// AuthTokenQueryParam is the query parameter name for the JWT token in redirect URLs.
	AuthTokenQueryParam = "auth_token"
)

const (
	OpLogin = iota
	OpOTPValidation
	OpPing
	OpOTPGeneration
	OpOTPVerification
	OpOTPDisable
	OpAPIKeyLogin
	OpRegistration
	OpEmailVerification
	OpResendEmailVerification
	OpDeleteAccount
	OpPasswordResetRequest
	OpPasswordResetConfirm
	OpPasswordUpdate
	OpGetAccount
	OpUpdateProfile
	OpGetAvatar
	OpUploadAvatar
	OpUpdateEmail
	OpGetPermissions
	OpGetQuota
	OpGetQuotaHistory
)

const defaultOperationName = "operation"

// DefaultEndpoint is the default API endpoint for the account service.
const DefaultEndpoint = "account.pinner.xyz"

// operationString maps operation IDs to their string names.
var operationString = map[int]string{
	OpLogin:        "login",
	OpOTPValidation: "OTP validation",
	OpPing:         "ping",
	OpOTPGeneration: "OTP generation",
	OpOTPVerification: "OTP verification",
	OpOTPDisable:   "OTP disable",
	OpAPIKeyLogin:  "API key login",
	OpRegistration: "registration",
	OpEmailVerification: "email verification",
	OpResendEmailVerification: "resend email verification",
	OpDeleteAccount: "account deletion",
	OpPasswordResetRequest: "password reset request",
	OpPasswordResetConfirm: "password reset confirm",
	OpPasswordUpdate: "password update",
	OpGetAccount:   "get account",
	OpUpdateProfile: "update profile",
	OpGetAvatar:    "get avatar",
	OpUploadAvatar: "upload avatar",
	OpUpdateEmail:  "update email",
	OpGetPermissions: "get account permissions",
	OpGetQuota:      "get quota status",
	OpGetQuotaHistory: "get quota history",
}

// errorFactory is a helper for creating errors with optional ErrUnauthorized wrapping.
type errorFactory struct {
	wrapErr bool
	message string
}

// Error creates the actual error.
func (ef errorFactory) Error() error {
	if ef.wrapErr {
		return fmt.Errorf("%w: %s", ErrUnauthorized, ef.message)
	}
	return fmt.Errorf("%s", ef.message)
}

// authErr creates an error factory that wraps with ErrUnauthorized.
func authErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, message: msg}
}

// plainErr creates an error factory without wrapping.
func plainErr(msg string) errorFactory {
	return errorFactory{wrapErr: false, message: msg}
}

// httpErrorMessages maps operation IDs to their custom status code error messages.
// This provides a centralized, DRY way to handle HTTP error responses.
var httpErrorMessages = map[int]map[int]errorFactory{
	OpLogin: {
		http.StatusUnauthorized: authErr("invalid login credentials"),
	},
	OpOTPValidation: {
		http.StatusBadRequest:   plainErr("invalid OTP code"),
		http.StatusUnauthorized: authErr("invalid or expired 2FA session"),
	},
	OpPing: {
		http.StatusUnauthorized: authErr("invalid JWT token"),
	},
	OpOTPGeneration: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpOTPVerification: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid OTP code"),
	},
	OpOTPDisable: {
		http.StatusUnauthorized: authErr("authentication required or invalid password"),
	},
	OpAPIKeyLogin: {
		http.StatusUnauthorized: authErr("invalid API key"),
		http.StatusForbidden:    plainErr("account is pending deletion"),
	},
	OpRegistration: {
		http.StatusConflict: plainErr("user already exists with this email"),
	},
	OpEmailVerification: {
		http.StatusBadRequest: plainErr("invalid verification token or email"),
		http.StatusNotFound:   plainErr("user not found"),
	},
	OpResendEmailVerification: {
		http.StatusBadRequest: plainErr("invalid email address"),
		http.StatusNotFound:   plainErr("user not found"),
	},
	OpDeleteAccount: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("cannot delete account"),
		http.StatusNotFound:     plainErr("account not found"),
	},
	OpPasswordResetRequest: {
		http.StatusBadRequest: plainErr("invalid email address"),
		http.StatusNotFound:   plainErr("user not found"),
	},
	OpPasswordResetConfirm: {
		http.StatusBadRequest: plainErr("invalid or expired reset token"),
		http.StatusNotFound:   plainErr("user not found"),
	},
	OpPasswordUpdate: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid password"),
		http.StatusNotFound:     plainErr("user not found"),
	},
	OpGetAccount: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("account not found"),
	},
	OpUpdateProfile: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid profile data"),
		http.StatusNotFound:     plainErr("user not found"),
	},
	OpGetAvatar: {
		http.StatusBadRequest:   plainErr("bad request"),
		http.StatusNotFound:     plainErr("avatar not found"),
	},
	OpUploadAvatar: {
		http.StatusBadRequest:plainErr("bad request"),
		http.StatusNotFound:    plainErr("not found"),
	},
	OpUpdateEmail: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest: plainErr("invalid email or password"),
		http.StatusNotFound:   plainErr("user not found"),
	},
	OpGetPermissions: {
		http.StatusBadRequest:  plainErr("bad request"),
		http.StatusNotFound:     plainErr("permissions not found"),
	},
	OpGetQuota: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("bad request"),
		http.StatusNotFound:     plainErr("quota not found"),
	},
	OpGetQuotaHistory: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid date parameters"),
		http.StatusNotFound:     plainErr("quota history not found"),
	},
}

// IsSettled returns true if the operation is in a settled state (finished, no longer being processed).
func (s OperationStatus) IsSettled() bool {
	return s == OperationStatusCompleted || s == OperationStatusFailed || s == OperationStatusError
}

// String returns the string representation of the operation status.
func (s OperationStatus) String() string {
	return string(s)
}

// APIKey represents an API key for the account.
// Embeds the generated client.CreateAPIKeyResponse to reuse all fields.
type APIKey struct {
	client.CreateAPIKeyResponse
}

// Operation represents an account operation (upload, pin, etc.).
// Embeds the generated client.OperationDetailResponse to reuse all fields.
type Operation struct {
	client.OperationDetailResponse
}

// OperationListItem represents an operation from a list response.
type OperationListItem struct {
	client.OperationListItem
}

// OperationFilters represents the available filter options for operations.
// Embeds the generated client.OperationFiltersResponse to reuse all fields.
type OperationFilters struct {
	client.OperationFiltersResponse
}

// OperationFilterItem represents a filter item (e.g., a specific status, protocol, or operation type).
type OperationFilterItem struct {
	client.OperationFilterItem
}

// AccountInfo represents user account information.
// Embeds the generated client.AccountInfoResponse to reuse all fields.
type AccountInfo struct {
	client.AccountInfoResponse
}

// AccountPermissions represents the access control policies and model for the authenticated user.
// Embeds the generated client.AccountPermissionsResponse to reuse all fields.
type AccountPermissions struct {
	client.AccountPermissionsResponse
}

// QuotaTypeStatus represents the usage status for a single quota type (upload/download/storage).
// Embeds the generated client.QuotaTypeStatus to reuse all fields.
type QuotaTypeStatus struct {
	client.QuotaTypeStatus
}

// QuotaStatus represents the current quota status for the authenticated user.
// Embeds the generated client.QuotaStatusResponse to reuse all fields.
type QuotaStatus struct {
	client.QuotaStatusResponse
}

// UsagePoint represents a data point in quota history with timestamp and byte count.
// Embeds the generated client.UsagePoint to reuse all fields.
type UsagePoint struct {
	client.UsagePoint
}

// QuotaHistory represents historical quota usage data for charting and analytics.
// Embeds the generated client.QuotaHistoryResponse to reuse all fields.
type QuotaHistory struct {
	client.QuotaHistoryResponse
}

// Filter is a filter for listing operations (legacy, use ListOptions instead).
type Filter struct {
	Status OperationStatus
	CID    string
	Limit  int
}

// ListOptions provides options for listing operations with filters, sorting, and pagination.
type ListOptions struct {
	Filters    []queryutil.CrudFilter
	Sorts      []queryutil.Sort
	Pagination *queryutil.Pagination
	Search     string
}

// ListOption is a function that modifies ListOptions.
type ListOption func(*ListOptions)

// WithFilters adds filters to the list options.
func WithFilters(filters ...queryutil.CrudFilter) ListOption {
	return func(opts *ListOptions) {
		opts.Filters = filters
	}
}

// WithSorts adds sorting to the list options.
func WithSorts(sorts ...queryutil.Sort) ListOption {
	return func(opts *ListOptions) {
		opts.Sorts = sorts
	}
}

// WithPagination adds pagination to the list options.
func WithPagination(pagination *queryutil.Pagination) ListOption {
	return func(opts *ListOptions) {
		opts.Pagination = pagination
	}
}

// WithSearch adds a search term to the list options.
func WithSearch(search string) ListOption {
	return func(opts *ListOptions) {
		opts.Search = search
	}
}

// PollOption is an option for WaitForOperation.
type PollOption func(*pollConfig)

type pollConfig struct {
	interval      time.Duration
	timeout       time.Duration
	settledStates []OperationStatus
}

// WithPollInterval sets the polling interval.
func WithPollInterval(d time.Duration) PollOption {
	return func(c *pollConfig) {
		c.interval = d
	}
}

// WithPollTimeout sets the polling timeout.
func WithPollTimeout(d time.Duration) PollOption {
	return func(c *pollConfig) {
		c.timeout = d
	}
}

// WithPollSettledStates sets the operation statuses that are considered "settled".
// If not provided, defaults to [OperationStatusCompleted, OperationStatusFailed, OperationStatusError].
func WithPollSettledStates(states ...OperationStatus) PollOption {
	return func(c *pollConfig) {
		c.settledStates = states
	}
}

// defaultPollConfig returns the default polling configuration.
func defaultPollConfig() *pollConfig {
	return &pollConfig{
		interval:      2 * time.Second,
		timeout:       5 * time.Minute,
		settledStates: DefaultSettledStates,
	}
}

// checkResponseWithBody validates the HTTP status code and returns a formatted error with body
func checkResponseWithBody(statusCode int, body []byte, operation string) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf("%s failed with status %d: %s", operation, statusCode, string(body))
	}
	return nil
}

// handleResponse processes an HTTP response using the global error message map.
// op: the operation ID (used to lookup custom error messages)
// successCodes: status codes that indicate success (e.g., []int{http.StatusOK})
// Returns nil for success codes, custom error from global map, or generic error with body.
func handleResponse(statusCode int, body []byte, op int, successCodes []int) error {
	// Check if status code is in success codes
	for _, code := range successCodes {
		if statusCode == code {
			return nil
		}
	}

	// Check for custom error message in global map
	if errorMessages, ok := httpErrorMessages[op]; ok {
		if factory, ok := errorMessages[statusCode]; ok {
			return factory.Error()
		}
	}

	// Get operation name for generic error
	opName := operationString[op]
	if opName == "" {
		opName = defaultOperationName
	}

	// Generic error with body
	return fmt.Errorf("%s failed with status %d: %s", opName, statusCode, string(body))
}

// validateJSON200 validates the HTTP status code and JSON200 field, returning the data if valid
func validateJSON200[T any](statusCode int, body []byte, json200 *T, nilMsg string) (*T, error) {
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", ErrUnauthorized)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("failed with status %d: %s", statusCode, string(body))
	}
	if json200 == nil {
		return nil, fmt.Errorf("%s", nilMsg)
	}
	return json200, nil
}

// newAPIKey creates a new APIKey from a CreateAPIKeyResponse.
func newAPIKey(data client.CreateAPIKeyResponse) *APIKey {
	return &APIKey{CreateAPIKeyResponse: data}
}

// NewAPIKey creates an APIKey with the given name and token.
func NewAPIKey(name, token string) *APIKey {
	return &APIKey{CreateAPIKeyResponse: client.CreateAPIKeyResponse{
		Name:  name,
		Token: token,
	}}
}

// NewLoginResult creates a LoginResult with the given parameters.
func NewLoginResult(token string, otpRequired bool, intermediateJWT string) *LoginResult {
	return &LoginResult{
		Token:           token,
		OTPRequired:     otpRequired,
		IntermediateJWT: intermediateJWT,
	}
}

// newOperation creates a new Operation from an OperationDetailResponse.
func newOperation(data client.OperationDetailResponse) *Operation {
	return &Operation{OperationDetailResponse: data}
}

// LoginResult contains the result of a login attempt.
type LoginResult struct {
	Token           string
	OTPRequired     bool
	IntermediateJWT string
}

// AccountAPI provides account management and operation tracking functionality.
type AccountAPI interface {
	// Authentication
	// Login authenticates with email/password and returns a JWT token.
	// If 2FA is required, the returned token is an intermediate JWT and OTPRequired will be true.
	Login(ctx context.Context, email, password string) (*LoginResult, error)

	// LoginWithAPIKey authenticates using an API key and returns a JWT token.
	// The API key should be passed as the token value (JWT issued when the API key was created).
	LoginWithAPIKey(ctx context.Context, apiKey string) (string, error)

	// ValidateOTP completes 2FA login using an intermediate JWT and OTP code.
	// Returns the final JWT token on success.
	ValidateOTP(ctx context.Context, intermediateJWT, otp string) (string, error)

	// Register creates a new user account.
	Register(ctx context.Context, email, firstName, lastName, password string) error

	// VerifyEmail confirms a user's email address with a verification token.
	VerifyEmail(ctx context.Context, email, token string) error

	// ResendVerifyEmail resends the email verification link to the user's email address.
	ResendVerifyEmail(ctx context.Context, email string) error

	// Ping verifies the JWT token is valid.
	Ping(ctx context.Context) error

	// Two-Factor Authentication
	// GenerateOTP generates a new OTP secret for 2FA setup.
	GenerateOTP(ctx context.Context) (string, error)

	// VerifyOTP verifies an OTP code and enables 2FA for the account.
	VerifyOTP(ctx context.Context, otp string) error

	// DisableOTP disables 2FA for the account.
	DisableOTP(ctx context.Context, password string) error

	// API Keys
	// CreateAPIKey creates a new API key for the account.
	CreateAPIKey(ctx context.Context, name string) (*APIKey, error)

	// ListAPIKeys retrieves a list of API keys for the authenticated user.
	ListAPIKeys(ctx context.Context, opts ...ListOption) ([]*APIKey, error)

	// DeleteAPIKey deletes a specific API key for the authenticated user.
	DeleteAPIKey(ctx context.Context, keyID string) error

	// DeleteAccount initiates the process to delete the authenticated user's account.
	DeleteAccount(ctx context.Context) error

	// Account info
	// UploadLimit returns the account's upload limit in bytes.
	// This determines the threshold for using TUS resumable uploads.
	UploadLimit(ctx context.Context) (int64, error)

	// Operations
	// GetOperation retrieves details of a specific operation.
	GetOperation(ctx context.Context, id int64) (*Operation, error)

	// ListOperations lists operations with optional filters, sorting, and pagination.
	ListOperations(ctx context.Context, opts ...ListOption) ([]*Operation, error)

	// GetOperationFilters retrieves available filter values for operations.
	GetOperationFilters(ctx context.Context) (*OperationFilters, error)

	// WaitForOperation polls an operation until it reaches a settled state.
	// Settled states are: completed, failed, error.
	WaitForOperation(ctx context.Context, id int64, opts ...PollOption) (*Operation, error)

	// Password Management
	// RequestPasswordReset initiates the password reset process by sending a reset link to the user's email.
	RequestPasswordReset(ctx context.Context, email string) error

	// ConfirmPasswordReset resets the user's password using a token received via email.
	ConfirmPasswordReset(ctx context.Context, email, token, newPassword string) error

	// UpdatePassword updates the authenticated user's password.
	UpdatePassword(ctx context.Context, currentPassword, newPassword string) error

	// Profile Management
	// GetAccount retrieves information about the authenticated user's account.
	GetAccount(ctx context.Context) (*AccountInfo, error)

	// UpdateProfile updates the authenticated user's profile information (first name and last name).
	UpdateProfile(ctx context.Context, firstName, lastName string) error

	// GetAvatar retrieves the authenticated user's profile picture.
	GetAvatar(ctx context.Context) ([]byte, error)

	// UploadAvatar uploads a profile picture/avatar for the authenticated user.
	UploadAvatar(ctx context.Context, file []byte) error

	// UpdateEmail updates the authenticated user's email address.
	UpdateEmail(ctx context.Context, email, password string) error

	// GetPermissions retrieves the access control policies and model for the authenticated user.
	GetPermissions(ctx context.Context) (*AccountPermissions, error)

	// Quota Management
	// GetQuota retrieves the current quota status including upload and download usage, limits, and remaining allowance for the authenticated user.
	GetQuota(ctx context.Context) (*QuotaStatus, error)

	// GetQuotaHistory retrieves historical quota usage data for charting and analytics.
	// startDate and endDate should be in RFC3339 format.
	// usageType should be "upload", "download", or "storage".
	GetQuotaHistory(ctx context.Context, startDate, endDate, usageType string) (*QuotaHistory, error)
}

// Client implements AccountAPI using the generated OpenAPI client.
type Client struct {
	client         client.ClientWithResponsesInterface
	jwt            string
	disableRedirect bool
}

// clientConfig holds the configuration for creating a new Client.
type clientConfig struct {
	endpoint       string
	jwt            string
	httpClient     *http.Client
	hostOverride   *internalhttp.HostOverride
	disableRedirect bool
}

// defaultClientConfig returns a clientConfig with sensible defaults.
func defaultClientConfig() *clientConfig {
	return &clientConfig{
		endpoint: DefaultEndpoint,
	}
}

// ClientOption is an option for configuring the AccountAPI client.
type ClientOption func(*clientConfig)

// WithJWT sets the JWT token for authentication.
// This token will be added to the Authorization header as "Bearer {token}".
func WithJWT(jwt string) ClientOption {
	return func(cfg *clientConfig) {
		cfg.jwt = jwt
	}
}

// WithEndpoint sets the API endpoint URL.
// If not provided, DefaultEndpoint will be used.
func WithEndpoint(endpoint string) ClientOption {
	return func(cfg *clientConfig) {
		cfg.endpoint = endpoint
	}
}

// WithDisableFollowRedirect disables automatic HTTP redirects.
// This is useful for testing scenarios where you need to inspect redirect responses.
func WithDisableFollowRedirect() ClientOption {
	return func(cfg *clientConfig) {
		cfg.disableRedirect = true
	}
}

// WithHostOverride sets up host header override for testing with vhosts.
// This allows connecting to a specific IP address while sending a different hostname in the Host header.
//
// Parameters:
//   - host: The hostname to use in the Host header (e.g., "account.pinner.xyz")
//   - target: The IP address:port to connect to (e.g., "127.0.0.1:8080")
//
// Example:
//
//	client := account.NewClient(
//	    account.WithHostOverride("account.pinner.xyz", "127.0.0.1:8080"),
//	)
func WithHostOverride(host, target string) ClientOption {
	return func(cfg *clientConfig) {
		cfg.hostOverride = &internalhttp.HostOverride{
			Host:   host,
			Target: target,
		}
	}
}

// NewClient creates a new AccountAPI client.
// Uses DefaultEndpoint if no endpoint is provided via WithEndpoint.
func NewClient(opts ...ClientOption) AccountAPI {
	cfg := defaultClientConfig()

	// Apply options to configure the client
	for _, opt := range opts {
		opt(cfg)
	}

	// Create the Client first so we can reference it in the CheckRedirect closure
	clientWrapper := &Client{
		client:         nil,
		jwt:            cfg.jwt,
		disableRedirect: cfg.disableRedirect,
	}

	// Build client options from config
	clientOpts := []client.ClientOption{}

	// Add JWT request editor if provided
	if cfg.jwt != "" {
		clientOpts = append(clientOpts, client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+cfg.jwt)
			return nil
		}))
	}

	// Create HTTP client with redirect control and optional host override using shared utilities
	httpClient := internalhttp.BuildHTTPClient(&clientWrapper.disableRedirect, cfg.hostOverride)

	// Add the HTTP client to client options
	clientOpts = append(clientOpts, client.WithHTTPClient(httpClient))

	c, _ := client.NewClientWithResponses(cfg.endpoint, clientOpts...)
	clientWrapper.client = c
	return clientWrapper
}

// NewClientWithDefaults creates a new AccountAPI client with default settings (for testing).
func NewClientWithDefaults(genClient client.ClientWithResponsesInterface) AccountAPI {
	return &Client{client: genClient}
}

// disableRedirects temporarily disables HTTP redirect following for the next request.
// This is thread-safe and should be used with enableRedirects in a defer pattern.
func (c *Client) disableRedirects() {
	c.disableRedirect = true
}

// enableRedirects re-enables HTTP redirect following after a disableRedirects call.
// This is thread-safe and should be used in a defer pattern.
func (c *Client) enableRedirects() {
	c.disableRedirect = false
}

// Login authenticates with email/password and returns a login result.
// If 2FA is enabled for the account, OTPRequired will be true and the token
// is an intermediate JWT that must be used with ValidateOTP.
func (c *Client) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	reqBody := client.LoginRequest{
		Email:    email,
		Password: password,
	}

	resp, err := c.client.PostApiAuthLoginWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send login request: %w", err)
	}

	// Validate response using the global error map (login returns 200 for OTP, 302 redirect for non-OTP)
	if err := handleResponse(resp.StatusCode(), resp.Body, OpLogin, []int{http.StatusOK, http.StatusFound}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil || resp.JSON200.Token == "" {
		return nil, fmt.Errorf("login response did not contain a token")
	}

	result := &LoginResult{
		Token:           resp.JSON200.Token,
		OTPRequired:     resp.JSON200.Otp != nil && *resp.JSON200.Otp,
		IntermediateJWT: resp.JSON200.Token,
	}

	return result, nil
}

// LoginWithAPIKey authenticates using an API key and returns a JWT token.
// The API key should be passed as a JWT token (the token returned when the API key was created).
func (c *Client) LoginWithAPIKey(ctx context.Context, apiKey string) (string, error) {
	resp, err := c.client.PostApiAuthKeyWithResponse(ctx, &client.PostApiAuthKeyParams{
		Authorization: &apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send API key login request: %w", err)
	}

	// Validate response using the global error map
	if err := handleResponse(resp.StatusCode(), resp.Body, OpAPIKeyLogin, []int{http.StatusOK}); err != nil {
		return "", err
	}

	if resp.JSON200 == nil || resp.JSON200.Token == "" {
		return "", fmt.Errorf("API key login response did not contain a token")
	}

	return resp.JSON200.Token, nil
}

// ValidateOTP completes 2FA login using an intermediate JWT and OTP code.
// Returns the final JWT token on success.
func (c *Client) ValidateOTP(ctx context.Context, intermediateJWT, otp string) (string, error) {
	reqBody := client.OTPValidateRequest{
		Otp: otp,
	}

	// Temporarily disable redirect following to capture the 302 response
	// The OTP validate endpoint returns 302 with Location header containing the JWT
	originalState := c.disableRedirect
	c.disableRedirect = true
	defer func() { c.disableRedirect = originalState }()

	// Use the client with a request editor to add the intermediate JWT
	resp, err := c.client.PostApiAuthOtpValidateWithResponse(ctx, reqBody,
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+intermediateJWT)
			return nil
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to send OTP validate request: %w", err)
	}

	// Validate response using the global error map
	if err := handleResponse(resp.StatusCode(), resp.Body, OpOTPValidation, []int{http.StatusFound}); err != nil {
		return "", err
	}

	location := resp.HTTPResponse.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("OTP validation successful but no redirect location provided")
	}

	// The portal sets the final JWT in a cookie. For CLI, we need to extract it.
	// The cookie name is typically "auth" or similar. We'll look for JWT in Set-Cookie.
	cookies := resp.HTTPResponse.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == AuthTokenCookie && cookie.Value != "" {
			return cookie.Value, nil
		}
	}

	// If no cookie found, try to extract from Location header (may contain JWT as query param)
	parsedURL, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect location: %w", err)
	}
	token := parsedURL.Query().Get(AuthTokenQueryParam)
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("OTP validation successful but unable to extract final JWT from response")
}

// Ping verifies the JWT token is valid.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.client.PostApiAuthPingWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to send ping request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpPing, []int{http.StatusOK})
}

// GenerateOTP generates a new OTP secret for 2FA setup.
func (c *Client) GenerateOTP(ctx context.Context) (string, error) {
	resp, err := c.client.PostApiAuthOtpGenerateWithResponse(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to send OTP generate request: %w", err)
	}

	if err := handleResponse(resp.StatusCode(), resp.Body, OpOTPGeneration, []int{http.StatusOK}); err != nil {
		return "", err
	}

	if resp.JSON200 == nil || resp.JSON200.Otp == "" {
		return "", fmt.Errorf("OTP generation response did not contain a secret")
	}

	return resp.JSON200.Otp, nil
}

// VerifyOTP verifies an OTP code and enables 2FA for the account.
func (c *Client) VerifyOTP(ctx context.Context, otp string) error {
	reqBody := client.OTPVerifyRequest{
		Otp: otp,
	}

	resp, err := c.client.PostApiAuthOtpVerifyWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send OTP verify request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpOTPVerification, []int{http.StatusNoContent})
}

// DisableOTP disables 2FA for the account.
func (c *Client) DisableOTP(ctx context.Context, password string) error {
	reqBody := client.OTPDisableRequest{
		Password: password,
	}

	resp, err := c.client.PostApiAuthOtpDisableWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send OTP disable request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpOTPDisable, []int{http.StatusNoContent})
}

// Register creates a new user account.
func (c *Client) Register(ctx context.Context, email, firstName, lastName, password string) error {
	reqBody := client.RegisterRequest{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Password:  password,
	}

	resp, err := c.client.PostApiAuthRegisterWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send register request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpRegistration, []int{http.StatusCreated, http.StatusOK})
}

// VerifyEmail confirms a user's email address with a verification token.
func (c *Client) VerifyEmail(ctx context.Context, email, token string) error {
	reqBody := client.VerifyEmailRequest{
		Email: email,
		Token: token,
	}

	resp, err := c.client.PostApiAccountVerifyEmailWithResponse(ctx, nil, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send verify email request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpEmailVerification, []int{http.StatusOK})
}

// ResendVerifyEmail resends the email verification link to the user's email address.
func (c *Client) ResendVerifyEmail(ctx context.Context, email string) error {
	reqBody := client.ResendVerifyEmailRequest{
		Email: email,
	}

	resp, err := c.client.PostApiAccountVerifyEmailResendWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send resend verify email request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpResendEmailVerification, []int{http.StatusOK})
}

// CreateAPIKey creates a new API key for the account.
func (c *Client) CreateAPIKey(ctx context.Context, name string) (*APIKey, error) {
	reqBody := client.APIKeyCreateRequest{
		Name: name,
	}

	resp, err := c.client.PostApiAccountKeysWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send create API key request: %w", err)
	}

	data, err := validateJSON200(resp.StatusCode(), resp.Body, resp.JSON200, "create API key response did not contain data")
	if err != nil {
		return nil, err
	}

	return newAPIKey(*data), nil
}

// ListAPIKeys retrieves a list of API keys for the authenticated user.
func (c *Client) ListAPIKeys(ctx context.Context, opts ...ListOption) ([]*APIKey, error) {
	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Build request parameters
	params := &client.GetApiAccountKeysParams{}
	if options.Pagination != nil {
		params.UnderscoreStart = &options.Pagination.Start
		if options.Pagination.End != 0 {
			params.UnderscoreEnd = &options.Pagination.End
		}
	}

	resp, err := c.client.GetApiAccountKeysWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send list API keys request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to list API keys with status %d: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list API keys response did not contain data")
	}

	keys := lo.Map(resp.JSON200.Data, func(key client.APIKeyResponse, _ int) *APIKey {
		return &APIKey{CreateAPIKeyResponse: client.CreateAPIKeyResponse{
			Name:  key.Name,
			Token: "",
			Uuid:  key.Uuid,
		}}
	})

	return keys, nil
}

// DeleteAPIKey deletes a specific API key for the authenticated user.
func (c *Client) DeleteAPIKey(ctx context.Context, keyID string) error {
	_uuid, err := uuid.Parse(keyID)
	if err != nil {
		return fmt.Errorf("invalid API key UUID: %w", err)
	}
	resp, err := c.client.DeleteApiAccountKeysKeyIDWithResponse(ctx, _uuid)
	if err != nil {
		return fmt.Errorf("failed to send delete API key request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to delete API key with status %d: %s", resp.StatusCode(), string(resp.Body))
	}

	return nil
}

// DeleteAccount initiates the process to delete the authenticated user's account.
func (c *Client) DeleteAccount(ctx context.Context) error {
	resp, err := c.client.DeleteApiAccountWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to send delete account request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpDeleteAccount, []int{http.StatusOK})
}

// UploadLimit returns the account's upload limit in bytes.
func (c *Client) UploadLimit(ctx context.Context) (int64, error) {
	resp, err := c.client.GetApiUploadLimitWithResponse(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to send upload limit request: %w", err)
	}

	data, err := validateJSON200(resp.StatusCode(), resp.Body, resp.JSON200, "upload limit response did not contain data")
	if err != nil {
		return 0, err
	}

	return int64(data.Limit), nil
}

// GetOperation retrieves details of a specific operation.
func (c *Client) GetOperation(ctx context.Context, id int64) (*Operation, error) {
	resp, err := c.client.GetApiOperationsIdWithResponse(ctx, int(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get operation %d: %w", id, err)
	}

	data, err := validateJSON200(resp.StatusCode(), resp.Body, resp.JSON200, "operation response did not contain data")
	if err != nil {
		return nil, err
	}

	return newOperation(*data), nil
}

// ListOperations lists operations with optional filters, sorting, and pagination.
func (c *Client) ListOperations(ctx context.Context, opts ...ListOption) ([]*Operation, error) {
	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Use serializers to generate query parameters
	ser := serializer.NewQueryParamSerializer()

	// Serialize pagination
	pagination := options.Pagination
	if pagination == nil {
		pagination = &queryutil.DefaultPagination
	}
	paginationValues, _ := ser.SerializePagination(*pagination)

	// Serialize sorts
	sortValues, _ := ser.SerializeSorts(options.Sorts)

	// Serialize filters
	filterValues, _ := ser.SerializeFilters(options.Filters)

	// Use request editor to dynamically add query parameters
	reqEditor := func(ctx context.Context, req *http.Request) error {
		query := req.URL.Query()

		// Add search parameter
		if options.Search != "" {
			query.Set("q", options.Search)
		}

		// Add pagination parameters
		for key, values := range paginationValues {
			for _, v := range values {
				query.Add(key, v)
			}
		}

		// Add sort parameters
		for key, values := range sortValues {
			for _, v := range values {
				query.Add(key, v)
			}
		}

		// Add filter parameters
		for key, values := range filterValues {
			for _, v := range values {
				query.Add(key, v)
			}
		}

		req.URL.RawQuery = query.Encode()
		return nil
	}

	resp, err := c.client.GetApiOperationsWithResponse(ctx, &client.GetApiOperationsParams{}, reqEditor)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}

	data, err := validateJSON200(resp.StatusCode(), resp.Body, resp.JSON200, "operations response did not contain data")
	if err != nil {
		return nil, err
	}

	ops := lo.Map(data.Data, func(op client.OperationListItem, _ int) *Operation {
		return newOperation(operationListItemToDetail(op))
	})
	return ops, nil
}

// GetOperationFilters retrieves available filter values for operations.
func (c *Client) GetOperationFilters(ctx context.Context) (*OperationFilters, error) {
	resp, err := c.client.GetApiOperationsFiltersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation filters: %w", err)
	}

	data, err := validateJSON200(resp.StatusCode(), resp.Body, resp.JSON200, "operation filters response did not contain data")
	if err != nil {
		return nil, err
	}

	return &OperationFilters{OperationFiltersResponse: data.Data}, nil
}

// operationListItemToDetail converts an OperationListItem to OperationDetailResponse.
func operationListItemToDetail(op client.OperationListItem) client.OperationDetailResponse {
	return client.OperationDetailResponse{
		Cid:                   op.Cid,
		CurrentStep:           op.CurrentStep,
		Error:                 op.Error,
		EstimatedCompletionAt: op.EstimatedCompletionAt,
		Id:                    op.Id,
		Operation:             op.Operation,
		OperationDisplayName:  op.OperationDisplayName,
		ProgressPercent:       op.ProgressPercent,
		Protocol:              op.Protocol,
		ProtocolDisplayName:   op.ProtocolDisplayName,
		StartedAt:             op.StartedAt,
		Status:                op.Status,
		StatusDisplayName:     op.StatusDisplayName,
		StatusMessage:         op.StatusMessage,
		TotalSteps:            op.TotalSteps,
		UpdatedAt:             op.UpdatedAt,
	}
}

// IsSettled returns true if the operation is in a settled state.
func (op *Operation) IsSettled() bool {
	return OperationStatus(op.Status).IsSettled()
}

// WaitForOperation polls an operation until it reaches a settled state.
// Polling options match the TypeScript SDK's OperationPollingOptions:
//   - interval: polling interval in milliseconds (default: 2000ms)
//   - timeout: maximum time to wait in milliseconds (default: 300000ms = 5 minutes)
//   - settledStates: operation statuses considered "settled" (default: [completed, failed, error])
func (c *Client) WaitForOperation(ctx context.Context, id int64, opts ...PollOption) (*Operation, error) {
	cfg := defaultPollConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Create a map of settled states for O(1) lookup
	settledStatesMap := make(map[OperationStatus]bool)
	for _, state := range cfg.settledStates {
		settledStatesMap[state] = true
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: operation %d", ErrOperationTimeout, id)
		case <-ticker.C:
			op, err := c.GetOperation(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("failed to get operation %d: %w", id, err)
			}

			opStatus := OperationStatus(op.Status)
			if settledStatesMap[opStatus] {
				return op, nil
			}
		}
	}
}

// RequestPasswordReset initiates the password reset process by sending a reset link to the user's email.
func (c *Client) RequestPasswordReset(ctx context.Context, email string) error {
	reqBody := client.PasswordResetRequest{
		Email: email,
	}

	resp, err := c.client.PostApiAccountPasswordResetRequestWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to send password reset request: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpPasswordResetRequest, []int{http.StatusOK})
}

// ConfirmPasswordReset resets the user's password using a token received via email.
func (c *Client) ConfirmPasswordReset(ctx context.Context, email, token, newPassword string) error {
	reqBody := client.PasswordResetVerifyRequest{
		Email:    email,
		Token:    token,
		Password: newPassword,
	}

	resp, err := c.client.PostApiAccountPasswordResetConfirmWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to confirm password reset: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpPasswordResetConfirm, []int{http.StatusOK})
}

// UpdatePassword updates the authenticated user's password.
func (c *Client) UpdatePassword(ctx context.Context, currentPassword, newPassword string) error {
	reqBody := client.UpdatePasswordRequest{
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	}

	resp, err := c.client.PostApiAccountUpdatePasswordWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpPasswordUpdate, []int{http.StatusOK})
}

// GetAccount retrieves information about the authenticated user's account.
func (c *Client) GetAccount(ctx context.Context) (*AccountInfo, error) {
	resp, err := c.client.GetApiAccountWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account information: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, handleResponse(resp.StatusCode(), resp.Body, OpGetAccount, []int{http.StatusOK})
	}

	return &AccountInfo{AccountInfoResponse: *resp.JSON200}, nil
}

// UpdateProfile updates the authenticated user's profile information (first name and last name).
// Passing empty strings for firstName or lastName will leave those fields unchanged.
func (c *Client) UpdateProfile(ctx context.Context, firstName, lastName string) error {
	reqBody := client.UpdateProfileRequest{}

	if firstName != "" {
		reqBody.FirstName = &firstName
	}
	if lastName != "" {
		reqBody.LastName = &lastName
	}

	resp, err := c.client.PatchApiAccountWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpUpdateProfile, []int{http.StatusOK})
}

// GetAvatar retrieves the authenticated user's profile picture.
// Returns the image data as bytes.
func (c *Client) GetAvatar(ctx context.Context) ([]byte, error) {
	resp, err := c.client.GetApiAccountAvatarWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get avatar: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, handleResponse(resp.StatusCode(), resp.Body, OpGetAvatar, []int{http.StatusOK})
	}

	return resp.Body, nil
}

// UploadAvatar uploads a profile picture/avatar for the authenticated user.
// The file content should be the raw image data.
func (c *Client) UploadAvatar(ctx context.Context, file []byte) error {
	resp, err := c.client.PostApiAccountAvatarWithBodyWithResponse(ctx, http.DetectContentType(file), bytes.NewReader(file))
	if err != nil {
		return fmt.Errorf("failed to upload avatar: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpUploadAvatar, []int{http.StatusOK, http.StatusNoContent})
}

// UpdateEmail updates the authenticated user's email address.
// Requires the user's current password for verification.
func (c *Client) UpdateEmail(ctx context.Context, email, password string) error {
	reqBody := client.UpdateEmailRequest{
		Email:    email,
		Password: password,
	}

	resp, err := c.client.PostApiAccountUpdateEmailWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	return handleResponse(resp.StatusCode(), resp.Body, OpUpdateEmail, []int{http.StatusOK})
}

// GetPermissions retrieves the access control policies and model for the authenticated user.
func (c *Client) GetPermissions(ctx context.Context) (*AccountPermissions, error) {
	resp, err := c.client.GetApiAccountPermissionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account permissions: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, handleResponse(resp.StatusCode(), resp.Body, OpGetPermissions, []int{http.StatusOK})
	}

	return &AccountPermissions{AccountPermissionsResponse: *resp.JSON200}, nil
}

// GetQuota retrieves the current quota status including upload and download usage, limits, and remaining allowance for the authenticated user.
func (c *Client) GetQuota(ctx context.Context) (*QuotaStatus, error) {
	resp, err := c.client.GetApiAccountQuotaWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get quota status: %w", err)
	}

	if err := handleResponse(resp.StatusCode(), resp.Body, OpGetQuota, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get quota response did not contain data")
	}

	return &QuotaStatus{QuotaStatusResponse: *resp.JSON200}, nil
}

// GetQuotaHistory retrieves historical quota usage data for charting and analytics.
// startDate and endDate should be in RFC3339 format.
// usageType should be "upload", "download", or "bandwidth".
func (c *Client) GetQuotaHistory(ctx context.Context, startDate, endDate, usageType string) (*QuotaHistory, error) {
	params := &client.GetApiAccountQuotaHistoryParams{}
	if startDate != "" {
		params.StartDate = &startDate
	}
	if endDate != "" {
		params.EndDate = &endDate
	}
	if usageType != "" {
		params.Type = &usageType
	}

	resp, err := c.client.GetApiAccountQuotaHistoryWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get quota history: %w", err)
	}

	if err := handleResponse(resp.StatusCode(), resp.Body, OpGetQuotaHistory, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get quota history response did not contain data")
	}

	return &QuotaHistory{QuotaHistoryResponse: *resp.JSON200}, nil
}

// CreateDownloadRateLimiter returns a rate limiter function that checks download quota before allowing downloads.
// This is intended for use with external SDKs (e.g., IPFS SDK) to integrate quota checking.
//
// The returned RateLimiterFunc will:
// - Call GetQuota to fetch current download usage
// - Return true if the requested size is within the remaining download quota
// - Return true if download quota is unlimited (Remaining is nil)
// - Return false if quota is insufficient
// - Return an error if quota cannot be retrieved
func CreateDownloadRateLimiter(client AccountAPI) RateLimiterFunc {
	return func(ctx context.Context, size int64) (bool, error) {
		quota, err := client.GetQuota(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to check download quota: %w", err)
		}

		// If Remaining is nil, it means unlimited quota
		if quota.Download.Remaining == nil {
			return true, nil
		}

		// Check if requested size is within remaining quota
		return int64(*quota.Download.Remaining) >= size, nil
	}
}
