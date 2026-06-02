package admin

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"

	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// Profiling error sentinels
var (
	errProfilingFailed      = internalhttp.PlainError("profiling operation failed")
	errInvalidProfilingRate = internalhttp.PlainError("invalid profiling rate")
)

const (
	// Profiling operation identifiers for error message mapping
	OpProfilingIndex = 400 + iota
	OpProfilingBlockProfile
	OpProfilingSetBlockRate
	OpProfilingCmdline
	OpProfilingGoroutine
	OpProfilingHeap
	OpProfilingMutexProfile
	OpProfilingSetMutexFraction
	OpProfilingCPUProfile
	OpProfilingStatus
	OpProfilingSymbol
	OpProfilingThreadcreate
	OpProfilingTrace
)

const defaultProfilingOperationName = "profiling operation"

// ErrProfilingDefault is a generic profiling error type.
var ErrProfilingDefault = errors.New("profiling operation failed")

// profilingOperationString maps profiling operation IDs to their string names.
var profilingOperationString = map[int]string{
	OpProfilingIndex:          "pprof index",
	OpProfilingBlockProfile:   "block profile",
	OpProfilingSetBlockRate:   "set block profile rate",
	OpProfilingCmdline:        "cmdline",
	OpProfilingGoroutine:      "goroutine profile",
	OpProfilingHeap:           "heap profile",
	OpProfilingMutexProfile:   "mutex profile",
	OpProfilingSetMutexFraction: "set mutex profile fraction",
	OpProfilingCPUProfile:     "cpu profile",
	OpProfilingStatus:         "profiling status",
	OpProfilingSymbol:         "symbol lookup",
	OpProfilingThreadcreate:   "thread create profile",
	OpProfilingTrace:          "execution trace",
}

// profilingHTTPErrorMessages maps profiling operation IDs to their custom status code error messages.
var profilingHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpProfilingIndex: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingBlockProfile: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingSetBlockRate: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusBadRequest:  errInvalidProfilingRate,
	},
	OpProfilingCmdline: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingGoroutine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingHeap: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingMutexProfile: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingSetMutexFraction: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusBadRequest:  errInvalidProfilingRate,
	},
	OpProfilingCPUProfile: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingStatus: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingSymbol: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingThreadcreate: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
	OpProfilingTrace: {
		stdhttp.StatusUnauthorized: errAuthRequired,
	},
}

// profilingOpHandler is the shared operation handler for profiling operations (lazily initialized).
var profilingOpHandler = initProfilingOpHandler()

// initProfilingOpHandler initializes the OpHandler with profiling operation mappings.
func initProfilingOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultProfilingOperationName

	for opID, name := range profilingOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range profilingHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handleProfilingResponse wraps OpHandler.HandleResponse.
func handleProfilingResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return profilingOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validateProfilingJSON200 wraps OpHandler.ValidateJSON200.
func validateProfilingJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(profilingOpHandler, respStatusCode, body, json200, op)
}

// ProfilingStatus represents the current profiling configuration.
// Embeds the generated admin.ProfilingStatusResponse to reuse all fields.
type ProfilingStatus struct {
	admin.ProfilingStatusResponse
}

// ProfilingService provides methods for managing Go runtime profiling via pprof.
type ProfilingService struct {
	client admin.ClientWithResponsesInterface
}

// SetRequestExecutor sets the underlying admin client for the profiling service.
// Used for testing with mock clients.
func (p *ProfilingService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	p.client = client
}

// GetProfileIndex returns the pprof index page listing available profiles.
func (p *ProfilingService) GetProfileIndex(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pprof index: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingIndex, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetBlockProfile returns the block profile data.
func (p *ProfilingService) GetBlockProfile(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofBlockWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get block profile: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingBlockProfile, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// SetBlockProfileRate sets the block profiling rate.
// A rate of 0 disables block profiling, 1 captures all events, higher values sample.
func (p *ProfilingService) SetBlockProfileRate(ctx context.Context, rate int) error {
	resp, err := p.client.PutApiDebugPprofBlockWithResponse(ctx, admin.BlockProfileRequest{Rate: rate})
	if err != nil {
		return fmt.Errorf("failed to set block profile rate: %w", err)
	}

	return handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingSetBlockRate, []int{stdhttp.StatusNoContent})
}

// GetCmdline returns the command line of the running program.
func (p *ProfilingService) GetCmdline(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofCmdlineWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmdline: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingCmdline, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetGoroutineProfile returns stack traces of all current goroutines.
func (p *ProfilingService) GetGoroutineProfile(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofGoroutineWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get goroutine profile: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingGoroutine, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetHeapProfile returns a sampling of memory allocations.
func (p *ProfilingService) GetHeapProfile(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofHeapWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get heap profile: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingHeap, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetMutexProfile returns stack traces of holders of contended mutexes.
func (p *ProfilingService) GetMutexProfile(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofMutexWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mutex profile: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingMutexProfile, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// SetMutexProfileFraction sets the mutex profiling fraction.
// A fraction of 0 disables mutex profiling, 1 captures all events, 100 samples 1%.
func (p *ProfilingService) SetMutexProfileFraction(ctx context.Context, fraction int) error {
	resp, err := p.client.PutApiDebugPprofMutexWithResponse(ctx, admin.MutexProfileRequest{Fraction: fraction})
	if err != nil {
		return fmt.Errorf("failed to set mutex profile fraction: %w", err)
	}

	return handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingSetMutexFraction, []int{stdhttp.StatusNoContent})
}

// GetCPUProfile returns a CPU profile for the specified duration.
func (p *ProfilingService) GetCPUProfile(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofProfileWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cpu profile: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingCPUProfile, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetStatus returns the current block and mutex profiling rates.
func (p *ProfilingService) GetStatus(ctx context.Context) (*ProfilingStatus, error) {
	resp, err := p.client.GetApiDebugPprofStatusWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiling status: %w", err)
	}

	data, err := validateProfilingJSON200(resp.StatusCode(), resp.JSON200, OpProfilingStatus)
	if err != nil {
		return nil, err
	}

	return &ProfilingStatus{ProfilingStatusResponse: *data}, nil
}

// GetSymbol looks up program counters and returns function names.
func (p *ProfilingService) GetSymbol(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofSymbolWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingSymbol, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetThreadcreate returns stack traces that led to the creation of new OS threads.
func (p *ProfilingService) GetThreadcreate(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofThreadcreateWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get threadcreate profile: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingThreadcreate, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetTrace returns an execution trace.
func (p *ProfilingService) GetTrace(ctx context.Context) ([]byte, error) {
	resp, err := p.client.GetApiDebugPprofTraceWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	if err := handleProfilingResponse(resp.StatusCode(), resp.Body, OpProfilingTrace, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	return resp.Body, nil
}
