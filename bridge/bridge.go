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
	chans map[string]string
	users map[string]string
	hist  map[string][]string
}

func New(user_token, bot_token string, debug bool) Bridge {
	b := Bridge{}
	b.trace = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)
	b.api_user = slack.New(user_token, slack.OptionDebug(false))
	b.api_bot = slack.New(bot_token, slack.OptionDebug(false))
	b.rtm = b.api_bot.NewRTM()
	b.chat.chans = make(map[string]string)
	b.chat.users = make(map[string]string)
	b.chat.hist = make(map[string][]string)
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

func (b *Bridge) sendMessages(hist map[string][]string, arg ...string) {
	send := func(channel, value string) {
		hist := strings.Split(value, ";")
		name, text := strings.Title(hist[0]), strings.TrimSpace(hist[1])
		b.sendMessage(channel, name, text)
	}
	if len(arg) > 0 {
		channel := arg[0]
		if _, ok := hist[channel]; ok == true {
			for _, value := range hist[channel] {
				defer send(channel, value)
			}
		}
	} else {
		for channel := range hist {
			for _, value := range hist[channel] {
				defer send(channel, value)
			}
		}
	}
}

func (b *Bridge) getChannels() {
	if list, err := b.api_bot.GetChannels(true); err == nil {
		for _, channel := range list {
			b.chat.chans[channel.Name] = channel.ID
		}
	} else {
		b.trace.Printf("ERROR: %s\n", err)
	}
}

func (b *Bridge) getMessages() {
	param := slack.NewHistoryParameters()
	param.Count = 10
	for key, _ := range b.chat.chans {
		if hist, err := b.api_user.GetChannelHistory(b.chat.chans[key], param); err == nil {
			for _, msg := range hist.Messages {
				if _, ok := b.chat.users[msg.User]; !ok {
					user, _ := b.api_bot.GetUserInfo(msg.User)
					b.chat.users[msg.User] = user.Name
				}
				text := fmt.Sprintf("%s;%s\n", b.chat.users[msg.User], msg.Text)
				b.chat.hist[key] = append(b.chat.hist[key], text)
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
	}
}
