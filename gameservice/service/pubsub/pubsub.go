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

func playerKey(id string) string {
	return fmt.Sprintf("player:%s", id)
}

func New(redis *redis.Client, tracer opentracing.Tracer, logger log.Factory) *PubSub {
	return &PubSub{
		redis:  redis,
		tracer: tracer,
		logger: logger,
	}
}

func (p *PubSub) Bind(ctx context.Context, remote, playerId string) error {
	return p.redis.Set(playerKey(playerId), remote, 0).Err()
}

func (p *PubSub) Unbind(ctx context.Context, playerId string) {
	p.redis.Del(playerKey(playerId))
}

func (p *PubSub) Publish(ctx context.Context, channel string, msg interface{}) {
	ctx, span := p.startSpan(ctx, "Publish/"+channel)
	if span != nil {
		defer span.Finish()
	}

	logger := p.logger.For(ctx).With(log.String("channel", channel))

	data, err := json.Marshal(msg)

	cmd := p.redis.Publish(channel, data)

	val, err := cmd.Result()

	logger.Info(cmd.Name(), log.Int64("param.received", val), log.Error(err))
}

func (p *PubSub) Room(id string) *room {
	subs, err := p.GetPlayers(id)
	if err != nil {
		return nil
	}

	return &room{
		id:     id,
		subs:   subs,
		redis:  p.redis,
		logger: p.logger,
		pubsub: p,
	}
}

func (p *PubSub) ToPlayer(ctx context.Context, id string, msg interface{}) {
	remote, err := p.redis.Get(playerKey(id)).Result()
	if err != nil || remote == "" {
		// no such user connected to pubsub
		return
	}

	p.Publish(ctx, remote, msg)
}

func (p *PubSub) AddToRoom(ctx context.Context, roomId, player string) {
	key := roomKey(roomId)
	ctx, span := p.startSpan(ctx, key)
	if span != nil {
		defer span.Finish()
	}

	logger := p.logger.For(ctx).With(log.String("room", key))

	cmd := p.redis.SAdd(key, player)

	val, err := cmd.Result()

	if err != nil {
		logger.Error(err)
	}

	logger.Info(cmd, log.Int64("param.received", val))
}
func (p *PubSub) RemoveFromRoom(ctx context.Context, roomId, player string) {
	key := roomKey(roomId)
	ctx, span := p.startSpan(ctx, key)
	if span != nil {
		defer span.Finish()
	}

	logger := p.logger.For(ctx).With(log.String("room", key))

	cmd := p.redis.SRem(key, player)

	val, err := cmd.Result()

	if err != nil {
		logger.Error(err)
	}

	logger.Info(cmd, log.Int64("param.received", val))
}

func (p *PubSub) GetPlayers(roomId string) ([]string, error) {
	return p.redis.SMembers(roomKey(roomId)).Result()
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
