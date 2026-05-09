package admin

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"

	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// Website error sentinels
var (
	errWebsiteNotFound      = internalhttp.PlainError("website not found")
	errCannotBlockWebsite   = internalhttp.PlainError("cannot block website")
	errCannotUnblockWebsite = internalhttp.PlainError("cannot unblock website")
)

const (
	// Website operation identifiers for error message mapping
	OpWebsiteBlockWebsite = 300 + iota
	OpWebsiteUnblockWebsite
)

const defaultWebsiteOperationName = "website operation"

// ErrWebsiteDefault is a generic website error type.
var ErrWebsiteDefault = errors.New("website operation failed")

// websiteOperationString maps website operation IDs to their string names.
var websiteOperationString = map[int]string{
	OpWebsiteBlockWebsite:   "block website",
	OpWebsiteUnblockWebsite: "unblock website",
}

// websiteHTTPErrorMessages maps website operation IDs to their custom status code error messages.
var websiteHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpWebsiteBlockWebsite: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusNotFound:    errWebsiteNotFound,
		stdhttp.StatusBadRequest:  errCannotBlockWebsite,
	},
	OpWebsiteUnblockWebsite: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusNotFound:    errWebsiteNotFound,
		stdhttp.StatusBadRequest:  errCannotUnblockWebsite,
	},
}

// websiteOpHandler is the shared operation handler for website operations (lazily initialized).
var websiteOpHandler = initWebsiteOpHandler()

// initWebsiteOpHandler initializes the OpHandler with website operation mappings.
func initWebsiteOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultWebsiteOperationName

	for opID, name := range websiteOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range websiteHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handleWebsiteResponse wraps OpHandler.HandleResponse.
func handleWebsiteResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return websiteOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validateWebsiteJSON200 wraps OpHandler.ValidateJSON200.
func validateWebsiteJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(websiteOpHandler, respStatusCode, body, json200, op)
}

// Website represents a website managed by the IPFS hosting service.
// Embeds the generated admin.WebsiteResponse to reuse all fields.
type Website struct {
	admin.WebsiteResponse
}

// WebsiteService provides methods for managing IPFS websites.
type WebsiteService struct {
	client admin.ClientWithResponsesInterface
}

// SetRequestExecutor sets the underlying admin client for the website service.
// Used for testing with mock clients.
func (w *WebsiteService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	w.client = client
}

// BlockWebsite blocks a website by its ID.
func (w *WebsiteService) BlockWebsite(ctx context.Context, id string) (*Website, error) {
	resp, err := w.client.PostApiIpfsWebsitesIdBlockWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to block website: %w", err)
	}

	data, err := validateWebsiteJSON200(resp.StatusCode(), resp.JSON200, OpWebsiteBlockWebsite)
	if err != nil {
		return nil, err
	}

	return &Website{WebsiteResponse: *data}, nil
}

// UnblockWebsite unblocks a website by its ID.
func (w *WebsiteService) UnblockWebsite(ctx context.Context, id string) (*Website, error) {
	resp, err := w.client.PostApiIpfsWebsitesIdUnblockWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to unblock website: %w", err)
	}

	data, err := validateWebsiteJSON200(resp.StatusCode(), resp.JSON200, OpWebsiteUnblockWebsite)
	if err != nil {
		return nil, err
	}

	return &Website{WebsiteResponse: *data}, nil
}
