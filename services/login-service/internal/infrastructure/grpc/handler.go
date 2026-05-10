package grpc

import (
	"context"
	"ride-sharing/services/login-service/internal/service"
	pb "ride-sharing/shared/proto/login"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginGRPCHandler struct {
	pb.UnimplementedLoginServiceServer
	authService *service.AuthService
}

func NewGRPCHandler(grpcServer *grpc.Server, authService *service.AuthService) {
	handler := &LoginGRPCHandler{
		authService: authService,
	}
	pb.RegisterLoginServiceServer(grpcServer, handler)
}

func (h *LoginGRPCHandler) Signup(ctx context.Context, req *pb.SignupRequest) (*pb.SignupResponse, error) {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password, and name are required")
	}

	if len(req.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	user, accessToken, refreshToken, err := h.authService.Signup(ctx, req.Email, req.Password, req.Name, req.PhoneNumber)
	if err != nil {
		switch err {
		case service.ErrEmailAlreadyExists:
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		default:
			return nil, status.Error(codes.Internal, "failed to create account")
		}
	}

	return &pb.SignupResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &pb.User{
			Id:          user.ID,
			Email:       user.Email,
			Name:        user.Name,
			PhoneNumber: user.PhoneNumber,
		},
	}, nil
}

func (h *LoginGRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	user, accessToken, refreshToken, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		default:
			return nil, status.Error(codes.Internal, "failed to login")
		}
	}

	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &pb.User{
			Id:          user.ID,
			Email:       user.Email,
			Name:        user.Name,
			PhoneNumber: user.PhoneNumber,
		},
	}, nil
}

func (h *LoginGRPCHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	accessToken, refreshToken, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
		default:
			return nil, status.Error(codes.Internal, "failed to refresh token")
		}
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *LoginGRPCHandler) GoogleAuth(ctx context.Context, req *pb.GoogleAuthRequest) (*pb.GoogleAuthResponse, error) {
	if req.IdToken == "" {
		return nil, status.Error(codes.InvalidArgument, "id token is required")
	}

	user, accessToken, refreshToken, err := h.authService.GoogleAuth(ctx, req.IdToken)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			return nil, status.Error(codes.Unauthenticated, "invalid google token")
		default:
			return nil, status.Error(codes.Internal, "failed to authenticate with google")
		}
	}

	return &pb.GoogleAuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &pb.User{
			Id:          user.ID,
			Email:       user.Email,
			Name:        user.Name,
			PhoneNumber: user.PhoneNumber,
		},
	}, nil
}
