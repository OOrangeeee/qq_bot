package service

import (
	"GitHubBot/internal/database"
	"GitHubBot/internal/model"
	service "GitHubBot/internal/service/github"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
	"strings"
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
	if repos, ok := matchGithubGet(message); ok {
		var ans string
		for _, repoName := range repos {
			repo, err := database.Redis.GetRepo(repoName)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": "获取仓库信息失败",
				})
			}
			ansTmp, err := service.GetInfoOfRepo(repo.Url)
			if err != nil {
				ans += fmt.Sprintf("+++++\n获取仓库 %s 信息失败\n+++++\n", repoName)
				ans += "\n"
				continue
			}
			ans += ansTmp
			ans += "\n"
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": ans,
		})
	} else if setItem, ok := matchGithubSet(message); ok {
		repoName := setItem[0]
		repoUrl := setItem[1]
		err := database.Redis.AddNewRepo(&model.GbRepos{
			RepoName: repoName,
			Url:      repoUrl,
		})
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "添加仓库信息失败",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": fmt.Sprintf("橙子报告！添加仓库 %s 成功！！！", repoName),
		})
	} else if strings.EqualFold(message, "/gb-get-all") {
		// 获取所有仓库信息
		allNames, err := database.Redis.GetAllReposName()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "获取所有仓库名失败",
			})
		}
		var ans string
		for _, name := range allNames {
			repo, err := database.Redis.GetRepo(name)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": "获取仓库信息失败",
				})
			}
			ansTmp, err := service.GetInfoOfRepo(repo.Url)
			if err != nil {
				ans += fmt.Sprintf("+++++\n获取仓库 %s 信息失败\n+++++\n", name)
				continue
			}
			ans += ansTmp
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": ans,
		})
	} else if delItem, ok := matchGithubDel(message); ok {
		// 删除仓库信息
		for _, repoName := range delItem {
			repo, err := database.Redis.GetRepo(repoName)
			if err != nil {
				continue
			}
			if repo == nil {
				continue
			}
			err = database.Redis.DeleteRepo(repo)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": "删除仓库信息失败",
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": "橙子报告！删除仓库信息成功！！！",
		})
	} else if strings.Contains(message, "宁静") || strings.Contains(message, "nxj") || strings.Contains(message, "宁小静") || strings.Contains(message, "柠檬头") || strings.Contains(message, "柠檬") || strings.Contains(message, "lemon") || strings.Contains(message, "Lemon") || strings.Contains(message, "nmt") {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": "你刚刚是提到宁静了吗？其实啊！宁静是个大笨蛋！！宇宙超级无敌大笨蛋哇嘎嘎嘎嘎嘎！！！",
		})
	} else if strings.Contains(message, "狗") || strings.Contains(message, "dog") || strings.Contains(message, "Dog") {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": "狗狗是人类的好朋友，不要随便说狗狗坏话哦！",
		})
	} else if strings.Contains(message, "晋晨曦") || strings.Contains(message, "晋馋曦") || strings.Contains(message, "橙子") || strings.Contains(message, "小橙狗") || strings.Contains(message, "小馋狗") || strings.Contains(message, "jcx") || strings.Contains(message, "Jcx") {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": "你刚刚是提到晋晨曦了吗？你怎么认识世界上最可爱的快乐小狗的？他真的超级聪明，超级可爱，超级快乐哦！",
		})
	} else {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "无效指令",
		})
	}
}

// matchGithubGet 检查字符串是否符合 "/gb-get *****" 模式，并返回匹配后的非空格字符串切片和匹配结果
func matchGithubGet(input string) ([]string, bool) {
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
	// 编译正则表达式，用于捕获非空格字符串
	regex, err := regexp.Compile(`^/gb-del\s+(\S+)$`)
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
