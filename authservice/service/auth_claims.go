package service

import "github.com/dgrijalva/jwt-go"

type AuthClaims struct {
	UserId   string
	Username string
	jwt.StandardClaims
}
