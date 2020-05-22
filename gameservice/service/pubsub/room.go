package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Handzo/gogame/common/log"
	"github.com/go-redis/redis"
)

type room struct {
	id     string
	redis  *redis.Client
	logger log.Factory
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

	payload, err := json.Marshal(msg)
	if err != nil {
		// TODO: log error
		fmt.Println(err)
	}

	for _, sub := range subs {
		r.redis.Publish(sub, payload)
	}
}

func getKey(id string) string {
	return fmt.Sprintf("room:%s", id)
}
