package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	"GitHubBot/internal/model"
	githubService "GitHubBot/internal/service/github"
	llmService "GitHubBot/internal/service/llm"
	weatherService "GitHubBot/internal/service/weather"
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
	// dividingLine
	dividingLine := "🍊🍊🍊🍊🍊\n"
	eventTmp := c.Get("event")
	event, ok := eventTmp.(*model.Event)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": "event解析失败",
		})
	}
	message := event.Message
	messageType := event.MessageType
	fromIdInt := event.UserId
	groupIdInt := event.GroupId
	fromId := strconv.Itoa(int(fromIdInt))
	if repos, ok := matchGithubGet(message); ok {
		var ans string
		if messageType == "group" {
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
				ans += dividingLine
				continue
			}
			repo, err := database.Redis.GetRepo(repoName)
			if (err != nil && !errors.Is(err, gorm.ErrRecordNotFound)) || repo == nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！从数据库获取仓库信息失败呜呜呜",
				})
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				ans += fmt.Sprintf("+++++\n数据库中没有仓库 %s 信息\n+++++\n", repoName)
				ans += dividingLine
				continue
			}
			ansTmp, err := githubService.GetInfoOfRepo(repo.RepoName, repo.Url)
			if err != nil {
				ans += fmt.Sprintf("+++++\n获取仓库 %s 信息失败, Url为 %s \n+++++\n", repoName, repo.Url)
				ans += dividingLine
				continue
			}
			ans += ansTmp
			ans += dividingLine
		}
		ans = ans[:len(ans)-len(dividingLine)]
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if strings.EqualFold(message, "/gb-get-all") {
		var ans string
		if messageType == "group" {
			ans += "[CQ:at,qq=" + fromId + "]\n"
		}
		names, err := database.Redis.GetAllReposNames()
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！获取所有仓库名失败呜呜呜",
			})
		}
		for _, name := range names {
			repo, err := database.Redis.GetRepo(name)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！获取仓库信息失败呜呜呜",
				})
			}
			ansTmp, err := githubService.GetInfoOfRepo(repo.RepoName, repo.Url)
			if err != nil {
				ans += fmt.Sprintf("获取仓库 %s 信息失败, Url为 %s \n", repo.RepoName, repo.Url)
				ans += "\n"
				continue
			}
			ans += ansTmp
			ans += dividingLine
		}
		// 删除最后一个分割线
		ans = ans[:len(ans)-len(dividingLine)]
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
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
			RepoName: repoName,
			Url:      repoUrl,
		})
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！数据库中已经有这个仓库信息了呜呜呜",
			})
		}
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), fmt.Sprintf("[CQ:at,qq=%s]\n"+"橙子报告！添加仓库 %s 成功！！！", fromId, repoName))
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), fmt.Sprintf("橙子报告！添加仓库 %s 成功！！！", repoName))
			if err != nil {
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
		if messageType == "group" {
			ans += "[CQ:at,qq=" + fromId + "]\n"
		}
		for _, name := range allNames {
			ans += fmt.Sprintf("+++++\n%s\n", name)
		}
		if len(allNames) == 0 {
			ans += "数据库中没有仓库信息"
		} else {
			ans += "+++++"
		}
		if messageType == "group" {
			err = SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if delItem, ok := matchGithubDel(message); ok {
		// 删除仓库信息
		var ans string
		if messageType == "group" {
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
				ans += dividingLine
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
			ans += dividingLine
		}
		ans = ans[:len(ans)-len(dividingLine)]
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if strings.EqualFold(message, "/chat-clear") {
		// 转化qq为整数
		qq, err := strconv.Atoi(config.Config.AppConfig.QQ.BotQQ)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！qq转换失败呜呜呜",
			})
		}
		// 清空聊天记录
		getMessages, err := database.Redis.GetMessages(int(fromIdInt), qq)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！从数据库获取消息失败呜呜呜",
			})
		}
		sendMessages, err := database.Redis.GetMessages(qq, int(fromIdInt))
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！从数据库获取消息失败呜呜呜",
			})
		}
		if getMessages != nil {
			for _, messageTmp := range *getMessages {
				err = database.Redis.DeleteMessage(messageTmp)
				if err != nil {
					return c.JSON(http.StatusOK, map[string]interface{}{
						"reply": "橙子报告！删除消息失败呜呜呜",
					})
				}
			}
		}
		if sendMessages != nil {
			for _, messageTmp := range *sendMessages {
				err = database.Redis.DeleteMessage(messageTmp)
				if err != nil {
					return c.JSON(http.StatusOK, map[string]interface{}{
						"reply": "橙子报告！删除消息失败呜呜呜",
					})
				}
			}
		}
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), fmt.Sprintf("[CQ:at,qq=%s]\n"+"橙子报告！清空聊天记录成功！！！", fromId))
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), "橙子报告！清空聊天记录成功！！！")
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if stringTmp, ok := matchWeatherSet(message); ok {
		city := stringTmp[0]
		address := stringTmp[1]
		diliCode, err := weatherService.GetDiLiCode(address, city)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！获取地理编码失败呜呜呜",
			})
		}
		err = database.Redis.AddNewCity(&database.City{
			City:     city,
			DiLiCode: diliCode,
		})
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！城市信息已经存在呜呜呜",
			})
		}
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), fmt.Sprintf("[CQ:at,qq=%s]\n"+"橙子报告！添加城市 %s 成功！！！", fromId, city))
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), fmt.Sprintf("橙子报告！添加城市 %s 成功！！！", city))
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if cities, ok := matchWeatherGet(message); ok {
		var ans string
		if messageType == "group" {
			ans += "[CQ:at,qq=" + fromId + "]\n"
		}
		for _, city := range cities {
			cityInfo, err := database.Redis.GetCity(city)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				ans += fmt.Sprintf("+++++\n从数据库获取城市 %s 信息失败\n+++++\n", city)
				ans += dividingLine
				continue
			} else if errors.Is(err, gorm.ErrRecordNotFound) || cityInfo == nil {
				ans += fmt.Sprintf("+++++\n数据库中没有城市 %s 信息\n+++++\n", city)
				ans += dividingLine
				continue
			}
			code := cityInfo.DiLiCode
			weatherInfo, err := weatherService.GetWeather(code, "all")
			if err != nil {
				ans += fmt.Sprintf("+++++\n获取城市 %s 天气信息失败\n+++++\n", city)
				ans += dividingLine
				continue
			}
			ans += weatherInfo
			ans += dividingLine
		}
		ans = ans[:len(ans)-len(dividingLine)]
		if messageType == "group" {
			err := SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		} else {
			err := SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
			if err != nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！回复消息失败呜呜呜",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else if city, ok := matchChatWeatherGet(message); ok {
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
		cityInfo, err := database.Redis.GetCity(city)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！从数据库获取城市信息失败呜呜呜",
			})
		} else if errors.Is(err, gorm.ErrRecordNotFound) || cityInfo == nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！数据库中没有城市信息呜呜呜",
			})
		}
		code := cityInfo.DiLiCode
		weatherInfo, err := weatherService.GetWeather(code, "all")
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！获取天气信息失败呜呜呜",
			})
		}
		var ans string
		if messageType == "group" {
			if strings.Contains(message, "[CQ:at,qq="+config.Config.AppConfig.QQ.BotQQ+"]") {
				ans += "[CQ:at,qq=" + fromId + "]\n"
			} else {
				return c.JSON(http.StatusOK, map[string]interface{}{})
			}
		}
		messageSend = append(messageSend, llmService.Message{
			Role:    "user",
			Content: weatherInfo + config.Config.AppConfig.Llm.WeatherMessage,
		})
		ansTmp, err := llmService.SendMessage(config.Config.AppConfig.Llm.Secret, messageSend)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！发送消息失败呜呜呜",
			})
		}
		ans += ansTmp
		if messageType == "group" {
			err = SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
		}
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！回复消息失败呜呜呜",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{})
	} else {
		// 转化qq为整数
		qq, err := strconv.Atoi(config.Config.AppConfig.QQ.BotQQ)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！qq转换失败呜呜呜",
			})
		}
		var ans string
		if messageType == "group" {
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
		getMessages, err := database.Redis.GetMessages(int(fromIdInt), qq)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！从数据库获取消息失败呜呜呜",
			})
		}
		sendMessages, err := database.Redis.GetMessages(qq, int(fromIdInt))
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！从数据库获取消息失败呜呜呜",
			})
		}
		if getMessages != nil && sendMessages != nil {
			if len(*getMessages) != len(*sendMessages) {
				log.Log.WithFields(logrus.Fields{
					"len(*getMessages)":  len(*getMessages),
					"len(*sendMessages)": len(*sendMessages),
				}).Error("从数据库获取消息有误")
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！从数据库获取消息有误呜呜呜",
				})
			}
			length := len(*getMessages)
			if length > 3 {
				length = 3
			}
			for i := length - 1; i >= 0; i-- {
				messageTmp1 := (*getMessages)[i]
				messageSend = append(messageSend, llmService.Message{
					Role:    "user",
					Content: messageTmp1.Text,
				})
				messageTmp2 := (*sendMessages)[i]
				messageSend = append(messageSend, llmService.Message{
					Role:    "assistant",
					Content: messageTmp2.Text,
				})
			}
		}
		messageSend = append(messageSend, llmService.Message{
			Role:    "user",
			Content: message,
		})
		ansTmp, err := llmService.SendMessage(config.Config.AppConfig.Llm.Secret, messageSend)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！发送消息失败呜呜呜",
			})
		}
		newMessage := &database.Message{
			FromId: int(fromIdInt),
			ToId:   qq,
			Text:   message,
			Time:   time.Now(),
		}
		err = database.Redis.AddNewMessage(newMessage)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！存储消息失败呜呜呜",
			})
		}
		newMessage = &database.Message{
			FromId: qq,
			ToId:   int(fromIdInt),
			Text:   ansTmp,
			Time:   time.Now(),
		}
		err = database.Redis.AddNewMessage(newMessage)
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！存储消息失败呜呜呜",
			})
		}
		ans += ansTmp
		if messageType == "group" {
			err = SendMessageToQQ("group", int(fromIdInt), int(groupIdInt), ans)
		} else {
			err = SendMessageToQQ("private", int(fromIdInt), int(groupIdInt), ans)
		}
		if err != nil {
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

// matchWeatherSet 检查字符串是否符合 "/weather-set <city> <location>" 模式，并返回匹配后的非空格字符串切片和匹配结果
func matchWeatherSet(input string) ([]string, bool) {
	// 编译正则表达式，用于确保格式正确，只有两个字符串跟随 `/weather-set`
	regex, err := regexp.Compile(`^/weather-set\s+(\S+)\s+(\S+)$`)
	if err != nil {
		fmt.Println("Invalid regex:", err)
		return nil, false
	}
	// 检查字符串是否精确匹配正则表达式
	if regex.MatchString(input) {
		// 使用 FindStringSubmatch 获取所有匹配的部分，其中第一个元素是整个匹配，后续元素是捕获的子表达式
		matches := regex.FindStringSubmatch(input)
		// 检查是否确实有两个捕获组（加上整个匹配的元素共三个）
		if len(matches) == 3 {
			// 返回捕获的字符串和 true
			return matches[1:], true
		}
	}
	// 如果不符合正则表达式，或捕获组数量不对，返回空切片和 false
	return nil, false
}

// matchWeatherGet 检查字符串是否符合 "/gb-get *****" 模式，并返回匹配后的非空格字符串切片和匹配结果
func matchWeatherGet(input string) ([]string, bool) {
	// 编译正则表达式，用于捕获非空格字符串
	regex, err := regexp.Compile(`^/weather-get\s+(.+)$`)
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

// matchChatWeatherGet 检查字符串是否符合 "/chat-weather-get *****" 模式，并返回匹配后的非空格字符串和匹配结果
func matchChatWeatherGet(input string) (string, bool) {
	// 编译正则表达式，确保格式正确：`/chat-weather-get` 后仅跟一个字符串
	regex, err := regexp.Compile(`^/chat-weather-get\s+(\S+)$`)
	if err != nil {
		fmt.Println("Invalid regex:", err)
		return "", false
	}
	// 检查字符串是否精确匹配正则表达式
	if regex.MatchString(input) {
		// 使用 FindStringSubmatch 获取所有匹配的部分，其中第一个元素是整个匹配，第二个元素是捕获的子表达式
		matches := regex.FindStringSubmatch(input)
		// 检查是否确实只有一个捕获组（加上整个匹配的元素共两个）
		if len(matches) == 2 {
			// 返回捕获的字符串和 true
			return matches[1], true
		}
	}
	// 如果不符合正则表达式，或捕获组数量不对，返回空切片和 false
	return "", false
}
