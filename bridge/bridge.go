// Package bridge forwards chat messages from Slack to Keybase.
package bridge

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	utime "time"

	"github.com/nlopes/slack"
)

type Bridge struct {
	trace *log.Logger
	api   messenger
	chat  chat
}

type messenger struct {
	skuser *(slack.Client)
	skbot  *(slack.Client)
	skrtm  *(slack.RTM)
	kb     *Keybase
}

type chat struct {
	chans  map[string]string
	users  map[string]string
	hist   map[string][]message
	wspace string
}

type message struct {
	time    utime.Time
	channel string
	name    string
	text    string
}

// New initializes the Slack connection and returns an object of type Bridge.
// It takes the user and bot OAuth access tokens from Slack as inputs.
// The debug status flag enables debug information on standard output.
func New(user_token, bot_token string, debug bool) Bridge {
	b := Bridge{}
	b.trace = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)
	b.api.skuser = slack.New(user_token, slack.OptionDebug(false))
	b.api.skbot = slack.New(bot_token, slack.OptionDebug(false))
	b.api.skrtm = b.api.skbot.NewRTM()
	b.api.kb = NewKeybase()
	b.chat.chans = make(map[string]string)
	b.chat.users = make(map[string]string)
	b.chat.hist = make(map[string][]message)
	if !debug {
		b.trace.SetOutput(ioutil.Discard)
	}
	return b
}

// Start listens for incoming and outgoing events in an endless loop.
// Chat messages sent to Slack will be forwarded to Keybase.
func (b *Bridge) Start() {
	go b.api.skrtm.ManageConnection()
	go func() {
		for msg := range b.api.skrtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				b.trace.Print("INFO: Slack connection established")
				b.chat.wspace = ev.Info.Team.Domain
				b.getChannels()
				b.getMessages()
			case *slack.HelloEvent:
				b.trace.Print("INFO: Slack history synchronized with Keybase")
			case *slack.MessageEvent:
				uInfo, _ := b.api.skbot.GetUserInfo(ev.User)
				cInfo, _ := b.api.skbot.GetChannelInfo(ev.Channel)
				msg := message{b.timestamp(ev.Timestamp), cInfo.Name, strings.Title(uInfo.Name), ev.Text}
				b.sendMessage(msg)
			case *slack.RTMError:
				b.trace.Printf("ERROR: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				b.trace.Print("ERROR: Slack credentials invalid")
				break
			}
		}
	}()
}

// Stop closes the connection by terminating all threads running in the background.
// This method shall be executed before the main program exits.
func (b *Bridge) Stop() {
	b.api.skrtm.Disconnect()
	fmt.Println()
	b.trace.Print("INFO: Closing connection")
}

// sendMessage sends a chat message to Keybase.
// Input argument is an object of type message with channel, time, name and text information.
func (b *Bridge) sendMessage(msg message) {
	if result, err := b.api.kb.SendChannelMessage(b.chat.wspace, msg); err == nil {
		b.trace.Printf("INFO: %s %+v\n", result, msg)
	} else {
		b.trace.Printf("ERROR: %s %s\n", result, err)
	}
}

// sendMessages sends the complete or partial chat history to Keybase.
// Input argument is the chat history and optionally a channel name.
func (b *Bridge) sendMessages(hist map[string][]message, arg ...string) {
	if len(arg) > 0 {
		channel := arg[0]
		if _, ok := hist[channel]; ok == true {
			for _, msg := range hist[channel] {
				defer b.sendMessage(msg)
			}
		} else {
			b.trace.Printf("ERROR: History not available for channel #%s\n", channel)
		}
	} else {
		for channel := range hist {
			for _, msg := range hist[channel] {
				defer b.sendMessage(msg)
			}
		}
	}
}

// getMessages performs a chat history synchronization between the Slack and Keybase.
// Any messages which have not been sent from Slack yet are forwarded to Keybase.
func (b *Bridge) getMessages() {
	sync := func(hist *slack.History, channel string) int {
		b.chat.hist[channel] = []message{}
		for _, msg := range hist.Messages {
			if _, ok := b.chat.users[msg.User]; !ok {
				user, _ := b.api.skbot.GetUserInfo(msg.User)
				b.chat.users[msg.User] = strings.Title(user.Name)
			}
			meta := message{b.timestamp(msg.Msg.Timestamp), channel, b.chat.users[msg.User], msg.Text}
			b.chat.hist[channel] = append(b.chat.hist[channel], meta)
		}
		return len(b.chat.hist[channel])
	}
	param := slack.HistoryParameters{}
	skmsg, kbmsg := message{}, message{}
	for channel, _ := range b.chat.chans {
		b.trace.Printf("INFO: Synchronizing channel \"%s\"\n", channel)
		param = slack.NewHistoryParameters()
		param.Count = 1
		if hist, err := b.api.skuser.GetChannelHistory(b.chat.chans[channel], param); err == nil {
			if num := sync(hist, channel); num > 0 {
				skmsg = b.chat.hist[channel][0]
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
		if hist, err := b.api.kb.GetChannelHistory(b.chat.wspace, channel, param); err == nil {
			if num := len(hist[channel]); num > 0 {
				kbmsg = hist[channel][0]
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
		if eq := reflect.DeepEqual(skmsg, kbmsg); eq == false {
			if (message{} != kbmsg) {
				param = slack.NewHistoryParameters()
				param.Oldest = strconv.FormatInt(kbmsg.time.Unix(), 10)
			} else {
				param = slack.NewHistoryParameters()
				param.Count = 10
			}
			if hist, err := b.api.skuser.GetChannelHistory(b.chat.chans[channel], param); err == nil {
				_ = sync(hist, channel)
				b.sendMessages(b.chat.hist, channel)
			} else {
				b.trace.Printf("ERROR: %s\n", err)
			}
		}
	}
}

// getChannels creates a map of channels that are available in the Slack workspace.
// The channel ID is saved over the channel name.
func (b *Bridge) getChannels() {
	if list, err := b.api.skbot.GetChannels(true); err == nil {
		for _, channel := range list {
			b.chat.chans[channel.Name] = channel.ID
		}
	} else {
		b.trace.Printf("ERROR: %s\n", err)
	}
}

func (b *Bridge) timestamp(time string) utime.Time {
	temp, _ := strconv.ParseInt(strings.Split(time, ".")[0], 10, 64)
	return utime.Unix(temp, 0)
}
