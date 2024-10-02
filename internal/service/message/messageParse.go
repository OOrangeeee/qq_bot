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
			if err != nil || repo == nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"reply": "橙子报告！从数据库获取仓库信息失败呜呜呜",
				})
			}
			ansTmp, err := service.GetInfoOfRepo(repo.RepoName, repo.Url)
			if err != nil {
				ans += fmt.Sprintf("+++++\n获取仓库 %s 信息失败, Url为 %s \n+++++\n", repoName, repo.Url)
				ans += "\n"
				continue
			}
			ans += ansTmp
			ans += "-\n"
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
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！数据库中已经有这个仓库信息了呜呜呜",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": fmt.Sprintf("橙子报告！添加仓库 %s 成功！！！", repoName),
		})
	} else if strings.EqualFold(message, "/gb-get-names") {
		// 获取所有仓库名
		allNames, err := database.Redis.GetAllReposNames()
		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"reply": "橙子报告！获取所有仓库名失败呜呜呜",
			})
		}
		var ans string
		for _, name := range allNames {
			ans += fmt.Sprintf("+++++\n%s\n", name)
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": ans,
		})
	} else if delItem, ok := matchGithubDel(message); ok {
		// 删除仓库信息
		var ans string
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
			if err != nil {
				ans += fmt.Sprintf("+++++\n从数据库获取仓库 %s 信息失败\n+++++\n", repoName)
			}
			if repo == nil {
				ans += fmt.Sprintf("+++++\n数据库中没有仓库 %s 信息\n+++++\n", repoName)
			}
			err = database.Redis.DeleteRepo(repo)
			if err != nil {
				ans += fmt.Sprintf("+++++\n删除仓库 %s 信息失败\n+++++\n", repoName)
			}
			ans += fmt.Sprintf("+++++\n删除仓库 %s 信息成功\n+++++\n", repoName)
			ans += "-\n"
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": ans,
		})
	} else {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"reply": "无效指令",
		})
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
