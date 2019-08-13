package bridge

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	utime "time"

	"github.com/nlopes/slack"
)

type Keybase struct {
	response api
	history  map[string][]message
}

type api struct {
	Result struct {
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

type history = slack.HistoryParameters

func NewKeybase() *Keybase {
	kb := Keybase{}
	kb.history = make(map[string][]message)
	return &kb
}

func (kb *Keybase) GetChannelHistory(team, channel string, param history) (hist map[string][]message, err error) {
	var resp []byte
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
						"num":%d
					}
				}
			}
		}`, team, channel, param.Count)
	cmd := "keybase"
	args := []string{
		"chat",
		"api",
		"-m",
		opt,
	}
	if resp, err = exec.Command(cmd, args...).Output(); err != nil {
		return kb.history, err
	}
	if err := json.Unmarshal(resp, &kb.response); err != nil {
		return kb.history, err
	}
	for idx := 0; idx < param.Count; idx++ {
		meta := make([]string, 0)
		body := kb.response.Result.Messages[idx].Msg.Content.Text.Body
		re := regexp.MustCompile(`\[([^\[\]]*)\]`)
		if submatches := re.FindAllString(body, -1); len(submatches) > 0 {
			for _, element := range submatches {
				element = strings.Trim(element, "[")
				element = strings.Trim(element, "]")
				meta = append(meta, element)
			}
			time, _ := utime.Parse("2006-01-02 15:04:05.999999999 -0700 MST", meta[0])
			name := meta[1]
			text := ""
			text = strings.Split(body, "["+meta[1]+"]")[1]
			text = strings.TrimSpace(text)
			msg := message{time, channel, name, text}
			kb.history[channel] = append(kb.history[channel], msg)
		}
	}
	return kb.history, nil
}
