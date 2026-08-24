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

// Platform domain error sentinels
var (
	errPlatformDomainNotFound    = internalhttp.NotFoundError("platform domain not found")
	errInvalidPlatformDomain     = internalhttp.BadRequestError("invalid platform domain data")
	errInvalidPlatformDomainBind = internalhttp.BadRequestError("invalid platform domain bind request")
)

const (
	// Platform domain operation identifiers for error message mapping
	OpPlatformDomainList = 500 + iota
	OpPlatformDomainRegister
	OpPlatformDomainDelete
	OpPlatformDomainUpdate
	OpPlatformDomainBind
)

const defaultPlatformDomainOperationName = "platform domain operation"

// ErrPlatformDomainDefault is a generic platform domain error type.
var ErrPlatformDomainDefault = errors.New("platform domain operation failed")

// platformDomainOperationString maps platform domain operation IDs to their string names.
var platformDomainOperationString = map[int]string{
	OpPlatformDomainList:     "list platform domains",
	OpPlatformDomainRegister: "register platform domain",
	OpPlatformDomainDelete:   "delete platform domain",
	OpPlatformDomainUpdate:   "update platform domain",
	OpPlatformDomainBind:     "bind website to platform domain",
}

// platformDomainHTTPErrorMessages maps platform domain operation IDs to their custom status code error messages.
var platformDomainHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpPlatformDomainList: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpPlatformDomainRegister: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPlatformDomain,
	},
	OpPlatformDomainDelete: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPlatformDomainNotFound,
	},
	OpPlatformDomainUpdate: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPlatformDomain,
		stdhttp.StatusNotFound:     errPlatformDomainNotFound,
	},
	OpPlatformDomainBind: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPlatformDomainBind,
		stdhttp.StatusNotFound:     errPlatformDomainNotFound,
	},
}

// platformDomainOpHandler is the shared operation handler for platform domain operations (lazily initialized).
var platformDomainOpHandler = initPlatformDomainOpHandler()

// initPlatformDomainOpHandler initializes the OpHandler with platform domain operation mappings.
func initPlatformDomainOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultPlatformDomainOperationName

	for opID, name := range platformDomainOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range platformDomainHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handlePlatformDomainResponse wraps OpHandler.HandleResponse.
func handlePlatformDomainResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return platformDomainOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validatePlatformDomainJSON200 wraps OpHandler.ValidateJSON200.
func validatePlatformDomainJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(platformDomainOpHandler, respStatusCode, body, json200, op)
}

// validatePlatformDomainJSON201 wraps OpHandler.ValidateJSON201.
func validatePlatformDomainJSON201[T any](respStatusCode int, json201 *T, nilMsg string, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := []byte(nilMsg)
	return internalhttp.ValidateJSON201(platformDomainOpHandler, respStatusCode, body, json201, op)
}

// Type aliases for generated admin client types.
type (
	PlatformDomainRequest       = admin.PlatformDomainRequest
	PlatformDomainUpdateRequest = admin.PlatformDomainUpdateRequest
	PlatformDomainBindRequest   = admin.PlatformDomainBindRequest
)

// PlatformDomain represents a platform-owned root domain that users can claim
// subdomains under. Embeds the generated admin.PlatformDomainResponse.
type PlatformDomain struct {
	admin.PlatformDomainResponse
}

// RootDomain represents the root apex domain of a platform domain after an
// operator-owned website is bound to it. Embeds the generated admin.DomainResponse.
type RootDomain struct {
	admin.DomainResponse
}

// PlatformDomainService provides methods for managing platform domains.
type PlatformDomainService struct {
	client admin.ClientWithResponsesInterface
}

// SetRequestExecutor sets the underlying admin client for the platform domain
// service. Used for testing with mock clients.
func (p *PlatformDomainService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	p.client = client
}

// ListPlatformDomains lists all registered platform-owned root domains,
// including disabled ones.
func (p *PlatformDomainService) ListPlatformDomains(ctx context.Context) ([]*PlatformDomain, int, error) {
	resp, err := p.client.GetApiIpfsPlatformDomainsWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list platform domains: %w", err)
	}

	data, err := validatePlatformDomainJSON200(resp.StatusCode(), resp.JSON200, OpPlatformDomainList)
	if err != nil {
		return nil, 0, err
	}

	domains := lo.Map(data.Data, func(d admin.PlatformDomainResponse, _ int) *PlatformDomain {
		return &PlatformDomain{PlatformDomainResponse: d}
	})

	return domains, data.Total, nil
}

// RegisterPlatformDomain registers a platform-owned root domain that users can
// claim free subdomains under.
func (p *PlatformDomainService) RegisterPlatformDomain(ctx context.Context, req *PlatformDomainRequest) (*PlatformDomain, error) {
	resp, err := p.client.PostApiIpfsPlatformDomainsWithResponse(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to register platform domain: %w", err)
	}

	data, err := validatePlatformDomainJSON201(resp.StatusCode(), resp.JSON201, "register platform domain response did not contain data", OpPlatformDomainRegister)
	if err != nil {
		return nil, err
	}

	return &PlatformDomain{PlatformDomainResponse: *data}, nil
}

// DeletePlatformDomain removes a registered platform root. Existing subdomain
// bindings remain but can no longer be reconciled as platform subdomains.
func (p *PlatformDomainService) DeletePlatformDomain(ctx context.Context, id string) error {
	resp, err := p.client.DeleteApiIpfsPlatformDomainsIdWithResponse(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete platform domain: %w", err)
	}

	return handlePlatformDomainResponse(resp.StatusCode(), resp.Body, OpPlatformDomainDelete, []int{stdhttp.StatusNoContent})
}

// UpdatePlatformDomain enables or disables a registered platform root.
// Disabling prevents new claims but does not delete existing bindings.
func (p *PlatformDomainService) UpdatePlatformDomain(ctx context.Context, id string, req *PlatformDomainUpdateRequest) (*PlatformDomain, error) {
	resp, err := p.client.PatchApiIpfsPlatformDomainsIdWithResponse(ctx, id, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to update platform domain: %w", err)
	}

	data, err := validatePlatformDomainJSON200(resp.StatusCode(), resp.JSON200, OpPlatformDomainUpdate)
	if err != nil {
		return nil, err
	}

	return &PlatformDomain{PlatformDomainResponse: *data}, nil
}

// BindWebsiteToPlatformDomain binds an operator-owned website directly to the
// root apex of a platform domain (e.g. "pinner.site"). The platform root's DNS
// zone is auto-created on first use.
func (p *PlatformDomainService) BindWebsiteToPlatformDomain(ctx context.Context, id string, req *PlatformDomainBindRequest) (*RootDomain, error) {
	resp, err := p.client.PostApiIpfsPlatformDomainsIdBindWithResponse(ctx, id, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to bind website to platform domain: %w", err)
	}

	data, err := validatePlatformDomainJSON200(resp.StatusCode(), resp.JSON200, OpPlatformDomainBind)
	if err != nil {
		return nil, err
	}

	return &RootDomain{DomainResponse: *data}, nil
}
