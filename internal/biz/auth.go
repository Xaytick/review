package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingJwtToken = errors.Unauthorized("UNAUTHORIZED", "JWT token is missing")
	ErrUserNotFound    = errors.Unauthorized("UNAUTHORIZED", "User not found in token")
	ErrRoleInvalid     = errors.Unauthorized("UNAUTHORIZED", "Role is invalid")
)

type authedUser struct {
	UserID  int64
	Role    string
	StoreID int64
}

// Helper to get user info from context
func userFromContext(ctx context.Context) (*authedUser, error) {
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, ErrMissingJwtToken
	}
	mapClaims, ok := claims.(jwtv5.MapClaims)
	if !ok {
		return nil, ErrUserNotFound
	}

	userIDFloat, ok := mapClaims["user_id"].(float64)
	if !ok {
		return nil, ErrUserNotFound
	}

	role, ok := mapClaims["role"].(string)
	if !ok {
		return nil, ErrRoleInvalid
	}

	user := &authedUser{
		UserID: int64(userIDFloat),
		Role:   role,
	}

	// StoreID is optional, only for merchants
	if storeIDFloat, ok := mapClaims["store_id"].(float64); ok {
		user.StoreID = int64(storeIDFloat)
	}

	return user, nil
}
