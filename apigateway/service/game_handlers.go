package service

import (
	"context"

	gamepb "github.com/Handzo/gogame/gameservice/proto"
)

func (this apiService) OpenSession(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.OpenSession(ctx, req.(*gamepb.OpenSessionRequest))
}

func (this apiService) CloseSession(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.CloseSession(ctx, req.(*gamepb.CloseSessionRequest))
}

func (this apiService) ChangePassword(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.ChangePassword(ctx, req.(*gamepb.ChangePasswordRequest))
}

func (this apiService) CreateTable(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.CreateTable(ctx, req.(*gamepb.CreateTableRequest))
}

func (this apiService) GetOpenTables(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.GetOpenTables(ctx, req.(*gamepb.GetOpenTablesRequest))
}

func (this apiService) JoinTable(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.JoinTable(ctx, req.(*gamepb.JoinTableRequest))
}

func (this apiService) BecomeParticipant(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.BecomeParticipant(ctx, req.(*gamepb.BecomeParticipantRequest))
}

func (this apiService) Ready(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.Ready(ctx, req.(*gamepb.ReadyRequest))
}

func (this apiService) MakeMove(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.MakeMove(ctx, req.(*gamepb.MakeMoveRequest))
}
