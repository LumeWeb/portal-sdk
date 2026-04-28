package admin

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"time"

	"github.com/samber/lo"
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
	OpQuotaGetStats
	OpQuotaReconcile
	OpQuotaCleanup
	OpQuotaListUserConfigs
	OpQuotaUpdateUserConfig
	OpQuotaResetUserPlan
)

const defaultQuotaOperationName = "quota operation"

// ErrQuotaDefault is returned when a quota operation fails without a more specific error.
var ErrQuotaDefault = errors.New("quota operation failed")

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
	OpQuotaGetStats:            "get quota statistics",
	OpQuotaReconcile:       "reconcile quota",
	OpQuotaCleanup:         "cleanup quota",
	OpQuotaListUserConfigs:  "list user quota configs",
	OpQuotaUpdateUserConfig: "update user quota config",
	OpQuotaResetUserPlan:    "reset user quota plan",
}

// httpErrorMessages maps quota operation IDs to their custom status code error messages.
var quotaHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpQuotaListPlans: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid request parameters"),
	},
	OpQuotaCreatePlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid plan data"),
		stdhttp.StatusConflict:    internalhttp.PlainError("plan with this name already exists"),
	},
	OpQuotaGetPlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("plan not found"),
	},
	OpQuotaUpdatePlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid plan data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("plan not found"),
	},
	OpQuotaDeletePlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("plan not found"),
		stdhttp.StatusConflict:     internalhttp.PlainError("cannot delete plan in use"),
	},
	OpQuotaSetDefaultPlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("plan not found"),
	},
	OpQuotaListAllowances: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid request parameters"),
	},
	OpQuotaCreateAllowance: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid allowance data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("user not found"),
	},
	OpQuotaUpdateAllowance: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid allowance data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("allowance not found"),
	},
	OpQuotaDeleteAllowance: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("allowance not found"),
	},
	OpQuotaGetStats: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
	},
	OpQuotaReconcile: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
	},
	OpQuotaCleanup: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
	},
	OpQuotaListUserConfigs: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid request parameters"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("invalid endpoint"),
	},
	OpQuotaUpdateUserConfig: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid user configuration data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("user not found"),
	},
	OpQuotaResetUserPlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("user not found"),
	},
}

// quotaOpHandler is the shared operation handler for quota operations (lazily initialized).
var quotaOpHandler = initQuotaOpHandler()

// initQuotaOpHandler initializes the OpHandler with quota operation mappings.
func initQuotaOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultQuotaOperationName

	for opID, name := range quotaOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range quotaHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handleQuotaResponse wraps OpHandler.HandleResponse.
func handleQuotaResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return quotaOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validateQuotaJSON200 wraps OpHandler.ValidateJSON200.
func validateQuotaJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(quotaOpHandler, respStatusCode, body, json200, op)
}

// validateQuotaJSON201 wraps OpHandler.ValidateJSON201.
func validateQuotaJSON201[T any](respStatusCode int, json201 *T, nilMsg string, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := []byte(nilMsg)
	return internalhttp.ValidateJSON201(quotaOpHandler, respStatusCode, body, json201, op)
}
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

// UserQuotaConfig represents a user's quota configuration.
// Embeds the generated admin.UserQuotaConfigResponse to reuse all fields.
type UserQuotaConfig struct {
	admin.UserQuotaConfigResponse
}

// SystemStats represents system-wide quota statistics.
type SystemStats struct {
	admin.SystemStatsResponse
}

// UserQuotaConfigUpdate represents a user quota configuration update request.
// This is an alias of the generated type for convenience.
type UserQuotaConfigUpdate = admin.UserQuotaConfigUpdateRequest

// QuotaService provides methods for managing quotas.
type QuotaService struct {
	client   admin.ClientWithResponsesInterface
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
			UploadLimitBytes:   limits.UploadLimitBytes,
			UploadThreshold:    limits.UploadThreshold,
			DownloadLimitBytes: limits.DownloadLimitBytes,
			DownloadThreshold:  limits.DownloadThreshold,
			StorageLimitBytes:  limits.StorageLimitBytes,
			StorageThreshold:   limits.StorageThreshold,
			WindowDuration:     limits.WindowDuration,
			WindowStartHour:    limits.WindowStartHour,
			WindowTimezone:     limits.WindowTimezone,
			WindowType:         limits.WindowType,
		},
	}
}

// QuotaLimits defines quota limit configuration.
type QuotaLimits struct {
	UploadLimitBytes   int
	UploadThreshold    int
	DownloadLimitBytes int
	DownloadThreshold  int
	StorageLimitBytes  int
	StorageThreshold   int
	WindowDuration     int
	WindowStartHour    int
	WindowTimezone     string
	WindowType         string
}

// ListPlans lists all quota plans.
func (q *QuotaService) ListPlans(ctx context.Context) ([]*QuotaPlan, int, error) {
	resp, err := q.client.GetApiQuotaPlansWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list plans: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaListPlans, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list plans response did not contain data")
	}

	plans := lo.Map(resp.JSON200.Data, func(plan admin.QuotaPlanResponse, _ int) *QuotaPlan {
		return &QuotaPlan{QuotaPlanResponse: plan}
	})

	return plans, resp.JSON200.Total, nil
}

// CreatePlan creates a new quota plan.
func (q *QuotaService) CreatePlan(ctx context.Context, plan *QuotaPlan) (*QuotaPlan, error) {
	reqBody := admin.QuotaPlanRequest{
		Name:               plan.Name,
		Description:        plan.Description,
		UploadLimitBytes:   plan.UploadLimitBytes,
		UploadThreshold:    plan.UploadThreshold,
		DownloadLimitBytes: plan.DownloadLimitBytes,
		DownloadThreshold:  plan.DownloadThreshold,
		StorageLimitBytes:  plan.StorageLimitBytes,
		StorageThreshold:   plan.StorageThreshold,
		WindowDuration:     plan.WindowDuration,
		WindowStartHour:    plan.WindowStartHour,
		WindowTimezone:     plan.WindowTimezone,
		WindowType:         plan.WindowType,
		IsActive:           plan.IsActive,
	}

	resp, err := q.client.PostApiQuotaPlansWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	data, err := validateQuotaJSON201(resp.StatusCode(), resp.JSON201, "create plan response did not contain data", OpQuotaCreatePlan)
	if err != nil {
		return nil, err
	}

	return &QuotaPlan{QuotaPlanResponse: *data}, nil
}

// GetPlan retrieves a quota plan by ID.
func (q *QuotaService) GetPlan(ctx context.Context, planID string) (*QuotaPlan, error) {
	resp, err := q.client.GetApiQuotaPlansPlanIDWithResponse(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
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
		UploadLimitBytes:   plan.UploadLimitBytes,
		UploadThreshold:    plan.UploadThreshold,
		DownloadLimitBytes: plan.DownloadLimitBytes,
		DownloadThreshold:  plan.DownloadThreshold,
		StorageLimitBytes:  plan.StorageLimitBytes,
		StorageThreshold:   plan.StorageThreshold,
		WindowDuration:     plan.WindowDuration,
		WindowStartHour:    plan.WindowStartHour,
		WindowTimezone:     plan.WindowTimezone,
		WindowType:         plan.WindowType,
		IsActive:           plan.IsActive,
	}

	resp, err := q.client.PutApiQuotaPlansPlanIDWithResponse(ctx, planID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
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
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaDeletePlan, []int{stdhttp.StatusNoContent})
}

// SetDefaultPlan sets a quota plan as the default for new users.
func (q *QuotaService) SetDefaultPlan(ctx context.Context, planID string) error {
	resp, err := q.client.PostApiQuotaPlansPlanIDDefaultWithResponse(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to set default plan: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaSetDefaultPlan, []int{stdhttp.StatusNoContent})
}

// ListAllowances lists all quota allowances.
func (q *QuotaService) ListAllowances(ctx context.Context) ([]*QuotaAllowance, int, error) {
	resp, err := q.client.GetApiQuotaAllowancesWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list allowances: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaListAllowances, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list allowances response did not contain data")
	}

	allowances := lo.Map(resp.JSON200.Data, func(grant admin.AllowanceGrantResponse, _ int) *QuotaAllowance {
		return &QuotaAllowance{AllowanceGrantResponse: grant}
	})

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
		return nil, fmt.Errorf("failed to create allowance: %w", err)
	}

	data, err := validateQuotaJSON201(resp.StatusCode(), resp.JSON201, "create allowance response did not contain data", OpQuotaCreateAllowance)
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
		return nil, fmt.Errorf("failed to update allowance: %w", err)
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
		return fmt.Errorf("failed to delete allowance: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaDeleteAllowance, []int{stdhttp.StatusNoContent})
}

// GetStats retrieves system-wide quota statistics.
func (q *QuotaService) GetStats(ctx context.Context) (*SystemStats, error) {
	resp, err := q.client.GetApiQuotaSystemStatsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
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
		return "", 0, fmt.Errorf("failed to reconcile: %w", err)
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
		return 0, fmt.Errorf("failed to cleanup: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaCleanup, []int{stdhttp.StatusOK}); err != nil {
		return 0, err
	}

	if resp.JSON200 == nil {
		return 0, fmt.Errorf("cleanup response did not contain data")
	}

	return resp.JSON200.RecordsDeleted, nil
}

// ListUserConfigs lists all user quota configurations with pagination.
func (q *QuotaService) ListUserConfigs(ctx context.Context) ([]*UserQuotaConfig, int, error) {
	resp, err := q.client.GetApiQuotaUserConfigsWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list user configs: %w", err)
	}

	if err := handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaListUserConfigs, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	configs := lo.Map(resp.JSON200.Data, func(config admin.UserQuotaConfigResponse, _ int) *UserQuotaConfig {
		return &UserQuotaConfig{UserQuotaConfigResponse: config}
	})

	return configs, resp.JSON200.Total, nil
}

// UpdateUserConfig updates a user's quota configuration.
func (q *QuotaService) UpdateUserConfig(ctx context.Context, userID int, config *admin.UserQuotaConfigUpdateRequest) (*UserQuotaConfig, error) {
	resp, err := q.client.PutApiQuotaUserConfigsUserIDWithResponse(ctx, fmt.Sprintf("%d", userID), *config)
	if err != nil {
		return nil, fmt.Errorf("failed to update user config: %w", err)
	}

	data, err := validateQuotaJSON200(resp.StatusCode(), resp.JSON200, OpQuotaUpdateUserConfig)
	if err != nil {
		return nil, err
	}

	return &UserQuotaConfig{UserQuotaConfigResponse: *data}, nil
}

// ResetUserPlan removes a user's assigned quota plan (sets to NULL).
func (q *QuotaService) ResetUserPlan(ctx context.Context, userID int) error {
	resp, err := q.client.DeleteApiQuotaUserConfigsUserIDPlanWithResponse(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		return fmt.Errorf("failed to reset user plan: %w", err)
	}

	return handleQuotaResponse(resp.StatusCode(), resp.Body, OpQuotaResetUserPlan, []int{stdhttp.StatusNoContent})
}

// SetRequestExecutor sets the underlying admin client for the quota service.
// Used for testing with mock clients.
func (q *QuotaService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	q.client = client
}
