package util

import (
	"GitHubBot/internal/model"
	"encoding/json"
)

func ParseEvent(data []byte) (*model.Event, error) {
	var event model.Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		// 如果解析出错，返回一个新的零值结构体和错误信息
		return &model.Event{}, err
	}
	return &event, nil
}
