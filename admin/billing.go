package admin

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"time"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// Billing error sentinels - DRY error factories reused across operation mappings
var (
	// Auth errors
	errAuthRequired            = internalhttp.AuthError("authentication required")
	errInsufficientPermissions = internalhttp.PlainError("insufficient permissions")

	// Credit errors
	errInvalidCreditData = internalhttp.PlainError("invalid credit data")
	errCreditNotFound    = internalhttp.PlainError("credit not found")

	// User errors
	errUserNotFound = internalhttp.PlainError("user not found")

	// Purge errors
	errInvalidPurgeRequest = internalhttp.PlainError("invalid purge request")

	// Price line errors
	errInvalidPriceLineData = internalhttp.PlainError("invalid price line data")
	errPriceLineNotFound    = internalhttp.PlainError("price line not found")

	// Pricing plan errors
	errInvalidPricingPlanData = internalhttp.PlainError("invalid pricing plan data")
	errPricingPlanNotFound    = internalhttp.PlainError("pricing plan not found")

	// Pricing plan period errors
	errInvalidPricingPlanPeriodData = internalhttp.PlainError("invalid pricing plan period data")
	errPricingPlanPeriodNotFound    = internalhttp.PlainError("pricing plan period not found")

	// Subscriber errors
	errSubscriberNotFound = internalhttp.PlainError("subscriber not found")

	// Gateway errors
	errGatewayNotFound = internalhttp.PlainError("gateway not found")

	// Subscription errors
	errInvalidCancellationRequest    = internalhttp.PlainError("invalid cancellation request")
	errScheduledCancellationNotFound = internalhttp.PlainError("no scheduled cancellation found")
	errCannotAbortCancellation       = internalhttp.PlainError("cannot abort cancellation")
	errInvalidPlanChangeRequest      = internalhttp.PlainError("invalid plan change request")
	errCannotPauseSubscription       = internalhttp.PlainError("subscription cannot be paused")
	errCannotResumeSubscription      = internalhttp.PlainError("subscription cannot be resumed")
	errNoActiveSubscription          = internalhttp.PlainError("no active subscription found")
	errNoPausedSubscription          = internalhttp.PlainError("no paused subscription found")

	// Plan errors
	errInvalidPlanData         = internalhttp.PlainError("invalid plan data")
	errPriceLineOrPlanNotFound = internalhttp.PlainError("price line or plan not found")
	errInvalidPositionData     = internalhttp.PlainError("invalid position data")
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
	OpBillingGetPriceLine
	OpBillingUpdatePriceLine
	OpBillingDeletePriceLine
	OpBillingListPricingPlans
	OpBillingCreatePricingPlan
	OpBillingUpdatePricingPlan
	OpBillingGetPricingPlan
	OpBillingDeletePricingPlan
	OpBillingListPricingPlanPeriods
	OpBillingCreatePricingPlanPeriod
	OpBillingGetPricingPlanPeriod
	OpBillingUpdatePricingPlanPeriod
	OpBillingDeletePricingPlanPeriod
	OpBillingListSubscribers
	OpBillingGetSubscriber
	OpBillingListGatewaySubscribers
	OpBillingGetUserSubscribers
	OpBillingCancelUserSubscription
	OpBillingAbortUserSubscriptionCancellation
	OpBillingChangeUserPlan
	OpBillingPauseUserSubscription
	OpBillingResumeUserSubscription
	OpBillingAddPlanToPriceLine
	OpBillingDeletePlanFromPriceLine
	OpBillingUpdatePlanPosition
	OpBillingSyncPricingPlan
	OpBillingSyncAllPricingPlans
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
	OpBillingGetPriceLine:            "get price line",
	OpBillingUpdatePriceLine:         "update price line",
	OpBillingDeletePriceLine:         "delete price line",
	OpBillingListPricingPlans:        "list pricing plans",
	OpBillingCreatePricingPlan:       "create pricing plan",
	OpBillingUpdatePricingPlan:       "update pricing plan",
	OpBillingGetPricingPlan:          "get pricing plan",
	OpBillingDeletePricingPlan:       "delete pricing plan",
	OpBillingListPricingPlanPeriods:  "list pricing plan periods",
	OpBillingCreatePricingPlanPeriod: "create pricing plan period",
	OpBillingGetPricingPlanPeriod:    "get pricing plan period",
	OpBillingUpdatePricingPlanPeriod: "update pricing plan period",
	OpBillingDeletePricingPlanPeriod: "delete pricing plan period",
	OpBillingListSubscribers:          "list subscribers",
	OpBillingGetSubscriber:            "get subscriber",
	OpBillingListGatewaySubscribers:   "list gateway subscribers",
	OpBillingGetUserSubscribers:       "get user subscribers",
	OpBillingCancelUserSubscription:             "cancel user subscription",
	OpBillingAbortUserSubscriptionCancellation: "abort user subscription cancellation",
	OpBillingChangeUserPlan:                     "change user plan",
	OpBillingPauseUserSubscription:              "pause user subscription",
	OpBillingResumeUserSubscription:             "resume user subscription",
	OpBillingAddPlanToPriceLine:       "add plan to price line",
	OpBillingDeletePlanFromPriceLine:  "delete plan from price line",
	OpBillingUpdatePlanPosition:       "update plan position",
	OpBillingSyncPricingPlan:          "sync pricing plan to gateway",
	OpBillingSyncAllPricingPlans:      "sync all pricing plans to gateways",
}

// httpErrorMessages maps billing operation IDs to their custom status code error messages.
var billingHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpBillingListCredits: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpBillingCreateCredit: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidCreditData,
		stdhttp.StatusNotFound:     errUserNotFound,
	},
	OpBillingGetCredit: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errCreditNotFound,
	},
	OpBillingDeleteCredit: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errCreditNotFound,
	},
	OpBillingRestoreCredit: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errCreditNotFound,
	},
	OpBillingPurgeCredits: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPurgeRequest,
	},
	OpBillingGetUserBalance: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errUserNotFound,
	},
	OpBillingGetUserDeletedCredits: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errUserNotFound,
	},
	OpBillingListPriceLines: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpBillingCreatePriceLine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPriceLineData,
	},
	OpBillingGetPriceLine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPriceLineNotFound,
	},
	OpBillingUpdatePriceLine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPriceLineData,
		stdhttp.StatusNotFound:     errPriceLineNotFound,
	},
	OpBillingDeletePriceLine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPriceLineNotFound,
	},
	OpBillingListPricingPlans: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpBillingCreatePricingPlan: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPricingPlanData,
	},
	OpBillingUpdatePricingPlan: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPricingPlanData,
		stdhttp.StatusNotFound:     errPricingPlanNotFound,
	},
	OpBillingGetPricingPlan: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPricingPlanNotFound,
	},
	OpBillingDeletePricingPlan: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPricingPlanNotFound,
	},
	OpBillingListPricingPlanPeriods: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpBillingCreatePricingPlanPeriod: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPricingPlanPeriodData,
	},
	OpBillingGetPricingPlanPeriod: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPricingPlanPeriodNotFound,
	},
	OpBillingUpdatePricingPlanPeriod: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errInvalidPricingPlanPeriodData,
		stdhttp.StatusNotFound:     errPricingPlanPeriodNotFound,
	},
	OpBillingDeletePricingPlanPeriod: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPricingPlanPeriodNotFound,
	},
	OpBillingListSubscribers: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpBillingGetSubscriber: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errSubscriberNotFound,
	},
	OpBillingListGatewaySubscribers: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errGatewayNotFound,
	},
	OpBillingGetUserSubscribers: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errUserNotFound,
	},
	OpBillingCancelUserSubscription: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errUserNotFound,
		stdhttp.StatusBadRequest:   errInvalidCancellationRequest,
	},
	OpBillingAbortUserSubscriptionCancellation: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errScheduledCancellationNotFound,
		stdhttp.StatusBadRequest:   errCannotAbortCancellation,
	},
	OpBillingChangeUserPlan: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errUserNotFound,
		stdhttp.StatusBadRequest:   errInvalidPlanChangeRequest,
	},
	OpBillingPauseUserSubscription: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errNoActiveSubscription,
		stdhttp.StatusBadRequest:   errCannotPauseSubscription,
	},
	OpBillingResumeUserSubscription: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errNoPausedSubscription,
		stdhttp.StatusBadRequest:   errCannotResumeSubscription,
	},
	OpBillingAddPlanToPriceLine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPriceLineNotFound,
		stdhttp.StatusBadRequest:   errInvalidPlanData,
	},
	OpBillingDeletePlanFromPriceLine: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPriceLineNotFound,
		stdhttp.StatusBadRequest:   errInvalidPlanData,
	},
	OpBillingUpdatePlanPosition: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPriceLineOrPlanNotFound,
		stdhttp.StatusBadRequest:   errInvalidPositionData,
	},
	OpBillingSyncPricingPlan: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errPricingPlanNotFound,
		stdhttp.StatusBadRequest:   errInvalidPricingPlanData,
	},
	OpBillingSyncAllPricingPlans: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
}

// billingOpHandler is the shared operation handler for billing operations (lazily initialized).
var billingOpHandler = initBillingOpHandler()

// initBillingOpHandler initializes the OpHandler with billing operation mappings.
func initBillingOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultBillingOperationName

	for opID, name := range billingOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range billingHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handleBillingResponse wraps OpHandler.HandleResponse.
func handleBillingResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return billingOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validateBillingJSON200 wraps OpHandler.ValidateJSON200.
func validateBillingJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(billingOpHandler, respStatusCode, body, json200, op)
}

// validateBillingJSON201 wraps OpHandler.ValidateJSON201.
func validateBillingJSON201[T any](respStatusCode int, json201 *T, nilMsg string, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := []byte(nilMsg)
	return internalhttp.ValidateJSON201(billingOpHandler, respStatusCode, body, json201, op)
}

// ErrBillingDefault is a generic billing error type.
var ErrBillingDefault = errors.New("billing operation failed")

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

// CreditCreateRequest represents a request to create a credit.
// This is an alias of the generated type for convenience.
type CreditCreateRequest = admin.CreditCreateRequest

// AddPlanToPriceLineRequest represents a request to add a plan to a price line.
type AddPlanToPriceLineRequest = admin.AddPlanToPriceLineRequest

// UpdatePlanPositionRequest represents a request to update a plan's position.
type UpdatePlanPositionRequest = admin.UpdatePlanPositionRequest

// CreditPurgeRequest represents a request to purge soft-deleted credits.
// This is an alias of the generated type for convenience.
type CreditPurgeRequest = admin.CreditPurgeRequest

// PriceLine represents a price line entry.
// Embeds the generated admin.PriceLineResponse to reuse all fields.
type PriceLine struct {
	admin.PriceLineResponse
}

// PriceLineDetailResponse represents a detailed price line with its associated plans.
type PriceLineDetailResponse struct {
	CreatedAt   time.Time          `json:"created_at"`
	Description string             `json:"description"`
	Id          int                `json:"id"`
	IsActive    bool               `json:"is_active"`
	IsDefault   bool               `json:"is_default"`
	Name        string             `json:"name"`
	Plans       []*PricingPlanItem `json:"plans,omitempty"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// PriceLineCreateRequest represents a request to create a new price line.
// This is an alias of the generated type for convenience.
type PriceLineCreateRequest = admin.PriceLineCreateRequest

// PriceLineUpdateRequest represents a request to update an existing price line.
// This is an alias of the generated type for convenience.
type PriceLineUpdateRequest = admin.PriceLineUpdateRequest

// PricingPlan represents a pricing plan entry.
// Embeds the generated admin.PricingPlanResponse to reuse all fields.
type PricingPlan struct {
	admin.PricingPlanResponse
}

// PricingPlanCreateRequest represents a request to create a new pricing plan.
type PricingPlanCreateRequest struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Features       []string            `json:"features"`
	Currency       string              `json:"currency"`
	IsActive       bool                `json:"is_active"`
	IsPublic       bool                `json:"is_public"`
	Position       *int                `json:"position,omitempty"`
	PricelineId    *int                `json:"priceline_id,omitempty"`
	PricingPeriods []PricingPlanPeriod `json:"pricing_periods"`
}

// toInternal converts to the generated type for API calls.
func (r *PricingPlanCreateRequest) toInternal() admin.PricingPlanCreateRequest {
	return admin.PricingPlanCreateRequest{
		Name:           r.Name,
		Description:    r.Description,
		Features:       r.Features,
		Currency:       r.Currency,
		IsActive:       r.IsActive,
		IsPublic:       r.IsPublic,
		Position:       r.Position,
		PricelineId:    r.PricelineId,
		PricingPeriods: lo.Map(r.PricingPeriods, func(p PricingPlanPeriod, _ int) admin.PricingPeriodCreateInput {
			return p.toCreateInput()
		}),
	}
}

// PricingPlanItem represents a lightweight pricing plan item (used in lists).
// Embeds the generated admin.PricingPlanItem to reuse all fields.
type PricingPlanItem struct {
	admin.PricingPlanItem
}

// PricingPlanUpdateRequest represents a request to update an existing pricing plan.
type PricingPlanUpdateRequest struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Features       []string            `json:"features"`
	Currency       string              `json:"currency"`
	IsActive       bool                `json:"is_active"`
	IsPublic       bool                `json:"is_public"`
	PricingPeriods []PricingPlanPeriod `json:"pricing_periods"`
}

// toInternal converts to the generated type for API calls.
func (r *PricingPlanUpdateRequest) toInternal() admin.PricingPlanUpdateRequest {
	return admin.PricingPlanUpdateRequest{
		Name:           r.Name,
		Description:    r.Description,
		Features:       r.Features,
		Currency:       r.Currency,
		IsActive:       r.IsActive,
		IsPublic:       r.IsPublic,
		PricingPeriods: lo.Map(r.PricingPeriods, func(p PricingPlanPeriod, _ int) admin.PricingPeriodInput {
			return p.toInput()
		}),
	}
}

// PricingPlanPeriod represents a pricing plan period entry.
// Embeds the generated admin.PricingPlanPeriodDTO to reuse all fields.
type PricingPlanPeriod struct {
	admin.PricingPlanPeriodDTO
}

// toCreateInput converts PricingPlanPeriod to PricingPeriodCreateInput for API calls.
func (p PricingPlanPeriod) toCreateInput() admin.PricingPeriodCreateInput {
	return admin.PricingPeriodCreateInput{
		AllowFree:   &p.AllowFree,
		Cadence:     p.Cadence,
		IsActive:    p.IsActive,
		PriceUsd:    p.PriceUsd,
		QuotaPlanId: p.QuotaPlanId,
		RollingDays: p.RollingDays,
	}
}

// toInput converts PricingPlanPeriod to PricingPeriodInput for API calls.
func (p PricingPlanPeriod) toInput() admin.PricingPeriodInput {
	id := p.Id
	return admin.PricingPeriodInput{
		AllowFree:   &p.AllowFree,
		Cadence:     p.Cadence,
		Id:          &id,
		IsActive:    p.IsActive,
		PriceUsd:    p.PriceUsd,
		QuotaPlanId: p.QuotaPlanId,
		RollingDays: p.RollingDays,
	}
}

// PricingPlanPeriodCreateRequest represents a request to create a new pricing plan period.
// This is an alias of the generated type for convenience.
type PricingPlanPeriodCreateRequest = admin.PricingPlanPeriodCreateRequest

// PricingPlanPeriodUpdateRequest represents a request to update an existing pricing plan period.
// This is an alias of the generated type for convenience.
type PricingPlanPeriodUpdateRequest = admin.PricingPlanPeriodUpdateRequest

// APIEndpointInfo describes an API endpoint returned by management operations.
type APIEndpointInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Subscriber represents a billing subscription subscriber.
type Subscriber struct {
	BillingPeriodEnd    *time.Time `json:"billing_period_end"`
	BillingPeriodStart  *time.Time `json:"billing_period_start"`
	CancelledAt         *time.Time `json:"cancelled_at"`
	CreatedAt           time.Time  `json:"created_at"`
	ExternalId          string     `json:"external_id"`
	GatewayType         string     `json:"gateway_type"`
	Id                  int        `json:"id"`
	IsActive            bool       `json:"is_active"`
	PaymentStatus       *string    `json:"payment_status"`
	PausedAt            *time.Time `json:"paused_at"`
	PreviousPlanId      *int       `json:"previous_plan_id"`
	PricingPlanPeriodId *int       `json:"pricing_plan_period_id"`
	SubscriptionId      string     `json:"subscription_id"`
	UpdatedAt           time.Time  `json:"updated_at"`
	UserId              int        `json:"user_id"`
	WillCancelAt        *time.Time `json:"will_cancel_at"`
}

// fromInternal populates the Subscriber from the generated admin type.
func (s *Subscriber) fromInternal(item admin.SubscriberItem) {
	s.BillingPeriodEnd = item.BillingPeriodEnd
	s.BillingPeriodStart = item.BillingPeriodStart
	s.CancelledAt = item.CancelledAt
	s.CreatedAt = item.CreatedAt
	s.ExternalId = item.ExternalId
	s.GatewayType = item.GatewayType
	s.Id = item.Id
	s.IsActive = item.IsActive
	s.PaymentStatus = item.PaymentStatus
	s.PausedAt = item.PausedAt
	s.PreviousPlanId = item.PreviousPlanId
	s.PricingPlanPeriodId = item.PricingPlanPeriodId
	s.SubscriptionId = item.SubscriptionId
	s.UpdatedAt = item.UpdatedAt
	s.UserId = item.UserId
	s.WillCancelAt = item.WillCancelAt
}

// ManagementResult represents the result of a billing management operation.
type ManagementResult struct {
	Action               string           `json:"action"`
	ApiEndpoint          *APIEndpointInfo  `json:"api_endpoint,omitempty"`
	CanAbort             bool             `json:"can_abort"`
	ConfirmationMessage  *string          `json:"confirmation_message"`
	EffectiveTime        *time.Time       `json:"effective_time"`
	ErrorMessage         *string          `json:"error_message"`
	RequiresConfirmation bool             `json:"requires_confirmation"`
	Status               string           `json:"status"`
	Url                  *string          `json:"url"`
}

// fromInternal populates the ManagementResult from the generated admin type.
func (m *ManagementResult) fromInternal(resp admin.ManagementResultResponse) {
	m.Action = resp.Action
	m.CanAbort = resp.CanAbort
	m.ConfirmationMessage = resp.ConfirmationMessage
	m.EffectiveTime = resp.EffectiveTime
	m.ErrorMessage = resp.ErrorMessage
	m.RequiresConfirmation = resp.RequiresConfirmation
	m.Status = resp.Status
	m.Url = resp.Url
	if resp.ApiEndpoint != nil {
		m.ApiEndpoint = &APIEndpointInfo{
			Method: resp.ApiEndpoint.Method,
			Path:   resp.ApiEndpoint.Path,
		}
	}
}

// Decimal is an alias for the decimal.Decimal type used in billing calculations.
type Decimal = decimal.Decimal

// PlanChangeResult represents the result of a plan change operation.
type PlanChangeResult struct {
	Action        string     `json:"action"`
	ChargeDue     Decimal    `json:"charge_due"`
	CheckoutLink  *string    `json:"checkout_link"`
	CreditApplied Decimal    `json:"credit_applied"`
	EffectiveDate *time.Time `json:"effective_date"`
}

// fromInternal populates the PlanChangeResult from the generated admin type.
func (p *PlanChangeResult) fromInternal(resp admin.PlanChangeResultResponse) {
	p.Action = resp.Action
	p.ChargeDue = resp.ChargeDue
	p.CheckoutLink = resp.CheckoutLink
	p.CreditApplied = resp.CreditApplied
	p.EffectiveDate = resp.EffectiveDate
}

// CancelSubscriptionRequest represents a request to cancel a subscription.
// This is an alias of the generated type for convenience.
type CancelSubscriptionRequest = admin.PostApiBillingUsersUserIdSubscriptionsCancelJSONRequestBody

// ChangePlanRequest represents a request to change a subscription plan.
// This is an alias of the generated type for convenience.
type ChangePlanRequest = admin.PostApiBillingUsersUserIdSubscriptionsChangePlanJSONRequestBody

// BillingService provides methods for managing billing operations.
type BillingService struct {
	client admin.ClientWithResponsesInterface
}

// ListCredits lists all credits with optional filtering.
func (b *BillingService) ListCredits(ctx context.Context, params *GetApiBillingCreditsParams) ([]*CreditItem, int, error) {
	var adminParams *admin.GetApiBillingCreditsParams
	if params != nil {
		adminParams = (*admin.GetApiBillingCreditsParams)(params)
	}

	resp, err := b.client.GetApiBillingCreditsWithResponse(ctx, adminParams)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list credits: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListCredits, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list credits response did not contain data")
	}

	credits := lo.Map(resp.JSON200.Data, func(credit admin.CreditItem, _ int) *CreditItem {
		return &CreditItem{CreditItem: credit}
	})

	return credits, resp.JSON200.Total, nil
}

// CreateCredit creates a new credit entry.
func (b *BillingService) CreateCredit(ctx context.Context, req *CreditCreateRequest) (*Credit, error) {
	resp, err := b.client.PostApiBillingCreditsWithResponse(ctx, *req)
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
	resp, err := b.client.PostApiBillingCreditsPurgeWithResponse(ctx, *req)
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
func (b *BillingService) GetUserDeletedCredits(ctx context.Context, userID string, params *GetApiBillingUsersUserIdDeletedCreditsParams) ([]*CreditItem, int, error) {
	var adminParams *admin.GetApiBillingUsersUserIdDeletedCreditsParams
	if params != nil {
		adminParams = (*admin.GetApiBillingUsersUserIdDeletedCreditsParams)(params)
	}

	resp, err := b.client.GetApiBillingUsersUserIdDeletedCreditsWithResponse(ctx, userID, adminParams)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user deleted credits: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingGetUserDeletedCredits, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("get user deleted credits response did not contain data")
	}

	credits := lo.Map(resp.JSON200.Data, func(credit admin.CreditItem, _ int) *CreditItem {
		return &CreditItem{CreditItem: credit}
	})

	return credits, resp.JSON200.Total, nil
}

// ListPriceLines lists all price lines.
func (b *BillingService) ListPriceLines(ctx context.Context) ([]*PriceLine, int, error) {
	resp, err := b.client.GetApiBillingPriceLinesWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list price lines: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListPriceLines, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list price lines response did not contain data")
	}

	lines := lo.Map(resp.JSON200.Data, func(line admin.PriceLineResponse, _ int) *PriceLine {
		return &PriceLine{PriceLineResponse: line}
	})

	return lines, resp.JSON200.Total, nil
}

// CreatePriceLine creates a new price line.
func (b *BillingService) CreatePriceLine(ctx context.Context, req *PriceLineCreateRequest) (*PriceLine, error) {
	resp, err := b.client.PostApiBillingPriceLinesWithResponse(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to create price line: %w", err)
	}

	data, err := validateBillingJSON201(resp.StatusCode(), resp.JSON201, "create price line response did not contain data", OpBillingCreatePriceLine)
	if err != nil {
		return nil, err
	}

	return &PriceLine{PriceLineResponse: *data}, nil
}

// GetPriceLine retrieves a price line by ID with its associated plans.
func (b *BillingService) GetPriceLine(ctx context.Context, priceLineID string) (*PriceLineDetailResponse, error) {
	resp, err := b.client.GetApiBillingPriceLinesIdWithResponse(ctx, priceLineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get price line: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingGetPriceLine)
	if err != nil {
		return nil, err
	}

	// Convert internal plans to public PricingPlanItem type
	var plans []*PricingPlanItem
	if data.Plans != nil {
		plans = lo.Map(*data.Plans, func(plan admin.PricingPlanItem, _ int) *PricingPlanItem {
			return &PricingPlanItem{PricingPlanItem: plan}
		})
	}

	return &PriceLineDetailResponse{
		CreatedAt:   data.CreatedAt,
		Description: data.Description,
		Id:          data.Id,
		IsActive:    data.IsActive,
		IsDefault:   data.IsDefault,
		Name:        data.Name,
		Plans:       plans,
		UpdatedAt:   data.UpdatedAt,
	}, nil
}

// SyncPricingPlan triggers immediate synchronization of a pricing plan with payment gateway.
func (b *BillingService) SyncPricingPlan(ctx context.Context, planID string) error {
	resp, err := b.client.PostApiBillingPlansIdSyncWithResponse(ctx, planID)
	if err != nil {
		return fmt.Errorf("failed to sync pricing plan: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingSyncPricingPlan, []int{stdhttp.StatusOK})
}

// SyncAllPricingPlans triggers synchronization of all pricing plans with payment gateways.
func (b *BillingService) SyncAllPricingPlans(ctx context.Context) error {
	resp, err := b.client.PostApiBillingPricingPlansSyncAllWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync all pricing plans: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingSyncAllPricingPlans, []int{stdhttp.StatusOK})
}

// UpdatePriceLine updates an existing price line.
func (b *BillingService) UpdatePriceLine(ctx context.Context, priceLineID string, req *PriceLineUpdateRequest) (*PriceLine, error) {
	resp, err := b.client.PutApiBillingPriceLinesIdWithResponse(ctx, priceLineID, *req)
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
func (b *BillingService) ListPricingPlans(ctx context.Context) ([]*PricingPlanItem, int, error) {
	resp, err := b.client.GetApiBillingPricingPlansWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pricing plans: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListPricingPlans, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list pricing plans response did not contain data")
	}

	plans := lo.Map(resp.JSON200.Data, func(plan admin.PricingPlanItem, _ int) *PricingPlanItem {
		return &PricingPlanItem{PricingPlanItem: plan}
	})

	return plans, resp.JSON200.Total, nil
}

// CreatePricingPlan creates a new pricing plan.
func (b *BillingService) CreatePricingPlan(ctx context.Context, req *PricingPlanCreateRequest) (*PricingPlan, error) {
	resp, err := b.client.PostApiBillingPricingPlansWithResponse(ctx, req.toInternal())
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
	resp, err := b.client.PutApiBillingPricingPlansIdWithResponse(ctx, planID, req.toInternal())
	if err != nil {
		return nil, fmt.Errorf("failed to update pricing plan: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingUpdatePricingPlan)
	if err != nil {
		return nil, err
	}

	return &PricingPlan{PricingPlanResponse: *data}, nil
}

// GetPricingPlan retrieves a pricing plan by ID with its pricing periods.
func (b *BillingService) GetPricingPlan(ctx context.Context, planID string) (*PricingPlan, error) {
	resp, err := b.client.GetApiBillingPricingPlansIdWithResponse(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing plan: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingGetPricingPlan)
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
func (b *BillingService) ListPricingPlanPeriods(ctx context.Context) ([]*PricingPlanPeriod, int, error) {
	resp, err := b.client.GetApiBillingPricingPlanPeriodsWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pricing plan periods: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListPricingPlanPeriods, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list pricing plan periods response did not contain data")
	}

	periods := lo.Map(resp.JSON200.Data, func(period admin.PricingPlanPeriodDTO, _ int) *PricingPlanPeriod {
		return &PricingPlanPeriod{PricingPlanPeriodDTO: period}
	})

	return periods, resp.JSON200.Total, nil
}

// CreatePricingPlanPeriod creates a new pricing plan period.
func (b *BillingService) CreatePricingPlanPeriod(ctx context.Context, req *PricingPlanPeriodCreateRequest) (*PricingPlanPeriod, error) {
	resp, err := b.client.PostApiBillingPricingPlanPeriodsWithResponse(ctx, *req)
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
	resp, err := b.client.PutApiBillingPricingPlanPeriodsIdWithResponse(ctx, periodID, *req)
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

// ListSubscribers lists all subscribers across all gateways.
func (b *BillingService) ListSubscribers(ctx context.Context) ([]*Subscriber, int, error) {
	resp, err := b.client.GetApiBillingSubscribersWithResponse(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subscribers: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListSubscribers, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list subscribers response did not contain data")
	}

	subscribers := lo.Map(resp.JSON200.Data, func(sub admin.SubscriberItem, _ int) *Subscriber {
		s := &Subscriber{}
		s.fromInternal(sub)
		return s
	})

	return subscribers, resp.JSON200.Total, nil
}

// GetSubscriber retrieves a specific subscriber by ID.
func (b *BillingService) GetSubscriber(ctx context.Context, subscriberID string) (*Subscriber, error) {
	resp, err := b.client.GetApiBillingSubscribersIdWithResponse(ctx, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingGetSubscriber)
	if err != nil {
		return nil, err
	}

	s := &Subscriber{}
	s.fromInternal(admin.SubscriberItem{
		BillingPeriodEnd:    data.BillingPeriodEnd,
		BillingPeriodStart:  data.BillingPeriodStart,
		CancelledAt:         data.CancelledAt,
		CreatedAt:           data.CreatedAt,
		ExternalId:          data.ExternalId,
		GatewayType:         data.GatewayType,
		Id:                  data.Id,
		IsActive:            data.IsActive,
		PaymentStatus:       data.PaymentStatus,
		PreviousPlanId:      data.PreviousPlanId,
		PricingPlanPeriodId: data.PricingPlanPeriodId,
		SubscriptionId:      data.SubscriptionId,
		UpdatedAt:           data.UpdatedAt,
		UserId:              data.UserId,
		WillCancelAt:        data.WillCancelAt,
	})
	return s, nil
}

// ListGatewaySubscribers lists subscribers for a specific gateway.
func (b *BillingService) ListGatewaySubscribers(ctx context.Context, gatewayID string) ([]*Subscriber, int, error) {
	resp, err := b.client.GetApiBillingGatewaysGatewayIdSubscribersWithResponse(ctx, gatewayID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list gateway subscribers: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingListGatewaySubscribers, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("list gateway subscribers response did not contain data")
	}

	subscribers := lo.Map(resp.JSON200.Data, func(sub admin.SubscriberItem, _ int) *Subscriber {
		s := &Subscriber{}
		s.fromInternal(sub)
		return s
	})

	return subscribers, resp.JSON200.Total, nil
}

// GetUserSubscribers retrieves subscribers for a specific user.
func (b *BillingService) GetUserSubscribers(ctx context.Context, userID string) ([]*Subscriber, int, error) {
	resp, err := b.client.GetApiBillingUsersUserIdSubscribersWithResponse(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user subscribers: %w", err)
	}

	if err := handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingGetUserSubscribers, []int{stdhttp.StatusOK}); err != nil {
		return nil, 0, err
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("get user subscribers response did not contain data")
	}

	subscribers := lo.Map(resp.JSON200.Data, func(sub admin.SubscriberItem, _ int) *Subscriber {
		s := &Subscriber{}
		s.fromInternal(sub)
		return s
	})

	return subscribers, resp.JSON200.Total, nil
}

// CancelUserSubscription cancels a user's subscription.
func (b *BillingService) CancelUserSubscription(ctx context.Context, userID string, req *CancelSubscriptionRequest) (*ManagementResult, error) {
	resp, err := b.client.PostApiBillingUsersUserIdSubscriptionsCancelWithResponse(ctx, userID, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel subscription: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingCancelUserSubscription)
	if err != nil {
		return nil, err
	}

	result := &ManagementResult{}
	result.fromInternal(*data)
	return result, nil
}

// AbortUserSubscriptionCancellation aborts a scheduled subscription cancellation,
// restoring the subscription to active status.
func (b *BillingService) AbortUserSubscriptionCancellation(ctx context.Context, userID string) (*ManagementResult, error) {
	resp, err := b.client.PostApiBillingUsersUserIdSubscriptionsCancelAbortWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to abort subscription cancellation: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingAbortUserSubscriptionCancellation)
	if err != nil {
		return nil, err
	}

	result := &ManagementResult{}
	result.fromInternal(*data)
	return result, nil
}

// ChangeUserPlan changes a user's subscription plan.
func (b *BillingService) ChangeUserPlan(ctx context.Context, userID string, req *ChangePlanRequest) (*PlanChangeResult, error) {
	resp, err := b.client.PostApiBillingUsersUserIdSubscriptionsChangePlanWithResponse(ctx, userID, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to change user plan: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingChangeUserPlan)
	if err != nil {
		return nil, err
	}

	result := &PlanChangeResult{}
	result.fromInternal(*data)
	return result, nil
}

// PauseUserSubscription pauses a user's subscription.
func (b *BillingService) PauseUserSubscription(ctx context.Context, userID string) (*ManagementResult, error) {
	resp, err := b.client.PostApiBillingUsersUserIdSubscriptionsPauseWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to pause subscription: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingPauseUserSubscription)
	if err != nil {
		return nil, err
	}

	result := &ManagementResult{}
	result.fromInternal(*data)
	return result, nil
}

// ResumeUserSubscription resumes a user's paused subscription.
func (b *BillingService) ResumeUserSubscription(ctx context.Context, userID string) (*ManagementResult, error) {
	resp, err := b.client.PostApiBillingUsersUserIdSubscriptionsResumeWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resume subscription: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingResumeUserSubscription)
	if err != nil {
		return nil, err
	}

	result := &ManagementResult{}
	result.fromInternal(*data)
	return result, nil
}

// AddPlanToPriceLine adds a pricing plan to a price line.
func (b *BillingService) AddPlanToPriceLine(ctx context.Context, priceLineID string, req *AddPlanToPriceLineRequest) (*PriceLineDetailResponse, error) {
	resp, err := b.client.PostApiBillingPriceLinesIdPlanWithResponse(ctx, priceLineID, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to add plan to price line: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingAddPlanToPriceLine)
	if err != nil {
		return nil, err
	}

	// Convert internal plans to public PricingPlanItem type
	var plans []*PricingPlanItem
	if data.Plans != nil {
		plans = lo.Map(*data.Plans, func(plan admin.PricingPlanItem, _ int) *PricingPlanItem {
			return &PricingPlanItem{PricingPlanItem: plan}
		})
	}

	return &PriceLineDetailResponse{
		CreatedAt:   data.CreatedAt,
		Description: data.Description,
		Id:          data.Id,
		IsActive:    data.IsActive,
		IsDefault:   data.IsDefault,
		Name:        data.Name,
		Plans:       plans,
		UpdatedAt:   data.UpdatedAt,
	}, nil
}

// DeletePlanFromPriceLine removes a pricing plan from a price line.
func (b *BillingService) DeletePlanFromPriceLine(ctx context.Context, priceLineID, planID string) error {
	resp, err := b.client.DeleteApiBillingPriceLinesIdPlansPlanIdWithResponse(ctx, priceLineID, planID)
	if err != nil {
		return fmt.Errorf("failed to delete plan from price line: %w", err)
	}

	return handleBillingResponse(resp.StatusCode(), resp.Body, OpBillingDeletePlanFromPriceLine, []int{stdhttp.StatusNoContent})
}

// UpdatePlanPosition updates the position of a plan in a price line.
func (b *BillingService) UpdatePlanPosition(ctx context.Context, priceLineID, planID string, req *UpdatePlanPositionRequest) (*PriceLineDetailResponse, error) {
	resp, err := b.client.PutApiBillingPriceLinesIdPlansPlanIdWithResponse(ctx, priceLineID, planID, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan position: %w", err)
	}

	data, err := validateBillingJSON200(resp.StatusCode(), resp.JSON200, OpBillingUpdatePlanPosition)
	if err != nil {
		return nil, err
	}

	// Convert internal plans to public PricingPlanItem type
	var plans []*PricingPlanItem
	if data.Plans != nil {
		plans = lo.Map(*data.Plans, func(plan admin.PricingPlanItem, _ int) *PricingPlanItem {
			return &PricingPlanItem{PricingPlanItem: plan}
		})
	}

	return &PriceLineDetailResponse{
		CreatedAt:   data.CreatedAt,
		Description: data.Description,
		Id:          data.Id,
		IsActive:    data.IsActive,
		IsDefault:   data.IsDefault,
		Name:        data.Name,
		Plans:       plans,
		UpdatedAt:   data.UpdatedAt,
	}, nil
}
