package repository

import (
	"context"

	"github.com/Handzo/gogame/gameservice/repository/model"
)

type GameRepository interface {
	SelectOrInsertPlayer(context.Context, *model.Player) (bool, error)
	CreateSession(context.Context, *model.Session) error
	UpdateSessions(context.Context, ...*model.Session) error
}
