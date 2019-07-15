package bridge

import (
	"fmt"
	"io/ioutil"
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

func New(token string, debug bool) Bridge {
	b := Bridge{}
	b.trace = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)
	b.api = slack.New(token, slack.OptionDebug(false))
	b.rtm = b.api.NewRTM()
	if !debug {
		b.trace.SetOutput(ioutil.Discard)
	}
	return b
}

func (b *Bridge) Start() {
	go b.rtm.ManageConnection()
	go func() {
		for msg := range b.rtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				b.trace.Print("INFO: Accepting messages")
			case *slack.MessageEvent:
				user, _ := b.api.GetUserInfo(ev.User)
				channel, _ := b.api.GetChannelInfo(ev.Channel)
				str := fmt.Sprintf("#%s [%s] %s", channel.Name, strings.Title(user.Name), ev.Text)
				b.trace.Print(str)
			case *slack.RTMError:
				str := fmt.Sprintf("ERROR: %s\n", ev.Error())
				b.trace.Print(str)
			case *slack.InvalidAuthEvent:
				b.trace.Print("ERROR: Invalid credentials")
				break
			}
		}
	}()
}

func (b *Bridge) Stop() {
	b.rtm.Disconnect()
	fmt.Println()
	b.trace.Print("INFO: Closing connection")
}
