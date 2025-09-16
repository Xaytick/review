package biz

import (
	"context"
	"fmt"
	"strconv"

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

	// Robustly parse user_id: prefer string to avoid precision loss
	var userID int64
	if v, ok := mapClaims["user_id"]; ok {
		switch t := v.(type) {
		case string:
			id, err := strconv.ParseInt(t, 10, 64)
			if err == nil {
				userID = id
			}
		case float64:
			userID = int64(t)
		default:
			// best-effort stringify
			if s, ok := v.(fmt.Stringer); ok {
				if id, err := strconv.ParseInt(s.String(), 10, 64); err == nil {
					userID = id
				}
			}
		}
	}
	if userID == 0 {
		return nil, ErrUserNotFound
	}

	role, ok := mapClaims["role"].(string)
	if !ok {
		return nil, ErrRoleInvalid
	}

	user := &authedUser{
		UserID: userID,
		Role:   role,
	}

	// StoreID is optional, only for merchants. Prefer string.
	if v, ok := mapClaims["store_id"]; ok {
		switch t := v.(type) {
		case string:
			if id, err := strconv.ParseInt(t, 10, 64); err == nil {
				user.StoreID = id
			}
		case float64:
			user.StoreID = int64(t)
		}
	}

	return user, nil
}
