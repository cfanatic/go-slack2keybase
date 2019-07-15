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

func (b *Bridge) Start() {
	go b.rtm.ManageConnection()
	go func() {
		for msg := range b.rtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				b.trace("INFO: Accepting messages")
			case *slack.MessageEvent:
				user, _ := b.api.GetUserInfo(ev.User)
				channel, _ := b.api.GetChannelInfo(ev.Channel)
				str := fmt.Sprintf("#%s [%s] %s", channel.Name, strings.Title(user.Name), ev.Text)
				b.trace(str)
			case *slack.RTMError:
				str := fmt.Sprintf("ERROR: %s\n", ev.Error())
				b.trace(str)
			case *slack.InvalidAuthEvent:
				b.trace("ERROR: Invalid credentials")
				break
			}
		}
	}()
}

func (b *Bridge) Stop() {
	b.rtm.Disconnect()
	fmt.Println()
	b.trace("INFO: Closing connection")
}
