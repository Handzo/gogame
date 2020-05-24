package repository

import (
	"context"

	"github.com/Handzo/gogame/gameservice/repository/model"
)

type GameRepository interface {
	Update(context.Context, interface{}, ...string) error
	Select(context.Context, interface{}, ...string) error
	Insert(context.Context, interface{}) error
	SelectOrInsertPlayer(context.Context, *model.Player) (bool, error)
	CreateSession(context.Context, *model.Session) error
	GetOpenedSessionForRemote(context.Context, string) (*model.Session, error)
	CreateTable(context.Context, string, string, uint32) (*model.Table, error)
	FindTable(context.Context, string) (*model.Table, error)
	FindTableWithPlayer(context.Context, string) (*model.Table, error)
	FindCurrentRoundForTable(context.Context, string) (*model.Round, error)
	FindCurrentDealForTable(context.Context, string) (*model.Deal, error)
	FindCurrentDealOrderForTable(context.Context, string) (*model.DealOrder, error)
	FindParticipantWithOrder(context.Context, string, int) (*model.Participant, error)
}
