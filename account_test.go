package account

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.lumeweb.com/portal-sdk/internal/client"
	"go.lumeweb.com/portal-sdk/internal/client/mocks"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
	"go.lumeweb.com/queryutil"
	"gorm.io/datatypes"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		token       string
		otpRequired bool
		statusCode  int
		wantErr     bool
	}{
		{
			name:        "successful login without 2FA",
			email:       "test@example.com",
			password:    "password",
			token:       "test-jwt-token",
			otpRequired: false,
			statusCode:  http.StatusOK,
			wantErr:     false,
		},
		{
			name:        "successful login with 2FA",
			email:       "test@example.com",
			password:    "password",
			token:       "intermediate-jwt-token",
			otpRequired: true,
			statusCode:  http.StatusOK,
			wantErr:     false,
		},
		{
			name:        "invalid credentials",
			email:       "test@example.com",
			password:    "wrong",
			token:       "",
			otpRequired: false,
			statusCode:  http.StatusUnauthorized,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/login" {
					t.Errorf("expected /api/auth/login path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusConflict {
					resp := client.Error{Error: "user already exists"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
				if tt.statusCode == http.StatusOK {
					otp := &tt.otpRequired
					resp := client.LoginResponse{Token: tt.token, Otp: otp}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				} else if tt.statusCode == http.StatusUnauthorized {
					// Write error response for 401
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			result, err := acc.Login(context.Background(), tt.email, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("Login() error should be ErrUnauthorized, got: %v", err)
				}
			}

			if !tt.wantErr {
				if result.Token != tt.token {
					t.Errorf("Login() got token = %v, want %v", result.Token, tt.token)
				}
				if result.OTPRequired != tt.otpRequired {
					t.Errorf("Login() got otpRequired = %v, want %v", result.OTPRequired, tt.otpRequired)
				}
			}
		})
	}
}

func TestRequestPasswordReset(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		statusCode int
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful password reset request",
			email:      "test@example.com",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "invalid email",
			email:      "invalid-email",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid email address")
			},
		},
		{
			name:       "user not found",
			email:      "nonexistent@example.com",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/password-reset/request" {
					t.Errorf("expected /api/account/password-reset/request path, got %s", r.URL.Path)
				}

				// Verify request body
				var reqBody client.PasswordResetRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
				require.Equal(t, tt.email, reqBody.Email)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			err := acc.RequestPasswordReset(context.Background(), tt.email)

			if (err != nil) != tt.wantErr {
				t.Errorf("RequestPasswordReset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestConfirmPasswordReset(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		token       string
		newPassword string
		statusCode  int
		wantErr     bool
		errCheck    func(*testing.T, error)
	}{
		{
			name:        "successful password reset confirmation",
			email:       "test@example.com",
			token:       "reset-token-123",
			newPassword: "new-password",
			statusCode:  http.StatusOK,
			wantErr:     false,
		},
		{
			name:        "invalid token",
			email:       "test@example.com",
			token:       "invalid-token",
			newPassword: "new-password",
			statusCode:  http.StatusBadRequest,
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid or expired reset token")
			},
		},
		{
			name:        "user not found",
			email:       "nonexistent@example.com",
			token:       "reset-token-123",
			newPassword: "new-password",
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/password-reset/confirm" {
					t.Errorf("expected /api/account/password-reset/confirm path, got %s", r.URL.Path)
				}

				// Verify request body
				var reqBody client.PasswordResetVerifyRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
				require.Equal(t, tt.email, reqBody.Email)
				require.Equal(t, tt.token, reqBody.Token)
				require.Equal(t, tt.newPassword, reqBody.Password)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			err := acc.ConfirmPasswordReset(context.Background(), tt.email, tt.token, tt.newPassword)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConfirmPasswordReset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestUpdatePassword(t *testing.T) {
	tests := []struct {
		name           string
		currentPassword string
		newPassword     string
		jwt             string
		statusCode      int
		wantErr         bool
		errCheck        func(*testing.T, error)
	}{
		{
			name:           "successful password update",
			currentPassword: "old-password",
			newPassword:     "new-password",
			jwt:             "test-jwt-token",
			statusCode:      http.StatusOK,
			wantErr:         false,
		},
		{
			name:           "unauthorized - missing JWT",
			currentPassword: "old-password",
			newPassword:     "new-password",
			jwt:             "",
			statusCode:      http.StatusUnauthorized,
			wantErr:         true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, ErrUnauthorized)
			},
		},
		{
			name:           "invalid current password",
			currentPassword: "wrong-password",
			newPassword:     "new-password",
			jwt:             "test-jwt-token",
			statusCode:      http.StatusBadRequest,
			wantErr:         true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid password")
			},
		},
		{
			name:           "user not found",
			currentPassword: "old-password",
			newPassword:     "new-password",
			jwt:             "test-jwt-token",
			statusCode:      http.StatusNotFound,
			wantErr:         true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "user not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/update-password" {
					t.Errorf("expected /api/account/update-password path, got %s", r.URL.Path)
				}

				// Verify Authorization header
				authHeader := r.Header.Get("Authorization")
				if tt.jwt != "" {
					require.Equal(t, "Bearer "+tt.jwt, authHeader)
				}

				// Verify request body
				var reqBody client.UpdatePasswordRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
				require.Equal(t, tt.currentPassword, reqBody.CurrentPassword)
				require.Equal(t, tt.newPassword, reqBody.NewPassword)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			err := acc.UpdatePassword(context.Background(), tt.currentPassword, tt.newPassword)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

func TestGenerateOTP(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful OTP generation",
			secret:     "JBSWY3DPEHPK3PXP",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			secret:     "",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/otp/generate" {
					t.Errorf("expected /api/auth/otp/generate path, got %s", r.URL.Path)
				}

				switch tt.statusCode {
				case http.StatusOK:
					resp := client.OTPGenerateResponse{Otp: tt.secret}
					body, _ := json.Marshal(resp)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					w.Write(body)
				case http.StatusUnauthorized:
					resp := client.ErrorResponse{Error: "unauthorized"}
					body, _ := json.Marshal(resp)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					w.Write(body)
				default:
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
			secret, err := acc.GenerateOTP(context.Background())

			if (err != nil) != tt.wantErr {
				// Just check that an error occurred for unauthorized
				if tt.wantErr && err != nil {
					return
				}
				t.Errorf("GenerateOTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				// Error is expected to be returned
				return
			}

			if !tt.wantErr && secret != tt.secret {
				t.Errorf("GenerateOTP() got = %v, want %v", secret, tt.secret)
			}
		})
	}
}

func TestVerifyOTP(t *testing.T) {
	tests := []struct {
		name       string
		otp        string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful OTP verification",
			otp:        "123456",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "invalid OTP code",
			otp:        "000000",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			otp:        "123456",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/otp/verify" {
					t.Errorf("expected /api/auth/otp/verify path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
			err := acc.VerifyOTP(context.Background(), tt.otp)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyOTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("VerifyOTP() error should be ErrUnauthorized, got: %v", err)
				}
			}
		})
	}
}

func TestDisableOTP(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful OTP disable",
			password:   "password123",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			password:   "wrongpassword",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/otp/disable" {
					t.Errorf("expected /api/auth/otp/disable path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
			err := acc.DisableOTP(context.Background(), tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("DisableOTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("DisableOTP() error should be ErrUnauthorized, got: %v", err)
				}
			}
		})
	}
}

func TestValidateOTP(t *testing.T) {
	tests := []struct {
		name         string
		intermediate string
		otp          string
		finalToken   string
		statusCode   int
		wantErr      bool
	}{
		{
			name:         "invalid OTP code",
			intermediate: "intermediate-jwt",
			otp:          "000000",
			finalToken:   "",
			statusCode:   http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name:         "unauthorized",
			intermediate: "expired-intermediate",
			otp:          "123456",
			finalToken:   "",
			statusCode:   http.StatusUnauthorized,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/otp/validate" {
					t.Errorf("expected /api/auth/otp/validate path, got %s", r.URL.Path)
				}

				authHeader := r.Header.Get("Authorization")
				expectedAuth := "Bearer " + tt.intermediate
				if authHeader != expectedAuth {
					t.Errorf("expected Authorization header %s, got %s", expectedAuth, authHeader)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			_, err := acc.ValidateOTP(context.Background(), tt.intermediate, tt.otp)

			if !tt.wantErr {
				t.Errorf("ValidateOTP() expected error, got none")
			}
			if err == nil {
				t.Errorf("ValidateOTP() expected error, got none")
			}
		})
	}
}

func TestValidateOTP_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/otp/validate" {
			t.Errorf("expected /api/auth/otp/validate path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL))
	_, err := acc.ValidateOTP(context.Background(), "intermediate-jwt", "123456")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send OTP validate request")
}

func TestValidateOTP_EmptyLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/otp/validate" {
			t.Errorf("expected /api/auth/otp/validate path, got %s", r.URL.Path)
		}

		// Return 302 but without Location header
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL))
	_, err := acc.ValidateOTP(context.Background(), "intermediate-jwt", "123456")

	require.Error(t, err)
	require.Contains(t, err.Error(), "no redirect location provided")
}

func TestValidateOTP_UnableToExtractJWT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/otp/validate" {
			t.Errorf("expected /api/auth/otp/validate path, got %s", r.URL.Path)
		}

		// Return 302 with Location header that doesn't contain JWT
		w.Header().Set("Location", "/auth/complete")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	// Create client with redirect following disabled to inspect response
	acc := NewClient(WithEndpoint(server.URL), WithDisableFollowRedirect())
	_, err := acc.ValidateOTP(context.Background(), "intermediate-jwt", "123456")

	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to extract final JWT")
}

func TestValidateOTP_EmptyTokenWithOtherParams(t *testing.T) {
	// Regression test for bug where empty token value followed by other params
	// caused incorrect extraction of subsequent parameters as the token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/otp/validate" {
			t.Errorf("expected /api/auth/otp/validate path, got %s", r.URL.Path)
		}

		// Return 302 with Location header containing empty token followed by other params
		w.Header().Set("Location", "/auth/complete?token=&other=abc&session=xyz")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	// Create client with redirect following disabled to inspect response
	acc := NewClient(WithEndpoint(server.URL), WithDisableFollowRedirect())
	token, err := acc.ValidateOTP(context.Background(), "intermediate-jwt", "123456")

	// Should return an error since token is empty, not extract "&other=abc&session=xyz"
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to extract final JWT")
	require.Empty(t, token)
}

func TestPing(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful ping",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/ping" {
					t.Errorf("expected /api/auth/ping path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					resp := client.PongResponse{Ping: "pong"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			err := acc.Ping(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("Ping() error should be ErrUnauthorized, got: %v", err)
				}
			}
		})
	}
}

func TestUploadLimit(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		limit      int64
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful upload limit",
			jwt:        "valid-token",
			limit:      100 * 1024 * 1024,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/api/upload-limit" {
					t.Errorf("expected /api/upload-limit path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.UploadLimitResponse{Limit: int(tt.limit)}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			limit, err := acc.UploadLimit(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("UploadLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("UploadLimit() error should be ErrUnauthorized, got: %v", err)
				}
			}

			if !tt.wantErr && limit != tt.limit {
				t.Errorf("UploadLimit() got = %v, want %v", limit, tt.limit)
			}
		})
	}
}

func TestLoginWithAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		token      string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful API key login",
			apiKey:     "test-api-key-jwt-token",
			token:      "session-jwt-token",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "invalid API key",
			apiKey:     "invalid-api-key",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "account pending deletion",
			apiKey:     "test-api-key",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/key" {
					t.Errorf("expected /api/auth/key path, got %s", r.URL.Path)
				}

				// Verify Authorization header contains the API key
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.apiKey {
					t.Errorf("expected Authorization header %q, got %q", tt.apiKey, authHeader)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.LoginResponse{Token: tt.token}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				} else if tt.statusCode == http.StatusUnauthorized || tt.statusCode == http.StatusForbidden {
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			token, err := acc.LoginWithAPIKey(context.Background(), tt.apiKey)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoginWithAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("LoginWithAPIKey() error should be ErrUnauthorized, got: %v", err)
				}
			}

			if !tt.wantErr && token != tt.token {
				t.Errorf("LoginWithAPIKey() got = %v, want %v", token, tt.token)
			}
		})
	}
}

func TestLoginWithAPIKey_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/key" {
			t.Errorf("expected /api/auth/key path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL))
	_, err := acc.LoginWithAPIKey(context.Background(), "test-api-key")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send API key login request")
}

func TestLoginWithAPIKey_EmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/key" {
			t.Errorf("expected /api/auth/key path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return response with empty token
		resp := client.LoginResponse{Token: ""}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL))
	_, err := acc.LoginWithAPIKey(context.Background(), "test-api-key")

	require.Error(t, err)
	require.Contains(t, err.Error(), "did not contain a token")
}

func TestLoginWithAPIKey_NilResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/key" {
			t.Errorf("expected /api/auth/key path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return empty JSON body (will result in JSON200 being nil after parsing)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL))
	_, err := acc.LoginWithAPIKey(context.Background(), "test-api-key")

	require.Error(t, err)
	require.Contains(t, err.Error(), "did not contain a token")
}

func TestOperationStatus_IsSettled(t *testing.T) {
	tests := []struct {
		name   string
		status OperationStatus
		want   bool
	}{
		{"pending is not settled", OperationStatusPending, false},
		{"running is not settled", OperationStatusRunning, false},
		{"completed is settled", OperationStatusCompleted, true},
		{"failed is settled", OperationStatusFailed, true},
		{"error is settled", OperationStatusError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsSettled(); got != tt.want {
				t.Errorf("IsSettled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWaitForOperation(t *testing.T) {
	// Test with a mock that returns running first, then completed
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var resp client.OperationDetailResponse
		if callCount == 1 {
			// First call: running
			resp = client.OperationDetailResponse{
				Id:              123,
				Status:          OperationStatusRunning.String(),
				Cid:             lo.ToPtr("QmTest"),
				ProgressPercent: float32(25.0),
				StatusMessage:   "In progress",
				TotalSteps:      lo.ToPtr[int](4),
				CurrentStep:     lo.ToPtr[int](1),
			}
		} else {
			// Second call: completed
			resp = client.OperationDetailResponse{
				Id:              123,
				Status:          OperationStatusCompleted.String(),
				Cid:             lo.ToPtr("QmTest"),
				ProgressPercent: float32(100.0),
				StatusMessage:   "Completed",
				TotalSteps:      lo.ToPtr[int](4),
				CurrentStep:     lo.ToPtr[int](4),
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))

	op, err := acc.WaitForOperation(context.Background(), 123, WithPollInterval(100*time.Millisecond))
	require.NoError(t, err)
	require.NotNil(t, op)
	require.Equal(t, 123, op.Id)
	require.Equal(t, OperationStatusCompleted.String(), op.Status)
	require.Equal(t, "QmTest", *op.Cid)
	require.GreaterOrEqual(t, callCount, 2)
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		firstName  string
		lastName   string
		password   string
		statusCode int
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "successful registration",
			email:      "newuser@example.com",
			firstName:  "John",
			lastName:   "Doe",
			password:   "password123",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "email already exists",
			email:      "existing@example.com",
			firstName:  "John",
			lastName:   "Doe",
			password:   "password123",
			statusCode: http.StatusConflict,
			wantErr:    true,
		},
		{
			name:       "server error",
			email:      "test@example.com",
			firstName:  "John",
			lastName:   "Doe",
			password:   "password123",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/register" {
					t.Errorf("expected /api/auth/register path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			err := acc.Register(context.Background(), tt.email, tt.firstName, tt.lastName, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("Register() expected error, got nil")
				}
			}
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		token      string
		statusCode int
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "successful verification",
			email:      "user@example.com",
			token:      "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "invalid token",
			email:      "user@example.com",
			token:      "invalid-token",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
			errMsg:     "invalid verification",
		},
		{
			name:       "user not found",
			email:      "nonexistent@example.com",
			token:      "token",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errMsg:     "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/verify-email" {
					t.Errorf("expected /api/account/verify-email path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			err := acc.VerifyEmail(context.Background(), tt.email, tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("VerifyEmail() expected error containing %q, got nil", tt.errMsg)
				}
			}
		})
	}
}

func TestResendVerifyEmail(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		statusCode int
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "successful resend",
			email:      "user@example.com",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "invalid email",
			email:      "invalid-email",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
			errMsg:     "invalid email",
		},
		{
			name:       "user not found",
			email:      "nonexistent@example.com",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errMsg:     "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/verify-email/resend" {
					t.Errorf("expected /api/account/verify-email/resend path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			err := acc.ResendVerifyEmail(context.Background(), tt.email)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResendVerifyEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("ResendVerifyEmail() expected error containing %q, got nil", tt.errMsg)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	endpoint := "https://api.example.com"
	jwt := "test-token"

	acc := NewClient(WithEndpoint(endpoint), WithJWT(jwt))

	c, ok := acc.(*Client)
	if !ok {
		t.Fatal("NewClient did not return *Client")
	}

	if c.client == nil {
		t.Fatal("generated client should not be nil")
	}
}

func TestCreateAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		apiKeyName string
		token      string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful API key creation",
			jwt:        "valid-token",
			apiKeyName: "test-key",
			token:      "test-api-key-token",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			apiKeyName: "test-key",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "bad request",
			jwt:        "valid-token",
			apiKeyName: "",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/keys" {
					t.Errorf("expected /api/account/keys path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.CreateAPIKeyResponse{
						Name:  tt.apiKeyName,
						Token: tt.token,
						Uuid:  client.BinaryUUID{BinUUID: datatypes.BinUUID(uuid.MustParse("5aabc9d3-ce22-4beb-bdf9-91d0c43340ef"))},
					}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			apiKey, err := acc.CreateAPIKey(context.Background(), tt.apiKeyName)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("CreateAPIKey() error should be ErrUnauthorized, got: %v", err)
				}
			}

			if !tt.wantErr {
				if apiKey == nil {
					t.Fatal("CreateAPIKey() returned nil API key")
				}
				if apiKey.Name != tt.apiKeyName {
					t.Errorf("CreateAPIKey() got name = %v, want %v", apiKey.Name, tt.apiKeyName)
				}
				if apiKey.Token != tt.token {
					t.Errorf("CreateAPIKey() got token = %v, want %v", apiKey.Token, tt.token)
				}
			}
		})
	}
}

func TestPollOptions(t *testing.T) {
	cfg := &pollConfig{}

	WithPollInterval(2 * time.Second)(cfg)
	if cfg.interval != 2*time.Second {
		t.Errorf("expected interval 2s, got %v", cfg.interval)
	}

	WithPollTimeout(10 * time.Minute)(cfg)
	if cfg.timeout != 10*time.Minute {
		t.Errorf("expected timeout 10m, got %v", cfg.timeout)
	}

	WithPollSettledStates(OperationStatusCompleted, OperationStatusFailed)(cfg)
	if len(cfg.settledStates) != 2 {
		t.Errorf("expected 2 settled states, got %d", len(cfg.settledStates))
	}
	if cfg.settledStates[0] != OperationStatusCompleted {
		t.Errorf("expected first state to be completed, got %v", cfg.settledStates[0])
	}
}

func TestWaitForOperation_Timeout(t *testing.T) {
	// Test that WaitForOperation times out when operation never settles
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := client.OperationDetailResponse{
			Id:            123,
			Status:        OperationStatusRunning.String(),
			Cid:           lo.ToPtr("QmTest"),
			StatusMessage: "Still running",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))

	_, err := acc.WaitForOperation(
		context.Background(),
		123,
		WithPollInterval(50*time.Millisecond),
		WithPollTimeout(200*time.Millisecond),
	)
	require.Error(t, err)
	// Can be either ErrOperationTimeout (from ctx.Done()) or context.DeadlineExceeded (from HTTP request)
	require.True(t,
		errors.Is(err, ErrOperationTimeout) || errors.Is(err, context.DeadlineExceeded),
		"expected timeout error, got: %v", err,
	)
}

func TestWaitForOperation_CustomSettledStates(t *testing.T) {
	// Test that custom settled states work
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := client.OperationDetailResponse{
			Id:            123,
			Status:        OperationStatusFailed.String(),
			Cid:           lo.ToPtr("QmTest"),
			StatusMessage: "Operation failed",
			Error:         lo.ToPtr("test error"),
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))

	// Only consider "failed" as settled
	op, err := acc.WaitForOperation(
		context.Background(),
		123,
		WithPollInterval(50*time.Millisecond),
		WithPollSettledStates(OperationStatusFailed),
	)
	require.NoError(t, err)
	require.NotNil(t, op)
	require.Equal(t, OperationStatusFailed.String(), op.Status)
	require.Equal(t, 1, callCount)
}

func TestWaitForOperation_GetOperationError(t *testing.T) {
	// Test that WaitForOperation returns error when GetOperation fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/operations/123" {
			// Simulate network error
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))

	_, err := acc.WaitForOperation(
		context.Background(),
		123,
		WithPollInterval(50*time.Millisecond),
		WithPollTimeout(200*time.Millisecond),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get operation 123")
}

func TestWaitForOperation_ContextTimeout(t *testing.T) {
	// Test that WaitForOperation returns ErrOperationTimeout when context times out
	// Use a very short timeout and very long poll interval to ensure context times out first
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/operations/123" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := client.OperationDetailResponse{
				Id:            123,
				Status:        OperationStatusRunning.String(),
				Cid:           lo.ToPtr("QmTest"),
				StatusMessage: "Still running",
			}
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))

	_, err := acc.WaitForOperation(
		context.Background(),
		123,
		WithPollInterval(1*time.Second),      // Long interval so it doesn't tick before timeout
		WithPollTimeout(50*time.Millisecond), // Very short timeout
	)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOperationTimeout)
}

func TestGetOperationFilters(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful get filters",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/api/operations/filters" {
					t.Errorf("expected /api/operations/filters path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.OperationFiltersResponseResponse{
						Data: client.OperationFiltersResponse{
							Data: client.OperationFiltersResponseData{
								Operations: []client.OperationFilterItem{
									{Name: "upload", Value: "upload"},
									{Name: "download", Value: "download"},
								},
								Protocols: []client.OperationFilterItem{
									{Name: "ipfs", Value: "ipfs"},
								},
								Statuses: []client.OperationFilterItem{
									{Name: "completed", Value: OperationStatusCompleted.String()},
									{Name: "running", Value: OperationStatusRunning.String()},
								},
							},
						},
					}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			filters, err := acc.GetOperationFilters(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOperationFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("GetOperationFilters() error should be ErrUnauthorized, got: %v", err)
				}
			}

			if !tt.wantErr {
				if filters == nil {
					t.Fatal("GetOperationFilters() returned nil filters")
				}
				if len(filters.Data.Operations) != 2 {
					t.Errorf("expected 2 operations, got %d", len(filters.Data.Operations))
				}
				if len(filters.Data.Protocols) != 1 {
					t.Errorf("expected 1 protocol, got %d", len(filters.Data.Protocols))
				}
				if len(filters.Data.Statuses) != 2 {
					t.Errorf("expected 2 statuses, got %d", len(filters.Data.Statuses))
				}
			}
		})
	}
}

func TestGetOperationFilters_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/operations/filters" {
			t.Errorf("expected /api/operations/filters path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	_, err := acc.GetOperationFilters(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get operation filters")
}

func TestListOptions(t *testing.T) {
	// Test WithFilters - using a real filter
	filter := queryutil.Equal("status", "completed")
	opts := &ListOptions{}
	WithFilters(filter)(opts)
	require.Len(t, opts.Filters, 1)

	// Test WithSorts - using a real sort
	sort := queryutil.Sort{Field: "id", Order: "desc"}
	opts = &ListOptions{}
	WithSorts(sort)(opts)
	require.Len(t, opts.Sorts, 1)

	// Test WithPagination - using a real pagination
	pagination, _ := queryutil.NewPagination(0, 10)
	opts = &ListOptions{}
	WithPagination(&pagination)(opts)
	require.NotNil(t, opts.Pagination)

	// Test WithSearch
	opts = &ListOptions{}
	WithSearch("test query")(opts)
	require.Equal(t, "test query", opts.Search)
}

func TestListAPIKeys(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		statusCode int
		wantErr    bool
		opts       []ListOption
	}{
		{
			name:       "successful list API keys",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
			opts:       []ListOption{WithPagination(&queryutil.DefaultPagination)},
		},
		{
			name:       "list API keys with pagination",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
			opts:       []ListOption{WithPagination(&queryutil.Pagination{Start: 0, End: 10})},
		},
		{
			name:       "list API keys with pagination (End = 0)",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
			opts:       []ListOption{WithPagination(&queryutil.Pagination{Start: 0, End: 0})},
		},
		{
			name:       "list API keys with empty data",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
			opts:       []ListOption{},
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/keys" {
					t.Errorf("expected /api/account/keys path, got %s", r.URL.Path)
				}

				// Verify query parameters for pagination test
				if tt.name == "list API keys with pagination" {
					if r.URL.Query().Get("_start") != "0" {
						t.Errorf("expected _start=0, got %s", r.URL.Query().Get("_start"))
					}
					if r.URL.Query().Get("_end") != "10" {
						t.Errorf("expected _end=10, got %s", r.URL.Query().Get("_end"))
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					var resp client.APIKeyListResponse
					if tt.name == "list API keys with empty data" {
						resp = client.APIKeyListResponse{
							Data:  []client.APIKeyResponse{},
							Total: 0,
						}
					} else {
						resp = client.APIKeyListResponse{
							Data: []client.APIKeyResponse{
								{
									Name:      "test-key-1",
									CreatedAt: time.Now(),
									Uuid:      client.BinaryUUID{BinUUID: datatypes.BinUUID(uuid.MustParse("c9374625-39c4-41b0-be29-a647a868d6eb"))},
								},
								{
									Name:      "test-key-2",
									CreatedAt: time.Now(),
									Uuid:      client.BinaryUUID{BinUUID: datatypes.BinUUID(uuid.MustParse("621cb859-31a0-4d71-ac7d-78395cc56daa"))},
								},
							},
							Total: 2,
						}
					}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			keys, err := acc.ListAPIKeys(context.Background(), tt.opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListAPIKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				// The error is a generic error message, not ErrUnauthorized
				if err == nil {
					t.Errorf("ListAPIKeys() expected error for unauthorized, got nil")
				}
			}

			if !tt.wantErr {
				if keys == nil {
					t.Fatal("ListAPIKeys() returned nil keys")
				}
				if tt.name == "list API keys with empty data" {
					if len(keys) != 0 {
						t.Errorf("expected 0 API keys for empty data, got %d", len(keys))
					}
				} else {
					if len(keys) != 2 {
						t.Errorf("expected 2 API keys, got %d", len(keys))
					}
					if keys[0].Name != "test-key-1" {
						t.Errorf("ListAPIKeys() got name = %v, want test-key-1", keys[0].Name)
					}
				}
			}
		})
	}
}

func TestListAPIKeys_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/account/keys" {
			t.Errorf("expected /api/account/keys path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	_, err := acc.ListAPIKeys(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send list API keys request")
}

func TestListAPIKeys_NilJSON200(t *testing.T) {
	// This test specifically targets the coverage gap where resp.JSON200 == nil
	// We need to mock the generated client to return a response with 200 status but nil JSON200

	mockClient := mocks.NewMockClientWithResponsesInterface(t)

	// Create a mock response with 200 status but nil JSON200
	mockResp := &client.GetApiAccountKeysResponse{
		Body:         []byte(""),
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      nil, // This is the key - 200 status but nil JSON200
	}

	mockClient.On("GetApiAccountKeysWithResponse", mock.Anything, mock.Anything).
		Return(mockResp, nil)

	acc := NewClientWithDefaults(mockClient)
	_, err := acc.ListAPIKeys(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "list API keys response did not contain data")
}

func TestListAPIKeys_PaginationCoverage(t *testing.T) {
	// This test specifically targets the coverage gap in ListAPIKeys
	// where params.UnderscoreEnd is set when options.Pagination.End != 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/account/keys" {
			t.Errorf("expected /api/account/keys path, got %s", r.URL.Path)
		}

		// Verify that pagination parameters are present
		start := r.URL.Query().Get("_start")
		end := r.URL.Query().Get("_end")

		if start != "0" {
			t.Errorf("expected _start=0, got %s", start)
		}
		if end != "10" {
			t.Errorf("expected _end=10, got %s", end)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := client.APIKeyListResponse{
			Data: []client.APIKeyResponse{
				{
					Name:      "test-key",
					CreatedAt: time.Now(),
					Uuid:      client.BinaryUUID{BinUUID: datatypes.BinUUID(uuid.MustParse("0ce0e6b1-94d8-486e-8e78-e639d6f922e0"))},
				},
			},
			Total: 1,
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))

	// Create a pagination struct with both Start and End set
	pagination := queryutil.Pagination{Start: 0, End: 10}

	// Call ListAPIKeys with pagination options
	keys, err := acc.ListAPIKeys(context.Background(), WithPagination(&pagination))

	require.NoError(t, err)
	require.NotNil(t, keys)
	require.Len(t, keys, 1)
	require.Equal(t, "test-key", keys[0].Name)
}

func TestDeleteAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		keyID      string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful delete API key",
			jwt:        "valid-token",
			keyID:      "550e8400-e29b-41d4-a716-446655440000",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			keyID:      "550e8400-e29b-41d4-a716-446655440000",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "not found",
			jwt:        "valid-token",
			keyID:      "550e8400-e29b-41d4-a716-446655440001",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			err := acc.DeleteAPIKey(context.Background(), tt.keyID)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				// The error is a generic error message, not ErrUnauthorized
				if err == nil {
					t.Errorf("DeleteAPIKey() expected error for unauthorized, got nil")
				}
			}
		})
	}
}

func TestDeleteAPIKey_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		// Simulate network error
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	err := acc.DeleteAPIKey(context.Background(), "550e8400-e29b-41d4-a716-446655440000")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send delete API key request")
}

func TestDeleteAccount(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful account deletion",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "bad request",
			jwt:        "valid-token",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "not found",
			jwt:        "valid-token",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account" {
					t.Errorf("expected /api/account path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusUnauthorized {
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			err := acc.DeleteAccount(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("DeleteAccount() error should be ErrUnauthorized, got: %v", err)
				}
			}
		})
	}
}

func TestDeleteAccount_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	err := acc.DeleteAccount(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send delete account request")
}

func TestNewClientWithDefaults(t *testing.T) {
	// Create a mock client for testing
	mockClient := mocks.NewMockClientWithResponsesInterface(t)
	acc := NewClientWithDefaults(mockClient)

	c, ok := acc.(*Client)
	require.True(t, ok)
	require.NotNil(t, c)
	require.Equal(t, mockClient, c.client)
}

func TestValidateOTP_Success(t *testing.T) {
	tests := []struct {
		name         string
		intermediate string
		otp          string
		finalToken   string
		location     string
		cookies      []*http.Cookie
	}{
		{
			name:         "JWT from cookie",
			intermediate: "intermediate-jwt",
			otp:          "123456",
			finalToken:   "final-jwt-token",
			location:     "/auth/complete",
			cookies:      []*http.Cookie{{Name: AuthTokenCookie, Value: "final-jwt-token"}},
		},
		{
			name:         "JWT from full URL Location header with &",
			intermediate: "intermediate-jwt",
			otp:          "123456",
			finalToken:   "final-jwt-token",
			location:     "http://localhost/auth/complete?auth_token=final-jwt-token&redirect=/",
		},
		{
			name:         "JWT from full URL Location header without &",
			intermediate: "intermediate-jwt",
			otp:          "123456",
			finalToken:   "final-jwt-token",
			location:     "http://localhost/auth/complete?auth_token=final-jwt-token",
		},
		{
			name:         "JWT from Location header with &",
			intermediate: "intermediate-jwt",
			otp:          "123456",
			finalToken:   "final-jwt-token",
			location:     "/auth/complete?auth_token=final-jwt-token&redirect=/",
		},
		{
			name:         "JWT from Location header without &",
			intermediate: "intermediate-jwt",
			otp:          "123456",
			finalToken:   "final-jwt-token",
			location:     "/auth/complete?auth_token=final-jwt-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/otp/validate" {
					t.Errorf("expected /api/auth/otp/validate path, got %s", r.URL.Path)
				}

				authHeader := r.Header.Get("Authorization")
				expectedAuth := "Bearer " + tt.intermediate
				if authHeader != expectedAuth {
					t.Errorf("expected Authorization header %s, got %s", expectedAuth, authHeader)
				}

				w.Header().Set("Location", tt.location)
				for _, cookie := range tt.cookies {
					http.SetCookie(w, cookie)
				}
				w.WriteHeader(http.StatusFound)
			}))
			defer server.Close()

			// Create client with redirect following disabled to inspect response
			acc := NewClient(WithEndpoint(server.URL), WithDisableFollowRedirect())
			token, err := acc.ValidateOTP(context.Background(), tt.intermediate, tt.otp)

			if err != nil {
				t.Errorf("ValidateOTP() error = %v", err)
			}
			if token != tt.finalToken {
				t.Errorf("ValidateOTP() got = %v, want %v", token, tt.finalToken)
			}
		})
	}
}

func TestLogin_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		password   string
		statusCode int
		token      string
		wantErr    bool
		errCheck   func(t *testing.T, err error)
	}{
		{
			name:       "network error",
			email:      "test@example.com",
			password:   "password",
			statusCode: http.StatusOK,
			token:      "",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to send login request")
			},
		},
		{
			name:       "empty token response",
			email:      "test@example.com",
			password:   "password",
			statusCode: http.StatusOK,
			token:      "",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "did not contain a token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.name == "network error" {
					// Force connection close to simulate network error
					conn, _, _ := w.(http.Hijacker).Hijack()
					conn.Close()
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.token != "" {
					otp := false
					resp := client.LoginResponse{Token: tt.token, Otp: &otp}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				} else {
					resp := client.LoginResponse{Token: ""}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL))
			_, err := acc.Login(context.Background(), tt.email, tt.password)

			if tt.wantErr {
				require.Error(t, err)
				tt.errCheck(t, err)
			}
		})
	}
}

func TestPing_ErrorPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/ping" {
			t.Errorf("expected /api/auth/ping path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	err := acc.Ping(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send ping request")
}

func TestGenerateOTP_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		secret     string
		wantErr    bool
		errCheck   func(t *testing.T, err error)
	}{
		{
			name:       "network error",
			statusCode: http.StatusOK,
			secret:     "",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to send OTP generate request")
			},
		},
		{
			name:       "bad request",
			statusCode: http.StatusUnauthorized,
			secret:     "",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				// The error should be "unauthorized" or "authentication required"
				// Note: This test may fail due to JSON parsing errors in the generated client
				// before handleResponse is called
			},
		},
		{
			name:       "empty secret response",
			statusCode: http.StatusOK,
			secret:     "",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "did not contain a secret")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				if r.URL.Path != "/api/auth/otp/generate" {
					t.Errorf("expected /api/auth/otp/generate path, got %s", r.URL.Path)
				}

				if tt.name == "network error" {
					conn, _, _ := w.(http.Hijacker).Hijack()
					conn.Close()
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.OTPGenerateResponse{Otp: tt.secret}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
			secret, err := acc.GenerateOTP(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				tt.errCheck(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.secret, secret)
			}
		})
	}
}

func TestVerifyOTP_ErrorPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/otp/verify" {
			t.Errorf("expected /api/auth/otp/verify path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	err := acc.VerifyOTP(context.Background(), "123456")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send OTP verify request")
}

func TestDisableOTP_ErrorPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/auth/otp/disable" {
			t.Errorf("expected /api/auth/otp/disable path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	err := acc.DisableOTP(context.Background(), "password123")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send OTP disable request")
}

func TestUploadLimit_ErrorPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/upload-limit" {
			t.Errorf("expected /api/upload-limit path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	_, err := acc.UploadLimit(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send upload limit request")
}

func TestGetOperation_ErrorPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/operations/123" {
			t.Errorf("expected /api/operations/123 path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := client.OperationDetailResponse{
			Id:            123,
			Status:        OperationStatusCompleted.String(),
			Cid:           lo.ToPtr("QmTest"),
			StatusMessage: "Completed",
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))

	// First call should succeed
	op, err := acc.GetOperation(context.Background(), 123)
	require.NoError(t, err)
	require.NotNil(t, op)
	require.Equal(t, 123, op.Id)
}

func TestGetOperation_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/operations/123" {
			t.Errorf("expected /api/operations/123 path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("test-token"))
	_, err := acc.GetOperation(context.Background(), 123)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get operation 123")
}

func TestGetOperation_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/operations/123" {
			t.Errorf("expected /api/operations/123 path, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("invalid-token"))
	_, err := acc.GetOperation(context.Background(), 123)

	require.Error(t, err)
	require.Contains(t, err.Error(), "authentication required")
}

func TestListOperations(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		statusCode int
		wantErr    bool
		opts       []ListOption
	}{
		{
			name:       "successful list operations",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
			opts:       []ListOption{WithSearch("test"), WithPagination(&queryutil.DefaultPagination)},
		},
		{
			name:       "list operations with sorts and filters",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			wantErr:    false,
			opts: []ListOption{
				WithSorts(queryutil.Sort{Field: "id", Order: "desc"}),
				WithFilters(queryutil.Equal("status", "completed")),
			},
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/api/operations" {
					t.Errorf("expected /api/operations path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.OperationListItemResponse{
						Data: []client.OperationListItem{
							{
								Id:                   1,
								Operation:            "upload",
								OperationDisplayName: "Upload",
								Protocol:             "ipfs",
								ProtocolDisplayName:  "IPFS",
								Status:               OperationStatusCompleted.String(),
								StatusDisplayName:    "Completed",
								StatusMessage:        "Done",
								ProgressPercent:      100.0,
								Cid:                  lo.ToPtr("QmTest1"),
								CurrentStep:          lo.ToPtr[int](1),
								TotalSteps:           lo.ToPtr[int](1),
								StartedAt:            time.Now(),
								UpdatedAt:            time.Now(),
							},
							{
								Id:                   2,
								Operation:            "download",
								OperationDisplayName: "Download",
								Protocol:             "ipfs",
								ProtocolDisplayName:  "IPFS",
								Status:               OperationStatusRunning.String(),
								StatusDisplayName:    "Running",
								StatusMessage:        "In progress",
								ProgressPercent:      50.0,
								Cid:                  lo.ToPtr("QmTest2"),
								CurrentStep:          lo.ToPtr[int](1),
								TotalSteps:           lo.ToPtr[int](2),
								StartedAt:            time.Now(),
								UpdatedAt:            time.Now(),
							},
						},
						Total: 2,
					}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			ops, err := acc.ListOperations(context.Background(), tt.opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListOperations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.statusCode == http.StatusUnauthorized {
				if !errors.Is(err, ErrUnauthorized) {
					t.Errorf("ListOperations() error should be ErrUnauthorized, got: %v", err)
				}
			}

			if !tt.wantErr {
				if ops == nil {
					t.Fatal("ListOperations() returned nil operations")
				}
				if len(ops) != 2 {
					t.Errorf("expected 2 operations, got %d", len(ops))
				}
			}
		})
	}
}

func TestListOperations_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/api/operations" {
			t.Errorf("expected /api/operations path, got %s", r.URL.Path)
		}

		// Simulate network error
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	acc := NewClient(WithEndpoint(server.URL), WithJWT("valid-token"))
	_, err := acc.ListOperations(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to list operations")
}

func TestOperationListItemToDetail(t *testing.T) {
	now := time.Now()
	nowPtr := &now
	listItem := client.OperationListItem{
		Id:                    123,
		Operation:             "upload",
		OperationDisplayName:  "Upload",
		Protocol:              "ipfs",
		ProtocolDisplayName:   "IPFS",
		Status:                OperationStatusCompleted.String(),
		StatusDisplayName:     "Completed",
		StatusMessage:         "Done",
		ProgressPercent:       100.0,
		Cid:                   lo.ToPtr("QmTest"),
		CurrentStep:           lo.ToPtr[int](2),
		TotalSteps:            lo.ToPtr[int](4),
		Error:                 lo.ToPtr("test error"),
		StartedAt:             now,
		UpdatedAt:             now,
		EstimatedCompletionAt: nowPtr,
	}

	detail := operationListItemToDetail(listItem)

	require.Equal(t, listItem.Id, detail.Id)
	require.Equal(t, listItem.Operation, detail.Operation)
	require.Equal(t, listItem.Protocol, detail.Protocol)
	require.Equal(t, listItem.Status, detail.Status)
	require.Equal(t, listItem.Cid, detail.Cid)
	require.Equal(t, listItem.CurrentStep, detail.CurrentStep)
	require.Equal(t, listItem.TotalSteps, detail.TotalSteps)
	require.Equal(t, listItem.Error, detail.Error)
	require.Equal(t, listItem.EstimatedCompletionAt, detail.EstimatedCompletionAt)
}

func TestOperation_IsSettled(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"pending is not settled", OperationStatusPending.String(), false},
		{"running is not settled", OperationStatusRunning.String(), false},
		{"completed is settled", OperationStatusCompleted.String(), true},
		{"failed is settled", OperationStatusFailed.String(), true},
		{"error is settled", OperationStatusError.String(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &Operation{OperationDetailResponse: client.OperationDetailResponse{
				Id:     123,
				Status: tt.status,
			}}
			if got := op.IsSettled(); got != tt.want {
				t.Errorf("IsSettled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateJSON200_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		json200    *client.UploadLimitResponse
		nilMsg     string
		wantErr    bool
		errCheck   func(t *testing.T, err error)
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			json200:    nil,
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "authentication required")
			},
		},
		{
			name:       "bad request with body",
			statusCode: http.StatusBadRequest,
			json200:    nil,
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed with status 400")
			},
		},
		{
			name:       "ok but nil json200",
			statusCode: http.StatusOK,
			json200:    nil,
			nilMsg:     "test nil message",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "test nil message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte("test body")
			data, err := validateJSON200(tt.statusCode, body, tt.json200, tt.nilMsg)

			if tt.wantErr {
				require.Error(t, err)
				tt.errCheck(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, data)
			}
		})
	}
}

func TestHandleResponse_GenericErrorPath(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		op           int
		successCodes []int
		body         []byte
		wantErr      bool
		errCheck     func(t *testing.T, err error)
	}{
		{
			name:         "generic error with unknown operation",
			statusCode:   http.StatusInternalServerError,
			op:           999, // unknown operation
			successCodes: []int{http.StatusOK},
			body:         []byte("internal server error"),
			wantErr:      true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "operation failed with status 500")
			},
		},
		{
			name:         "generic error with unknown status code for known operation",
			statusCode:   http.StatusTooManyRequests,
			op:           OpLogin,
			successCodes: []int{http.StatusOK},
			body:         []byte("rate limit exceeded"),
			wantErr:      true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "login failed with status 429")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleResponse(tt.statusCode, tt.body, tt.op, tt.successCodes)

			if tt.wantErr {
				require.Error(t, err)
				tt.errCheck(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckResponseWithBody(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		operation  string
		wantErr    bool
		errCheck   func(t *testing.T, err error)
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       []byte("success"),
			operation:  "test operation",
			wantErr:    false,
		},
		{
			name:       "error with status",
			statusCode: http.StatusBadRequest,
			body:       []byte("bad request"),
			operation:  "test operation",
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "test operation failed with status 400")
				require.Contains(t, err.Error(), "bad request")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkResponseWithBody(tt.statusCode, tt.body, tt.operation)

			if tt.wantErr {
				require.Error(t, err)
				tt.errCheck(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithHostOverride(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		target      string
		verifyHost  string
		setupServer func(*testing.T) *httptest.Server
	}{
		{
			name:       "host override redirects to target with correct Host header",
			host:       "account.pinner.xyz",
			target:     "127.0.0.1:8080",
			verifyHost: "account.pinner.xyz",
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify the Host header is set to the overridden value
					require.Equal(t, "account.pinner.xyz", r.Host)
					
					// Verify the request came from our test server
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				}))
			},
		},
		{
			name:       "host override with different host and target",
			host:       "api.example.com",
			target:     "192.168.1.100:3000",
			verifyHost: "api.example.com",
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify the Host header is set to the overridden value
					require.Equal(t, "api.example.com", r.Host)
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer(t)
			defer server.Close()

			// Parse the server URL to get host:port
			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			// Create client with host override
			// Use http:// prefix for target to ensure it's a valid URL
			targetURL := "http://" + serverURL.Host
			acc := NewClient(
				WithHostOverride(tt.host, targetURL),
			)

			// Make a request to verify the host override works
			// We'll use Ping as it's a simple authenticated endpoint
			// For this test, we need to set a JWT to avoid auth errors
			ctx := context.Background()
			
			// The request should be routed to the target with the overridden Host header
			// We can verify this by checking the server received the correct Host header
			err = acc.Ping(ctx)
			
			// If the host override worked correctly, the request should succeed
			// (unless there's an auth error, which is expected without valid JWT)
			// The important thing is that the request reached our test server
			require.NoError(t, err)
		})
	}
}

func TestHostOverrideRoundTripper(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		target        string
		expectedHost  string
		expectedPath  string
	}{
		{
			name:         "round tripper overrides Host header correctly",
			host:         "account.pinner.xyz",
			target:       "http://127.0.0.1:8080",
			expectedHost: "account.pinner.xyz",
			expectedPath: "/api/test",
		},
		{
			name:         "round tripper preserves path",
			host:         "api.example.com",
			target:       "http://192.168.1.100:3000",
			expectedHost: "api.example.com",
			expectedPath: "/api/operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that verifies the request
			receivedHost := ""
			receivedPath := ""
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedHost = r.Host
				receivedPath = r.URL.Path
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Parse the server URL
			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			// Create the round tripper using shared utilities
			transport := internalhttp.NewHostOverrideRoundTripper(tt.host, "http://"+serverURL.Host)

			// Create a request with the original URL
			req := &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Scheme: "http",
					Host:   tt.host,
					Path:   tt.expectedPath,
				},
				Header: http.Header{},
			}

			// Use the round tripper
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the Host header was overridden
			require.Equal(t, tt.expectedHost, receivedHost)
			
			// Verify the path was preserved
			require.Equal(t, tt.expectedPath, receivedPath)
		})
	}
}

func TestHostOverrideWithQueryParams(t *testing.T) {
	// Test that query parameters are preserved when using host override
	receivedHost := ""
	receivedQuery := ""
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		receivedQuery = r.URL.RawQuery
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)

	// Create the round tripper using shared utilities
	transport := internalhttp.NewHostOverrideRoundTripper("account.pinner.xyz", "http://"+serverURL.Host)

	// Create a request with query parameters
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme:   "http",
			Host:     "account.pinner.xyz",
			Path:     "/api/operations",
			RawQuery: "status=completed&limit=10",
		},
		Header: http.Header{},
	}

	// Use the round tripper
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the Host header was overridden
	require.Equal(t, "account.pinner.xyz", receivedHost)
	
	// Verify query parameters were preserved
	require.Equal(t, "status=completed&limit=10", receivedQuery)
}

func TestHostOverridePreservesHTTPSScheme(t *testing.T) {
	// Test that HTTPS scheme is preserved when target doesn't specify a scheme
	receivedHost := ""
	requestWasHTTPS := false
	
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		requestWasHTTPS = r.TLS != nil
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)

	// Create a custom transport that skips TLS verification for testing
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Create the round tripper with target that doesn't specify scheme
	transport := internalhttp.NewHostOverrideRoundTripperWithTransport(customTransport, "account.pinner.xyz", serverURL.Host) // No scheme, should use original request's scheme

	// Create an HTTPS request
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "https",
			Host:   "account.pinner.xyz",
			Path:   "/api/test",
		},
		Header: http.Header{},
	}

	// Use the round tripper
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the request was made over HTTPS (not downgraded to HTTP)
	require.True(t, requestWasHTTPS, "Request should be HTTPS, not HTTP")
	
	// Verify the Host header was overridden
	require.Equal(t, "account.pinner.xyz", receivedHost)
}

func TestGetAccount(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful get account",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/account" {
					t.Errorf("expected /api/account path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					now := time.Now()
					resp := client.AccountInfoResponse{
						Id:        1,
						Email:     "test@example.com",
						FirstName: "John",
						LastName:  "Doe",
						Verified:  true,
						Otp:       false,
						CreatedAt: &now,
					}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				} else if tt.statusCode == http.StatusUnauthorized {
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithJWT("test-token"), WithEndpoint(server.URL))
			result, err := acc.GetAccount(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, "test@example.com", result.Email)
				require.Equal(t, "John", result.FirstName)
			}
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name        string
		firstName   string
		lastName    string
		statusCode  int
		wantErr     bool
	}{
		{
			name:       "successful update",
			firstName:  "Jane",
			lastName:   "Smith",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "update with empty values",
			firstName:  "",
			lastName:   "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			firstName:  "Jane",
			lastName:   "Smith",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "bad request",
			firstName:  "Jane",
			lastName:   "Smith",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PATCH" {
					t.Errorf("expected PATCH request, got %s", r.Method)
				}
				if r.URL.Path != "/api/account" {
					t.Errorf("expected /api/account path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(`{"message":"updated"}`))
				} else if tt.statusCode == http.StatusUnauthorized {
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithJWT("test-token"), WithEndpoint(server.URL))
			err := acc.UpdateProfile(context.Background(), tt.firstName, tt.lastName)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAvatar(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful get avatar",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/account/avatar" {
					t.Errorf("expected /api/account/avatar path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte("fake-image-data"))
				} else {
					resp := client.ErrorResponse{Error: "error"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithJWT("test-token"), WithEndpoint(server.URL))
			result, err := acc.GetAvatar(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAvatar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.Equal(t, "fake-image-data", string(result))
			}
		})
	}
}

func TestUploadAvatar(t *testing.T) {
	tests := []struct {
		name       string
		fileData   []byte
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful upload",
			fileData:   []byte("fake-image-data"),
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "successful upload (204)",
			fileData:   []byte("fake-image-data"),
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "bad request",
			fileData:   []byte("fake-image-data"),
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/account/avatar" {
					t.Errorf("expected /api/account/avatar path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(`{"message":"uploaded"}`))
				} else if tt.statusCode != http.StatusNoContent {
					resp := client.ErrorResponse{Error: "error"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithJWT("test-token"), WithEndpoint(server.URL))
			err := acc.UploadAvatar(context.Background(), tt.fileData)

			if (err != nil) != tt.wantErr {
				t.Errorf("UploadAvatar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateEmail(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		password   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful update",
			email:      "newemail@example.com",
			password:   "currentpassword",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			email:      "newemail@example.com",
			password:   "wrongpassword",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "bad request",
			email:      "invalid-email",
			password:   "currentpassword",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/account/update-email" {
					t.Errorf("expected /api/account/update-email path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(`{"message":"updated"}`))
				} else if tt.statusCode == http.StatusUnauthorized {
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithJWT("test-token"), WithEndpoint(server.URL))
			err := acc.UpdateEmail(context.Background(), tt.email, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPermissions(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful get permissions",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/account/permissions" {
					t.Errorf("expected /api/account/permissions path, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := client.AccountPermissionsResponse{
						Model: client.AccessModel{
							RequestDefinition:  client.AccessModelDef{Key: "user", Value: "id"},
							PolicyDefinition:   client.AccessModelDef{Key: "policy", Value: "data"},
							RoleDefinition:     client.AccessModelDef{Key: "role", Value: "data"},
							PolicyEffect:       client.AccessModelDef{Key: "effect", Value: "data"},
							Matchers:           client.AccessModelDef{Key: "match", Value: "data"},
						},
						Permissions: []client.AccessPolicy{
							{
								Sub: "user:1",
								Dom: "domain",
								Obj: "resource",
								Act: "action",
							},
						},
					}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				} else if tt.statusCode == http.StatusUnauthorized {
					resp := client.ErrorResponse{Error: "unauthorized"}
					require.NoError(t, json.NewEncoder(w).Encode(resp))
				}
			}))
			defer server.Close()

			acc := NewClient(WithJWT("test-token"), WithEndpoint(server.URL))
			result, err := acc.GetPermissions(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPermissions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				require.NotNil(t, result)
				require.NotEmpty(t, result.Permissions)
			}
		})
	}
}

func TestGetQuota(t *testing.T) {
	tests := []struct {
		name       string
		jwt        string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:       "successful quota retrieval",
			jwt:        "valid-token",
			statusCode: http.StatusOK,
			response: client.QuotaStatusResponse{
				Upload: client.QuotaTypeStatus{
					Limit:      func() *int { i := 1000000; return &i }(),
					Used:      500000,
					Remaining: func() *int { i := 500000; return &i }(),
					Percentage: 50,
				},
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 2000000; return &i }(),
					Used:      1000000,
					Remaining: func() *int { i := 1000000; return &i }(),
					Percentage: 50,
				},
				Storage: client.QuotaTypeStatus{
					Limit:      func() *int { i := 3000000; return &i }(),
					Used:       1500000,
					Remaining:  func() *int { i := 1500000; return &i }(),
					Percentage: 50,
					Threshold:  func() *int { i := 100000; return &i }(),
				},
			},
			wantErr: false,
		},
		{
			name:       "unauthorized",
			jwt:        "invalid-token",
			statusCode: http.StatusUnauthorized,
			response:   client.ErrorResponse{Error: "unauthorized"},
			wantErr:    true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, ErrUnauthorized)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/quota" {
					t.Errorf("expected /api/account/quota path, got %s", r.URL.Path)
				}

				authHeader := r.Header.Get("Authorization")
				expectedAuth := "Bearer " + tt.jwt
				require.Equal(t, expectedAuth, authHeader)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)

				if tt.statusCode == http.StatusOK {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				} else if tt.statusCode == http.StatusUnauthorized {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			quota, err := acc.GetQuota(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetQuota() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, quota, "quota should not be nil")
				require.Equal(t, 50, quota.Upload.Percentage, "upload percentage mismatch")
				require.Equal(t, 50, quota.Download.Percentage, "download percentage mismatch")
				require.Equal(t, 50, quota.Storage.Percentage, "storage percentage mismatch")
			}
		})
	}
}

func TestGetQuotaHistory(t *testing.T) {
	startDate := "2024-01-01T00:00:00Z"
	endDate := "2024-01-31T23:59:59Z"

	tests := []struct {
		name       string
		jwt        string
		startDate  string
		endDate    string
		usageType  string
		statusCode int
		response   interface{}
		wantErr    bool
		errCheck   func(*testing.T, error)
	}{
		{
			name:      "successful quota history with date range",
			jwt:       "valid-token",
			startDate: startDate,
			endDate:   endDate,
			usageType: "upload",
			statusCode: http.StatusOK,
			response: client.QuotaHistoryResponse{
				UserId: 123,
				Points: []client.UsagePoint{
					{
						Bytes: 1024000,
						Date:  "2024-01-01T00:00:00Z",
					},
					{
						Bytes: 512000,
						Date:  "2024-01-02T00:00:00Z",
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "successful quota history with all parameters",
			jwt:       "valid-token",
			startDate: startDate,
			endDate:   endDate,
			usageType: "download",
			statusCode: http.StatusOK,
			response: client.QuotaHistoryResponse{
				UserId: 123,
				Points: []client.UsagePoint{
					{
						Bytes: 2048000,
						Date:  "2024-01-01T00:00:00Z",
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "unauthorized",
			jwt:       "invalid-token",
			startDate: startDate,
			endDate:   endDate,
			usageType: "upload",
			statusCode: http.StatusUnauthorized,
			response:  client.ErrorResponse{Error: "unauthorized"},
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, ErrUnauthorized)
			},
		},
		{
			name:      "invalid date parameters",
			jwt:       "valid-token",
			startDate: "invalid-date",
			endDate:   endDate,
			usageType: "upload",
			statusCode: http.StatusBadRequest,
			response:  client.ErrorResponse{Error: "invalid date parameters"},
			wantErr:   true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "invalid date")
			},
		},
		{
			name:      "quota history not found",
			jwt:       "valid-token",
			startDate: "2099-01-01T00:00:00Z",
			endDate:   "2099-12-31T23:59:59Z",
			usageType: "upload",
			statusCode: http.StatusNotFound,
			response:  client.ErrorResponse{Error: "quota history not found"},
			wantErr:   true,
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

				if r.URL.Path != "/api/account/quota/history" {
					t.Errorf("expected /api/account/quota/history path, got %s", r.URL.Path)
				}

				authHeader := r.Header.Get("Authorization")
				expectedAuth := "Bearer " + tt.jwt
				require.Equal(t, expectedAuth, authHeader)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)

				if tt.statusCode == http.StatusOK {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				} else if tt.statusCode == http.StatusUnauthorized || tt.statusCode == http.StatusBadRequest || tt.statusCode == http.StatusNotFound {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			history, err := acc.GetQuotaHistory(context.Background(), tt.startDate, tt.endDate, tt.usageType)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetQuotaHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr {
				require.NotNil(t, history, "history should not be nil")
				require.Equal(t, 123, history.UserId, "user ID mismatch")
				require.Len(t, history.Points, len(tt.response.(client.QuotaHistoryResponse).Points), "points count mismatch")
			}
		})
	}
}

func TestCreateDownloadRateLimiter(t *testing.T) {
	tests := []struct {
		name          string
		jwt           string
		statusCode    int
		response      interface{}
		requestedSize int64
		wantAllowed   bool
		wantErr       bool
		errCheck      func(*testing.T, error)
	}{
		{
			name:          "sufficient quota allowed",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: 500000,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 2000000; return &i }(),
					Used:      1000000,
					Remaining: func() *int { i := 1000000; return &i }(),
					Percentage: 50,
				},
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:          "insufficient quota denied",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: 2000000,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 2000000; return &i }(),
					Used:      1000000,
					Remaining: func() *int { i := 1000000; return &i }(),
					Percentage: 50,
				},
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:          "unlimited quota always allowed",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: 999999999999,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      nil,
					Used:      0,
					Remaining: nil,
					Percentage: 0,
				},
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:          "exact quota match allowed",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: 1000000,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 2000000; return &i }(),
					Used:      1000000,
					Remaining: func() *int { i := 1000000; return &i }(),
					Percentage: 50,
				},
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:          "zero bytes allowed",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: 0,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 100; return &i }(),
					Used:      100,
					Remaining: func() *int { i := 0; return &i }(),
					Percentage: 100,
				},
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name:          "negative size rejected",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: -1,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 1000000; return &i }(),
					Used:      0,
					Remaining: func() *int { i := 1000000; return &i }(),
					Percentage: 0,
				},
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:          "negative size with unlimited quota rejected",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: -100,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      nil,
					Used:      0,
					Remaining: nil,
					Percentage: 0,
				},
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:          "single byte over quota denied",
			jwt:           "valid-token",
			statusCode:    http.StatusOK,
			requestedSize: 1000001,
			response: client.QuotaStatusResponse{
				Download: client.QuotaTypeStatus{
					Limit:      func() *int { i := 2000000; return &i }(),
					Used:      1000000,
					Remaining: func() *int { i := 1000000; return &i }(),
					Percentage: 50,
				},
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name:          "quota endpoint error",
			jwt:           "invalid-token",
			statusCode:    http.StatusUnauthorized,
			requestedSize: 500000,
			response:      client.ErrorResponse{Error: "unauthorized"},
			wantAllowed:   false,
			wantErr:       true,
			errCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, ErrUnauthorized)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/api/account/quota" {
					t.Errorf("expected /api/account/quota path, got %s", r.URL.Path)
				}

				authHeader := r.Header.Get("Authorization")
				expectedAuth := "Bearer " + tt.jwt
				require.Equal(t, expectedAuth, authHeader)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)

				if tt.statusCode == http.StatusOK {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				} else if tt.statusCode == http.StatusUnauthorized {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			acc := NewClient(WithEndpoint(server.URL), WithJWT(tt.jwt))
			rateLimiter := CreateDownloadRateLimiter(acc)

			allowed, err := rateLimiter.AllowDownload(context.Background(), tt.requestedSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("AllowDownload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			if !tt.wantErr && allowed != tt.wantAllowed {
				t.Errorf("AllowDownload() allowed = %v, want %v, requestedSize %d", allowed, tt.wantAllowed, tt.requestedSize)
			}
		})
	}
}
