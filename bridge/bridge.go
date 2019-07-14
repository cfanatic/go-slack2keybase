package bridge

import (
	"fmt"
	"strings"

	"github.com/nlopes/slack"
)

type Bridge struct {
	trace func(...interface{})
	api   *(slack.Client)
	rtm   *(slack.RTM)
}

func New(token string, trace func(...interface{})) (bridge Bridge) {
	bridge.trace = trace
	bridge.api = slack.New(token, slack.OptionDebug(false))
	bridge.rtm = bridge.api.NewRTM()
	return
}

func (bridge *Bridge) Start() {
	go bridge.rtm.ManageConnection()
	go func() {
		for msg := range bridge.rtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				bridge.trace("INFO: Accepting messages")
			case *slack.MessageEvent:
				user, _ := bridge.api.GetUserInfo(ev.User)
				channel, _ := bridge.api.GetChannelInfo(ev.Channel)
				str := fmt.Sprintf("#%s [%s] %s", channel.Name, strings.Title(user.Name), ev.Text)
				bridge.trace(str)
			case *slack.RTMError:
				str := fmt.Sprintf("ERROR: %s\n", ev.Error())
				bridge.trace(str)
			case *slack.InvalidAuthEvent:
				bridge.trace("ERROR: Invalid credentials")
				break
			}
		}
	}()
}

func (bridge *Bridge) Stop() {
	bridge.rtm.Disconnect()
	fmt.Println()
	bridge.trace("INFO: Closing connection")
}
