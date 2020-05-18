package pubsub

import (
	"context"
	"encoding/json"

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
