package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lumeweb.com/portal-sdk/internal/admin"
)

func TestPlatformDomainService_ListPlatformDomains(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list platform domains",
			statusCode: http.StatusOK,
			response: admin.PlatformDomainListResponse{
				Total: 2,
				Data: []admin.PlatformDomainResponse{
					{Id: 1, Domain: "example.com", Namespace: "users", ZoneId: 5, Enabled: true},
					{Id: 2, Domain: "example.org", Namespace: "orgs", ZoneId: 6, Enabled: false},
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
				require.Equal(t, "/api/ipfs/platform-domains", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			domains, total, err := client.PlatformDomains().ListPlatformDomains(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPlatformDomains() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, 2, total)
				require.Len(t, domains, 2)
				require.Equal(t, "example.com", domains[0].Domain)
				require.True(t, domains[0].Enabled)
				require.Equal(t, 6, domains[1].ZoneId)
				require.False(t, domains[1].Enabled)
			}
		})
	}
}

func TestPlatformDomainService_RegisterPlatformDomain(t *testing.T) {
	enabled := true

	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful register platform domain",
			statusCode: http.StatusCreated,
			response: admin.PlatformDomainResponse{
				Id: 3, Domain: "example.net", Namespace: "devs", ZoneId: 7, Enabled: true,
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
			name:       "invalid platform domain",
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "invalid platform domain data"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid platform domain data")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/ipfs/platform-domains", r.URL.Path)

				var body admin.PlatformDomainRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "example.net", body.Domain)
				require.Equal(t, "devs", body.Namespace)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			domain, err := client.PlatformDomains().RegisterPlatformDomain(context.Background(), &admin.PlatformDomainRequest{
				Domain: "example.net", Namespace: "devs", Enabled: &enabled,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterPlatformDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, domain)
				require.Equal(t, 3, domain.Id)
				require.Equal(t, "example.net", domain.Domain)
				require.True(t, domain.Enabled)
			}
		})
	}
}

func TestPlatformDomainService_DeletePlatformDomain(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful delete platform domain",
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
			name:       "platform domain not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "platform domain not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "platform domain not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "DELETE", r.Method)
				require.Equal(t, "/api/ipfs/platform-domains/42", r.URL.Path)

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

			err = client.PlatformDomains().DeletePlatformDomain(context.Background(), "42")

			if (err != nil) != tt.wantErr {
				t.Errorf("DeletePlatformDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestPlatformDomainService_UpdatePlatformDomain(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful update platform domain",
			statusCode: http.StatusOK,
			response: admin.PlatformDomainResponse{
				Id: 3, Domain: "example.net", Namespace: "devs", ZoneId: 7, Enabled: false,
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
			name:       "platform domain not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "platform domain not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "platform domain not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "PATCH", r.Method)
				require.Equal(t, "/api/ipfs/platform-domains/3", r.URL.Path)

				var body admin.PlatformDomainUpdateRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.False(t, body.Enabled)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			domain, err := client.PlatformDomains().UpdatePlatformDomain(context.Background(), "3", &admin.PlatformDomainUpdateRequest{Enabled: false})

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePlatformDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, domain)
				require.Equal(t, 3, domain.Id)
				require.False(t, domain.Enabled)
			}
		})
	}
}

func TestPlatformDomainService_BindWebsiteToPlatformDomain(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful bind website to platform domain",
			statusCode: http.StatusOK,
			response: admin.DomainResponse{
				Id: 9, Domain: "platform.example.com", Namespace: "users", DnsHostingEnabled: true,
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
			name:       "invalid bind request",
			statusCode: http.StatusBadRequest,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "invalid platform domain bind request"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid platform domain bind request")
			},
		},
		{
			name:       "platform domain not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "platform domain not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "platform domain not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/ipfs/platform-domains/5/bind", r.URL.Path)

				var body admin.PlatformDomainBindRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, 42, body.WebsiteId)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			rootDomain, err := client.PlatformDomains().BindWebsiteToPlatformDomain(context.Background(), "5", &admin.PlatformDomainBindRequest{WebsiteId: 42})

			if (err != nil) != tt.wantErr {
				t.Errorf("BindWebsiteToPlatformDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, rootDomain)
				require.Equal(t, 9, rootDomain.Id)
				require.Equal(t, "platform.example.com", rootDomain.Domain)
				require.True(t, rootDomain.DnsHostingEnabled)
			}
		})
	}
}
