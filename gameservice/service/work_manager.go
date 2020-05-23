package service

import (
	"context"
	"errors"

	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/rmq"
	"github.com/opentracing/opentracing-go"
)

type WorkManager struct {
	worker   *rmq.Worker
	tracer   opentracing.Tracer
	logger   log.Factory
	handlers map[string]taskHandler
}

type taskHandler func(context.Context, *rmq.Task) error

func NewWorkManager(worker *rmq.Worker, tracer opentracing.Tracer, logger log.Factory) *WorkManager {
	return &WorkManager{
		worker:   worker,
		tracer:   tracer,
		logger:   logger,
		handlers: make(map[string]taskHandler),
	}
}

func (w *WorkManager) Start() {
	w.worker.Start()

	for task := range w.worker.Channel() {
		w.process(task)
	}
}

func (w *WorkManager) AddTask(task *rmq.Task) error {
	return w.worker.AddTask(task)
}

func (w *WorkManager) process(task *rmq.Task) {
	ctx := context.WithValue(context.Background(), "c", "b")
	span, ctx, logger := w.logger.StartForWithTracer(ctx, w.tracer, "Worker/"+task.Callback)
	defer span.Finish()

	logger.Info("New task received", log.String("task", task.Callback), log.String("topic", task.Topic))

	if h, ok := w.handlers[task.Callback]; ok {
		err := h(ctx, task)
		if err != nil {
			logger.Error(err)
		}
	} else {
		logger.Warn("No handlers for task", log.String("task", task.Callback))
	}
}

func (w *WorkManager) Register(task string, handler taskHandler) error {
	if _, ok := w.handlers[task]; ok {
		return errors.New("Task already registered")
	}

	w.handlers[task] = handler
	return nil
}
