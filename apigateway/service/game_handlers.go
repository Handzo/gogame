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

func (this apiService) CreateTable(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.CreateTable(ctx, req.(*gamepb.CreateTableRequest))
}

func (this apiService) JoinTable(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.JoinTable(ctx, req.(*gamepb.JoinTableRequest))
}

func (this apiService) MakeMove(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.MakeMove(ctx, req.(*gamepb.MakeMoveRequest))
}
