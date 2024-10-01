package service

import (
	"GitHubBot/internal/log"
	"GitHubBot/internal/model"
	"GitHubBot/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"io"
)

func GetEvent(c echo.Context) (string, error) {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("读取请求体失败")
	}
	var event *model.Event
	event, err = util.ParseEvent(body)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("解析事件失败")
		return "", err
	}
	c.Set("event", event)
	return event.MessageType, nil
}
