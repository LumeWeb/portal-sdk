package admin

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"github.com/samber/lo"
	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

const (
	// Billing operation identifiers for error message mapping
	OpBillingListCredits = 200 + iota
	OpBillingCreateCredit
	OpBillingGetCredit
	OpBillingDeleteCredit
	OpBillingRestoreCredit
	OpBillingPurgeCredits
	OpBillingGetUserBalance
	OpBillingGetUserDeletedCredits
	OpBillingListPriceLines
	OpBillingCreatePriceLine
	OpBillingUpdatePriceLine
	OpBillingDeletePriceLine
	OpBillingListPricingPlans
	OpBillingCreatePricingPlan
	OpBillingUpdatePricingPlan
	OpBillingDeletePricingPlan
	OpBillingListPricingPlanPeriods
	OpBillingCreatePricingPlanPeriod
	OpBillingGetPricingPlanPeriod
	OpBillingUpdatePricingPlanPeriod
	OpBillingDeletePricingPlanPeriod
)

const defaultBillingOperationName = "billing operation"

// operationString maps billing operation IDs to their string names.
var billingOperationString = map[int]string{
	OpBillingListCredits:             "list billing credits",
	OpBillingCreateCredit:            "create billing credit",
	OpBillingGetCredit:               "get billing credit",
	OpBillingDeleteCredit:            "delete billing credit",
	OpBillingRestoreCredit:           "restore billing credit",
	OpBillingPurgeCredits:            "purge billing credits",
	OpBillingGetUserBalance:          "get user balance",
	OpBillingGetUserDeletedCredits:   "get user deleted credits",
	OpBillingListPriceLines:          "list price lines",
	OpBillingCreatePriceLine:         "create price line",
	OpBillingUpdatePriceLine:         "update price line",
	OpBillingDeletePriceLine:         "delete price line",
	OpBillingListPricingPlans:        "list pricing plans",
	OpBillingCreatePricingPlan:       "create pricing plan",
	OpBillingUpdatePricingPlan:       "update pricing plan",
	OpBillingDeletePricingPlan:       "delete pricing plan",
	OpBillingListPricingPlanPeriods:  "list pricing plan periods",
	OpBillingCreatePricingPlanPeriod: "create pricing plan period",
	OpBillingGetPricingPlanPeriod:    "get pricing plan period",
	OpBillingUpdatePricingPlanPeriod: "update pricing plan period",
	OpBillingDeletePricingPlanPeriod: "delete pricing plan period",
}

// httpErrorMessages maps billing operation IDs to their custom status code error messages.
var billingHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpBillingListCredits: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
	},
	OpBillingCreateCredit: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid credit data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("user not found"),
	},
	OpBillingGetCredit: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("credit not found"),
	},
	OpBillingDeleteCredit: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("credit not found"),
	},
	OpBillingRestoreCredit: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("credit not found"),
	},
	OpBillingPurgeCredits: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid purge request"),
	},
	OpBillingGetUserBalance: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("user not found"),
	},
	OpBillingGetUserDeletedCredits: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("user not found"),
	},
	OpBillingListPriceLines: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
	},
	OpBillingCreatePriceLine: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid price line data"),
	},
	OpBillingUpdatePriceLine: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid price line data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("price line not found"),
	},
	OpBillingDeletePriceLine: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("price line not found"),
	},
	OpBillingListPricingPlans: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
	},
	OpBillingCreatePricingPlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid pricing plan data"),
	},
	OpBillingUpdatePricingPlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid pricing plan data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("pricing plan not found"),
	},
	OpBillingDeletePricingPlan: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("pricing plan not found"),
	},
	OpBillingListPricingPlanPeriods: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
	},
	OpBillingCreatePricingPlanPeriod: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid pricing plan period data"),
	},
	OpBillingGetPricingPlanPeriod: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("pricing plan period not found"),
	},
	OpBillingUpdatePricingPlanPeriod: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusBadRequest:   internalhttp.PlainError("invalid pricing plan period data"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("pricing plan period not found"),
	},
	OpBillingDeletePricingPlanPeriod: {
		stdhttp.StatusUnauthorized: internalhttp.AuthError("authentication required"),
		stdhttp.StatusForbidden:    internalhttp.PlainError("insufficient permissions"),
		stdhttp.StatusNotFound:     internalhttp.PlainError("pricing plan period not found"),
	},
}

// handleBillingResponse processes an HTTP response using the billing error message map.
// op: the operation ID (used to lookup custom error messages)
// successCodes: status codes that indicate success (e.g., []int{stdhttp.StatusOK})
// Returns nil for success codes, custom error from global map, or generic error with body.
func handleBillingResponse(statusCode int, body []byte, op int, successCodes []int) error {
	// Check if status code is in success codes
	for _, code := range successCodes {
		if statusCode == code {
			return nil
		}
	}

	// Check for custom error message in global map
	if errorMessages, ok := billingHTTPErrorMessages[op]; ok {
		if factory, ok := errorMessages[statusCode]; ok {
			return factory.Error()
		}
	}

	// Get operation name for generic error
	opName := billingOperationString[op]
	if opName == "" {
		opName = defaultBillingOperationName
	}

	// Generic error with body
	return fmt.Errorf("%s failed with status %d: %s", opName, statusCode, string(body))
}

// validateBillingJSON200 validates HTTP 200 responses with JSON200 data.
func validateBillingJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	if respStatusCode != stdhttp.StatusOK {
		// Check for custom error message in global map first
		if errorMessages, ok := billingHTTPErrorMessages[op]; ok {
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

// validateBillingJSON201 validates HTTP 201 responses with JSON201 data.
func validateBillingJSON201[T any](respStatusCode int, json201 *T, nilMsg string, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	if respStatusCode != stdhttp.StatusCreated {
		// Check for custom error message in global map first
		if errorMessages, ok := billingHTTPErrorMessages[op]; ok {
			if factory, ok := errorMessages[respStatusCode]; ok {
				return nil, factory.Error()
			}
		}
		return nil, fmt.Errorf("expected status 201, got %d", respStatusCode)
	}
	if json201 == nil {
		return nil, fmt.Errorf("%s", nilMsg)
	}
	return json201, nil
}

// ErrBillingDefault is a generic billing error type.
var ErrBillingDefault = fmt.Errorf("billing operation failed")

// Credit represents a billing credit entry.
// Embeds the generated admin.CreditResponse to reuse all fields.
type Credit struct {
	admin.CreditResponse
}

// CreditItem represents a lightweight billing credit item (used in lists).
// Embeds the generated admin.CreditItem to reuse all fields.
type CreditItem struct {
	admin.CreditItem
}

// UserBalance represents a user's current balance.
// Embeds the generated admin.BalanceResponse to reuse all fields.
type UserBalance struct {
	admin.BalanceResponse
}

// CreditCreateRequest represents a request to create a new credit.
type CreditCreateRequest struct {
	UserID          int
	Amount          string
	TransactionType string
	Direction       string
	Description     string
	ReferenceID     string
	ReferenceType   string
}

// CreditPurgeRequest represents a request to purge soft-deleted credits.
type CreditPurgeRequest struct {
	OlderThan string // Duration string (e.g., "30d", "24h", "1y")
}

// PriceLine represents a price line entry.
// Embeds the generated admin.PriceLineResponse to reuse all fields.
type PriceLine struct {
	admin.PriceLineResponse
}

// PriceLineCreateRequest represents a request to create a new price line.
type PriceLineCreateRequest struct {
	Name        string
	Description string
	IsActive    bool
	IsDefault   bool
}

// PriceLineUpdateRequest represents a request to update an existing price line.
type PriceLineUpdateRequest struct {
	Name        string
	Description string
	IsActive    bool
	IsDefault   bool
}

// PricingPlan represents a pricing plan entry.
// Embeds the generated admin.PricingPlanResponse to reuse all fields.
type PricingPlan struct {
	admin.PricingPlanResponse
}

// PricingPlanCreateRequest represents a request to create a new pricing plan.
type PricingPlanCreateRequest struct {
	Name           string
	Description    string
	Currency       string
	IsActive       bool
	IsPublic       bool
	PricingPeriods []PricingPlanPeriod
}

// PricingPlanItem represents a lightweight pricing plan item (used in lists).
// Embeds the generated admin.PricingPlanItem to reuse all fields.
type PricingPlanItem struct {
	admin.PricingPlanItem
}

// PricingPlanUpdateRequest represents a request to update an existing pricing plan.
type PricingPlanUpdateRequest struct {
	Name           string
	Description    string
	Currency       string
	IsActive       bool
	IsPublic       bool
	PricingPeriods []PricingPlanPeriod
}

// PricingPlanPeriod represents a pricing plan period entry.
// Embeds the generated admin.PricingPlanPeriodDTO to reuse all fields.
type PricingPlanPeriod struct {
	admin.PricingPlanPeriodDTO
}

// PricingPlanPeriodCreateRequest represents a request to create a new pricing plan period.
type PricingPlanPeriodCreateRequest struct {
	Cadence       string
	PriceUsd      float32
	PricingPlanId int
	QuotaPlanId   int
	RollingDays   *int // Optional
}

// PricingPlanPeriodUpdateRequest represents a request to update an existing pricing plan period.
type PricingPlanPeriodUpdateRequest struct {
	Cadence     string
	PriceUsd    float32
	QuotaPlanId int
	RollingDays *int // Optional
}

// BillingService provides methods for managing billing operations.
type BillingService struct {
	client admin.ClientWithResponsesInterface
}

// ListCredits lists all credits with optional filtering.
func (b *BillingService) ListCredits(ctx context.Context, params *GetApiBillingCreditsParams) ([]*CreditItem, error) {
	var adminParams *admin.GetApiBillingCreditsParams
	if params != nil {
		adminParams = (*admin.GetApiBillingCreditsParams)(params)
	}

	resp, err := b.client.GetApiBillingCreditsWithResponse(ctx, adminParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list credits: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListCredits, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list credits response did not contain data")
	}

	credits := lo.Map(resp.JSON200.Data, func(credit admin.CreditItem, _ int) *CreditItem {
		return &CreditItem{CreditItem: credit}
	})

	return credits, nil
}

// CreateCredit creates a new credit entry.
func (b *BillingService) CreateCredit(ctx context.Context, req *CreditCreateRequest) (*Credit, error) {
	reqBody := admin.CreditCreateRequest{
		UserId:          req.UserID,
		Amount:          req.Amount,
		TransactionType: req.TransactionType,
		Direction:       req.Direction,
	}

	// Set optional fields if provided
	if req.Description != "" {
		reqBody.Description = &req.Description
	}
	if req.ReferenceID != "" {
		reqBody.ReferenceId = &req.ReferenceID
	}
	if req.ReferenceType != "" {
		reqBody.ReferenceType = &req.ReferenceType
	}

	resp, err := b.client.PostApiBillingCreditsWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create credit: %w", err)
	}

	data, err := validateBillingJSON201(resp.StatusCode(), resp.JSON201, "create credit response did not contain data", OpBillingCreateCredit)
	if err != nil {
		return nil, err
	}

	return &Credit{CreditResponse: *data}, nil
}

// GetCredit retrieves a credit by ID.
func (b *BillingService) GetCredit(ctx context.Context, creditID string) (*Credit, error) {
	resp, err := b.client.GetApiBillingCreditsIdWithResponse(ctx, creditID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credit: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingGetCredit)
	if err != nil {
		return nil, err
	}

	return &Credit{CreditResponse: *data}, nil
}

// DeleteCredit soft deletes a credit by ID.
func (b *BillingService) DeleteCredit(ctx context.Context, creditID string) error {
	resp, err := b.client.DeleteApiBillingCreditsIdWithResponse(ctx, creditID)
	if err != nil {
		return fmt.Errorf("failed to delete credit: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingDeleteCredit, []int{stdhttp.StatusNoContent})
}

// RestoreCredit restores a soft-deleted credit by ID.
func (b *BillingService) RestoreCredit(ctx context.Context, creditID string) (*Credit, error) {
	resp, err := b.client.PostApiBillingCreditsIdRestoreWithResponse(ctx, creditID)
	if err != nil {
		return nil, fmt.Errorf("failed to restore credit: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingRestoreCredit)
	if err != nil {
		return nil, err
	}

	return &Credit{CreditResponse: *data}, nil
}

// PurgeCredits permanently removes soft-deleted credits older than specified duration.
func (b *BillingService) PurgeCredits(ctx context.Context, req *CreditPurgeRequest) (int, error) {
	reqBody := admin.CreditPurgeRequest{
		OlderThan: req.OlderThan,
	}

	resp, err := b.client.PostApiBillingCreditsPurgeWithResponse(ctx, reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to purge credits: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingPurgeCredits, []int{stdhttp.StatusOK}); err != nil {
		return 0, err
	}

	if resp.JSON200 == nil {
		return 0, fmt.Errorf("purge credits response did not contain data")
	}

	return resp.JSON200.PurgedCount, nil
}

// GetUserBalance retrieves the current balance for a user.
func (b *BillingService) GetUserBalance(ctx context.Context, userID string) (*UserBalance, error) {
	resp, err := b.client.GetApiBillingUsersUserIdBalanceWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingGetUserBalance)
	if err != nil {
		return nil, err
	}

	return &UserBalance{BalanceResponse: *data}, nil
}

// GetUserDeletedCredits retrieves soft-deleted credits for a user.
func (b *BillingService) GetUserDeletedCredits(ctx context.Context, userID string, params *GetApiBillingUsersUserIdDeletedCreditsParams) ([]*CreditItem, error) {
	var adminParams *admin.GetApiBillingUsersUserIdDeletedCreditsParams
	if params != nil {
		adminParams = (*admin.GetApiBillingUsersUserIdDeletedCreditsParams)(params)
	}

	resp, err := b.client.GetApiBillingUsersUserIdDeletedCreditsWithResponse(ctx, userID, adminParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get user deleted credits: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingGetUserDeletedCredits, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get user deleted credits response did not contain data")
	}

	credits := lo.Map(resp.JSON200.Data, func(credit admin.CreditItem, _ int) *CreditItem {
		return &CreditItem{CreditItem: credit}
	})

	return credits, nil
}

// ListPriceLines lists all price lines.
func (b *BillingService) ListPriceLines(ctx context.Context) ([]*PriceLine, error) {
	resp, err := b.client.GetApiBillingPriceLinesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list price lines: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListPriceLines, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list price lines response did not contain data")
	}

	lines := lo.Map(resp.JSON200.Data, func(line admin.PriceLineResponse, _ int) *PriceLine {
		return &PriceLine{PriceLineResponse: line}
	})

	return lines, nil
}

// CreatePriceLine creates a new price line.
func (b *BillingService) CreatePriceLine(ctx context.Context, req *PriceLineCreateRequest) (*PriceLine, error) {
	reqBody := admin.PriceLineCreateRequest{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		IsDefault:   req.IsDefault,
	}

	resp, err := b.client.PostApiBillingPriceLinesWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create price line: %w", err)
	}

	data, err := validateBillingJSON201(resp.StatusCode(), resp.JSON201, "create price line response did not contain data", OpBillingCreatePriceLine)
	if err != nil {
		return nil, err
	}

	return &PriceLine{PriceLineResponse: *data}, nil
}

// UpdatePriceLine updates an existing price line.
func (b *BillingService) UpdatePriceLine(ctx context.Context, priceLineID string, req *PriceLineUpdateRequest) (*PriceLine, error) {
	reqBody := admin.PriceLineUpdateRequest{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		IsDefault:   req.IsDefault,
	}

	resp, err := b.client.PutApiBillingPriceLinesIdWithResponse(ctx, priceLineID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update price line: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingUpdatePriceLine)
	if err != nil {
		return nil, err
	}

	return &PriceLine{PriceLineResponse: *data}, nil
}

// DeletePriceLine deletes a price line by ID.
func (b *BillingService) DeletePriceLine(ctx context.Context, priceLineID string) error {
	resp, err := b.client.DeleteApiBillingPriceLinesIdWithResponse(ctx, priceLineID)
	if err != nil {
		return fmt.Errorf("failed to delete price line: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingDeletePriceLine, []int{stdhttp.StatusNoContent})
}

// ListPricingPlans lists all pricing plans.
func (b *BillingService) ListPricingPlans(ctx context.Context) ([]*PricingPlanItem, error) {
	resp, err := b.client.GetApiBillingPricingPlansWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pricing plans: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListPricingPlans, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list pricing plans response did not contain data")
	}

	plans := lo.Map(resp.JSON200.Data, func(plan admin.PricingPlanItem, _ int) *PricingPlanItem {
		return &PricingPlanItem{PricingPlanItem: plan}
	})

	return plans, nil
}

// CreatePricingPlan creates a new pricing plan.
func (b *BillingService) CreatePricingPlan(ctx context.Context, req *PricingPlanCreateRequest) (*PricingPlan, error) {
	periods := lo.Map(req.PricingPeriods, func(p PricingPlanPeriod, _ int) admin.PricingPlanPeriodDTO {
		return admin.PricingPlanPeriodDTO{
			Cadence:       p.Cadence,
			PriceUsd:      p.PriceUsd,
			PricingPlanId: p.PricingPlanId,
			QuotaPlanId:   p.QuotaPlanId,
			RollingDays:   p.RollingDays,
		}
	})

	reqBody := admin.PricingPlanCreateRequest{
		Name:           req.Name,
		Description:    req.Description,
		Currency:       req.Currency,
		IsActive:       req.IsActive,
		IsPublic:       req.IsPublic,
		PricingPeriods: periods,
	}

	resp, err := b.client.PostApiBillingPricingPlansWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create pricing plan: %w", err)
	}

	data, err := validateBillingJSON201(resp.StatusCode(), resp.JSON201, "create pricing plan response did not contain data", OpBillingCreatePricingPlan)
	if err != nil {
		return nil, err
	}

	return &PricingPlan{PricingPlanResponse: *data}, nil
}

// UpdatePricingPlan updates an existing pricing plan.
func (b *BillingService) UpdatePricingPlan(ctx context.Context, planID string, req *PricingPlanUpdateRequest) (*PricingPlan, error) {
	periods := lo.Map(req.PricingPeriods, func(p PricingPlanPeriod, _ int) admin.PricingPlanPeriodDTO {
		return admin.PricingPlanPeriodDTO{
			Cadence:       p.Cadence,
			PriceUsd:      p.PriceUsd,
			PricingPlanId: p.PricingPlanId,
			QuotaPlanId:   p.QuotaPlanId,
			RollingDays:   p.RollingDays,
		}
	})

	reqBody := admin.PricingPlanUpdateRequest{
		Name:           req.Name,
		Description:    req.Description,
		Currency:       req.Currency,
		IsActive:       req.IsActive,
		IsPublic:       req.IsPublic,
		PricingPeriods: periods,
	}

	resp, err := b.client.PutApiBillingPricingPlansIdWithResponse(ctx, planID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update pricing plan: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingUpdatePricingPlan)
	if err != nil {
		return nil, err
	}

	return &PricingPlan{PricingPlanResponse: *data}, nil
}

// DeletePricingPlan deletes a pricing plan by ID.
func (b *BillingService) DeletePricingPlan(ctx context.Context, planID string) error {
	resp, err := b.client.DeleteApiBillingPricingPlansIdWithResponse(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to delete pricing plan: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingDeletePricingPlan, []int{stdhttp.StatusNoContent})
}

// ListPricingPlanPeriods lists all pricing plan periods.
func (b *BillingService) ListPricingPlanPeriods(ctx context.Context) ([]*PricingPlanPeriod, error) {
	resp, err := b.client.GetApiBillingPricingPlanPeriodsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pricing plan periods: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListPricingPlanPeriods, []int{stdhttp.StatusOK}); err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list pricing plan periods response did not contain data")
	}

	periods := lo.Map(resp.JSON200.Data, func(period admin.PricingPlanPeriodDTO, _ int) *PricingPlanPeriod {
		return &PricingPlanPeriod{PricingPlanPeriodDTO: period}
	})

	return periods, nil
}

// CreatePricingPlanPeriod creates a new pricing plan period.
func (b *BillingService) CreatePricingPlanPeriod(ctx context.Context, req *PricingPlanPeriodCreateRequest) (*PricingPlanPeriod, error) {
	reqBody := admin.PricingPlanPeriodCreateRequest{
		Cadence:       req.Cadence,
		PriceUsd:      req.PriceUsd,
		PricingPlanId: req.PricingPlanId,
		QuotaPlanId:   req.QuotaPlanId,
		RollingDays:   req.RollingDays,
	}

	resp, err := b.client.PostApiBillingPricingPlanPeriodsWithResponse(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create pricing plan period: %w", err)
	}

	data, err := validateBillingJSON201(resp.StatusCode(), resp.JSON201, "create pricing plan period response did not contain data", OpBillingCreatePricingPlanPeriod)
	if err != nil {
		return nil, err
	}

	return &PricingPlanPeriod{PricingPlanPeriodDTO: *data}, nil
}

// GetPricingPlanPeriod retrieves a pricing plan period by ID.
func (b *BillingService) GetPricingPlanPeriod(ctx context.Context, periodID string) (*PricingPlanPeriod, error) {
	resp, err := b.client.GetApiBillingPricingPlanPeriodsIdWithResponse(ctx, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing plan period: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingGetPricingPlanPeriod)
	if err != nil {
		return nil, err
	}

	return &PricingPlanPeriod{PricingPlanPeriodDTO: *data}, nil
}

// UpdatePricingPlanPeriod updates an existing pricing plan period.
func (b *BillingService) UpdatePricingPlanPeriod(ctx context.Context, periodID string, req *PricingPlanPeriodUpdateRequest) (*PricingPlanPeriod, error) {
	reqBody := admin.PricingPlanPeriodUpdateRequest{
		Cadence:     req.Cadence,
		PriceUsd:    req.PriceUsd,
		QuotaPlanId: req.QuotaPlanId,
		RollingDays: req.RollingDays,
	}

	resp, err := b.client.PutApiBillingPricingPlanPeriodsIdWithResponse(ctx, periodID, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update pricing plan period: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingUpdatePricingPlanPeriod)
	if err != nil {
		return nil, err
	}

	return &PricingPlanPeriod{PricingPlanPeriodDTO: *data}, nil
}

// DeletePricingPlanPeriod deletes a pricing plan period by ID.
func (b *BillingService) DeletePricingPlanPeriod(ctx context.Context, periodID string) error {
	resp, err := b.client.DeleteApiBillingPricingPlanPeriodsIdWithResponse(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to delete pricing plan period: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingDeletePricingPlanPeriod, []int{stdhttp.StatusNoContent})
}

// GetApiBillingCreditsParams defines parameters for ListCredits.
// This type aliases the generated type for convenience.
type GetApiBillingCreditsParams admin.GetApiBillingCreditsParams

// GetApiBillingUsersUserIdDeletedCreditsParams defines parameters for GetUserDeletedCredits.
// This type aliases the generated type for convenience.
type GetApiBillingUsersUserIdDeletedCreditsParams admin.GetApiBillingUsersUserIdDeletedCreditsParams
