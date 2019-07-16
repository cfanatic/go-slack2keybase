package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/cfanatic/go-slack2keybase/bridge"
)

const oauth_user = "<INSERT_USER_TOKEN>"
const oauth_bot = "<INSERT_BOT_TOKEN>"

func main() {
	bridge := bridge.New(oauth_user, oauth_bot, true)
	bridge.Start()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		<-c
		bridge.Stop()
		break
	}
}
