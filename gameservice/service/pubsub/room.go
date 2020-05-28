package pubsub

import (
	"context"
	"fmt"

	"github.com/Handzo/gogame/common/log"
	"github.com/go-redis/redis"
)

type room struct {
	id     string
	subs   []string
	redis  *redis.Client
	logger log.Factory
	pubsub *PubSub
}

type subscriber struct {
	remote string
	redis  *redis.Client
	logger log.Factory
}

func (r room) Publish(ctx context.Context, msg interface{}) {
	players, err := r.pubsub.GetPlayers(r.id)
	if err != nil {
		// TODO: log error
		fmt.Println(err)
	}

	r.logger.For(ctx).Info("push to room players", log.Object("players", players))

	for _, player := range players {
		go r.pubsub.ToPlayer(ctx, player, msg)
	}
}

func roomKey(id string) string {
	return fmt.Sprintf("room:%s", id)
}
