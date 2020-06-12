package code

import "google.golang.org/grpc/status"

var (
	UserAlreadyExists       = status.Error(200, "user already exist")
	UserNotFound            = status.Error(201, "user not found")
	InvalidPassword         = status.Error(202, "invalid password")
	InvalidAuthInfo         = status.Error(203, "invalid auth info")
	InvalidToken            = status.Error(204, "invalid token")
	InvalidVerificationCode = status.Error(205, "invalid verification code")
)
