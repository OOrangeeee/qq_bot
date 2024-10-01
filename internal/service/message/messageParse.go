package service

import (
	"GitHubBot/internal/model"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
)

func MessageParse(c echo.Context) error {
	eventTmp := c.Get("event")
	event, ok := eventTmp.(*model.Event)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": "event解析失败",
		})
	}
	message := event.Message
	if repos, ok := checkPattern(message); ok {
		for _, repo := range repos {

		}
	}
}

// checkPattern 检查字符串是否符合 "/gb-get *****" 模式，并返回匹配后的非空格字符串切片和匹配结果
func checkPattern(input string) ([]string, bool) {
	// 编译正则表达式，用于捕获非空格字符串
	regex, err := regexp.Compile(`^/gb-get\s+(\S+)$`)
	if err != nil {
		fmt.Println("Invalid regex:", err)
		return nil, false
	}

	// 检查字符串是否匹配正则表达式
	if regex.MatchString(input) {
		// 使用 FindStringSubmatch 获取所有匹配的部分，其中第一个元素是整个匹配，后续元素是捕获的子表达式
		matches := regex.FindStringSubmatch(input)
		if len(matches) > 1 {
			// 返回捕获的子表达式（非空格字符串）
			return matches[1:], true
		}
	}
	return nil, false
}
