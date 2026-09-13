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

func TestUserService_ListUsers(t *testing.T) {
	tests := []struct {
		name       string
		params     *UserListParams
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful list users",
			params:     &UserListParams{UnderscoreStart: intPtr(0), UnderscoreEnd: intPtr(10)},
			statusCode: http.StatusOK,
			response: admin.UserListResponse{
				Total: 2,
				Data: []admin.UserResponse{
					{Id: 1, FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Verified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
					{Id: 2, FirstName: "Bob", LastName: "Jones", Email: "bob@example.com", Verified: false, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				},
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			params:     nil,
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "authentication required")
			},
		},
		{
			name:       "forbidden",
			params:     nil,
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
				require.Equal(t, "/api/users", r.URL.Path)

				if tt.params != nil {
					require.Equal(t, "0", r.URL.Query().Get("_start"))
					require.Equal(t, "10", r.URL.Query().Get("_end"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			users, total, err := client.Users().ListUsers(context.Background(), tt.params)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.Equal(t, 2, total)
				require.Len(t, users, 2)
				require.Equal(t, 1, users[0].Id)
				require.Equal(t, "alice@example.com", users[0].Email)
				require.True(t, users[0].Verified)
				require.False(t, users[1].Verified)
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful create user",
			statusCode: http.StatusCreated,
			response: admin.UserResponse{
				Id: 3, FirstName: "Carol", LastName: "White", Email: "carol@example.com", Role: "user", Verified: false, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "unauthorized"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "authentication required")
			},
		},
		{
			name:       "validation failed",
			statusCode: http.StatusUnprocessableEntity,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "user validation failed"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user validation failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/users", r.URL.Path)

				var body admin.UserCreateRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "carol@example.com", body.Email)
				require.Equal(t, "secret123", body.Password)
				require.False(t, body.VerifyEmail)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			user, err := client.Users().CreateUser(context.Background(), &UserCreateRequest{
				Email:       "carol@example.com",
				Password:    "secret123",
				VerifyEmail: false,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, user)
				require.Equal(t, 3, user.Id)
				require.Equal(t, "carol@example.com", user.Email)
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful get user",
			statusCode: http.StatusOK,
			response: admin.UserResponse{
				Id: 1, FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Role: "admin", Verified: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "user not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/api/users/1", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			user, err := client.Users().GetUser(context.Background(), 1)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, user)
				require.Equal(t, 1, user.Id)
				require.Equal(t, "admin", user.Role)
				require.True(t, user.Verified)
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful update user",
			statusCode: http.StatusOK,
			response: admin.UserResponse{
				Id: 1, FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Verified: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "user not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "PATCH", r.Method)
				require.Equal(t, "/api/users/1", r.URL.Path)

				var body admin.UserUpdateRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.NotNil(t, body.Verified)
				require.True(t, *body.Verified)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(tt.response))
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			verified := true
			user, err := client.Users().UpdateUser(context.Background(), 1, &UserUpdateRequest{
				Verified: &verified,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, user)
				require.Equal(t, 1, user.Id)
				require.True(t, user.Verified)
			}
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful delete user",
			statusCode: http.StatusNoContent,
			response:   nil,
			wantErr:    false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   admin.ErrorResponse{Error: admin.ErrorDetail{Reason: "user not found"}},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "DELETE", r.Method)
				require.Equal(t, "/api/users/1", r.URL.Path)

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					w.Header().Set("Content-Type", "application/json")
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			client, err := NewClient(WithEndpoint(server.URL))
			require.NoError(t, err)

			err = client.Users().DeleteUser(context.Background(), 1)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
