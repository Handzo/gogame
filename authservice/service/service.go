package service

import (
	"context"
	"time"

	"github.com/Handzo/gogame/authservice/code"
	pb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/authservice/repository"
	"github.com/Handzo/gogame/authservice/repository/model"
	"github.com/Handzo/gogame/authservice/repository/postgres"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	"github.com/dgrijalva/jwt-go"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
)

var (
	key = []byte("mysupersecretkey")
)

type authService struct {
	tracer opentracing.Tracer
	logger log.Factory
	repo   repository.AuthRepository
}

func NewService(tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) pb.AuthServiceServer {
	return &authService{
		tracer: tracer,
		logger: logger,
		repo: postgres.New(
			tracing.New("auth-db-pg", metricsFactory, logger),
			logger,
		),
	}
}

func (s *authService) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	if req.Username == "" || req.Password == "" || req.Email == "" {
		return nil, code.InvalidAuthInfo
	}

	user := &model.User{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	}
	if err := user.HashPassword(); err != nil {
		return nil, err
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.GetToken(ctx, user.Id, user.Username)
	if err != nil {
		return nil, err
	}

	return &pb.SignUpResponse{
		Token: token,
	}, nil
}

func (s *authService) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.SignInResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, code.InvalidAuthInfo
	}

	user, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if !user.ValidPassword(req.Password) {
		return nil, code.InvalidPassword
	}

	token, err := s.GetToken(ctx, user.Id, user.Username)
	if err != nil {
		return nil, err
	}

	return &pb.SignInResponse{
		Token: token,
	}, nil
}

// func (s *authService) GetUserInfo(ctx context.Context, req *pb.UserInfoRequest) (*pb.UserInfoResponse, error) {
// 	return &pb.UserInfoResponse{}, nil
// }

func (s *authService) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	token, _ := jwt.ParseWithClaims(req.Token, &AuthClaims{}, func(*jwt.Token) (interface{}, error) {
		return key, nil
	})

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return &pb.ValidateResponse{
			UserId:   claims.UserId,
			Username: claims.Username,
		}, nil
	}

	return nil, code.InvalidToken
}

func (s *authService) GetToken(ctx context.Context, userId, username string) (string, error) {
	expireToken := time.Now().Add(time.Hour * 1).Unix()

	jwttoken := jwt.NewWithClaims(jwt.SigningMethodHS256, AuthClaims{
		UserId:   userId,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireToken,
		},
	})

	return jwttoken.SignedString(key)
}

func (s *authService) GetVerificationCode(ctx context.Context, req *pb.GetVerificationCodeRequest) (*pb.GetVerificationCodeResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	code, err := s.repo.CreateVerificationCode(ctx, user)
	if err != nil {
		return nil, err
	}

	// TODO: send to email
	return &pb.GetVerificationCodeResponse{
		Code: code.Code,
	}, nil
}

func (s *authService) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	// TODO: validate password

	c, err := s.repo.GetVerificationCode(ctx, req.Code)
	if err != nil {
		return nil, code.InvalidVerificationCode
	}

	user := &model.User{}
	user.Id = c.UserId

	if err = s.repo.Select(ctx, user, "password"); err != nil {
		return nil, err
	}

	user.Password = req.NewPassword
	user.HashPassword()

	if err = s.repo.Update(ctx, user, "password"); err != nil {
		return nil, err
	}

	return &pb.ResetPasswordResponse{}, nil
}

func (s *authService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	user := &model.User{}
	user.Id = req.UserId

	if err := s.repo.Select(ctx, user); err != nil {
		return nil, code.UserNotFound
	}

	if !user.ValidPassword(req.OldPassword) {
		return nil, code.InvalidPassword
	}

	user.Password = req.NewPassword
	user.HashPassword()

	if err := s.repo.Update(ctx, user, "password"); err != nil {
		return nil, err
	}

	return &pb.ChangePasswordResponse{}, nil
}
