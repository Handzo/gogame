package repository

import (
	"context"

	"github.com/Handzo/gogame/authservice/repository/model"
)

type AuthRepository interface {
	Update(context.Context, interface{}, ...string) error
	Select(context.Context, interface{}, ...string) error
	Insert(context.Context, interface{}) error
	CreateUser(context.Context, *model.User) error
	GetUserByUsername(context.Context, string) (*model.User, error)
	GetUserByEmail(context.Context, string) (*model.User, error)
	GetUserById(context.Context, string) (*model.User, error)
	CreateVerificationCode(context.Context, *model.User) (*model.VerificationCode, error)
	GetVerificationCode(context.Context, string) (*model.VerificationCode, error)
}
