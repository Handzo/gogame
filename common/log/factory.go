package log

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type Factory struct {
	logger *logrus.Entry
}

func NewFactory(logger *logrus.Entry) Factory {
	return Factory{logger: logger}
}

func (b Factory) Bg() Logger {
	return logger{b.logger}
}

func (b Factory) For(ctx context.Context) Logger {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		// TODO for Jaeger span extract trace/span IDs as fields
		return spanLogger{span: span, logger: b.logger}
	}
	return b.Bg()
}

func (b Factory) StartFor(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context, Logger) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	// TODO for Jaeger span extract trace/span IDs as fields
	return span, ctx, spanLogger{span: span, logger: b.logger}
}

func (b Factory) StartForWithTracer(ctx context.Context, tracer opentracing.Tracer, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context, Logger) {
	span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, operationName, opts...)
	// TODO for Jaeger span extract trace/span IDs as fields
	return span, ctx, spanLogger{span: span, logger: b.logger}
}

func (b Factory) With(fields ...Field) Factory {
	return Factory{logger: b.logger.WithFields(FieldMap(fields...))}
}
