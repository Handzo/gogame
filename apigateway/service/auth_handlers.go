package service

import (
	"context"

	authpb "github.com/Handzo/gogame/authservice/proto"
)

func (this apiService) SignUp(ctx context.Context, req interface{}) (interface{}, error) {
	return this.authsvc.SignUp(ctx, req.(*authpb.SignUpRequest))
}

func (this apiService) SignIn(ctx context.Context, req interface{}) (interface{}, error) {
	return this.authsvc.SignIn(ctx, req.(*authpb.SignInRequest))
}

func (this apiService) GetVerificationCode(ctx context.Context, req interface{}) (interface{}, error) {
	return this.authsvc.GetVerificationCode(ctx, req.(*authpb.GetVerificationCodeRequest))
}

func (this apiService) ResetPassword(ctx context.Context, req interface{}) (interface{}, error) {
	return this.authsvc.ResetPassword(ctx, req.(*authpb.ResetPasswordRequest))
}
