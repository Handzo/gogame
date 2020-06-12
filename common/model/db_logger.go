package model

import (
	"context"
	"fmt"

	"github.com/Handzo/gogame/common/log"
	"github.com/go-pg/pg/v9"
	"github.com/opentracing/opentracing-go"
	tags "github.com/opentracing/opentracing-go/ext"
)

type dbLogger struct {
	tracer opentracing.Tracer
	span   opentracing.Span
}

func NewDBLogger(tracer opentracing.Tracer) *dbLogger {
	return &dbLogger{
		tracer: tracer,
	}
}

func (d dbLogger) BeforeQuery(ctx context.Context, q *pg.QueryEvent) (context.Context, error) {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span = d.tracer.StartSpan("QUERY", opentracing.ChildOf(span.Context()))
		tags.SpanKindRPCClient.Set(span)
		tags.PeerService.Set(span, "pg")
		query, _ := q.FormattedQuery()
		span.LogFields(log.String("query", query))
		span.SetTag("query", "pg")
		ctx = opentracing.ContextWithSpan(ctx, span)
	}
	return ctx, nil
}

func (d dbLogger) AfterQuery(ctx context.Context, q *pg.QueryEvent) error {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span.Finish()
	}
	fmt.Println(q.FormattedQuery())
	return nil
}
