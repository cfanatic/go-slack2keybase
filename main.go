package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cfanatic/go-slack2keybase/bridge"
)

const oauth = "<INSERT_BOT_TOKEN>"

var trace = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)

func main() {
	bridge := bridge.New(oauth, trace.Print)
	bridge.Start()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		<-c
		bridge.Stop()
		break
	}
}
