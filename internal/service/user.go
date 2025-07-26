package service

import (
	"context"
	"review/internal/biz"
	"time"

	pb "review/api/user/v1"
)

// UserService is a user service.
type UserService struct {
	pb.UnimplementedUserServer
	uc *biz.UserUsecase
}

// NewUserService new a user service.
func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

// Register implements api.user.v1.UserServer.
func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	user := &biz.User{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Role:     req.Role,
	}
	err := s.uc.Register(ctx, user)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterReply{Success: true, Message: "Registration successful"}, nil
}

// Login implements api.user.v1.UserServer.
func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
	token, err := s.uc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &pb.LoginReply{Token: token, Message: "Login successful"}, nil
}

// GetUserInfo implements api.user.v1.UserServer.
func (s *UserService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoReply, error) {
	user, err := s.uc.GetUserInfo(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	userInfo := &pb.UserInfo{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
	return &pb.GetUserInfoReply{UserInfo: userInfo}, nil
}

// UpdateUserInfo implements api.user.v1.UserServer.
func (s *UserService) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoRequest) (*pb.UpdateUserInfoReply, error) {
	// Note: In a real-world application, you'd get the UserID from the JWT token to prevent users from updating others' info.
	// Here we trust the request for simplicity.
	user := &biz.User{
		ID:       req.UserID,
		Username: req.Username,
		Email:    req.Email,
	}
	err := s.uc.UpdateUserInfo(ctx, user)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserInfoReply{Success: true, Message: "User info updated successfully"}, nil
}

// DeleteUser implements api.user.v1.UserServer.
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserReply, error) {
	err := s.uc.DeleteUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteUserReply{Success: true, Message: "User deleted successfully"}, nil
}

// GetUserList implements api.user.v1.UserServer.
func (s *UserService) GetUserList(ctx context.Context, req *pb.GetUserListRequest) (*pb.GetUserListReply, error) {
	users, total, err := s.uc.GetUserList(ctx, req.Offset, req.Limit)
	if err != nil {
		return nil, err
	}

	pbUsers := make([]*pb.UserInfo, len(users))
	for i, user := range users {
		pbUsers[i] = &pb.UserInfo{
			UserID:    user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &pb.GetUserListReply{Users: pbUsers, Total: total}, nil
}
