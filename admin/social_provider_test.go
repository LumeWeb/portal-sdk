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

func sampleProviderResponse() admin.SocialProviderResponse {
	return admin.SocialProviderResponse{
		Id:          1,
		ProviderId:  "github",
		DisplayName: "GitHub",
		ClientId:    "client-123",
		Scopes:      []string{"user:email"},
		AuthUrl:     "https://github.com/login/oauth/authorize",
		TokenUrl:    "https://github.com/login/oauth/access_token",
		UserUrl:     "https://api.github.com/user",
		UserIdKey:   "id",
		UserEmailKey: "email",
		UserNameKey:  "login",
		Enabled:     true,
		OrderIndex:  0,
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestSocialProviderService_ListSocialProviders(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list social providers",
			statusCode: http.StatusOK,
			response: admin.SocialProviderListResponse{
				Total: 2,
				Data: []admin.SocialProviderResponse{
					sampleProviderResponse(),
					{
						Id: 2, ProviderId: "google", DisplayName: "Google", ClientId: "client-456",
						Scopes: []string{"openid"}, AuthUrl: "https://accounts.google.com/o/oauth2/v2/auth",
						TokenUrl: "https://oauth2.googleapis.com/token", UserUrl: "https://www.googleapis.com/oauth2/v3/userinfo",
						UserIdKey: "sub", UserEmailKey: "email", UserNameKey: "name",
						Enabled: false, OrderIndex: 1,
						CreatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "forbidden"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "insufficient permissions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/social/providers", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			providers, total, err := client.SocialProviders().ListSocialProviders(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListSocialProviders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, 2, total)
				require.Len(t, providers, 2)
				require.Equal(t, "github", providers[0].ProviderId)
				require.Equal(t, "GitHub", providers[0].DisplayName)
				require.True(t, providers[0].Enabled)
				require.Equal(t, "google", providers[1].ProviderId)
				require.False(t, providers[1].Enabled)
			}
		})
	}
}

func TestSocialProviderService_CreateSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful create social provider",
			statusCode: http.StatusCreated,
			response:   sampleProviderResponse(),
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "invalid social provider",
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "invalid social login provider data"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid social login provider data")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/social/providers", r.URL.Path)

				var body admin.SocialProviderRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "github", body.ProviderId)
				require.Equal(t, "GitHub", body.DisplayName)
				require.Equal(t, "secret", body.ClientSecret)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			provider, err := client.SocialProviders().CreateSocialProvider(context.Background(), &admin.SocialProviderRequest{
				ProviderId:  "github",
				DisplayName: "GitHub",
				ClientId:    "client-123",
				ClientSecret: "secret",
				Scopes:      []string{"user:email"},
				AuthUrl:     "https://github.com/login/oauth/authorize",
				TokenUrl:    "https://github.com/login/oauth/access_token",
				UserUrl:     "https://api.github.com/user",
				UserIdKey:   "id",
				UserEmailKey: "email",
				UserNameKey:  "login",
				Enabled:     true,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, provider)
				require.Equal(t, 1, provider.Id)
				require.Equal(t, "github", provider.ProviderId)
			}
		})
	}
}

func TestSocialProviderService_GetSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful get social provider",
			statusCode: http.StatusOK,
			response:   sampleProviderResponse(),
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "social provider not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "social login provider not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/social/providers/1", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			provider, err := client.SocialProviders().GetSocialProvider(context.Background(), "1")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, provider)
				require.Equal(t, 1, provider.Id)
				require.Equal(t, "github", provider.ProviderId)
			}
		})
	}
}

func TestSocialProviderService_UpdateSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful update social provider",
			statusCode: http.StatusOK,
			response:   sampleProviderResponse(),
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "social provider not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "social login provider not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "PUT", r.Method)
				require.Equal(t, "/api/social/providers/1", r.URL.Path)

				var body admin.SocialProviderRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "github", body.ProviderId)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			provider, err := client.SocialProviders().UpdateSocialProvider(context.Background(), "1", &admin.SocialProviderRequest{
				ProviderId: "github", DisplayName: "GitHub", ClientId: "client-123",
				Scopes: []string{"user:email"}, AuthUrl: "https://github.com/login/oauth/authorize",
				TokenUrl: "https://github.com/login/oauth/access_token", UserUrl: "https://api.github.com/user",
				UserIdKey: "id", UserEmailKey: "email", UserNameKey: "login", Enabled: true,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, provider)
				require.Equal(t, 1, provider.Id)
				require.Equal(t, "github", provider.ProviderId)
			}
		})
	}
}

func TestSocialProviderService_DeleteSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful delete social provider",
			statusCode: http.StatusNoContent,
			response:   nil,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "social provider not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "social login provider not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "DELETE", r.Method)
				require.Equal(t, "/api/social/providers/1", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				if tt.response != nil {
					w.WriteHeader(tt.statusCode)
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
					return
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			err = client.SocialProviders().DeleteSocialProvider(context.Background(), "1")

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestSocialProviderService_EnableSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful enable social provider",
			statusCode: http.StatusOK,
			response:   sampleProviderResponse(),
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "social provider not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "social login provider not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/social/providers/1/enable", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			provider, err := client.SocialProviders().EnableSocialProvider(context.Background(), "1")

			if (err != nil) != tt.wantErr {
				t.Errorf("EnableSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, provider)
				require.Equal(t, 1, provider.Id)
				require.True(t, provider.Enabled)
			}
		})
	}
}

func TestSocialProviderService_DisableSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful disable social provider",
			statusCode: http.StatusOK,
			response:   sampleProviderResponse(),
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "social provider not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "social login provider not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/social/providers/1/disable", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			provider, err := client.SocialProviders().DisableSocialProvider(context.Background(), "1")

			if (err != nil) != tt.wantErr {
				t.Errorf("DisableSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, provider)
				require.Equal(t, 1, provider.Id)
			}
		})
	}
}

func TestSocialProviderService_CreateSocialProvider_NilRequest(t *testing.T) {
	client, err := NewClient(WithEndpoint("http://127.0.0.1:1"))
	require.NoError(t, err)

	provider, err := client.SocialProviders().CreateSocialProvider(context.Background(), nil)
	require.Nil(t, provider)
	require.ErrorIs(t, err, ErrBadRequest)
	require.ErrorContains(t, err, "social login provider request is required")
}

func TestSocialProviderService_UpdateSocialProvider_NilRequest(t *testing.T) {
	client, err := NewClient(WithEndpoint("http://127.0.0.1:1"))
	require.NoError(t, err)

	provider, err := client.SocialProviders().UpdateSocialProvider(context.Background(), "1", nil)
	require.Nil(t, provider)
	require.ErrorIs(t, err, ErrBadRequest)
	require.ErrorContains(t, err, "social login provider request is required")
}
