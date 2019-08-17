package bridge

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

type Keybase struct {
	response api
	history  map[string][]message
}

type api struct {
	Result struct {
		Message  string `json:"message"`
		ID       int    `json:"id"`
		Messages []struct {
			Msg struct {
				ID             int    `json:"id"`
				ConversationID string `json:"conversation_id"`
				Channel        struct {
					Name        string `json:"name"`
					Public      bool   `json:"public"`
					MembersType string `json:"members_type"`
					TopicType   string `json:"topic_type"`
					TopicName   string `json:"topic_name"`
				} `json:"channel"`
				Sender struct {
					UID        string `json:"uid"`
					Username   string `json:"username"`
					DeviceID   string `json:"device_id"`
					DeviceName string `json:"device_name"`
				} `json:"sender"`
				SentAt   int   `json:"sent_at"`
				SentAtMs int64 `json:"sent_at_ms"`
				Content  struct {
					Type string `json:"type"`
					Text struct {
						Body         string      `json:"body"`
						Payments     interface{} `json:"payments"`
						UserMentions interface{} `json:"userMentions"`
						TeamMentions interface{} `json:"teamMentions"`
					} `json:"text"`
				} `json:"content"`
				Prev []struct {
					ID   int    `json:"id"`
					Hash string `json:"hash"`
				} `json:"prev"`
				Unread         bool   `json:"unread"`
				ChannelMention string `json:"channel_mention"`
			} `json:"msg"`
		} `json:"messages"`
		Pagination struct {
			Next           string `json:"next"`
			Previous       string `json:"previous"`
			Num            int    `json:"num"`
			Last           bool   `json:"last"`
			ForceFirstPage bool   `json:"forceFirstPage"`
		} `json:"pagination"`
	} `json:"result"`
}

type command struct {
	name string
	arg  []string
}

type history = slack.HistoryParameters

func NewKeybase() *Keybase {
	kb := Keybase{}
	kb.history = make(map[string][]message)
	return &kb
}

func (kb *Keybase) GetChannelHistory(team, channel string, param history) (history map[string][]message, err error) {
	idx, id := 0, ""
	for idx < param.Count {
		if err := kb.getMessage(team, channel, id); err != nil {
			empty := make(map[string][]message)
			return empty, err
		}
		if kb.response.Result.Messages[0].Msg.Content.Type == "text" {
			meta := make([]string, 0)
			body := kb.response.Result.Messages[0].Msg.Content.Text.Body
			re := regexp.MustCompile(`\[([^\[\]]*)\]`)
			if submatches := re.FindAllString(body, -1); len(submatches) > 0 {
				for _, element := range submatches {
					element = strings.Trim(element, "[")
					element = strings.Trim(element, "]")
					meta = append(meta, element)
				}
				time := NewTimestamp(meta[0])
				name := meta[1]
				text := ""
				text = strings.Split(body, "["+meta[1]+"]")[1]
				text = strings.TrimSpace(text)
				kb.history[channel] = append(kb.history[channel], message{time, channel, name, text})
			}
			idx = idx + 1
		} else if kb.response.Result.Messages[0].Msg.ID > 1 {
			id = kb.response.Result.Pagination.Next
		} else {
			break
		}
	}
	return kb.history, nil
}

func (kb *Keybase) SendChannelMessage(team string, msg message) (result string, err error) {
	body, result := fmt.Sprintf("[%s] [%s] %s", msg.time, msg.name, msg.text), ""
	defer func() {
		result = kb.response.Result.Message
	}()
	if err := kb.sendMessage(team, msg.channel, body); err != nil {
		return result, err
	}
	return result, nil
}

func (kb *Keybase) sendMessage(team, channel, body string) (err error) {
	opt := fmt.Sprintf(`{
			"method":"send",
			"params":{
				"options":{
					"channel":{
						"name":"%s",
						"members_type":"team",
						"topic_name":"%s",
						"topic_type":"chat"
					},
					"message":{
						"body":"%s"
					}
				}
			}
		}`, team, channel, body)
	name := "keybase"
	arg := []string{
		"chat",
		"api",
		"-m",
		opt,
	}
	return kb.executeJSON(command{name, arg})
}

func (kb *Keybase) getMessage(team, channel, id string) (err error) {
	opt := fmt.Sprintf(`{
			"method":"read",
			"params":{
				"options":{
					"channel":{
						"name":"%s",
						"members_type":"team",
						"topic_name":"%s",
						"topic_type":"chat"
					},
					"pagination":{
						"num":1,
						"next":"%s"
					}
				}
			}
		}`, team, channel, id)
	name := "keybase"
	arg := []string{
		"chat",
		"api",
		"-m",
		opt,
	}
	return kb.executeJSON(command{name, arg})
}

func (kb *Keybase) executeJSON(cmd command) (err error) {
	response := []byte{}
	if response, err = exec.Command(cmd.name, cmd.arg...).Output(); err != nil {
		return err
	}
	if err = json.Unmarshal(response, &kb.response); err != nil {
		return err
	}
	return nil
}
