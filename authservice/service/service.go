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
	if req.Username == "" || req.Password == "" {
		return nil, code.InvalidAuthInfo
	}

	user := &model.User{
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

func (s *authService) GetUserInfo(ctx context.Context, req *pb.UserInfoRequest) (*pb.UserInfoResponse, error) {
	return &pb.UserInfoResponse{}, nil
}

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
