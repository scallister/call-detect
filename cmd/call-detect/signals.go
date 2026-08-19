package main

import (
	"os"
	"os/signal"
	"syscall"
)

func notifyQuit(quit func()) {
	if quit == nil {
		return
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		quit()
	}()
}
