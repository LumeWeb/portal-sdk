package admin

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"time"

	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

const (
	// Quota operation identifiers for error message mapping
	OpQuotaListPlans = 100 + iota
	OpQuotaCreatePlan
	OpQuotaGetPlan
	OpQuotaUpdatePlan
	OpQuotaDeletePlan
	OpQuotaSetDefaultPlan
	OpQuotaListAllowances
	OpQuotaCreateAllowance
	OpQuotaUpdateAllowance
	OpQuotaDeleteAllowance
	OpQuotaGetConfig
	OpQuotaUpdateConfig
	OpQuotaGetStats
	OpQuotaReconcile
	OpQuotaCleanup
)

const defaultQuotaOperationName = "quota operation"

// operationString maps quota operation IDs to their string names.
var quotaOperationString = map[int]string{
	OpQuotaListPlans:       "list quota plans",
	OpQuotaCreatePlan:      "create quota plan",
	OpQuotaGetPlan:         "get quota plan",
	OpQuotaUpdatePlan:      "update quota plan",
	OpQuotaDeletePlan:      "delete quota plan",
	OpQuotaSetDefaultPlan:  "set default quota plan",
	OpQuotaListAllowances:  "list quota allowances",
	OpQuotaCreateAllowance: "create quota allowance",
	OpQuotaUpdateAllowance: "update quota allowance",
	OpQuotaDeleteAllowance: "delete quota allowance",
	OpQuotaGetConfig:       "get quota configuration",
	OpQuotaUpdateConfig:    "update quota configuration",
	OpQuotaGetStats:        "get quota statistics",
	OpQuotaReconcile:       "reconcile quota",
	OpQuotaCleanup:         "cleanup quota",
}

// QuotaOperationErrorFactory is a helper for creating errors with optional ErrUnauthorized wrapping.
type QuotaOperationErrorFactory struct {
	wrapErr bool
	message string
}

// Error creates the actual error.
func (ef QuotaOperationErrorFactory) Error() error {
	if ef.wrapErr {
		return fmt.Errorf("%w: %s", internalhttp.ErrUnauthorized, ef.message)
	}
	return fmt.Errorf("%s", ef.message)
}

// quotaAuthErr creates an error factory that wraps with ErrUnauthorized.
func quotaAuthErr(msg string) QuotaOperationErrorFactory {
	return QuotaOperationErrorFactory{wrapErr: true, message: msg}
}

// quotaPlainErr creates an error factory without wrapping.
func quotaPlainErr(msg string) QuotaOperationErrorFactory {
	return QuotaOperationErrorFactory{wrapErr: false, message: msg}
}

// httpErrorMessages maps quota operation IDs to their custom status code error messages.
var quotaHTTPErrorMessages = map[int]map[int]QuotaOperationErrorFactory{
	OpQuotaListPlans: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid request parameters"),
	},
	OpQuotaCreatePlan: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid plan data"),
		stdhttp.StatusConflict:    quotaPlainErr("plan with this name already exists"),
	},
	OpQuotaGetPlan: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusNotFound:     quotaPlainErr("plan not found"),
	},
	OpQuotaUpdatePlan: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid plan data"),
		stdhttp.StatusNotFound:     quotaPlainErr("plan not found"),
	},
	OpQuotaDeletePlan: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusNotFound:     quotaPlainErr("plan not found"),
		stdhttp.StatusConflict:     quotaPlainErr("cannot delete plan in use"),
	},
	OpQuotaSetDefaultPlan: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusNotFound:     quotaPlainErr("plan not found"),
	},
	OpQuotaListAllowances: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid request parameters"),
	},
	OpQuotaCreateAllowance: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid allowance data"),
		stdhttp.StatusNotFound:     quotaPlainErr("user not found"),
	},
	OpQuotaUpdateAllowance: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid allowance data"),
		stdhttp.StatusNotFound:     quotaPlainErr("allowance not found"),
	},
	OpQuotaDeleteAllowance: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusNotFound:     quotaPlainErr("allowance not found"),
	},
	OpQuotaGetConfig: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
	},
	OpQuotaUpdateConfig: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
		stdhttp.StatusBadRequest:   quotaPlainErr("invalid configuration data"),
	},
	OpQuotaGetStats: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
	},
	OpQuotaReconcile: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
	},
	OpQuotaCleanup: {
		stdhttp.StatusUnauthorized: quotaAuthErr("authentication required"),
	},
}

// handleQuotaResponse processes an HTTP response using the quota error message map.
// op: the operation ID (used to lookup custom error messages)
// successCodes: status codes that indicate success (e.g., []int{stdhttp.StatusOK})
// Returns nil for success codes, custom error from global map, or generic error with body.
func handleQuotaResponse(statusCode int, body []byte, op int, successCodes []int) error {
	// Check if status code is in success codes
	for _, code := range successCodes {
		if statusCode == code {
			return nil
		}
	}

	// Check for custom error message in global map
	if errorMessages, ok := quotaHTTPErrorMessages[op]; ok {
		if factory, ok := errorMessages[statusCode]; ok {
			return factory.Error()
		}
	}

	// Get operation name for generic error
	opName := quotaOperationString[op]
	if opName == "" {
		opName = defaultQuotaOperationName
	}

	// Generic error with body
	return fmt.Errorf("%s failed with status %d: %s", opName, statusCode, string(body))
}

// validateQuotaJSON201 validates HTTP 201 responses with JSON201 data.
func validateQuotaJSON201[T any](respStatusCode int, json201 *T, nilMsg string) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	if respStatusCode != stdhttp.StatusCreated {
		return nil, fmt.Errorf("expected status 201, got %d", respStatusCode)
	}
	if json201 == nil {
		return nil, fmt.Errorf("%s", nilMsg)
	}
	return json201, nil
}

// validateQuotaJSON200 validates HTTP 200 responses with JSON200 data.
func validateQuotaJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	if respStatusCode != stdhttp.StatusOK {
		// Check for custom error message in global map first
		if errorMessages, ok := quotaHTTPErrorMessages[op]; ok {
			if factory, ok := errorMessages[respStatusCode]; ok {
				return nil, factory.Error()
			}
		}
		// Generic error if no custom message
		return nil, fmt.Errorf("expected status 200, got %d", respStatusCode)
	}
	if json200 == nil {
		return nil, fmt.Errorf("response body is required")
	}
	return json200, nil
}

// ErrQuotaDefault is a generic quota error type.
var ErrQuotaDefault = fmt.Errorf("quota operation failed")

// QuotaPlan represents a quota plan with limits and thresholds.
// Embeds the generated admin.QuotaPlanResponse to reuse all fields.
type QuotaPlan struct {
	admin.QuotaPlanResponse
}

// QuotaAllowance represents a quota allowance granted to a user.
// Embeds the generated admin.AllowanceGrantResponse to reuse all fields.
type QuotaAllowance struct {
	admin.AllowanceGrantResponse
}

// QuotaConfig represents the system-wide quota configuration.
// Embeds the generated admin.QuotaConfigResponse to reuse all fields.
type QuotaConfig struct {
	admin.QuotaConfigResponse
}

// SystemUsage represents system-wide usage statistics.
type SystemUsage struct {
	DownloadBytes int `json:"download_bytes"`
	StorageBytes  int `json:"storage_bytes"`
	UploadBytes   int `json:"upload_bytes"`
}

// SystemStats represents system-wide quota statistics.
type SystemStats struct {
	admin.SystemStatsResponse
}

// QuotaService provides methods for managing quotas.
type QuotaService struct {
	client   admin.ClientWithResponsesInterface
	jwt      string
	apiKey   string
}

// QuotaServerConfig holds configuration for quota service operations.
type QuotaServerConfig struct {
	jwt      string
	apiKey   string
	endpoint string
}

// NewQuotaPlan creates a new QuotaPlan with the given parameters.
func NewQuotaPlan(name, description string, limits QuotaLimits) *QuotaPlan {
	return &QuotaPlan{
		admin.QuotaPlanResponse{
			Name:               name,
			Description:        description,
			UploadDailyLimit:   limits.UploadDailyLimit,
			UploadTotalLimit:   limits.UploadTotalLimit,
			UploadThreshold:    limits.UploadThreshold,
			DownloadDailyLimit: limits.DownloadDailyLimit,
			DownloadTotalLimit: limits.DownloadTotalLimit,
			DownloadThreshold:  limits.DownloadThreshold,
			StorageLimit:       limits.StorageLimit,
			StorageThreshold:   limits.StorageThreshold,
		},
	}
}

// QuotaLimits defines quota limit configuration.
type QuotaLimits struct {
	UploadDailyLimit   int
	UploadTotalLimit   int
	UploadThreshold    int
	DownloadDailyLimit int
	DownloadTotalLimit int
	DownloadThreshold  int
	StorageLimit       int
	StorageThreshold   int
}

// ListPlans lists all quota plans.
func (q *QuotaService) ListPlans(ctx context.Context) ([]*QuotaPlan, int, error) {
	resp, err := q.client.GetApiQuotaPlansWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send list plans request: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaListPlans, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list plans response did not contain data")
	}

	plans := make([]*QuotaPlan, len(resp.JSON200.Plans))
	for i, plan := range resp.JSON200.Plans {
		plans[i] = &QuotaPlan{QuotaPlanResponse: plan}
	}

	return plans, resp.JSON200.Total, nil
}

// CreatePlan creates a new quota plan.
func (q *QuotaService) CreatePlan(ctx context.Context, plan *QuotaPlan) (*QuotaPlan, error) {
	reqBody := admin.QuotaPlanRequest{
		Name:               plan.Name,
		Description:        plan.Description,
		UploadDailyLimit:   plan.UploadDailyLimit,
		UploadTotalLimit:   plan.UploadTotalLimit,
		UploadThreshold:    plan.UploadThreshold,
		DownloadDailyLimit: plan.DownloadDailyLimit,
		DownloadTotalLimit: plan.DownloadTotalLimit,
		DownloadThreshold:  plan.DownloadThreshold,
		StorageLimit:       plan.StorageLimit,
		StorageThreshold:   plan.StorageThreshold,
		IsActive:           plan.IsActive,
	}

	resp, err := q.client.PostApiQuotaPlansWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send create plan request: %w", err)
	}

	data, err := validateQuotaJSON201(resp.StatusCode(), resp.JSON201, "create plan response did not contain data")
	if err != nil {
		return nil, err
	}

	return &QuotaPlan{QuotaPlanResponse: *data}, nil
}

// GetPlan retrieves a quota plan by ID.
func (q *QuotaService) GetPlan(ctx context.Context, planID string) (*QuotaPlan, error) {
	resp, err := q.client.GetApiQuotaPlansPlanIDWithResponse(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to send get plan request: %w", err)
	}

	data, err := validateQuotaJSON200(resp.StatusCode(), resp.JSON200, OpQuotaGetPlan)
	if err != nil {
		return nil, err
	}

	return &QuotaPlan{QuotaPlanResponse: *data}, nil
}

// UpdatePlan updates an existing quota plan.
func (q *QuotaService) UpdatePlan(ctx context.Context, planID string, plan *QuotaPlan) (*QuotaPlan, error) {
	reqBody := admin.QuotaPlanRequest{
		Name:               plan.Name,
		Description:        plan.Description,
		UploadDailyLimit:   plan.UploadDailyLimit,
		UploadTotalLimit:   plan.UploadTotalLimit,
		UploadThreshold:    plan.UploadThreshold,
		DownloadDailyLimit: plan.DownloadDailyLimit,
		DownloadTotalLimit: plan.DownloadTotalLimit,
		DownloadThreshold:  plan.DownloadThreshold,
		StorageLimit:       plan.StorageLimit,
		StorageThreshold:   plan.StorageThreshold,
		IsActive:           plan.IsActive,
	}

	resp, err := q.client.PutApiQuotaPlansPlanIDWithResponse(ctx, planID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send update plan request: %w", err)
	}

	data, err := validateQuotaJSON200(resp.StatusCode(), resp.JSON200, OpQuotaUpdatePlan)
	if err != nil {
		return nil, err
	}

	return &QuotaPlan{QuotaPlanResponse: *data}, nil
}

// DeletePlan deletes a quota plan.
func (q *QuotaService) DeletePlan(ctx context.Context, planID string) error {
	resp, err := q.client.DeleteApiQuotaPlansPlanIDWithResponse(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to send delete plan request: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaDeletePlan, []int{stdhttp.StatusNoContent})
}

// SetDefaultPlan sets a quota plan as the default for new users.
func (q *QuotaService) SetDefaultPlan(ctx context.Context, planID string) error {
	resp, err := q.client.PostApiQuotaPlansPlanIDDefaultWithResponse(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to send set default plan request: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaSetDefaultPlan, []int{stdhttp.StatusNoContent})
}

// ListAllowances lists all quota allowances for a user.
func (q *QuotaService) ListAllowances(ctx context.Context, userID int) ([]*QuotaAllowance, int, error) {
	resp, err := q.client.GetApiQuotaAllowancesWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send list allowances request: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaListAllowances, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list allowances response did not contain data")
	}

	allowances := make([]*QuotaAllowance, len(resp.JSON200.Grants))
	for i, grant := range resp.JSON200.Grants {
		allowances[i] = &QuotaAllowance{AllowanceGrantResponse: grant}
	}

	return allowances, resp.JSON200.Total, nil
}

// CreateAllowance creates a new quota allowance for a user.
func (q *QuotaService) CreateAllowance(ctx context.Context, userID int, source, allowanceType string, upload, download, storage int, expiryDate time.Time) (*QuotaAllowance, error) {
	reqBody := admin.AllowanceGrantRequest{
		UserId:     userID,
		Source:     source,
		Type:       allowanceType,
		Upload:     upload,
		Download:   download,
		Storage:    storage,
		ExpiryDate: expiryDate,
	}

	resp, err := q.client.PostApiQuotaAllowancesWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send create allowance request: %w", err)
	}

	data, err := validateQuotaJSON201(resp.StatusCode(), resp.JSON201, "create allowance response did not contain data")
	if err != nil {
		return nil, err
	}

	return &QuotaAllowance{AllowanceGrantResponse: *data}, nil
}

// UpdateAllowance updates an existing quota allowance.
func (q *QuotaService) UpdateAllowance(ctx context.Context, grantID string, userID int, source, allowanceType string, upload, download, storage int, expiryDate time.Time) (*QuotaAllowance, error) {
	reqBody := admin.AllowanceGrantRequest{
		UserId:     userID,
		Source:     source,
		Type:       allowanceType,
		Upload:     upload,
		Download:   download,
		Storage:    storage,
		ExpiryDate: expiryDate,
	}

	resp, err := q.client.PutApiQuotaAllowancesGrantIDWithResponse(ctx, grantID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send update allowance request: %w", err)
	}

	data, err := validateQuotaJSON200(resp.StatusCode(), resp.JSON200, OpQuotaUpdateAllowance)
	if err != nil {
		return nil, err
	}

	return &QuotaAllowance{AllowanceGrantResponse: *data}, nil
}

// DeleteAllowance deletes a quota allowance.
func (q *QuotaService) DeleteAllowance(ctx context.Context, grantID string) error {
	resp, err := q.client.DeleteApiQuotaAllowancesGrantIDWithResponse(ctx, grantID)
	if err != nil {
		return fmt.Errorf("failed to send delete allowance request: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaDeleteAllowance, []int{stdhttp.StatusNoContent})
}

// GetConfig retrieves the system quota configuration.
func (q *QuotaService) GetConfig(ctx context.Context) (*QuotaConfig, error) {
	resp, err := q.client.GetApiQuotaSystemConfigWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to send get config request: %w", err)
	}

	data, err := validateQuotaJSON200(resp.StatusCode(), resp.JSON200, OpQuotaGetConfig)
	if err != nil {
		return nil, err
	}

	return &QuotaConfig{QuotaConfigResponse: *data}, nil
}

// UpdateConfig updates the system quota configuration.
func (q *QuotaService) UpdateConfig(ctx context.Context, config *QuotaConfig) (*QuotaConfig, error) {
	reqBody := admin.QuotaConfigUpdateRequest{
		DefaultPlanId:          config.DefaultPlanId,
		EnableQuotaEnforcement: config.EnableQuotaEnforcement,
		StorageRetentionDays:   config.StorageRetentionDays,
	}

	resp, err := q.client.PutApiQuotaSystemConfigWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send update config request: %w", err)
	}

	data, err := validateQuotaJSON200(resp.StatusCode(), resp.JSON200, OpQuotaUpdateConfig)
	if err != nil {
		return nil, err
	}

	return &QuotaConfig{QuotaConfigResponse: *data}, nil
}

// GetStats retrieves system-wide quota statistics.
func (q *QuotaService) GetStats(ctx context.Context) (*SystemStats, error) {
	resp, err := q.client.GetApiQuotaSystemStatsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to send get stats request: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaGetStats, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get stats response did not contain data")
	}

	return &SystemStats{SystemStatsResponse: *resp.JSON200}, nil
}

// Reconcile performs quota reconciliation for users.
// If userID is nil, reconciles all users. If provided, reconciles only that user.
func (q *QuotaService) Reconcile(ctx context.Context, userID *int) (string, int, error) {
	reqBody := admin.ReconcileRequest{UserId: userID}

	resp, err := q.client.PostApiQuotaSystemReconcileWithResponse(ctx, reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send reconcile request: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaReconcile, []int{stdhttp.StatusOK}); err != nil {
		return "", 0, err
	}

	if resp.JSON200 == nil {
		return "", 0, fmt.Errorf("reconcile response did not contain data")
	}

	return resp.JSON200.Message, resp.JSON200.UsersProcessed, nil
}

// Cleanup performs quota cleanup based on retention policy.
func (q *QuotaService) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	reqBody := admin.CleanupRequest{RetentionDays: retentionDays}

	resp, err := q.client.PostApiQuotaSystemCleanupWithResponse(ctx, reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to send cleanup request: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaCleanup, []int{stdhttp.StatusOK}); err != nil {
		return 0, err
	}

	if resp.JSON200 == nil {
		return 0, fmt.Errorf("cleanup response did not contain data")
	}

	return resp.JSON200.RecordsDeleted, nil
}

// SetRequestExecutor sets the underlying admin client for the quota service.
// Used for testing with mock clients.
func (q *QuotaService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	q.client = client
}
