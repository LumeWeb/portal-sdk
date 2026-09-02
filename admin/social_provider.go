package admin

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"

	"github.com/samber/lo"
	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// Social provider error sentinels
var (
	errSocialProviderNotFound = internalhttp.NotFoundError("social login provider not found")
	errInvalidSocialProvider  = internalhttp.BadRequestError("invalid social login provider data")

	errInvalidSocialProviderRequest = internalhttp.BadRequestError("social login provider request is required")
)

const (
	// Social provider operation identifiers for error message mapping
	OpSocialProviderList = 600 + iota
	OpSocialProviderCreate
	OpSocialProviderGet
	OpSocialProviderUpdate
	OpSocialProviderDelete
	OpSocialProviderEnable
	OpSocialProviderDisable
)

const defaultSocialProviderOperationName = "social login provider operation"

// ErrSocialProviderDefault is a generic social provider error type.
var ErrSocialProviderDefault = errors.New("social login provider operation failed")

// socialProviderOperationString maps social provider operation IDs to their string names.
var socialProviderOperationString = map[int]string{
	OpSocialProviderList:    "list social login providers",
	OpSocialProviderCreate:  "create social login provider",
	OpSocialProviderGet:     "get social login provider",
	OpSocialProviderUpdate:  "update social login provider",
	OpSocialProviderDelete:  "delete social login provider",
	OpSocialProviderEnable:  "enable social login provider",
	OpSocialProviderDisable: "disable social login provider",
}

// socialProviderHTTPErrorMessages maps social provider operation IDs to their custom status code error messages.
var socialProviderHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpSocialProviderList: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpSocialProviderCreate: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidSocialProvider,
	},
	OpSocialProviderGet: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errSocialProviderNotFound,
	},
	OpSocialProviderUpdate: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidSocialProvider,
		stdhttp.StatusNotFound:     errSocialProviderNotFound,
	},
	OpSocialProviderDelete: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errSocialProviderNotFound,
	},
	OpSocialProviderEnable: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errSocialProviderNotFound,
	},
	OpSocialProviderDisable: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errSocialProviderNotFound,
	},
}

// socialProviderOpHandler is the shared operation handler for social provider operations (lazily initialized).
var socialProviderOpHandler = initSocialProviderOpHandler()

// initSocialProviderOpHandler initializes the OpHandler with social provider operation mappings.
func initSocialProviderOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultSocialProviderOperationName

	for opID, name := range socialProviderOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range socialProviderHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handleSocialProviderResponse wraps OpHandler.HandleResponse.
func handleSocialProviderResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return socialProviderOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validateSocialProviderJSON200 wraps OpHandler.ValidateJSON200.
func validateSocialProviderJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(socialProviderOpHandler, respStatusCode, body, json200, op)
}

// validateSocialProviderJSON201 wraps OpHandler.ValidateJSON201.
func validateSocialProviderJSON201[T any](respStatusCode int, json201 *T, nilMsg string, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := []byte(nilMsg)
	return internalhttp.ValidateJSON201(socialProviderOpHandler, respStatusCode, body, json201, op)
}

// Type aliases for generated admin client types.
type (
	SocialProviderRequest = admin.SocialProviderRequest
)

// SocialProvider represents a configured social login provider. Embeds the
// generated admin.SocialProviderResponse. The client secret is never returned
// by the API.
type SocialProvider struct {
	admin.SocialProviderResponse
}

// SocialProviderService provides methods for managing social login providers
// (full CRUD plus enable/disable).
type SocialProviderService struct {
	client admin.ClientWithResponsesInterface
}

// SetRequestExecutor sets the underlying admin client for the social provider
// service. Used for testing with mock clients.
func (s *SocialProviderService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	s.client = client
}

// ListSocialProviders lists all configured social login providers. Secrets are
// never returned.
func (s *SocialProviderService) ListSocialProviders(ctx context.Context) ([]*SocialProvider, int, error) {
	resp, err := s.client.GetApiSocialProvidersWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list social login providers: %w", err)
	}

	data, err := validateSocialProviderJSON200(resp.StatusCode(), resp.JSON200, OpSocialProviderList)
	if err != nil {
		return nil, 0, err
	}

	providers := lo.Map(data.Data, func(p admin.SocialProviderResponse, _ int) *SocialProvider {
		return &SocialProvider{SocialProviderResponse: p}
	})

	return providers, data.Total, nil
}

// CreateSocialProvider creates a new social login provider configuration.
func (s *SocialProviderService) CreateSocialProvider(ctx context.Context, req *SocialProviderRequest) (*SocialProvider, error) {
	if req == nil {
		return nil, errInvalidSocialProviderRequest.Error()
	}

	resp, err := s.client.PostApiSocialProvidersWithResponse(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to create social login provider: %w", err)
	}

	data, err := validateSocialProviderJSON201(resp.StatusCode(), resp.JSON201, "create social login provider response did not contain data", OpSocialProviderCreate)
	if err != nil {
		return nil, err
	}

	return &SocialProvider{SocialProviderResponse: *data}, nil
}

// GetSocialProvider returns a single social login provider by ID.
func (s *SocialProviderService) GetSocialProvider(ctx context.Context, id string) (*SocialProvider, error) {
	resp, err := s.client.GetApiSocialProvidersIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get social login provider: %w", err)
	}

	data, err := validateSocialProviderJSON200(resp.StatusCode(), resp.JSON200, OpSocialProviderGet)
	if err != nil {
		return nil, err
	}

	return &SocialProvider{SocialProviderResponse: *data}, nil
}

// UpdateSocialProvider updates an existing social login provider configuration.
func (s *SocialProviderService) UpdateSocialProvider(ctx context.Context, id string, req *SocialProviderRequest) (*SocialProvider, error) {
	if req == nil {
		return nil, errInvalidSocialProviderRequest.Error()
	}

	resp, err := s.client.PutApiSocialProvidersIdWithResponse(ctx, id, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to update social login provider: %w", err)
	}

	data, err := validateSocialProviderJSON200(resp.StatusCode(), resp.JSON200, OpSocialProviderUpdate)
	if err != nil {
		return nil, err
	}

	return &SocialProvider{SocialProviderResponse: *data}, nil
}

// DeleteSocialProvider removes a social login provider configuration by ID.
func (s *SocialProviderService) DeleteSocialProvider(ctx context.Context, id string) error {
	resp, err := s.client.DeleteApiSocialProvidersIdWithResponse(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete social login provider: %w", err)
	}

	return handleSocialProviderResponse(resp.StatusCode(), resp.Body, OpSocialProviderDelete, []int{stdhttp.StatusNoContent})
}

// EnableSocialProvider enables a previously disabled social login provider.
func (s *SocialProviderService) EnableSocialProvider(ctx context.Context, id string) (*SocialProvider, error) {
	resp, err := s.client.PostApiSocialProvidersIdEnableWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to enable social login provider: %w", err)
	}

	data, err := validateSocialProviderJSON200(resp.StatusCode(), resp.JSON200, OpSocialProviderEnable)
	if err != nil {
		return nil, err
	}

	return &SocialProvider{SocialProviderResponse: *data}, nil
}

// DisableSocialProvider disables a social login provider so it can no longer be
// used to authenticate.
func (s *SocialProviderService) DisableSocialProvider(ctx context.Context, id string) (*SocialProvider, error) {
	resp, err := s.client.PostApiSocialProvidersIdDisableWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to disable social login provider: %w", err)
	}

	data, err := validateSocialProviderJSON200(resp.StatusCode(), resp.JSON200, OpSocialProviderDisable)
	if err != nil {
		return nil, err
	}

	return &SocialProvider{SocialProviderResponse: *data}, nil
}
