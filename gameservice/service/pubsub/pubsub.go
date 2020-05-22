package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Handzo/gogame/common/log"
	"github.com/go-redis/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type PubSub struct {
	redis  *redis.Client
	tracer opentracing.Tracer
	logger log.Factory
}

func New(redis *redis.Client, tracer opentracing.Tracer, logger log.Factory) *PubSub {
	return &PubSub{
		redis:  redis,
		tracer: tracer,
		logger: logger,
	}
}

func (p *PubSub) Bind(ctx context.Context, remote, playerId string) error {
	return p.redis.Set(playerId, remote, 0).Err()
}

func (p *PubSub) Unbind(ctx context.Context, remote string) {
	p.redis.Del(remote)
}

func (p *PubSub) Publish(ctx context.Context, channel string, msg interface{}) {
	ctx, span := p.startSpan(ctx, channel)
	if span != nil {
		defer span.Finish()
	}

	logger := p.logger.For(ctx).With(log.String("channel", channel))
	logger.Info(msg)

	data, err := json.Marshal(msg)

	cmd := p.redis.Publish(channel, string(data))

	val, err := cmd.Result()

	logger.Info(cmd, log.Int64("param.received", val), log.Error(err))
}

func (p *PubSub) Room(id string) *room {
	return &room{id, p.redis, p.logger}
}

func (p *PubSub) ToPlayer(ctx context.Context, id string, msg interface{}) {
	remote, err := p.redis.Get(id).Result()
	if err != nil || remote == "" {
		// no such user connected to pubsub
		fmt.Println(err)
	}

	p.Publish(ctx, remote, msg)
}

func (p *PubSub) PublishToRoom(ctx context.Context, roomId string, msg interface{}) {
	ctx, span := p.startSpan(ctx, "room:"+roomId)
	if span != nil {
		defer span.Finish()
	}

	logger := p.logger.For(ctx)

	key := fmt.Sprintf("room:%s", roomId)

	logger.Info("get room members", log.String("room", roomId))

	remotes, err := p.redis.SMembers(key).Result()
	if err != nil {
		logger.Error(err)
	}

	for _, remote := range remotes {
		logger.Info("push to " + remote)
		p.Publish(ctx, remote, msg)
	}
}

func (p *PubSub) AddToRoom(ctx context.Context, roomId, remote string) {
	key := fmt.Sprintf("room:%s", roomId)
	ctx, span := p.startSpan(ctx, key)
	if span != nil {
		defer span.Finish()
	}

	logger := p.logger.For(ctx).With(log.String("room", key))

	cmd := p.redis.SAdd(key, remote)

	val, err := cmd.Result()

	if err != nil {
		logger.Error(err)
	}

	logger.Info(cmd, log.Int64("param.received", val))
}

func (p *PubSub) startSpan(ctx context.Context, channel string) (context.Context, opentracing.Span) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = p.tracer.StartSpan(channel, opentracing.ChildOf(span.Context()))
		span.SetTag("param.channel", channel)
		ext.SpanKindRPCClient.Set(span)
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	return ctx, span
}
