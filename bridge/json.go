package bridge

type KeybaseApi struct {
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
