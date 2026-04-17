package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lumeweb.com/portal-sdk/internal/admin"
)



func TestNewClient(t *testing.T) {
	t.Run("default client", func(t *testing.T) {
		client := NewClient()
		require.NotNil(t, client)
		require.NotNil(t, client.Quota())
	})

	t.Run("with endpoint", func(t *testing.T) {
		client := NewClient(WithEndpoint("https://admin.example.com"))
		require.NotNil(t, client)
	})

	t.Run("with host override", func(t *testing.T) {
		client := NewClient(WithHostOverride("admin.pinner.xyz", "http://127.0.0.1:8080"))
		require.NotNil(t, client)
	})

	t.Run("with JWT", func(t *testing.T) {
		client := NewClient(WithJWT("test-token"))
		require.NotNil(t, client)
	})

	t.Run("with API key", func(t *testing.T) {
		client := NewClient(WithAPIKey("test-api-key"))
		require.NotNil(t, client)
	})
}

func TestQuotaService_ListPlans(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list plans",
			statusCode: http.StatusOK,
			response: admin.PlanListResponse{
				Data: []admin.QuotaPlanResponse{
					{
						Name:               "Basic",
						UploadLimitBytes:   100,
						DownloadLimitBytes: 1000,
						StorageLimitBytes:  10000,
						Description:        "Basic plan",
					},
					{
						Name:               "Premium",
						UploadLimitBytes:   500,
						DownloadLimitBytes: 5000,
						StorageLimitBytes:  50000,
						Description:        "Premium plan",
					},
				},
				Total: 2,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/plans" {
					t.Errorf("expected /api/quota/plans path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			plans, total, err := client.Quota().ListPlans(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPlans() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, plans, 2)
				require.Equal(t, 2, total)
				require.Equal(t, "Basic", plans[0].Name)
				require.Equal(t, "Premium", plans[1].Name)
			}
		})
	}
}

func TestBillingService_CreatePriceLine(t *testing.T) {
	tests := []struct {
		name       string
		request    *PriceLineCreateRequest
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name: "successful create price line",
			request: &PriceLineCreateRequest{
				Name:        "Storage",
				Description: "Storage pricing",
				IsActive:    true,
				IsDefault:   false,
			},
			statusCode: http.StatusCreated,
			response: admin.PriceLineResponse{
				Id:          1,
				Name:        "Storage",
				Description: "Storage pricing",
				IsActive:    true,
				IsDefault:   false,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			request:    &PriceLineCreateRequest{},
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/price-lines" {
					t.Errorf("expected /api/billing/price-lines path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			line, err := client.Billing().CreatePriceLine(context.Background(), tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, line)
				require.Equal(t, "Storage", line.Name)
			}
		})
	}
}

func TestBillingService_UpdatePriceLine(t *testing.T) {
	tests := []struct {
		name        string
		priceLineID string
		request     *PriceLineUpdateRequest
		statusCode  int
		response    interface{}
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful update price line",
			priceLineID: "1",
			request: &PriceLineUpdateRequest{
				Name:        "Storage Updated",
				Description: "Storage pricing updated",
				IsActive:    true,
				IsDefault:   true,
			},
			statusCode: http.StatusOK,
			response: admin.PriceLineResponse{
				Id:          1,
				Name:        "Storage Updated",
				Description: "Storage pricing updated",
				IsActive:    true,
				IsDefault:   true,
			},
			wantErr: false,
		},
		{
			name:        "not found",
			priceLineID: "999",
			request:     &PriceLineUpdateRequest{},
			statusCode:  http.StatusNotFound,
			response:    admin.ErrorResponse{Error: "price line not found"},
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "price line not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/price-lines/"+tt.priceLineID {
					t.Errorf("expected /api/billing/price-lines/%s path, got %s", tt.priceLineID, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			line, err := client.Billing().UpdatePriceLine(context.Background(), tt.priceLineID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, line)
				require.Equal(t, "Storage Updated", line.Name)
			}
		})
	}
}

func TestBillingService_GetPriceLine(t *testing.T) {
	tests := []struct {
		name        string
		priceLineID string
		statusCode  int
		response    interface{}
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful get price line",
			priceLineID: "1",
			statusCode:  http.StatusOK,
			response: admin.PriceLineResponse{
				Id:          1,
				Name:        "Storage",
				Description: "Storage pricing",
				IsActive:    true,
				IsDefault:   false,
			},
			wantErr: false,
		},
		{
			name:        "not found",
			priceLineID: "999",
			statusCode:  http.StatusNotFound,
			response:     admin.ErrorResponse{Error: "price line not found"},
			wantErr:      true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "price line not found")
			},
		},
		{
			name:        "unauthorized",
			priceLineID: "1",
			statusCode:  http.StatusUnauthorized,
			response:     admin.ErrorResponse{Error: "unauthorized"},
			wantErr:      true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/price-lines/"+tt.priceLineID {
					t.Errorf("expected /api/billing/price-lines/%s path, got %s", tt.priceLineID, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			line, err := client.Billing().GetPriceLine(context.Background(), tt.priceLineID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, line)
				require.Equal(t, "Storage", line.Name)
			}
		})
	}
}

func TestBillingService_DeletePriceLine(t *testing.T) {
	tests := []struct {
		name        string
		priceLineID string
		statusCode  int
		response    interface{}
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful delete price line",
			priceLineID: "1",
			statusCode:  http.StatusNoContent,
			response:    nil,
			wantErr:     false,
		},
		{
			name:        "not found",
			priceLineID: "999",
			statusCode:  http.StatusNotFound,
			response:    admin.ErrorResponse{Error: "price line not found"},
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "price line not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/price-lines/"+tt.priceLineID {
					t.Errorf("expected /api/billing/price-lines/%s path, got %s", tt.priceLineID, r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					w.Header().Set("Content-Type", "application/json")
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Billing().DeletePriceLine(context.Background(), tt.priceLineID)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeletePriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestBillingService_ListPricingPlans(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list pricing plans",
			statusCode: http.StatusOK,
			response: admin.PricingPlansListResponse{
				Data: []admin.PricingPlanItem{
					{
						Id:          1,
						Name:        "Basic",
						Description: "Basic plan",
						Currency:    "USD",
						IsActive:    true,
					},
					{
						Id:          2,
						Name:        "Premium",
						Description: "Premium plan",
						Currency:    "USD",
						IsActive:    true,
					},
				},
				Total: 2,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/pricing-plans" {
					t.Errorf("expected /api/billing/pricing-plans path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			plans, total, err := client.Billing().ListPricingPlans(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPricingPlans() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, plans, 2)
				require.Equal(t, "Basic", plans[0].Name)
				require.Equal(t, "Premium", plans[1].Name)
				require.Equal(t, 2, total)
			}
		})
	}
}

func TestBillingService_CreatePricingPlan(t *testing.T) {
	tests := []struct {
		name       string
		request    *PricingPlanCreateRequest
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name: "successful create pricing plan",
			request: &PricingPlanCreateRequest{
				Name:           "Basic",
				Description:    "Basic plan",
				Currency:       "USD",
				IsActive:       true,
				IsPublic:       true,
				PricingPeriods: []PricingPlanPeriod{},
			},
			statusCode: http.StatusCreated,
			response: admin.PricingPlanResponse{
				Id:          1,
				Name:        "Basic",
				Description: "Basic plan",
				Currency:    "USD",
				IsActive:    true,
				IsPublic:    true,
			},
			wantErr: false,
		},
		{
			name: "successful create pricing plan with position and price line",
			request: &PricingPlanCreateRequest{
				Name:           "Premium",
				Description:    "Premium plan",
				Currency:       "USD",
				IsActive:       true,
				IsPublic:       true,
				Position:       new(2),
				PricelineId:    new(1),
				PricingPeriods: []PricingPlanPeriod{},
			},
			statusCode: http.StatusCreated,
			response: admin.PricingPlanResponse{
				Id:          2,
				Name:        "Premium",
				Description: "Premium plan",
				Currency:    "USD",
				IsActive:    true,
				IsPublic:    true,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			request:    &PricingPlanCreateRequest{},
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/pricing-plans" {
					t.Errorf("expected /api/billing/pricing-plans path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			plan, err := client.Billing().CreatePricingPlan(context.Background(), tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePricingPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, plan)
				require.Equal(t, tt.request.Name, plan.Name)
			}
		})
	}
}

func TestBillingService_DeletePricingPlan(t *testing.T) {
	tests := []struct {
		name    string
		planID  string
		respErr bool
	}{
		{
			name:    "successful delete pricing plan",
			planID:  "1",
			respErr: false,
		},
		{
			name:    "not found",
			planID:  "999",
			respErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/pricing-plans/"+tt.planID {
					t.Errorf("expected /api/billing/pricing-plans/%s path, got %s", tt.planID, r.URL.Path)
				}

				if tt.respErr {
					w.WriteHeader(http.StatusNotFound)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(admin.ErrorResponse{Error: "pricing plan not found"})
				} else {
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Billing().DeletePricingPlan(context.Background(), tt.planID)

			if tt.respErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "pricing plan not found")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBillingService_ListPricingPlanPeriods(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list pricing plan periods",
			statusCode: http.StatusOK,
			response: admin.PricingPlanPeriodsListResponse{
				Data: []admin.PricingPlanPeriodDTO{
					{
						Id:            1,
						Cadence:       "monthly",
						PriceUsd:      9.99,
						PricingPlanId: 1,
						QuotaPlanId:   1,
					},
					{
						Id:            2,
						Cadence:       "yearly",
						PriceUsd:      99.99,
						PricingPlanId: 1,
						QuotaPlanId:   1,
					},
				},
				Total: 2,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/pricing-plan-periods" {
					t.Errorf("expected /api/billing/pricing-plan-periods path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			periods, total, err := client.Billing().ListPricingPlanPeriods(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPricingPlanPeriods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, periods, 2)
				require.Equal(t, "monthly", periods[0].Cadence)
				require.Equal(t, "yearly", periods[1].Cadence)
				require.Equal(t, 2, total)
			}
		})
	}
}


// === Billing Service Tests ===

func TestBillingService_ListPriceLines(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list price lines",
			statusCode: http.StatusOK,
			response: admin.PriceLinesListResponse{
				Data: []admin.PriceLineResponse{
					{
						Id:          1,
						Name:        "Storage",
						Description: "Storage pricing",
						IsActive:    true,
						IsDefault:   false,
					},
					{
						Id:          2,
						Name:        "Bandwidth",
						Description: "Bandwidth pricing",
						IsActive:    true,
						IsDefault:   true,
					},
				},
				Total: 2,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/price-lines" {
					t.Errorf("expected /api/billing/price-lines path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			lines, total, err := client.Billing().ListPriceLines(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPriceLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, lines, 2)
				require.Equal(t, "Storage", lines[0].Name)
				require.Equal(t, "Bandwidth", lines[1].Name)
				require.Equal(t, 2, total)
			}
		})
	}
}

func TestQuotaService_ListUserConfigs(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list user configs",
			statusCode: http.StatusOK,
			response: admin.UserQuotaConfigListResponse{
				Data: []admin.UserQuotaConfigResponse{
					{
						Id:                 1,
						UserId:             100,
						UploadLimitBytes:   new(100),
						DownloadLimitBytes: new(1000),
						StorageLimitBytes:       new(10000),
						EnforcementPolicy:  "strict",
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						Id:                 2,
						UserId:             200,
						UploadLimitBytes:   new(500),
						DownloadLimitBytes: new(5000),
						StorageLimitBytes:       new(50000),
						EnforcementPolicy:  "lenient",
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
				Total: 2,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: "invalid request"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid request")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/user-configs" {
					t.Errorf("expected /api/quota/user-configs path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			configs, total, err := client.Quota().ListUserConfigs(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListUserConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, 2, total)
				require.Len(t, configs, 2)
				require.Equal(t, int(100), configs[0].UserId)
				require.Equal(t, "strict", configs[0].EnforcementPolicy)
				require.Equal(t, int(200), configs[1].UserId)
				require.Equal(t, "lenient", configs[1].EnforcementPolicy)
			}
		})
	}
}

func TestQuotaService_UpdateUserConfig(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		userID     int
		update     *admin.UserQuotaConfigUpdateRequest
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:   "successful update user config",
			userID: 100,
			update: &admin.UserQuotaConfigUpdateRequest{
				UploadLimitBytes:   new(1000),
				DownloadLimitBytes: new(10000),
				StorageLimitBytes:  new(100000),
				EnforcementPolicy:  new("strict"),
			},
			statusCode: http.StatusOK,
			response: admin.UserQuotaConfigResponse{
				Id:                1,
				UserId:            100,
				UploadLimitBytes:  new(1000),
				DownloadLimitBytes: new(10000),
				StorageLimitBytes:      new(100000),
				EnforcementPolicy: "strict",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
			wantErr: false,
		},
		{
			name:   "unauthorized",
			userID: 100,
			update: &admin.UserQuotaConfigUpdateRequest{},
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:   "user not found",
			userID: 999,
			update: &admin.UserQuotaConfigUpdateRequest{},
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "user not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name:   "bad request",
			userID: 100,
			update: &admin.UserQuotaConfigUpdateRequest{},
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: "invalid configuration"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "configuration")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/user-configs/100" && r.URL.Path != "/api/quota/user-configs/999" {
					t.Errorf("expected /api/quota/user-configs/{userID} path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			config, err := client.Quota().UpdateUserConfig(context.Background(), tt.userID, tt.update)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateUserConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, int(100), config.UserId)
				require.Equal(t, int(1000), *config.UploadLimitBytes)
				require.Equal(t, int(10000), *config.DownloadLimitBytes)
				require.Equal(t, int(100000), *config.StorageLimitBytes)
				require.Equal(t, "strict", config.EnforcementPolicy)
			}
		})
	}
}

func TestQuotaService_ResetUserPlan(t *testing.T) {
	tests := []struct {
		name       string
		userID     int
		statusCode int
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful reset user plan",
			userID:     100,
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			userID:     100,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "user not found",
			userID:     999,
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/user-configs/100/plan" && r.URL.Path != "/api/quota/user-configs/999/plan" {
					t.Errorf("expected /api/quota/user-configs/{userID}/plan path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				
				// Add response body for error cases
				if tt.statusCode == http.StatusNotFound {
					json.NewEncoder(w).Encode(admin.ErrorResponse{Error: "user not found"})
				} else if tt.statusCode == http.StatusUnauthorized {
					json.NewEncoder(w).Encode(admin.ErrorResponse{Error: "unauthorized"})
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Quota().ResetUserPlan(context.Background(), tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResetUserPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}





func TestQuotaService_CreatePlan(t *testing.T) {
	tests := []struct {
		name       string
		plan       *QuotaPlan
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name: "successful plan creation",
			plan: &QuotaPlan{
				QuotaPlanResponse: admin.QuotaPlanResponse{
					Name:               "Enterprise",
					UploadLimitBytes:   1000,
					DownloadLimitBytes: 10000,
					StorageLimitBytes:       100000,
					Description:        "Enterprise plan",
				},
			},
			statusCode: http.StatusCreated,
			response: admin.QuotaPlanResponse{
				Name:               "Enterprise",
				UploadLimitBytes:   1000,
				DownloadLimitBytes: 10000,
				StorageLimitBytes:       100000,
				Description:        "Enterprise plan",
			},
			wantErr: false,
		},
		{
			name: "unauthorized",
			plan: &QuotaPlan{
				QuotaPlanResponse: admin.QuotaPlanResponse{
					Name:               "Test",
					UploadLimitBytes:   100,
					DownloadLimitBytes: 1000,
					StorageLimitBytes:       10000,
				},
			},
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/plans" {
					t.Errorf("expected /api/quota/plans path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			plan, err := client.Quota().CreatePlan(context.Background(), tt.plan)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, "Enterprise", plan.Name)
				require.Equal(t, int(1000), plan.UploadLimitBytes)
			}
		})
	}
}

func TestQuotaService_UpdatePlan(t *testing.T) {
	tests := []struct {
		name       string
		planID     string
		plan       *QuotaPlan
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:   "successful plan update",
			planID: "plan-1",
			plan: &QuotaPlan{
				QuotaPlanResponse: admin.QuotaPlanResponse{
					Name:               "Enterprise Updated",
					UploadLimitBytes:   1500,
					DownloadLimitBytes: 15000,
					StorageLimitBytes:       150000,
					Description:        "Updated enterprise plan",
				},
			},
			statusCode: http.StatusOK,
			response: admin.QuotaPlanResponse{
				Name:               "Enterprise Updated",
				UploadLimitBytes:   1500,
				DownloadLimitBytes: 15000,
				StorageLimitBytes:       150000,
				Description:        "Updated enterprise plan",
			},
			wantErr: false,
		},
		{
			name:   "plan not found",
			planID: "nonexistent",
			plan: &QuotaPlan{
				QuotaPlanResponse: admin.QuotaPlanResponse{
					Name: "Updated Name",
				},
			},
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "plan not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "plan not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				expectedPath := "/api/quota/plans/" + tt.planID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			plan, err := client.Quota().UpdatePlan(context.Background(), tt.planID, tt.plan)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, "Enterprise Updated", plan.Name)
			}
		})
	}
}

func TestQuotaService_DeletePlan(t *testing.T) {
	tests := []struct {
		name       string
		planID     string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful plan deletion",
			planID:     "plan-1",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "plan not found",
			planID:     "nonexistent",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "plan not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "plan not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				expectedPath := "/api/quota/plans/" + tt.planID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					w.Header().Set("Content-Type", "application/json")
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Quota().DeletePlan(context.Background(), tt.planID)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeletePlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestQuotaService_GetPlan(t *testing.T) {
	tests := []struct {
		name       string
		planID     string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful get plan",
			planID:     "plan-1",
			statusCode: http.StatusOK,
			response: admin.QuotaPlanResponse{
				Name:               "Basic",
				UploadLimitBytes:   100,
				DownloadLimitBytes: 1000,
				StorageLimitBytes:       10000,
				Description:        "Basic plan",
			},
			wantErr: false,
		},
		{
			name:       "plan not found",
			planID:     "nonexistent",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "plan not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "plan not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := "/api/quota/plans/" + tt.planID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			plan, err := client.Quota().GetPlan(context.Background(), tt.planID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, "Basic", plan.Name)
			}
		})
	}
}

func TestQuotaService_ListAllowances(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list allowances",
			statusCode: http.StatusOK,
			response: admin.AllowanceListResponse{
				Data: []admin.AllowanceGrantResponse{
					{
						Id:         1,
						UserId:     123,
						Source:     "manual",
						Type:       "upload",
						Bytes:      20000,
						ExpiryDate: time.Now().Add(7 * 24 * time.Hour),
					},
				},
				Total: 1,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/allowances" {
					t.Errorf("expected /api/quota/allowances path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			allowances, _, err := client.Quota().ListAllowances(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListAllowances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, allowances, 1)
				require.Equal(t, int(123), allowances[0].UserId)
			}
		})
	}
}

func TestQuotaService_CreateAllowance(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	tests := []struct {
		name       string
		userID     int
		source     string
		allowance  string
		upload     int
		download   int
		storage    int
		expiryDate time.Time
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful allowance creation",
			userID:     123,
			source:     "manual",
			allowance:  "upload",
			upload:     100,
			download:   1000,
			storage:    10000,
			expiryDate: expiry,
			statusCode: http.StatusCreated,
			response: admin.AllowanceGrantResponse{
				Id:         1,
				UserId:     123,
				Source:     "manual",
				Type:       "upload",
				Bytes:      10000,
				ExpiryDate: expiry,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			userID:     123,
			source:     "manual",
			allowance:  "upload",
			upload:     100,
			download:   1000,
			storage:    10000,
			expiryDate: expiry,
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/allowances" {
					t.Errorf("expected /api/quota/allowances path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			allowance, err := client.Quota().CreateAllowance(context.Background(), tt.userID, tt.source, tt.allowance, tt.upload, tt.download, tt.storage, tt.expiryDate)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAllowance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, int(123), allowance.UserId)
			}
		})
	}
}

func TestQuotaService_UpdateAllowance(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	tests := []struct {
		name       string
		grantID    string
		userID     int
		source     string
		allowance  string
		upload     int
		download   int
		storage    int
		expiryDate time.Time
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful allowance update",
			grantID:    "allow-1",
			userID:     123,
			source:     "manual",
			allowance:  "upload",
			upload:     200,
			download:   2000,
			storage:    20000,
			expiryDate: expiry,
			statusCode: http.StatusOK,
			response: admin.AllowanceGrantResponse{
				Id:         1,
				UserId:     123,
				Source:     "manual",
				Type:       "upload",
				Bytes:      20000,
				ExpiryDate: expiry,
			},
			wantErr: false,
		},
		{
			name:       "allowance not found",
			grantID:    "nonexistent",
			userID:     123,
			source:     "manual",
			allowance:  "upload",
			upload:     100,
			download:   1000,
			storage:    10000,
			expiryDate: expiry,
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "allowance not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "allowance not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				expectedPath := "/api/quota/allowances/" + tt.grantID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			allowance, err := client.Quota().UpdateAllowance(context.Background(), tt.grantID, tt.userID, tt.source, tt.allowance, tt.upload, tt.download, tt.storage, tt.expiryDate)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAllowance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, int(123), allowance.UserId)
			}
		})
	}
}

func TestQuotaService_DeleteAllowance(t *testing.T) {
	tests := []struct {
		name        string
		allowanceID string
		statusCode  int
		response    interface{}
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful allowance deletion",
			allowanceID: "allow-1",
			statusCode:  http.StatusNoContent,
			wantErr:     false,
		},
		{
			name:        "allowance not found",
			allowanceID: "nonexistent",
			statusCode:  http.StatusNotFound,
			response:    admin.ErrorResponse{Error: "allowance not found"},
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "allowance not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				expectedPath := "/api/quota/allowances/" + tt.allowanceID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					w.Header().Set("Content-Type", "application/json")
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Quota().DeleteAllowance(context.Background(), tt.allowanceID)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteAllowance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestQuotaService_GetStats(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful get stats",
			statusCode: http.StatusOK,
			response: admin.SystemStatsResponse{
				TotalUsers:  100,
				ActiveUsers: 75,
				TotalPlans:  3,
				TotalUsageBytes: 100000000,
				CurrentUsage: admin.Usage{
					UploadBytes:   1000000,
					DownloadBytes: 10000000,
					StorageBytes:  100000000,
				},
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/system/stats" {
					t.Errorf("expected /api/quota/system/stats path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			stats, err := client.Quota().GetStats(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, int(100), stats.TotalUsers)
				require.Equal(t, int(75), stats.ActiveUsers)
			}
		})
	}
}
// New Subscriber API tests

func TestBillingService_ListSubscribers(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list subscribers",
			statusCode: http.StatusOK,
			response: admin.SubscribersListResponse{
				Data: []admin.SubscriberItem{
					{
						Id:                  1,
						UserId:              100,
						SubscriptionId:      "sub_123",
						ExternalId:          "ext_456",
						GatewayType:         "stripe",
						IsActive:            true,
						CreatedAt:           time.Now(),
						UpdatedAt:           time.Now(),
						PricingPlanPeriodId: lo.ToPtr[int](10),
					},
					{
						Id:                  2,
						UserId:              101,
						SubscriptionId:      "sub_124",
						ExternalId:          "ext_457",
						GatewayType:         "paypal",
						IsActive:            false,
						CancelledAt:         lo.ToPtr(time.Now()),
						CreatedAt:           time.Now(),
						UpdatedAt:           time.Now(),
					},
				},
				Total: 2,
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/billing/subscribers" {
					t.Errorf("expected /api/billing/subscribers path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			subs, total, err := client.Billing().ListSubscribers(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListSubscribers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, subs, 2)
				require.Equal(t, int(1), subs[0].Id)
				require.Equal(t, int(2), subs[1].Id)
				require.Equal(t, "stripe", subs[0].GatewayType)
				require.Equal(t, "paypal", subs[1].GatewayType)
				require.Equal(t, 2, total)
			}
		})
	}
}

func TestBillingService_GetSubscriber(t *testing.T) {
	tests := []struct {
		name        string
		subscriberID string
		statusCode  int
		response    interface{}
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful get subscriber",
			subscriberID: "1",
			statusCode:  http.StatusOK,
			response: admin.SubscriberResponse{
				Id:                  1,
				UserId:              100,
				SubscriptionId:      "sub_123",
				ExternalId:          "ext_456",
				GatewayType:         "stripe",
				IsActive:            true,
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
				PricingPlanPeriodId: lo.ToPtr[int](10),
				BillingPeriodStart:  lo.ToPtr(time.Now()),
				BillingPeriodEnd:    lo.ToPtr(time.Now().AddDate(0, 1, 0)),
			},
			wantErr: false,
		},
		{
			name:        "subscriber not found",
			subscriberID: "999",
			statusCode:  http.StatusNotFound,
			response:    admin.ErrorResponse{Error: "not found"},
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/subscribers/%s", tt.subscriberID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			sub, err := client.Billing().GetSubscriber(context.Background(), tt.subscriberID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSubscriber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, sub)
				require.Equal(t, int(1), sub.Id)
				require.Equal(t, "stripe", sub.GatewayType)
				require.Equal(t, true, sub.IsActive)
			}
		})
	}
}

func TestBillingService_ListGatewaySubscribers(t *testing.T) {
	tests := []struct {
		name       string
		gatewayID  string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list gateway subscribers",
			gatewayID:  "stripe",
			statusCode: http.StatusOK,
			response: admin.SubscribersListResponse{
				Data: []admin.SubscriberItem{
					{
						Id:             1,
						UserId:         100,
						SubscriptionId: "sub_123",
						ExternalId:     "ext_456",
						GatewayType:    "stripe",
						IsActive:       true,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					},
				},
				Total: 1,
			},
			wantErr: false,
		},
		{
			name:       "gateway not found",
			gatewayID:  "unknown",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "gateway not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/gateways/%s/subscribers", tt.gatewayID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			subs, total, err := client.Billing().ListGatewaySubscribers(context.Background(), tt.gatewayID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListGatewaySubscribers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, subs, 1)
				require.Equal(t, "stripe", subs[0].GatewayType)
				require.Equal(t, 1, total)
			}
		})
	}
}

func TestBillingService_GetUserSubscribers(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful get user subscribers",
			userID:     "100",
			statusCode: http.StatusOK,
			response: admin.SubscribersListResponse{
				Data: []admin.SubscriberItem{
					{
						Id:             1,
						UserId:         100,
						SubscriptionId: "sub_123",
						ExternalId:     "ext_456",
						GatewayType:    "stripe",
						IsActive:       true,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					},
				},
				Total: 1,
			},
			wantErr: false,
		},
		{
			name:       "user not found",
			userID:     "999",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "user not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/users/%s/subscribers", tt.userID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			subs, total, err := client.Billing().GetUserSubscribers(context.Background(), tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserSubscribers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, subs, 1)
				require.Equal(t, int(100), subs[0].UserId)
				require.Equal(t, 1, total)
			}
		})
	}
}

func TestBillingService_CancelUserSubscription(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		request    *CancelSubscriptionRequest
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:   "successful cancel subscription",
			userID: "100",
			request: &CancelSubscriptionRequest{
				Mode: lo.ToPtr("immediate"),
			},
			statusCode: http.StatusOK,
			response: admin.ManagementResultResponse{
				Action:               "cancel",
				RequiresConfirmation: false,
				ConfirmationMessage:  lo.ToPtr("Subscription cancelled immediately"),
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: "999",
			request: &CancelSubscriptionRequest{
				Mode: lo.ToPtr("end_of_period"),
			},
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "user not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/users/%s/subscriptions/cancel", tt.userID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			result, err := client.Billing().CancelUserSubscription(context.Background(), tt.userID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CancelUserSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, "cancel", result.Action)
				require.False(t, result.RequiresConfirmation)
			}
		})
	}
}

func TestBillingService_AbortUserSubscriptionCancellation(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful abort subscription cancellation",
			userID:     "100",
			statusCode: http.StatusOK,
			response: admin.ManagementResultResponse{
				Action:               "abort",
				RequiresConfirmation: false,
				ConfirmationMessage:  lo.ToPtr("Subscription cancellation aborted successfully"),
				CanAbort:             false,
				Status:               "active",
			},
			wantErr: false,
		},
		{
			name:       "user not found",
			userID:     "999",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "no scheduled cancellation found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no scheduled cancellation found")
			},
		},
		{
			name:       "unauthorized",
			userID:     "100",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
		},
		{
			name:       "forbidden - insufficient permissions",
			userID:     "100",
			statusCode: http.StatusForbidden,
			response:   admin.ErrorResponse{Error: "insufficient permissions"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/users/%s/subscriptions/cancel/abort", tt.userID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			result, err := client.Billing().AbortUserSubscriptionCancellation(context.Background(), tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("AbortUserSubscriptionCancellation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, tt.response.(admin.ManagementResultResponse).Action, result.Action)
			}
		})
	}
}

func TestBillingService_ChangeUserPlan(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		request    *ChangePlanRequest
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:   "successful change user plan",
			userID: "100",
			request: &ChangePlanRequest{
				PeriodId: 20,
			},
			statusCode: http.StatusOK,
			response: admin.PlanChangeResultResponse{
				Action:    "change_plan",
				ChargeDue: admin.Decimal{},
			},
			wantErr: false,
		},
		{
			name:   "invalid request",
			userID: "100",
			request: &ChangePlanRequest{
				PeriodId: 0,
			},
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: "invalid plan change request"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/users/%s/subscriptions/change-plan", tt.userID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			result, err := client.Billing().ChangeUserPlan(context.Background(), tt.userID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("ChangeUserPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, "change_plan", result.Action)
			}
		})
	}
}

func TestBillingService_PauseUserSubscription(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful pause subscription",
			userID:     "100",
			statusCode: http.StatusOK,
			response: admin.ManagementResultResponse{
				Action:               "pause",
				RequiresConfirmation: false,
				ConfirmationMessage:  lo.ToPtr("Subscription paused successfully"),
				Status:               "paused",
			},
			wantErr: false,
		},
		{
			name:       "user not found",
			userID:     "999",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "user not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name:       "subscription cannot be paused",
			userID:     "100",
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: "subscription cannot be paused"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "cannot be paused")
			},
		},
		{
			name:       "unauthorized",
			userID:     "100",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/users/%s/subscriptions/pause", tt.userID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			result, err := client.Billing().PauseUserSubscription(context.Background(), tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("PauseUserSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, "pause", result.Action)
				require.Equal(t, "paused", result.Status)
			}
		})
	}
}

func TestBillingService_ResumeUserSubscription(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful resume subscription",
			userID:     "100",
			statusCode: http.StatusOK,
			response: admin.ManagementResultResponse{
				Action:               "resume",
				RequiresConfirmation: false,
				ConfirmationMessage:  lo.ToPtr("Subscription resumed successfully"),
				Status:               "active",
			},
			wantErr: false,
		},
		{
			name:       "user not found",
			userID:     "999",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: "user not found"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name:       "subscription cannot be resumed",
			userID:     "100",
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: "subscription cannot be resumed"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "cannot be resumed")
			},
		},
		{
			name:       "unauthorized",
			userID:     "100",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/billing/users/%s/subscriptions/resume", tt.userID)
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			result, err := client.Billing().ResumeUserSubscription(context.Background(), tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResumeUserSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, "resume", result.Action)
				require.Equal(t, "active", result.Status)
			}
		})
	}
}

// TestBillingService_AddPlanToPriceLine tests the AddPlanToPriceLine method
func TestBillingService_AddPlanToPriceLine(t *testing.T) {
	tests := []struct {
		name       string
		priceLineID string
		request    *AddPlanToPriceLineRequest
		response   admin.PriceLineDetailResponse
		statusCode int
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful add plan to price line",
			priceLineID: "1",
			request: &AddPlanToPriceLineRequest{
				PlanId:   10,
				Position: 0,
			},
			response: admin.PriceLineDetailResponse{
				Id:   1,
				Name: "Price Line 1",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			priceLineID: "1",
			request:    &AddPlanToPriceLineRequest{},
			response:   admin.PriceLineDetailResponse{},
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "price line not found",
			priceLineID: "999",
			request:    &AddPlanToPriceLineRequest{},
			response:   admin.PriceLineDetailResponse{},
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				expectedPath := "/api/billing/price-lines/" + tt.priceLineID + "/plan"
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.wantErr {
					json.NewEncoder(w).Encode(admin.ErrorResponse{Error: "test error"})
				} else {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			result, err := client.Billing().AddPlanToPriceLine(context.Background(), tt.priceLineID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddPlanToPriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errCheck != nil {
					tt.errCheck(t, err)
				}
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.response.Name, result.Name)
		})
	}
}

// TestBillingService_DeletePlanFromPriceLine tests the DeletePlanFromPriceLine method
func TestBillingService_DeletePlanFromPriceLine(t *testing.T) {
	tests := []struct {
		name        string
		priceLineID string
		planID      string
		statusCode  int
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful delete plan from price line",
			priceLineID: "1",
			planID:      "10",
			statusCode:  http.StatusNoContent,
			wantErr:     false,
		},
		{
			name:        "unauthorized",
			priceLineID: "1",
			planID:      "10",
			statusCode:  http.StatusUnauthorized,
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:        "price line not found",
			priceLineID: "999",
			planID:      "10",
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "price line not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}
				expectedPath := "/api/billing/price-lines/" + tt.priceLineID + "/plans/" + tt.planID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Billing().DeletePlanFromPriceLine(context.Background(), tt.priceLineID, tt.planID)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeletePlanFromPriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

// TestBillingService_UpdatePlanPosition tests the UpdatePlanPosition method
func TestBillingService_UpdatePlanPosition(t *testing.T) {
	tests := []struct {
		name        string
		priceLineID string
		planID      string
		request     *UpdatePlanPositionRequest
		statusCode  int
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful update plan position",
			priceLineID: "1",
			planID:      "10",
			request:     &UpdatePlanPositionRequest{Position: 5},
			statusCode:  http.StatusNoContent,
			wantErr:     false,
		},
		{
			name:        "unauthorized",
			priceLineID: "1",
			planID:      "10",
			request:     &UpdatePlanPositionRequest{},
			statusCode:  http.StatusUnauthorized,
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:        "price line not found",
			priceLineID: "999",
			planID:      "10",
			request:     &UpdatePlanPositionRequest{},
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "price line or plan not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				expectedPath := "/api/billing/price-lines/" + tt.priceLineID + "/plans/" + tt.planID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			err := client.Billing().UpdatePlanPosition(context.Background(), tt.priceLineID, tt.planID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePlanPosition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

// TestBillingService_GetPriceLineDetail tests the GetPriceLine method with detailed response including plans
func TestBillingService_GetPriceLineDetail(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		priceLineID string
		statusCode  int
		response    admin.PriceLineDetailResponse
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful get price line with plans",
			priceLineID: "1",
			statusCode:  http.StatusOK,
			response: admin.PriceLineDetailResponse{
				Id:          1,
				Name:        "Storage",
				Description: "Storage pricing with plans",
				IsActive:    true,
				IsDefault:   false,
				CreatedAt:   now,
				UpdatedAt:   now,
				Plans: &[]admin.PricingPlanItem{
					{
						Id:          10,
						Name:        "Basic Storage",
						Description: "Basic storage plan",
						Currency:    "USD",
						IsActive:    true,
					},
					{
						Id:          11,
						Name:        "Premium Storage",
						Description: "Premium storage plan",
						Currency:    "USD",
						IsActive:    true,
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "price line not found",
			priceLineID: "999",
			statusCode:  http.StatusNotFound,
			response:    admin.PriceLineDetailResponse{},
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "price line not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				expectedPath := "/api/billing/price-lines/" + tt.priceLineID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				} else if tt.statusCode == http.StatusNotFound {
					json.NewEncoder(w).Encode(admin.ErrorResponse{Error: "price line not found"})
				}
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			line, err := client.Billing().GetPriceLine(context.Background(), tt.priceLineID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPriceLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, line)
				require.Equal(t, "Storage", line.Name)
				require.Equal(t, "Storage pricing with plans", line.Description)
				require.Equal(t, int(1), line.Id)
				require.Equal(t, true, line.IsActive)
				require.Equal(t, false, line.IsDefault)
				require.NotNil(t, line.Plans)
				require.Len(t, line.Plans, 2)
				require.Equal(t, int(10), line.Plans[0].Id)
				require.Equal(t, int(11), line.Plans[1].Id)
			}
		})
	}
}