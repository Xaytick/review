package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// User is a User model.
type User struct {
	ID        int64
	Username  string
	Password  string // Password is used for registration, not for storage.
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepo is a user repo.
type UserRepo interface {
	Register(ctx context.Context, u *User) error
	Login(ctx context.Context, username, password string) (string, error)
	GetUserInfo(ctx context.Context, id int64) (*User, error)
	UpdateUserInfo(ctx context.Context, u *User) error
	DeleteUser(ctx context.Context, id int64) error
	GetUserList(ctx context.Context, offset, limit int32) ([]*User, int64, error)
}

// UserUsecase is a User usecase.
type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

// NewUserUsecase new a User usecase.
func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// Register creates a User, and returns the new User.
func (uc *UserUsecase) Register(ctx context.Context, u *User) error {
	uc.log.WithContext(ctx).Debugf("Register: username=%s, email=%s, role=%s", u.Username, u.Email, u.Role)

	return uc.repo.Register(ctx, u)
}

// Login verifies user credentials and returns a token.
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (string, error) {
	uc.log.WithContext(ctx).Debugf("Login: username=%s", username)

	return uc.repo.Login(ctx, username, password)
}

// GetUserInfo gets a user's information.
func (uc *UserUsecase) GetUserInfo(ctx context.Context, id int64) (*User, error) {
	uc.log.WithContext(ctx).Debugf("GetUserInfo: id=%d", id)

	return uc.repo.GetUserInfo(ctx, id)
}

// UpdateUserInfo updates a user's information.
func (uc *UserUsecase) UpdateUserInfo(ctx context.Context, u *User) error {
	uc.log.WithContext(ctx).Debugf("UpdateUserInfo: id=%d", u.ID)
	
	return uc.repo.UpdateUserInfo(ctx, u)
}

// DeleteUser deletes a user.
func (uc *UserUsecase) DeleteUser(ctx context.Context, id int64) error {
	uc.log.WithContext(ctx).Debugf("DeleteUser: id=%d", id)
	
	return uc.repo.DeleteUser(ctx, id)
}

// GetUserList gets a list of users.
func (uc *UserUsecase) GetUserList(ctx context.Context, offset, limit int32) ([]*User, int64, error) {
	uc.log.WithContext(ctx).Debugf("GetUserList: offset=%d, limit=%d", offset, limit)

	return uc.repo.GetUserList(ctx, offset, limit)
}
