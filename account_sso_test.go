package account

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.lumeweb.com/portal-sdk/internal/client"
)

func TestListSocialProviders(t *testing.T) {
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
			response: []client.PublicProviderResponse{
				{ProviderId: "google", DisplayName: "Google", OrderIndex: 0},
				{ProviderId: "github", DisplayName: "GitHub", OrderIndex: 1},
			},
			wantErr: false,
		},
		{
			name:       "provider not enabled",
			statusCode: http.StatusNotFound,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "social login provider not enabled"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not enabled")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/account/auth/providers", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			providers, err := c.ListSocialProviders(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListSocialProviders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Len(t, providers, 2)
				require.Equal(t, "google", providers[0].ProviderId)
				require.Equal(t, "GitHub", providers[1].DisplayName)
			}
		})
	}
}

func TestListSocialLinks(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list social links",
			statusCode: http.StatusOK,
			response: client.SocialAccountListResponse{
				Total: 1,
				Data: []client.SocialAccountResponse{
					{Provider: "google", ProviderUserId: "u-123", Email: "user@example.com", CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/account/auth/links", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			links, total, err := c.ListSocialLinks(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListSocialLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, 1, total)
				require.Len(t, links, 1)
				require.Equal(t, "google", links[0].Provider)
				require.Equal(t, "user@example.com", links[0].Email)
			}
		})
	}
}

func TestSocialLogin(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		location   string
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful social login redirect",
			statusCode: http.StatusFound,
			location:   "https://accounts.google.com/o/oauth2/v2/auth?client_id=x",
			wantErr:    false,
		},
		{
			name:       "provider not enabled",
			statusCode: http.StatusNotFound,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "social login provider not enabled"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not enabled")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/account/auth/sso/google", r.URL.Path)
				require.Equal(t, "/dashboard", r.URL.Query().Get("return"))

				w.Header().Set("Content-Type", "application/json")
				if tt.location != "" {
					w.Header().Set("Location", tt.location)
				}
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			redirect, err := c.SocialLogin(context.Background(), "google", "/dashboard")

			if (err != nil) != tt.wantErr {
				t.Errorf("SocialLogin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, tt.location, redirect)
			}
		})
	}
}

func TestSocialLogout(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful social logout",
			statusCode: http.StatusTemporaryRedirect,
			wantErr:    false,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "invalid request"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid request")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/account/auth/sso/google/logout", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			err := c.SocialLogout(context.Background(), "google")

			if (err != nil) != tt.wantErr {
				t.Errorf("SocialLogout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestLinkSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		location   string
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful link redirect",
			statusCode: http.StatusFound,
			location:   "https://github.com/login/oauth/authorize?client_id=y",
			wantErr:    false,
		},
		{
			name:       "provider not enabled",
			statusCode: http.StatusNotFound,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "social login provider not enabled"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not enabled")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/account/auth/sso/github/link", r.URL.Path)
				require.Equal(t, "/settings", r.URL.Query().Get("return"))

				w.Header().Set("Content-Type", "application/json")
				if tt.location != "" {
					w.Header().Set("Location", tt.location)
				}
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			redirect, err := c.LinkSocialProvider(context.Background(), "github", "/settings")

			if (err != nil) != tt.wantErr {
				t.Errorf("LinkSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, tt.location, redirect)
			}
		})
	}
}

func TestUnlinkSocialProvider(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful unlink",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unauthorized")
			},
		},
		{
			name:       "provider not enabled",
			statusCode: http.StatusNotFound,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "social login provider not enabled"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not enabled")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "DELETE", r.Method)
				require.Equal(t, "/api/account/auth/sso/github", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				if tt.response != nil {
					w.WriteHeader(tt.statusCode)
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
					return
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			err := c.UnlinkSocialProvider(context.Background(), "github")

			if (err != nil) != tt.wantErr {
				t.Errorf("UnlinkSocialProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestSocialConsent(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful consent approve",
			statusCode: http.StatusOK,
			response:   client.SocialConsentResponse{RedirectUri: "/dashboard"},
			wantErr:    false,
		},
		{
			name:       "provider not enabled",
			statusCode: http.StatusNotFound,
			response:   client.ErrorResponse{Error: client.ErrorDetail{Reason: "social login provider not enabled"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "social login provider not enabled")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/account/auth/sso/google/consent", r.URL.Path)

				var body client.PostApiAccountAuthSsoProviderConsentJSONBody
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.True(t, body.Approve)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			c := NewClient(WithEndpoint(server.URL))

			redirect, err := c.SocialConsent(context.Background(), "google", true)

			if (err != nil) != tt.wantErr {
				t.Errorf("SocialConsent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, "/dashboard", redirect)
			}
		})
	}
}
