package meta

import (
	"context"
	"fmt"
	stdhttp "net/http"

	internalmeta "go.lumeweb.com/portal-sdk/internal/meta"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// Stats operation identifiers for error message mapping.
const (
	OpStatsGetAggregate = 400 + iota
	OpStatsGetProtocols
)

// Stats operation string mappings for error messages.
var statsOperationString = map[int]string{
	OpStatsGetAggregate:  "get aggregate stats",
	OpStatsGetProtocols:  "get protocol stats",
}

// Stats HTTP error messages per operation and status code.
var statsHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpStatsGetAggregate: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
	},
	OpStatsGetProtocols: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
	},
}

var statsOpHandler = initStatsOpHandler()

func initStatsOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = "stats operation"

	for opID, name := range statsOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range statsHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// AggregateStats wraps the generated AggregateStatsResponse to reuse all fields.
type AggregateStats struct {
	internalmeta.AggregateStatsResponse
}

// ProtocolStats wraps the generated ProtocolStatsResponse to reuse all fields.
type ProtocolStats struct {
	internalmeta.ProtocolStatsResponse
}

// StatsService provides access to aggregate and protocol statistics.
type StatsService struct {
	client internalmeta.ClientWithResponsesInterface
}

// GetAggregate retrieves aggregate statistics across all CIDs, pinners, and storage.
func (s *StatsService) GetAggregate(ctx context.Context) (*AggregateStats, error) {
	resp, err := s.client.GetApiStatsAggregateWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregate stats: %w", err)
	}

	data, err := internalhttp.ValidateJSON200(statsOpHandler, resp.StatusCode(), resp.Body, resp.JSON200, OpStatsGetAggregate)
	if err != nil {
		return nil, err
	}

	return &AggregateStats{AggregateStatsResponse: *data}, nil
}

// GetProtocols retrieves per-protocol statistics including pins, storage, and uploads.
func (s *StatsService) GetProtocols(ctx context.Context) (*ProtocolStats, error) {
	resp, err := s.client.GetApiStatsProtocolsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol stats: %w", err)
	}

	data, err := internalhttp.ValidateJSON200(statsOpHandler, resp.StatusCode(), resp.Body, resp.JSON200, OpStatsGetProtocols)
	if err != nil {
		return nil, err
	}

	return &ProtocolStats{ProtocolStatsResponse: *data}, nil
}
