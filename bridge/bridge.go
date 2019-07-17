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
	trace    *(log.Logger)
	api_user *(slack.Client)
	api_bot  *(slack.Client)
	rtm      *(slack.RTM)
	chat     Chat
}

type Chat struct {
	ids  map[string]string
	hist map[string][]string
}

func New(user_token, bot_token string, debug bool) Bridge {
	b := Bridge{}
	b.trace = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)
	b.api_user = slack.New(user_token, slack.OptionDebug(false))
	b.api_bot = slack.New(bot_token, slack.OptionDebug(false))
	b.rtm = b.api_bot.NewRTM()
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
				b.getMessages()
			case *slack.MessageEvent:
				uInfo, _ := b.api_bot.GetUserInfo(ev.User)
				cInfo, _ := b.api_bot.GetChannelInfo(ev.Channel)
				channel, name, text := cInfo.Name, strings.Title(uInfo.Name), ev.Text
				b.sendMessage(channel, name, text)
			case *slack.RTMError:
				b.trace.Printf("ERROR: %s\n", ev.Error())
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
	args := []string{
		"chat",
		"send",
		"asrg",
		fmt.Sprintf("[%s]  %s", name, text),
		fmt.Sprintf("--channel=%s", channel)}
	if err := exec.Command(cmd, args...).Run(); err == nil {
		b.trace.Printf("#%s [%s] %s\n", channel, name, text)
	} else {
		b.trace.Printf("ERROR: %s\n", err)
	}
}

func (b *Bridge) getChannels() {
	if list, err := b.api_bot.GetChannels(true); err == nil {
		b.chat.ids = make(map[string]string)
		for _, channel := range list {
			b.chat.ids[channel.Name] = channel.ID
		}
	} else {
		b.trace.Printf("ERROR: %s\n", err)
	}
}

func (b *Bridge) getMessages() {
	param := slack.NewHistoryParameters()
	param.Count = 10
	b.chat.hist = make(map[string][]string)
	for key, _ := range b.chat.ids {
		if hist, err := b.api_user.GetChannelHistory(b.chat.ids[key], param); err == nil {
			for _, msg := range hist.Messages {
				b.chat.hist[key] = append(b.chat.hist[key], msg.Text)
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
	}
}
