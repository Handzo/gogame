package log

import (
	"time"

	openlog "github.com/opentracing/opentracing-go/log"
)

type Field = openlog.Field

func String(key, val string) Field {
	return openlog.String(key, val)
}

func Bool(key string, val bool) Field {
	return openlog.Bool(key, val)
}

func Int(key string, val int) Field {
	return openlog.Int(key, val)
}

func Int32(key string, val int32) Field {
	return openlog.Int32(key, val)
}

func Int64(key string, val int64) Field {
	return openlog.Int64(key, val)
}

func Uint32(key string, val uint32) Field {
	return openlog.Uint32(key, val)
}

func Uint64(key string, val uint64) Field {
	return openlog.Uint64(key, val)
}

func Float32(key string, val float32) Field {
	return openlog.Float32(key, val)
}

func Float64(key string, val float64) Field {
	return openlog.Float64(key, val)
}

func Error(err error) Field {
	return openlog.Error(err)
}

func Object(key string, obj interface{}) Field {
	return openlog.Object(key, obj)
}

func Event(val string) Field {
	return String("event", val)
}

func Message(val string) Field {
	return String("message", val)
}

func Duration(dur time.Duration) Field {
	return String("duration", dur.String())
}
