package rmq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type Worker struct {
	redis               *redis.Client
	DelayWorkerInterval time.Duration
	UnackWorkerInterval time.Duration
	ErrorWorkerInterval time.Duration
	TaskTTL             time.Duration
	TaskTTR             time.Duration
	PoolCount           int64

	taskCh chan *Task
}

func NewWorker() *Worker {
	redis := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := redis.Ping().Result()
	if err != nil {
		panic(err)
	}

	return &Worker{
		redis:               redis,
		DelayWorkerInterval: 100 * time.Millisecond,
		UnackWorkerInterval: 300 * time.Millisecond,
		ErrorWorkerInterval: 300 * time.Millisecond,
		TaskTTL:             24 * time.Hour,
		TaskTTR:             3 * time.Second,
		PoolCount:           20,
		taskCh:              make(chan *Task, 20),
	}
}

func (w *Worker) Start() {
	go w.delayWorker()
	go w.unackWorker()
	go w.errorWorker()
}

func (w *Worker) Channel() <-chan *Task {
	return w.taskCh
}

func (w *Worker) AddTask(task *Task) error {
	task.CreateTime = time.Now()
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	err = w.redis.Set(task.Id, data, w.TaskTTL).Err()
	if err != nil {
		return err
	}

	return w.redis.ZAdd(DelayQueue, redis.Z{
		Score:  float64(task.ExecuteTime.UnixNano()),
		Member: task.Id,
	}).Err()
}

func (w *Worker) GetTask(id string) (*Task, error) {
	data, err := w.redis.Get(id).Result()
	if err != nil {
		return nil, err
	}
	var task Task
	return &task, json.Unmarshal([]byte(data), &task)
}

func (w *Worker) pool(queue string, from, to time.Time) ([]string, error) {
	return w.redis.ZRangeByScore(queue, redis.ZRangeBy{
		Min:    fmt.Sprintf("%d", from.UnixNano()),
		Max:    fmt.Sprintf("%d", to.UnixNano()),
		Offset: 0,
		Count:  w.PoolCount,
	}).Result()
}

func (w *Worker) delayToUnack(id string, score int64) (bool, error) {
	return w.taskTransfer(DelayQueue, UnackQueue, id, score)
}

func (w *Worker) unackToDelay(id string, score int64) (bool, error) {
	return w.taskTransfer(UnackQueue, DelayQueue, id, score)
}

func (w *Worker) errorToDelay(id string, score int64) (bool, error) {
	return w.taskTransfer(ErrorQueue, DelayQueue, id, score)
}

func (w *Worker) taskTransfer(from, to, id string, score int64) (bool, error) {
	count, err := w.redis.ZAdd(to, redis.Z{
		Score:  float64(score),
		Member: id,
	}).Result()

	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	err = w.redis.ZRem(from, id).Err()

	return true, err
}

func (w *Worker) unackToError(id string, score int64) error {
	err := w.redis.ZAdd(ErrorQueue, redis.Z{
		Score:  float64(score),
		Member: id,
	}).Err()

	if err != nil {
		return err
	}

	return w.redis.ZRem(UnackQueue, id).Err()
}

func (w *Worker) deleteTask(id string) error {
	if err := w.redis.Del(id).Err(); err != nil {
		return err
	}

	if err := w.redis.ZRem(DelayQueue, id).Err(); err != nil {
		return err
	}

	if err := w.redis.ZRem(UnackQueue, id).Err(); err != nil {
		return err
	}

	return w.redis.ZRem(ErrorQueue, id).Err()
}

func (w *Worker) delayWorker() {
	fmt.Println("Start delay worker")
	defer fmt.Println("Stop delay worker")

	ticker := time.NewTicker(w.DelayWorkerInterval)
	for _ = range ticker.C {
		go func() {
			now := time.Now()
			from := now.Add(-w.TaskTTL)
			// to := now.Add(-w.TaskTTR)

			// get delayed tasks
			ids, err := w.pool(DelayQueue, from, now)
			if err != nil {
				return
			}

			// run task
			for _, id := range ids {
				w.process(id)
			}
		}()
	}
}

func (w *Worker) unackWorker() {
	fmt.Println("Start unack worker")
	defer fmt.Println("Stop unack worker")

	ticker := time.NewTicker(w.UnackWorkerInterval)
	for _ = range ticker.C {
		go func() {
			now := time.Now()
			from := now.Add(-w.TaskTTL)

			// get from unack queue tasks
			ids, err := w.pool(UnackQueue, from, now)
			if err != nil {
				return
			}

			for _, id := range ids {
				w.unackToDelay(id, time.Now().UnixNano())
			}
		}()
	}
}

func (w *Worker) errorWorker() {
	fmt.Println("Start error worker")
	defer fmt.Println("Stop error worker")

	ticker := time.NewTicker(w.ErrorWorkerInterval)
	for _ = range ticker.C {
		go func() {
			now := time.Now()
			from := now.Add(-w.TaskTTL)

			// get from unack queue tasks
			ids, err := w.pool(ErrorQueue, from, now)
			if err != nil {
				return
			}

			for _, id := range ids {
				w.errorToDelay(id, time.Now().UnixNano())
			}
		}()
	}
}

func (w *Worker) process(id string) {
	// get task by id
	task, err := w.GetTask(id)
	if err != nil {
		fmt.Println(err)
		return
	}

	// push task from delay queue to unack
	got, err := w.delayToUnack(id, time.Now().Add(w.TaskTTR).UnixNano())
	if err != nil {
		fmt.Println("error transfer from delay to unack fail", err)
		return
	}

	if !got {
		return
	}

	w.taskCh <- task

	if err = w.deleteTask(id); err != nil {
		fmt.Println("error delete task fail", err)
		return
	}

	// process task
	// if error need retry?

	// delete task on success
}
