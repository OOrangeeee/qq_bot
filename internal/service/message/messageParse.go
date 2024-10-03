package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	"GitHubBot/internal/model"
	githubService "GitHubBot/internal/service/github"
	llmService "GitHubBot/internal/service/llm"
	"GitHubBot/internal/util"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	message_type := event.MessageType
	fromIdInt := event.UserId
	groupIdInt := event.GroupId
	fromId := strconv.Itoa(int(fromIdInt))
	if repos, ok := matchGithubGet(message); ok {
		var ans string
		if message_type == "group" {
			ans += "[CQ:at,qq=" + fromId + "]\n"
		}
		// 编写ans
		for _, repoName := range repos {
			ifExist, err := database.Redis.IfRepoExist(repoName)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！查询仓库是否存在失败呜呜呜",
				})
			}
			if !ifExist {
				ans += fmt.Sprintf("+++++\n数据库中没有仓库 %s 信息\n+++++\n", repoName)
				ans += "-\n"
				continue
			}
			repo, err := database.Redis.GetRepo(repoName)
			if (err != nil && !errors.Is(err, gorm.ErrRecordNotFound)) || repo == nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！从数据库获取仓库信息失败呜呜呜",
				})
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				ans += fmt.Sprintf("+++++\n数据库中没有仓库 %s 信息\n+++++\n", repoName)
				ans += "-\n"
				continue
			}
			ansTmp, err := githubService.GetInfoOfRepo(repo.RepoName, repo.Url)
			if err != nil {
				ans += fmt.Sprintf("+++++\n获取仓库 %s 信息失败, Url为 %s \n+++++\n", repoName, repo.Url)
				ans += "\n"
				continue
			}
			ans += ansTmp
			ans += "-\n"
		}
		if message_type == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if setItem, ok := matchGithubSet(message); ok {
		repoName := setItem[0]
		repoUrl := setItem[1]
		err := database.Redis.AddNewRepo(&database.GbRepos{
			Token:    util.GenerateUUID(),
			RepoName: repoName,
			Url:      repoUrl,
		})
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！数据库中已经有这个仓库信息了呜呜呜",
			})
		}
		if message_type == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), fmt.Sprintf("[CQ:at,qq=%s]\n"+"橙子报告！添加仓库 %s 成功！！！", fromId, repoName))
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), fmt.Sprintf("橙子报告！添加仓库 %s 成功！！！", repoName))
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if strings.EqualFold(message, "/gb-get-names") {
		// 获取所有仓库名
		allNames, err := database.Redis.GetAllReposNames()
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！获取所有仓库名失败呜呜呜",
			})
		}
		var ans string
		if message_type == "group" {
			ans += "[CQ:at,qq=" + fromId + "]\n"
		}
		for _, name := range allNames {
			ans += fmt.Sprintf("+++++\n%s\n", name)
		}
		if message_type == "group" {
			err = SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if delItem, ok := matchGithubDel(message); ok {
		// 删除仓库信息
		var ans string
		if message_type == "group" {
			ans += "[CQ:at,qq=" + fromId + "]\n"
		}
		for _, repoName := range delItem {
			ifExist, err := database.Redis.IfRepoExist(repoName)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！查询仓库是否存在失败呜呜呜",
				})
			}
			if !ifExist {
				ans += fmt.Sprintf("+++++\n数据库中没有仓库 %s 信息\n+++++\n", repoName)
				ans += "-\n"
				continue
			}
			repo, err := database.Redis.GetRepo(repoName)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				ans += fmt.Sprintf("+++++\n从数据库获取仓库 %s 信息失败\n+++++\n", repoName)
			} else if errors.Is(err, gorm.ErrRecordNotFound) || repo == nil {
				ans += fmt.Sprintf("+++++\n数据库中没有仓库 %s 信息\n+++++\n", repoName)
			}
			err = database.Redis.DeleteRepo(repo)
			if err != nil {
				ans += fmt.Sprintf("+++++\n删除仓库 %s 信息失败\n+++++\n", repoName)
			}
			ans += fmt.Sprintf("+++++\n删除仓库 %s 信息成功\n+++++\n", repoName)
			ans += "-\n"
		}
		if message_type == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("回复消息失败")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else {
		// 存储消息
		qq, err := strconv.Atoi(config.Config.AppConfig.QQ.BotQQ)
		log.Log.Info("qq:" + strconv.Itoa(qq))
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("qq转换失败")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！qq转换失败呜呜呜",
			})
		}
		newMessage := &database.Message{
			Token:  util.GenerateUUID(),
			FromId: int(fromIdInt),
			ToId:   qq,
			Text:   message,
			Time:   time.Now(),
		}
		err = database.Redis.AddNewMessage(newMessage)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("存储消息失败")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！存储消息失败呜呜呜",
			})
		}
		var ans string
		if message_type == "group" {
			if strings.Contains(message, "[CQ:at,qq="+config.Config.AppConfig.QQ.BotQQ+"]") {
				ans += "[CQ:at,qq=" + fromId + "]\n"
			} else {
				return c.JSON(http.StatusOK, map[string]interface{}{})
			}
		}
		var messageSend []llmService.Message
		messageSend = make([]llmService.Message, 0)
		messageSend = append(messageSend, llmService.Message{
			Role:    "user",
			Content: config.Config.AppConfig.Character.Describe,
		})
		vipStr := config.Config.AppConfig.Llm.VipQQ
		vips := strings.Split(vipStr, ",")
		isVip := false
		for _, vip := range vips {
			if vip == fromId {
				isVip = true
				break
			}
		}
		if isVip {
			messageSend = append(messageSend, llmService.Message{
				Role:    "user",
				Content: config.Config.AppConfig.Llm.VipMessage,
			})
		}
		messageSend = append(messageSend, llmService.Message{
			Role:    "user",
			Content: message,
		})
		getMessages, err := database.Redis.GetMessages(int(fromIdInt), qq)
		sendMessages, err := database.Redis.GetMessages(qq, int(fromIdInt))
		if getMessages != nil {
			for _, messageTmp1 := range *getMessages {
				messageSend = append(messageSend, llmService.Message{
					Role:    "user",
					Content: messageTmp1.Text,
				})
				log.Log.Info("user: " + messageTmp1.Text)
			}
		}
		if sendMessages != nil {
			for _, messageTmp2 := range *sendMessages {
				messageSend = append(messageSend, llmService.Message{
					Role:    "assistant",
					Content: messageTmp2.Text,
				})
				log.Log.Info("assistant: " + messageTmp2.Text)
			}
		}
		ansTmp, err := llmService.SendMessage(config.Config.AppConfig.Llm.Secret, messageSend)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("发送消息失败")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！发送消息失败呜呜呜",
			})
		}
		newMessage = &database.Message{
			Token:  util.GenerateUUID(),
			FromId: qq,
			ToId:   int(fromIdInt),
			Text:   ansTmp,
			Time:   time.Now(),
		}
		err = database.Redis.AddNewMessage(newMessage)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("存储消息失败")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！存储消息失败呜呜呜",
			})
		}
		ans += ansTmp
		if message_type == "group" {
			err = SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
		}
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("回复消息失败")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！回复消息失败呜呜呜",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	}
}

// matchGithubGet 检查字符串是否符合 "/gb-get *****" 模式，并返回匹配后的非空格字符串切片和匹配结果
func matchGithubGet(input string) ([]string, bool) {
	// 编译正则表达式，用于捕获非空格字符串
	regex, err := regexp.Compile(`^/gb-get\s+(.+)$`)
	if err != nil {
		fmt.Println("Invalid regex:", err)
		return nil, false
	}
	// 检查字符串是否匹配正则表达式
	if regex.MatchString(input) {
		// 使用 FindStringSubmatch 获取所有匹配的部分，其中第一个元素是整个匹配，后续元素是捕获的子表达式
		matches := regex.FindStringSubmatch(input)
		if len(matches) > 1 {
			// 使用 strings.Fields 分割字符串
			fields := strings.Fields(matches[1])
			return fields, true
		}
	}
	return nil, false
}

// matchGithubSet 检查字符串是否符合 "/gb-set <name> <GitHub URL>" 模式，并返回匹配后的非空格字符串切片和匹配结果
func matchGithubSet(input string) ([]string, bool) {
	// 定义正则表达式，匹配 /gb-set 后跟两个非空格字符串，第二个字符串必须是GitHub仓库的URL
	regex, err := regexp.Compile(`^/gb-set\s+(\S+)\s+(https://github\.com/\S+/\S+)$`)
	if err != nil {
		fmt.Println("Invalid regex:", err)
		return nil, false
	}

	// 检查字符串是否匹配正则表达式
	if regex.MatchString(input) {
		// 使用 FindStringSubmatch 获取所有匹配的部分，其中第一个元素是整个匹配，后续元素是捕获的子表达式
		matches := regex.FindStringSubmatch(input)
		if len(matches) > 2 {
			// 返回捕获的两个子表达式（name 和 GitHub 仓库 URL）
			return matches[1:], true
		}
	}

	return nil, false
}

// matchGithubDel 检查字符串是否符合 "/gb-del *****" 模式，并返回匹配后的非空格字符串切片和匹配结果
func matchGithubDel(input string) ([]string, bool) {
	// 编译正则表达式，用于捕获任意字符（包括空格）的字符串
	regex, err := regexp.Compile(`^/gb-del\s+(.+)`)
	if err != nil {
		fmt.Println("Invalid regex:", err)
		return nil, false
	}
	// 检查字符串是否匹配正则表达式
	if regex.MatchString(input) {
		// 使用 FindStringSubmatch 获取所有匹配的部分
		matches := regex.FindStringSubmatch(input)
		if len(matches) > 1 {
			// 使用 strings.Fields 分割字符串
			fields := strings.Fields(matches[1])
			return fields, true
		}
	}
	return nil, false
}
