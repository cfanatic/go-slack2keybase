package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/cfanatic/slack2keybase/bridge"
)

const oauth = "<YOUR_BOT_TOKEN>"

func main() {
	bridge := bridge.New(oauth)
	bridge.Start()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		<-c
		bridge.Stop()
		break
	}
}
