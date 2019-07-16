package bridge

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/nlopes/slack"
)

type Bridge struct {
	trace *(log.Logger)
	api   *(slack.Client)
	rtm   *(slack.RTM)
	chat  Chat
}

type Chat struct {
	channels []string
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
				b.trace.Print("INFO: Connection established")
				b.getChannels()
			case *slack.MessageEvent:
				uInfo, _ := b.api.GetUserInfo(ev.User)
				cInfo, _ := b.api.GetChannelInfo(ev.Channel)
				channel, name, text := cInfo.Name, strings.Title(uInfo.Name), ev.Text
				b.sendMessage(channel, name, text)
				str := fmt.Sprintf("#%s [%s] %s", channel, name, text)
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

func (b *Bridge) sendMessage(channel, name, text string) {
	cmd := "keybase"
	args := []string{"chat", "send", "asrg",
		fmt.Sprintf("[%s]  %s", name, text),
		fmt.Sprintf("--channel=%s", channel)}
	if err := exec.Command(cmd, args...).Run(); err != nil {
		b.trace.Print(err)
	}
}

func (b *Bridge) getChannels() {
	if list, err := b.api.GetChannels(true); err == nil {
		b.chat.channels = make([]string, 0, len(list))
		for _, channel := range list {
			b.chat.channels = append(b.chat.channels, channel.Name)
		}
	}
}
