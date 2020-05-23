package pubsub

import (
	"context"
	"fmt"

	"github.com/Handzo/gogame/common/log"
	"github.com/go-redis/redis"
)

type room struct {
	id     string
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
	key := getKey(r.id)

	subs, err := r.redis.SMembers(key).Result()
	if err != nil {
		// TODO: log error
		fmt.Println(err)
	}

	r.logger.For(ctx).Info("push to room players", log.Object("players", subs))

	for _, sub := range subs {
		go r.pubsub.ToPlayer(ctx, sub, msg)
	}
}

func (r room) publish(ctx context.Context, sub string, msg interface{}) {
	span, ctx, logger := r.logger.StartFor(ctx, "Publish/"+sub)
	defer span.Finish()

	logger.Info(msg.(string))

	r.redis.Publish(sub, msg)
}

func getKey(id string) string {
	return fmt.Sprintf("room:%s", id)
}
