package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// CommitInfo 定义提交信息的结构
type CommitInfo struct {
	Commit struct {
		Author struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

// Branch 定义分支信息的结构
type Branch struct {
	Name   string `json:"name"`
	Commit struct {
		Sha string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}

type GitHubHelperMainBranchForJson struct {
	Name             string `json:"MainBranchName"`
	CommitNum        int    `json:"MainBranchCommitNum"`
	LatestCommitTime string `json:"MainBranchLatestCommitTime"`
	MainContributor  string `json:"MainContributor"`
}

type GitHubHelperBranchForJson struct {
	Name             string `json:"BranchName"`
	LatestCommitMsg  string `json:"LatestCommitMsg"`
	LatestCommitTime string `json:"LatestCommitTime"`
}

// GitHubHelperInfoForJson 定义返回的 JSON 结构
type GitHubHelperInfoForJson struct {
	Name       string                        `json:"RepoName"`
	Url        string                        `json:"RepoUrl"`
	Owner      string                        `json:"Owner"`
	MainBranch GitHubHelperMainBranchForJson `json:"MainBranch"`
	Branches   []GitHubHelperBranchForJson   `json:"Branches"`
}

func GetJsonInfoOfRepo(url string) (string, error) {
	// 从 url 中提取 owner 和 repo 信息
	owner, repo := extractOwnerRepo(url)
	// 获取所有分支信息
	branches, err := getBranches(owner, repo)
	if err != nil {
		return "", err
	}
	// 优先处理 main 和 master 分支
	prioritizedBranch, branches := prioritizeMainOrMasterBranch(branches)
	// 获取所有提交信息
	commits, err := getAllCommits(owner, repo)
	if err != nil {
		return "", err
	}
	// 统计仓库的提交总数
	commitNum := len(commits)

	// 计算最新提交时间
	latestCommitTime := getLatestCommitFromCommits(commits)

	// 将最新提交时间转换为中国大陆时间
	loc, _ := time.LoadLocation("Asia/Shanghai")
	latestCommitTimeInChina := latestCommitTime.In(loc)

	// 获取仓库贡献最多的提交者
	mainContributor := getMainContributorFromCommits(commits)

	jsonBranches := make([]GitHubHelperBranchForJson, 0)

	for _, branch := range branches {
		lastCommitMsg, lastCommitDate, err := getBranchCommitInfo(branch.Commit.URL)
		if err != nil {
			return "", err
		}
		// 将最后一次提交时间转换为中国大陆时间
		lastCommitDateInChina := lastCommitDate.In(loc)
		jsonBranches = append(jsonBranches, GitHubHelperBranchForJson{
			Name:             branch.Name,
			LatestCommitMsg:  lastCommitMsg,
			LatestCommitTime: lastCommitDateInChina.Format("2006-01-02 15:04:05"),
		})
	}

	// 得到返回结果结构体
	jsonInfo := GitHubHelperInfoForJson{
		Name:  repo,
		Url:   url,
		Owner: owner,
		MainBranch: GitHubHelperMainBranchForJson{
			Name:             prioritizedBranch,
			CommitNum:        commitNum,
			LatestCommitTime: latestCommitTimeInChina.Format("2006-01-02 15:04:05"),
			MainContributor:  mainContributor,
		},
		Branches: jsonBranches,
	}

	// 将结果结构体转换为 JSON 字符串
	jsonBytes, err := json.Marshal(jsonInfo)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func GetInfoOfRepo(name, url string) (string, error) {
	// 从 url 中提取 owner 和 repo 信息
	owner, repo := extractOwnerRepo(url)

	// 获取所有分支信息
	branches, err := getBranches(owner, repo)
	if err != nil {
		return "", err
	}

	// 优先处理 main 和 master 分支
	prioritizedBranch, branches := prioritizeMainOrMasterBranch(branches)

	// 获取所有提交信息
	commits, err := getAllCommits(owner, repo)
	if err != nil {
		return "", err
	}

	// 统计仓库的提交总数
	commitNum := len(commits)

	// 计算最新提交时间
	latestCommitTime := getLatestCommitFromCommits(commits)

	// 将最新提交时间转换为中国大陆时间
	loc, _ := time.LoadLocation("Asia/Shanghai")
	latestCommitTimeInChina := latestCommitTime.In(loc)

	// 获取仓库贡献最多的提交者
	mainContributor := getMainContributorFromCommits(commits)

	// 拼接结果字符串
	result := "+++++\n"
	result += fmt.Sprintf("%s\n", name)
	result += fmt.Sprintf("仓库名称: %s\n", repo)
	result += fmt.Sprintf("仓库URL: %s\n", url)

	// 如果存在主分支，显示其信息
	if prioritizedBranch != "" {
		result += fmt.Sprintf("主分支名称: %s\nMain分支提交总数: %d\n最近一次提交: %s\n最佳贡献者: %s\n",
			prioritizedBranch, commitNum, latestCommitTimeInChina.Format("2006-01-02 15:04:05"), mainContributor)
	} else {
		result += "没有找到主分支（main 或 master）。\n"
	}

	// 遍历每个分支并获取最近一次提交的 message 和时间
	for _, branch := range branches {
		lastCommitMsg, lastCommitDate, err := getBranchCommitInfo(branch.Commit.URL)
		if err != nil {
			return "", err
		}
		// 将最后一次提交时间转换为中国大陆时间
		lastCommitDateInChina := lastCommitDate.In(loc)
		result += fmt.Sprintf("分支-%s: 最近一次提交: \"%s\" 于 %s\n", branch.Name, lastCommitMsg, lastCommitDateInChina.Format("2006-01-02 15:04:05"))
	}

	result += "+++++\n"
	return result, nil
}

// extractOwnerRepo 从 url 提取 owner 和 repo 信息
func extractOwnerRepo(url string) (string, string) {
	parts := strings.Split(url, "/")
	return parts[len(parts)-2], parts[len(parts)-1]
}

// getAllCommits 一次性获取仓库所有的提交信息，支持分页
func getAllCommits(owner, repo string) ([]CommitInfo, error) {
	var allCommits []CommitInfo
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf(config.Config.AppConfig.Github.ApiUrl+"/repos/%s/%s/commits?per_page=%d&page=%d", owner, repo, perPage, page)
		resp, err := makeGitHubRequest(url)
		if err != nil {
			return nil, err
		}

		var commits []CommitInfo
		if err := json.Unmarshal(resp, &commits); err != nil {
			return nil, err
		}

		if len(commits) == 0 {
			break
		}

		allCommits = append(allCommits, commits...)
		page++
	}

	return allCommits, nil
}

// getLatestCommitFromCommits 计算所有提交中的最新提交时间
func getLatestCommitFromCommits(commits []CommitInfo) time.Time {
	latestCommitTime := time.Time{}
	for _, commit := range commits {
		if commit.Commit.Author.Date.After(latestCommitTime) {
			latestCommitTime = commit.Commit.Author.Date
		}
	}
	return latestCommitTime
}

// getMainContributorFromCommits 计算仓库的贡献最多者
func getMainContributorFromCommits(commits []CommitInfo) string {
	contributors := make(map[string]int)
	for _, commit := range commits {
		contributors[commit.Author.Login]++
	}
	mainContributor := ""
	maxCommits := 0
	for contributor, count := range contributors {
		if count > maxCommits {
			mainContributor = contributor
			maxCommits = count
		}
	}
	return mainContributor
}

// getBranches 获取仓库的所有分支信息，支持分页
func getBranches(owner, repo string) ([]Branch, error) {
	var allBranches []Branch
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf(config.Config.AppConfig.Github.ApiUrl+"/repos/%s/%s/branches?per_page=%d&page=%d", owner, repo, perPage, page)
		resp, err := makeGitHubRequest(url)
		if err != nil {
			return nil, err
		}
		var branches []Branch
		if err := json.Unmarshal(resp, &branches); err != nil {
			return nil, err
		}
		if len(branches) == 0 {
			break
		}
		allBranches = append(allBranches, branches...)
		page++
	}
	return allBranches, nil
}

// prioritizeMainOrMasterBranch 优先处理 main 或 master 分支
func prioritizeMainOrMasterBranch(branches []Branch) (string, []Branch) {
	var mainBranch Branch
	var masterBranch Branch
	var otherBranches []Branch

	for _, branch := range branches {
		if branch.Name == "main" {
			mainBranch = branch
		} else if branch.Name == "master" {
			masterBranch = branch
		} else {
			otherBranches = append(otherBranches, branch)
		}
	}

	if mainBranch.Name != "" {
		return "main", append([]Branch{mainBranch}, otherBranches...)
	} else if masterBranch.Name != "" {
		return "master", append([]Branch{masterBranch}, otherBranches...)
	}
	return "", branches
}

// getBranchCommitInfo 使用分支的 Commit.URL 获取最新提交信息
func getBranchCommitInfo(commitURL string) (string, time.Time, error) {
	resp, err := makeGitHubRequest(commitURL)
	if err != nil {
		return "", time.Time{}, err
	}

	var commit CommitInfo
	if err := json.Unmarshal(resp, &commit); err != nil {
		return "", time.Time{}, err
	}

	return commit.Commit.Message, commit.Commit.Author.Date, nil
}

// makeGitHubRequest 封装 API 请求
func makeGitHubRequest(url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+config.Config.AppConfig.Github.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("关闭响应体失败")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data from GitHub API, status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
