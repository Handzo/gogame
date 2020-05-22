package rmq

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Task struct {
	Id          string
	Topic       string
	Callback    string
	Payload     string
	ExecuteTime time.Time
	CreateTime  time.Time
}

const (
	DelayQueue = "rmq_delay_queue"
	UnackQueue = "rmq_unack_queue"
	ErrorQueue = "rmq_error_queue"
)

type option struct {
	ExecuteTime time.Time
	Topic       string
}

type TaskOption func(*option)

func NewTask(callback, payload string, opts ...TaskOption) *Task {
	defaultOpt := &option{
		ExecuteTime: time.Now(),
	}

	for _, opt := range opts {
		opt(defaultOpt)
	}

	id := uuid.Must(uuid.NewV4())
	return &Task{
		Id:          id.String(),
		Topic:       defaultOpt.Topic,
		Callback:    callback,
		Payload:     payload,
		ExecuteTime: defaultOpt.ExecuteTime,
		CreateTime:  time.Now(),
	}
}

func WithExecTime(t time.Time) TaskOption {
	return func(o *option) {
		o.ExecuteTime = t
	}
}

func WithDelay(d time.Duration) TaskOption {
	return func(o *option) {
		o.ExecuteTime = time.Now().Add(d)
	}
}

func WithTopic(topic string) TaskOption {
	return func(o *option) {
		o.Topic = topic
	}
}
