package log

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Info(msg interface{}, fields ...Field)
	Error(msg interface{}, fields ...Field)
	Warn(msg interface{}, fields ...Field)
	Fatal(msg interface{}, fields ...Field)
	Infof(format string, msg interface{}, fields ...Field)
	Errorf(format string, msg interface{}, fields ...Field)
	Warnf(format string, msg interface{}, fields ...Field)
	Fatalf(format string, msg interface{}, fields ...Field)
	With(...Field) Logger
}

func NewEntry() *logrus.Entry {
	// TODO options
	l := logrus.New()
	l.Formatter = &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		// CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
		// 	return fmt.Sprintf("%s:%d", frame.Function, frame.Line), ""
		// },
	}

	// l.ReportCaller = true
	return logrus.NewEntry(l)
}

type logger struct {
	logger *logrus.Entry
}

func (l logger) Info(msg interface{}, fields ...Field) {
	l.logger.WithFields(FieldMap(fields...)).Info(msg)
}

func (l logger) Error(msg interface{}, fields ...Field) {
	l.logger.WithFields(FieldMap(fields...)).Error(msg)
}

func (l logger) Warn(msg interface{}, fields ...Field) {
	l.logger.WithFields(FieldMap(fields...)).Warn(msg)
}

func (l logger) Fatal(msg interface{}, fields ...Field) {
	l.logger.WithFields(FieldMap(fields...)).Fatal(msg)
}

func (l logger) Infof(format string, msg interface{}, fields ...Field) {
	l.Info(fmt.Sprintf(format, msg), fields...)
}

func (l logger) Errorf(format string, msg interface{}, fields ...Field) {
	l.Error(fmt.Sprintf(format, msg), fields...)
}

func (l logger) Warnf(format string, msg interface{}, fields ...Field) {
	l.Warn(fmt.Sprintf(format, msg), fields...)
}

func (l logger) Fatalf(format string, msg interface{}, fields ...Field) {
	l.Fatal(fmt.Sprintf(format, msg), fields...)
}

func (l logger) With(fields ...Field) Logger {
	entry := l.logger
	for _, f := range fields {
		entry = entry.WithField(f.Key(), f.Value())
	}
	return logger{entry}
}

func FieldMap(fields ...Field) logrus.Fields {
	fm := logrus.Fields{}
	for _, f := range fields {
		fm[f.Key()] = f.Value()
	}
	return fm
}
