package log

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
)

type spanLogger struct {
	logger *logrus.Entry
	span   opentracing.Span
}

func (sl spanLogger) Info(msg interface{}, fields ...Field) {
	sl.logToSpan("info", msg, fields...)
	sl.logger.WithFields(FieldMap(fields...)).Info(msg)
}

func (sl spanLogger) Error(msg interface{}, fields ...Field) {
	sl.logToSpan("error", msg, fields...)
	sl.logger.WithFields(FieldMap(fields...)).Error(msg)
}

func (sl spanLogger) Warn(msg interface{}, fields ...Field) {
	sl.logToSpan("warn", msg, fields...)
	sl.logger.WithFields(FieldMap(fields...)).Warn(msg)
}

func (sl spanLogger) Fatal(msg interface{}, fields ...Field) {
	sl.logToSpan("fatal", msg, fields...)
	ext.Error.Set(sl.span, true)
	sl.logger.WithFields(FieldMap(fields...)).Fatal(msg)
}

func (sl spanLogger) Infof(format string, msg interface{}, fields ...Field) {
	sl.Info(fmt.Sprintf(format, msg), fields...)
}

func (sl spanLogger) Errorf(format string, msg interface{}, fields ...Field) {
	sl.Error(fmt.Sprintf(format, msg), fields...)
}

func (sl spanLogger) Warnf(format string, msg interface{}, fields ...Field) {
	sl.Warn(fmt.Sprintf(format, msg), fields...)
}

func (sl spanLogger) Fatalf(format string, msg interface{}, fields ...Field) {
	sl.Fatal(fmt.Sprintf(format, msg), fields...)
}

func (sl spanLogger) With(fields ...Field) Logger {
	return spanLogger{logger: sl.logger.WithFields(FieldMap(fields...)), span: sl.span}
}

func (sl spanLogger) logToSpan(level string, msg interface{}, fields ...Field) {
	fields = append(fields, String("level", level), Object("content", msg))
	sl.span.LogFields(fields...)
}
