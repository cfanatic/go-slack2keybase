// Package bridge forwards chat messages from Slack to Keybase.
package bridge

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	utime "time"

	"github.com/nlopes/slack"
)

type Bridge struct {
	trace    *(log.Logger)
	api_user *(slack.Client)
	api_bot  *(slack.Client)
	rtm      *(slack.RTM)
	chat     chat
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
	b.api_user = slack.New(user_token, slack.OptionDebug(false))
	b.api_bot = slack.New(bot_token, slack.OptionDebug(false))
	b.rtm = b.api_bot.NewRTM()
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
	go b.rtm.ManageConnection()
	go func() {
		for msg := range b.rtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				b.trace.Print("INFO: Connection established")
				b.chat.wspace = ev.Info.Team.Domain
				b.getChannels()
				b.getMessages()
			case *slack.HelloEvent:
				b.trace.Print("INFO: Chat history synchronized")
			case *slack.MessageEvent:
				uInfo, _ := b.api_bot.GetUserInfo(ev.User)
				cInfo, _ := b.api_bot.GetChannelInfo(ev.Channel)
				msg := message{
					b.timestamp(ev.Timestamp),
					cInfo.Name,
					strings.Title(uInfo.Name),
					ev.Text,
				}
				b.sendMessage(msg)
			case *slack.RTMError:
				b.trace.Printf("ERROR: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				b.trace.Print("ERROR: Invalid credentials")
				break
			}
		}
	}()
}

// Stop closes the connection by terminating all threads running in the background.
// This method shall be executed before the main program exits.
func (b *Bridge) Stop() {
	b.rtm.Disconnect()
	fmt.Println()
	b.trace.Print("INFO: Closing connection")
}

// sendMessage sends a chat message to Keybase.
// Input arguments are the Slack channel, user name and text content.
func (b *Bridge) sendMessage(msg message) {
	cmd := "keybase"
	args := []string{
		"chat",
		"send",
		fmt.Sprintf("%s", b.chat.wspace),
		fmt.Sprintf("[%s] [%s] %s", msg.time, msg.name, msg.text),
		fmt.Sprintf("--channel=%s", msg.channel),
	}
	if err := exec.Command(cmd, args...).Run(); err == nil {
		b.trace.Printf("#%s [%s] [%s] %s\n", msg.channel, msg.time, msg.name, msg.text)
	} else {
		b.trace.Printf("ERROR: %s\n", err)
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
				user, _ := b.api_bot.GetUserInfo(msg.User)
				b.chat.users[msg.User] = strings.Title(user.Name)
			}
			meta := message{b.timestamp(msg.Msg.Timestamp), channel, b.chat.users[msg.User], msg.Text}
			b.chat.hist[channel] = append(b.chat.hist[channel], meta)
		}
		return len(b.chat.hist[channel])
	}
	param := slack.HistoryParameters{}
	for channel, _ := range b.chat.chans {
		b.trace.Printf("INFO: Synchronizing channel \"%s\"\n", channel)
		param = slack.NewHistoryParameters()
		param.Count = 1
		lastsk, lastkb := message{}, message{}
		if hist, err := b.api_user.GetChannelHistory(b.chat.chans[channel], param); err == nil {
			num := sync(hist, channel)
			if num > 0 {
				lastsk = b.chat.hist[channel][0]
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
		cmd := "keybase"
		args := []string{
			"chat",
			"api",
			"-m",
			fmt.Sprintf("{\"method\":\"read\",\"params\":{\"options\":{\"channel\":{\"name\":\"%s\",\"members_type\":\"team\",\"topic_name\":\"%s\",\"topic_type\":\"chat\"},\"pagination\":{\"num\":1}}}}", b.chat.wspace, channel),
		}
		if hist, err := exec.Command(cmd, args...).Output(); err == nil {
			response := KeybaseApi{}
			if err := json.Unmarshal(hist, &response); err == nil {
				meta := make([]string, 0)
				msg := response.Result.Messages[0].Msg.Content.Text.Body
				re := regexp.MustCompile(`\[([^\[\]]*)\]`)
				if submatches := re.FindAllString(msg, -1); len(submatches) > 0 {
					for _, element := range submatches {
						element = strings.Trim(element, "[")
						element = strings.Trim(element, "]")
						meta = append(meta, element)
					}
					time, _ := utime.Parse("2006-01-02 15:04:05.999999999 -0700 MST", meta[0])
					name := meta[1]
					text := ""
					text = strings.Split(msg, "["+meta[1]+"]")[1]
					text = strings.TrimSpace(text)
					lastkb.time, lastkb.channel, lastkb.name, lastkb.text = time, channel, name, text
				}
			} else {
				b.trace.Printf("ERROR: %s\n", err)
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
		if eq := reflect.DeepEqual(lastsk, lastkb); eq == false {
			if (message{} != lastkb) {
				param = slack.NewHistoryParameters()
				param.Oldest = strconv.FormatInt(lastkb.time.Unix(), 10)
			} else {
				param = slack.NewHistoryParameters()
				param.Count = 10
			}
			if hist, err := b.api_user.GetChannelHistory(b.chat.chans[channel], param); err == nil {
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
	if list, err := b.api_bot.GetChannels(true); err == nil {
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
