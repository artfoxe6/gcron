package main

import (
	"gcron"
	"gcron/config"
	"gcron/rpcService"
	"os"
	"os/signal"
	"syscall"
)

type Test struct {
	Name string
	//Height int
	//Sex    string
}

func main() {
	config.Load()
	go func() {
		rpcService.Run()
	}()
	scheduler := gcron.NewScheduler()
	scheduler.Start()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	scheduler.Stop <- 1
	rpcService.Stop()

}
