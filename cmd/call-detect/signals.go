package main

import (
	"os"
	"os/signal"
)

func notifyQuit(quit func()) {
	if quit == nil {
		return
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, quitSignals...)
	go func() {
		<-ch
		signal.Stop(ch)
		signal.Ignore(quitSignals...)
		quit()
	}()
}
