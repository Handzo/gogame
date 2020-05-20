package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Handzo/gogame/rmq"
)

func main() {
	worker := rmq.NewWorker()
	worker.Start()

	fmt.Println(os.Getpid())

	time.AfterFunc(time.Second*3, func() {
		task := rmq.NewTask("StartGame", "p1,p2,p3,p4", rmq.WithExecTime(time.Now().Add(time.Second*2)))
		fmt.Println("Addtask")
		worker.AddTask(task)
	})

	for t := range worker.Channel() {
		fmt.Printf("%+v\n", t)
		fmt.Println("diff ", time.Since(t.ExecuteTime))
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
