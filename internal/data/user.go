package data

import (
	"context"
	"errors"
	"review/internal/biz"
	"review/internal/data/model"
	"review/internal/data/query"
	"review/pkg/snowflake"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) Register(ctx context.Context, u *biz.User) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		r.log.WithContext(ctx).Errorf("failed to hash password: %v", err)
		return err
	}

	return r.data.q.Transaction(func(tx *query.Query) error {
		// 1. Create user with snowflake ID
		dbUser := &model.User{
			ID:           snowflake.GenID(),
			Username:     u.Username,
			PasswordHash: string(hashed),
			Role:         u.Role,
			Email:        u.Email,
		}
		if err := tx.User.WithContext(ctx).Create(dbUser); err != nil {
			return err
		}

		// 2. If user is a merchant, create a store for them
		if u.Role == "merchant" {
			store := &model.Store{
				StoreID: snowflake.GenID(),
				UserID:  dbUser.ID,
				Name:    u.Username + "'s Store", // Default store name
			}
			if err := tx.Store.WithContext(ctx).Create(store); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *userRepo) Login(ctx context.Context, username, password string) (string, error) {
	dbUser, err := r.data.q.WithContext(ctx).User.Where(r.data.q.User.Username.Eq(username)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("user not found")
		}
		r.log.WithContext(ctx).Errorf("failed to find user: %v", err)
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid password")
	}

	claims := jwt.MapClaims{
		"user_id":  dbUser.ID,
		"username": dbUser.Username,
		"role":     dbUser.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	// If the user is a merchant, find their store_id and add it to the claims
	if dbUser.Role == "merchant" {
		store, err := r.data.q.WithContext(ctx).Store.Where(r.data.q.Store.UserID.Eq(dbUser.ID)).First()
		if err != nil {
			// If store not found, it's an inconsistent data state, but we can choose to proceed without store_id
			r.log.WithContext(ctx).Warnf("could not find store for merchant user_id: %d, error: %v", dbUser.ID, err)
		} else {
			claims["store_id"] = store.StoreID
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// In a real application, the secret key should be loaded from config.
	signedToken, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		r.log.WithContext(ctx).Errorf("failed to sign token: %v", err)
		return "", err
	}
	return signedToken, nil
}

func (r *userRepo) GetUserInfo(ctx context.Context, id int64) (*biz.User, error) {
	dbUser, err := r.data.q.WithContext(ctx).User.Where(r.data.q.User.ID.Eq(id)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &biz.User{
		ID:        int64(dbUser.ID),
		Username:  dbUser.Username,
		Email:     dbUser.Email,
		Role:      dbUser.Role,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}, nil
}

func (r *userRepo) UpdateUserInfo(ctx context.Context, u *biz.User) error {
	result, err := r.data.q.WithContext(ctx).User.Where(r.data.q.User.ID.Eq(u.ID)).Updates(
		&model.User{
			Username: u.Username,
			Email:    u.Email,
		},
	)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found or no changes made")
	}
	return nil
}

func (r *userRepo) DeleteUser(ctx context.Context, id int64) error {
	result, err := r.data.q.WithContext(ctx).User.Where(r.data.q.User.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *userRepo) GetUserList(ctx context.Context, offset, limit int32) ([]*biz.User, int64, error) {
	dbUsers, err := r.data.q.WithContext(ctx).User.Offset(int(offset)).Limit(int(limit)).Find()
	if err != nil {
		return nil, 0, err
	}

	total, err := r.data.q.WithContext(ctx).User.Count()
	if err != nil {
		return nil, 0, err
	}

	bizUsers := make([]*biz.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		bizUsers[i] = &biz.User{
			ID:        int64(dbUser.ID),
			Username:  dbUser.Username,
			Email:     dbUser.Email,
			Role:      dbUser.Role,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
		}
	}

	return bizUsers, total, nil
}
