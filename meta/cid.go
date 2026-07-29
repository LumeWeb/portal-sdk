package meta

import (
	"context"
	"fmt"
	stdhttp "net/http"

	internalmeta "go.lumeweb.com/portal-sdk/internal/meta"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// CID operation identifiers for error message mapping.
const (
	OpCIDGetStats = 300 + iota
	OpCIDGetDAG
	OpCIDGetSiaObject
)

// CID operation string mappings for error messages.
var cidOperationString = map[int]string{
	OpCIDGetStats:     "get CID stats",
	OpCIDGetDAG:       "get CID DAG",
	OpCIDGetSiaObject: "get CID Sia object",
}

// CID HTTP error messages per operation and status code.
var cidHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpCIDGetStats: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("CID not found"),
	},
	OpCIDGetDAG: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("CID not found"),
	},
	OpCIDGetSiaObject: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("CID not found"),
	},
}

var cidOpHandler = initCIDOpHandler()

func initCIDOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = "CID metadata operation"

	for opID, name := range cidOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range cidHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// CIDStats wraps the generated CIDStatsResponse to reuse all fields.
type CIDStats struct {
	internalmeta.CIDStatsResponse
}

// DAGExport wraps the generated DAGExportResponse to reuse all fields.
type DAGExport struct {
	internalmeta.DAGExportResponse
}

// SiaObject wraps the generated CIDExportResponse to reuse all fields.
type SiaObject struct {
	internalmeta.CIDExportResponse
}

// CIDService provides access to CID metadata operations.
type CIDService struct {
	client internalmeta.ClientWithResponsesInterface
}

// GetStats retrieves metadata and statistics for a given CID.
func (s *CIDService) GetStats(ctx context.Context, cid string) (*CIDStats, error) {
	resp, err := s.client.GetApiStatsCidCidWithResponse(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("failed to get CID stats: %w", err)
	}

	data, err := internalhttp.ValidateJSON200(cidOpHandler, resp.StatusCode(), resp.Body, resp.JSON200, OpCIDGetStats)
	if err != nil {
		return nil, err
	}

	return &CIDStats{CIDStatsResponse: *data}, nil
}

// GetDAG retrieves the DAG structure for a given CID, including all blocks and links.
func (s *CIDService) GetDAG(ctx context.Context, cid string) (*DAGExport, error) {
	resp, err := s.client.GetApiExportCidCidDagWithResponse(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("failed to get CID DAG: %w", err)
	}

	data, err := internalhttp.ValidateJSON200(cidOpHandler, resp.StatusCode(), resp.Body, resp.JSON200, OpCIDGetDAG)
	if err != nil {
		return nil, err
	}

	return &DAGExport{DAGExportResponse: *data}, nil
}

// GetSiaObject retrieves the Sia object mapping for a given CID.
func (s *CIDService) GetSiaObject(ctx context.Context, cid string) (*SiaObject, error) {
	resp, err := s.client.GetApiExportCidCidSiaObjectWithResponse(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("failed to get CID Sia object: %w", err)
	}

	data, err := internalhttp.ValidateJSON200(cidOpHandler, resp.StatusCode(), resp.Body, resp.JSON200, OpCIDGetSiaObject)
	if err != nil {
		return nil, err
	}

	return &SiaObject{CIDExportResponse: *data}, nil
}
