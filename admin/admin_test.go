package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
				Plans: []admin.QuotaPlanResponse{
					{
						Name:               "Basic",
						UploadDailyLimit:   100,
						DownloadDailyLimit: 1000,
						StorageLimit:       10000,
						Description:        "Basic plan",
					},
					{
						Name:               "Premium",
						UploadDailyLimit:   500,
						DownloadDailyLimit: 5000,
						StorageLimit:       50000,
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

func ptrStr(s string) *string {
	return &s
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
					UploadDailyLimit:   1000,
					DownloadDailyLimit: 10000,
					StorageLimit:       100000,
					Description:        "Enterprise plan",
				},
			},
			statusCode: http.StatusCreated,
			response: admin.QuotaPlanResponse{
				Name:               "Enterprise",
				UploadDailyLimit:   1000,
				DownloadDailyLimit: 10000,
				StorageLimit:       100000,
				Description:        "Enterprise plan",
			},
			wantErr: false,
		},
		{
			name: "unauthorized",
			plan: &QuotaPlan{
				QuotaPlanResponse: admin.QuotaPlanResponse{
					Name:               "Test",
					UploadDailyLimit:   100,
					DownloadDailyLimit: 1000,
					StorageLimit:       10000,
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
				require.Equal(t, int(1000), plan.UploadDailyLimit)
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
					UploadDailyLimit:   1500,
					DownloadDailyLimit: 15000,
					StorageLimit:       150000,
					Description:        "Updated enterprise plan",
				},
			},
			statusCode: http.StatusOK,
			response: admin.QuotaPlanResponse{
				Name:               "Enterprise Updated",
				UploadDailyLimit:   1500,
				DownloadDailyLimit: 15000,
				StorageLimit:       150000,
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
				UploadDailyLimit:   100,
				DownloadDailyLimit: 1000,
				StorageLimit:       10000,
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
		userID     int
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list allowances",
			userID:     123,
			statusCode: http.StatusOK,
			response: admin.AllowanceListResponse{
				Grants: []admin.AllowanceGrantResponse{
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
			userID:     123,
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
			allowances, _, err := client.Quota().ListAllowances(context.Background(), tt.userID)

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

func TestQuotaService_GetConfig(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful get config",
			statusCode: http.StatusOK,
			response: admin.QuotaConfigResponse{
				DefaultPlanId:          1,
				DefaultPlanName:       ptrStr("Basic"),
				EnableQuotaEnforcement: true,
				StorageRetentionDays:  30,
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
				if r.URL.Path != "/api/quota/system/config" {
					t.Errorf("expected /api/quota/system/config path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			config, err := client.Quota().GetConfig(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.True(t, config.EnableQuotaEnforcement)
				require.Equal(t, int(1), config.DefaultPlanId)
			}
		})
	}
}

func TestQuotaService_UpdateConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     *QuotaConfig
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name: "successful config update",
			config: &QuotaConfig{
				QuotaConfigResponse: admin.QuotaConfigResponse{
					DefaultPlanId:          2,
					DefaultPlanName:       ptrStr("Premium"),
					EnableQuotaEnforcement: true,
					StorageRetentionDays:  60,
				},
			},
			statusCode: http.StatusOK,
			response: admin.QuotaConfigResponse{
				DefaultPlanId:          2,
				DefaultPlanName:       ptrStr("Premium"),
				EnableQuotaEnforcement: true,
				StorageRetentionDays:  60,
			},
			wantErr: false,
		},
		{
			name: "unauthorized",
			config: &QuotaConfig{
				QuotaConfigResponse: admin.QuotaConfigResponse{},
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
				if r.Method != "PUT" {
					t.Errorf("expected PUT request, got %s", r.Method)
				}
				if r.URL.Path != "/api/quota/system/config" {
					t.Errorf("expected /api/quota/system/config path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			config, err := client.Quota().UpdateConfig(context.Background(), tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, int(2), config.DefaultPlanId)
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
