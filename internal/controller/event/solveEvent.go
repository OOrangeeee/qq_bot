package controller

import (
	eventService "GitHubBot/internal/service/event"
	service "GitHubBot/internal/service/message"
	"github.com/labstack/echo/v4"
	"net/http"
)

func SolveEvent(c echo.Context) error {
	eventType, err := eventService.GetEvent(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}
	switch eventType {
	case "message":
		return service.MessageParse(c)
	default:
		return c.JSON(http.StatusNoContent, map[string]interface{}{
			"error": "未知事件类型",
		})
	}
}
