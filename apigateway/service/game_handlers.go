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
