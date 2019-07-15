package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/cfanatic/go-slack2keybase/bridge"
)

const oauth = "<INSERT_BOT_TOKEN>"

func main() {
	bridge := bridge.New(oauth, true)
	bridge.Start()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		<-c
		bridge.Stop()
		break
	}
}
