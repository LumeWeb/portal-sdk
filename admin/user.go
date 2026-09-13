package admin

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strconv"

	"github.com/samber/lo"
	"go.lumeweb.com/portal-sdk/internal/admin"
	internalhttp "go.lumeweb.com/portal-sdk/internal/http"
)

// User error sentinels
var (
	errAdminUserNotFound    = internalhttp.NotFoundError("user not found")
	errAdminInvalidUserData = internalhttp.BadRequestError("invalid user data")
	errInvalidUserRequest   = internalhttp.BadRequestError("user request is required")
)

const (
	// User operation identifiers for error message mapping
	OpUserList = 700 + iota
	OpUserCreate
	OpUserGet
	OpUserUpdate
	OpUserDelete
)

const defaultUserOperationName = "user operation"

// ErrUserDefault is a generic user error type.
var ErrUserDefault = errors.New("user operation failed")

// userOperationString maps user operation IDs to their string names.
var userOperationString = map[int]string{
	OpUserList:   "list users",
	OpUserCreate: "create user",
	OpUserGet:    "get user",
	OpUserUpdate: "update user",
	OpUserDelete: "delete user",
}

// userHTTPErrorMessages maps user operation IDs to their custom status code error messages.
var userHTTPErrorMessages = map[int]map[int]internalhttp.ErrorFactoryError{
	OpUserList: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
	},
	OpUserCreate: {
		stdhttp.StatusUnauthorized:        errAuthRequired,
		stdhttp.StatusForbidden:           errInsufficientPermissions,
		stdhttp.StatusBadRequest:          errAdminInvalidUserData,
		stdhttp.StatusUnprocessableEntity: internalhttp.BadRequestError("user validation failed"),
	},
	OpUserGet: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errAdminUserNotFound,
	},
	OpUserUpdate: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusBadRequest:   errAdminInvalidUserData,
		stdhttp.StatusNotFound:     errAdminUserNotFound,
	},
	OpUserDelete: {
		stdhttp.StatusUnauthorized: errAuthRequired,
		stdhttp.StatusForbidden:    errInsufficientPermissions,
		stdhttp.StatusNotFound:     errAdminUserNotFound,
	},
}

// userOpHandler is the shared operation handler for user operations (lazily initialized).
var userOpHandler = initUserOpHandler()

// initUserOpHandler initializes the OpHandler with user operation mappings.
func initUserOpHandler() *internalhttp.OpHandler {
	oh := internalhttp.NewOpHandler()
	oh.Default = defaultUserOperationName

	for opID, name := range userOperationString {
		oh.SetName(opID, name)
	}
	for opID, errorMap := range userHTTPErrorMessages {
		oh.AddOperation(opID, errorMap)
	}
	return oh
}

// handleUserResponse wraps OpHandler.HandleResponse.
func handleUserResponse(statusCode int, body []byte, op int, successCodes []int) error {
	return userOpHandler.HandleResponse(statusCode, body, op, successCodes)
}

// validateUserJSON200 wraps OpHandler.ValidateJSON200.
func validateUserJSON200[T any](respStatusCode int, json200 *T, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := fmt.Appendf(nil, "expected status 200, got %d", respStatusCode)
	return internalhttp.ValidateJSON200(userOpHandler, respStatusCode, body, json200, op)
}

// validateUserJSON201 wraps OpHandler.ValidateJSON201.
func validateUserJSON201[T any](respStatusCode int, json201 *T, nilMsg string, op int) (*T, error) {
	if respStatusCode == stdhttp.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", internalhttp.ErrUnauthorized)
	}
	body := []byte(nilMsg)
	return internalhttp.ValidateJSON201(userOpHandler, respStatusCode, body, json201, op)
}

// Type aliases for generated admin client types.
type (
	UserCreateRequest = admin.UserCreateRequest
	UserUpdateRequest = admin.UserUpdateRequest
	UserListParams    = admin.GetApiUsersParams
)

// User represents a portal account managed through the admin API. Embeds the
// generated admin.UserResponse. Authentication- and database-sensitive fields
// (password, verification tokens) are never returned by the API.
type User struct {
	admin.UserResponse
}

// UserService provides methods for managing portal users via the admin API.
type UserService struct {
	client admin.ClientWithResponsesInterface
}

// SetRequestExecutor sets the underlying admin client for the user service.
// Used for testing with mock clients.
func (u *UserService) SetRequestExecutor(client admin.ClientWithResponsesInterface) {
	u.client = client
}

// ListUsers returns a filterable, sortable, paginated list of portal users.
// Use params.Pagination fields (_start/_end), UnderscoreSort/UnderscoreOrder
// for sorting, and Email/Verified for exact-match filtering.
func (u *UserService) ListUsers(ctx context.Context, params *UserListParams) ([]*User, int, error) {
	resp, err := u.client.GetApiUsersWithResponse(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	data, err := validateUserJSON200(resp.StatusCode(), resp.JSON200, OpUserList)
	if err != nil {
		return nil, 0, err
	}

	users := lo.Map(data.Data, func(usr admin.UserResponse, _ int) *User {
		return &User{UserResponse: usr}
	})

	return users, data.Total, nil
}

// CreateUser creates a portal account with an email and password, plus optional
// names and verification-email behavior. The password is hashed server-side
// and never returned.
func (u *UserService) CreateUser(ctx context.Context, req *UserCreateRequest) (*User, error) {
	if req == nil {
		return nil, errInvalidUserRequest.Error()
	}

	resp, err := u.client.PostApiUsersWithResponse(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	data, err := validateUserJSON201(resp.StatusCode(), resp.JSON201, "create user response did not contain data", OpUserCreate)
	if err != nil {
		return nil, err
	}

	return &User{UserResponse: *data}, nil
}

// GetUser returns a single portal user by ID.
func (u *UserService) GetUser(ctx context.Context, id int) (*User, error) {
	resp, err := u.client.GetApiUsersIdWithResponse(ctx, strconv.Itoa(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	data, err := validateUserJSON200(resp.StatusCode(), resp.JSON200, OpUserGet)
	if err != nil {
		return nil, err
	}

	return &User{UserResponse: *data}, nil
}

// UpdateUser patches an existing portal user. Omitted fields are left
// unchanged.
func (u *UserService) UpdateUser(ctx context.Context, id int, req *UserUpdateRequest) (*User, error) {
	if req == nil {
		return nil, errInvalidUserRequest.Error()
	}

	resp, err := u.client.PatchApiUsersIdWithResponse(ctx, strconv.Itoa(id), *req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	data, err := validateUserJSON200(resp.StatusCode(), resp.JSON200, OpUserUpdate)
	if err != nil {
		return nil, err
	}

	return &User{UserResponse: *data}, nil
}

// DeleteUser removes a portal user account by ID.
func (u *UserService) DeleteUser(ctx context.Context, id int) error {
	resp, err := u.client.DeleteApiUsersIdWithResponse(ctx, strconv.Itoa(id))
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return handleUserResponse(resp.StatusCode(), resp.Body, OpUserDelete, []int{stdhttp.StatusNoContent})
}
