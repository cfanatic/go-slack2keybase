package bridge

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

type Bridge struct {
	trace *(log.Logger)
	api   *(slack.Client)
	rtm   *(slack.RTM)
}

func New(token string) (bridge Bridge) {
	bridge.trace = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)
	bridge.api = slack.New(token, slack.OptionDebug(false), slack.OptionLog(bridge.trace))
	bridge.rtm = bridge.api.NewRTM()
	return
}

func (bridge *Bridge) Start() {
	go bridge.rtm.ManageConnection()
	go func() {
		for msg := range bridge.rtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				bridge.trace.Print("INFO: Accepting messages")
			case *slack.MessageEvent:
				user, _ := bridge.api.GetUserInfo(ev.User)
				channel, _ := bridge.api.GetChannelInfo(ev.Channel)
				bridge.trace.Print("#", channel.Name, " [", strings.Title(user.Name), "] ", ev.Text)
			case *slack.RTMError:
				bridge.trace.Printf("ERROR: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				bridge.trace.Print("ERROR: Invalid credentials")
				break
			}
		}
	}()
}

func (bridge *Bridge) Stop() {
	bridge.rtm.Disconnect()
	fmt.Println()
	bridge.trace.Println("INFO: Closing connection")
}
