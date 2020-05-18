package repository

import (
	"context"
	"github.com/Handzo/gogame/authservice/repository/model"
)

type AuthRepository interface {
	CreateUser(context.Context, *model.User) (error)
	GetUserByUsername(context.Context, string) (*model.User, error)
	GetUserById(context.Context, string) (*model.User, error)
}
