package service

type RequestBody struct {
	Action      string `json:"action"`
	MessageType string `json:"message_type"`
	UserId      int    `json:"user_id"`
	Message     string `json:"message"`
	AutoEscape  bool   `json:"auto_escape"`
}

type ResponseBody struct {
	Data struct {
		MessageId int `json:"message_id"`
	} `json:"data"`
}
