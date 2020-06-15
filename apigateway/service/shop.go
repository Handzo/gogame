package service

import (
	"context"

	gamepb "github.com/Handzo/gogame/gameservice/proto"
)

func (this apiService) GetProducts(ctx context.Context, req interface{}) (interface{}, error) {
	return this.gamesvc.GetProducts(ctx, req.(*gamepb.GetProductsRequest))
}
