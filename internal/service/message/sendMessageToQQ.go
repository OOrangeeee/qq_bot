package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type RequestBody struct {
	Action      string `json:"action"`
	MessageType string `json:"message_type"`
	UserId      int    `json:"user_id"`
	GroupId     int    `json:"group_id"`
	Message     string `json:"message"`
	AutoEscape  bool   `json:"auto_escape"`
}

type ResponseBody struct {
	Data struct {
		MessageId int `json:"message_id"`
	} `json:"data"`
}

func SendMessageToQQ(messageType string, userId int, groupId int, message string) error {
	reqBody := RequestBody{
		Action:      "send_msg",
		MessageType: messageType,
		UserId:      userId,
		GroupId:     groupId,
		Message:     message,
		AutoEscape:  false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("请求数据序列化失败")
		return err
	}
	req, err := http.NewRequest("POST", config.Config.AppConfig.QQ.BotUrl+"/send_private_msg", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("创建请求失败")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.Config.AppConfig.QQ.BotToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("发送请求失败")
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("关闭请求失败")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("读取响应体失败")
		return err
	}
	var respBody ResponseBody
	if err := json.Unmarshal(body, &respBody); err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("响应体序列化失败")
		return err
	}
	// message_id目前没有用到，暂时不处理
	return nil
}
