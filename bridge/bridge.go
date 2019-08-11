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
	hist   map[string][]string
	wspace string
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
	b.chat.hist = make(map[string][]string)
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
				b.chat.wspace = ev.Info.Team.Domain
				b.getChannels()
				b.getMessages()
				b.trace.Print("INFO: Connection established")
			case *slack.MessageEvent:
				uInfo, _ := b.api_bot.GetUserInfo(ev.User)
				cInfo, _ := b.api_bot.GetChannelInfo(ev.Channel)
				channel, time, name, text := cInfo.Name, ev.Timestamp, strings.Title(uInfo.Name), ev.Text
				b.sendMessage(channel, time, name, text)
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
func (b *Bridge) sendMessage(channel, time, name, text string) {
	timestamp := b.timestamp(time)
	cmd := "keybase"
	args := []string{
		"chat",
		"send",
		fmt.Sprintf("%s", b.chat.wspace),
		fmt.Sprintf("[%s] [%s] %s", timestamp, name, text),
		fmt.Sprintf("--channel=%s", channel),
	}
	if err := exec.Command(cmd, args...).Run(); err == nil {
		b.trace.Printf("#%s [%s] [%s] %s\n", channel, timestamp, name, text)
	} else {
		b.trace.Printf("ERROR: %s\n", err)
	}
}

// sendMessages sends the complete or partial chat history to Keybase.
// Input argument is the chat history and optionally a channel name.
func (b *Bridge) sendMessages(hist map[string][]string, arg ...string) {
	send := func(channel, value string) {
		hist := strings.Split(value, ";")
		time, name, text := hist[0], hist[1], hist[2]
		b.sendMessage(channel, time, name, text)
	}
	if len(arg) > 0 {
		channel := arg[0]
		if _, ok := hist[channel]; ok == true {
			for _, value := range hist[channel] {
				defer send(channel, value)
			}
		} else {
			b.trace.Printf("ERROR: History not available for channel #%s\n", channel)
		}
	} else {
		for channel := range hist {
			for _, value := range hist[channel] {
				defer send(channel, value)
			}
		}
	}
}

// getMessages performs a chat history synchronization between the Slack and Keybase.
// Any messages which have not been sent from Slack yet are forwarded to Keybase.
func (b *Bridge) getMessages() {
	sync := func(hist *slack.History, key string) {
		for _, msg := range hist.Messages {
			if _, ok := b.chat.users[msg.User]; !ok {
				user, _ := b.api_bot.GetUserInfo(msg.User)
				b.chat.users[msg.User] = strings.Title(user.Name)
			}
			meta := fmt.Sprintf("%s;%s;%s", msg.Msg.Timestamp, b.chat.users[msg.User], msg.Text)
			b.chat.hist[key] = append(b.chat.hist[key], meta)
		}
	}
	for key, _ := range b.chat.chans {
		lastsk, lastkb := make(map[string]string), make(map[string]string)
		param := slack.NewHistoryParameters()
		param.Count = 1
		if hist, err := b.api_user.GetChannelHistory(b.chat.chans[key], param); err == nil {
			sync(hist, key)
			if len(b.chat.hist[key]) > 0 {
				meta := strings.Split(b.chat.hist[key][0], ";")
				time := fmt.Sprintf("%s", b.timestamp(meta[0]))
				lastsk["time"], lastsk["name"], lastsk["text"] = time, meta[1], meta[2]
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
		cmd := "keybase"
		args := []string{
			"chat",
			"api",
			"-m",
			fmt.Sprintf("{\"method\":\"read\",\"params\":{\"options\":{\"channel\":{\"name\":\"%s\",\"members_type\":\"team\",\"topic_name\":\"%s\",\"topic_type\":\"chat\"},\"pagination\":{\"num\":1}}}}", b.chat.wspace, key),
		}
		if hist, err := exec.Command(cmd, args...).Output(); err == nil {
			msg := Message{}
			if err := json.Unmarshal(hist, &msg); err == nil {
				meta := make([]string, 0)
				message := msg.Result.Messages[0].Msg.Content.Text.Body
				re := regexp.MustCompile(`\[([^\[\]]*)\]`)
				if submatches := re.FindAllString(message, -1); len(submatches) > 0 {
					for _, element := range submatches {
						element = strings.Trim(element, "[")
						element = strings.Trim(element, "]")
						meta = append(meta, element)
					}
					text := ""
					text = strings.Split(message, "["+meta[1]+"]")[1]
					text = strings.TrimSpace(text)
					lastkb["time"], lastkb["name"], lastkb["text"] = meta[0], meta[1], text
				}
			} else {
				b.trace.Printf("ERROR: %s\n", err)
			}
		} else {
			b.trace.Printf("ERROR: %s\n", err)
		}
		if eq := reflect.DeepEqual(lastsk, lastkb); eq == false {
			if len(lastkb) == 0 {
				b.trace.Printf("INFO: History in channel \"%s\" is empty", key)
			} else {
				b.trace.Printf("INFO: History in channel \"%s\" is out-of-date", key)
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
