package repository

import (
	"context"

	"github.com/Handzo/gogame/gameservice/repository/model"
)

type GameRepository interface {
	SelectOrInsertPlayer(context.Context, *model.Player) (bool, error)
	CreateSession(context.Context, *model.Session) error
	UpdateSessions(context.Context, ...*model.Session) error
	GetOpenedSessionForRemote(context.Context, string) (*model.Session, error)
	CreateTable(context.Context, string, uint32) (*model.Table, error)
	FindTable(context.Context, string) (*model.Table, error)
	FindPlayersTable(context.Context, string) (*model.Table, error)
	CreateParticipants(context.Context, ...*model.Participant) error
	JoinTable(context.Context, string, string) (int, error)
	Update(context.Context, interface{}, ...string) error
	Select(context.Context, interface{}, ...string) error
}
