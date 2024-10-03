package model

type Event struct {
	Time        uint   `json:"time"`
	SelfId      uint   `json:"self_id"`
	PostType    string `json:"post_type"`
	MessageType string `json:"message_type"`
	SubType     string `json:"sub_type"`
	MessageId   uint   `json:"message_id"`
	UserId      uint   `json:"user_id"`
	Message     string `json:"message"`
	RawMessage  string `json:"raw_message"`
	Font        int    `json:"font"`
	Sender      struct {
		Nickname string `json:"nickname"`
		Sex      string `json:"sex"`
		Age      int    `json:"age"`
	} `json:"sender"`
}
