package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// 全局变量存储 GitHub 认证 token
var Token = "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"

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

// RepoInfo 实现核心函数
func GetInfoOfRepo(url string) (string, error) {
	owner, repo := extractOwnerRepo(url)

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

	// 获取所有分支信息
	branches, err := getBranches(owner, repo)
	if err != nil {
		return "", err
	}

	// 获取仓库贡献最多的提交者
	mainContributor := getMainContributorFromCommits(commits)

	// 开始拼接结果字符串
	result := fmt.Sprintf("+++++\nName: %s\nMain Commit Num: %d\nLast Commit: %s\nBest Contributor: %s\n", repo, commitNum, latestCommitTimeInChina.Format("2006-01-02 15:04:05"), mainContributor)

	// 遍历每个分支并获取最近一次提交的 message 及最多贡献者
	for _, branch := range branches {
		lastCommitMsg, mostContributor := getBranchCommitInfo(commits, branch.Name)
		result += fmt.Sprintf("Branch-%s: last commit: \"%s\", Most Contributor: %s\n", branch.Name, lastCommitMsg, mostContributor)
	}

	result += "+++++\n"
	return result, nil
}

// extractOwnerRepo 从 url 提取 owner 和 repo 信息
func extractOwnerRepo(url string) (string, string) {
	parts := strings.Split(url, "/")
	return parts[len(parts)-2], parts[len(parts)-1]
}

// getAllCommits 一次性获取仓库所有的提交信息
func getAllCommits(owner, repo string) ([]CommitInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, repo)
	resp, err := makeGitHubRequest(url)
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	if err := json.Unmarshal(resp, &commits); err != nil {
		return nil, err
	}

	return commits, nil
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

// getBranches 获取仓库的所有分支信息
func getBranches(owner, repo string) ([]Branch, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", owner, repo)
	resp, err := makeGitHubRequest(url)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	if err := json.Unmarshal(resp, &branches); err != nil {
		return nil, err
	}

	return branches, nil
}

// getBranchCommitInfo 从所有提交中获取分支的最后一次提交信息及贡献最多者
func getBranchCommitInfo(commits []CommitInfo, branch string) (string, string) {
	branchCommits := []CommitInfo{}
	contributors := make(map[string]int)
	var lastCommit CommitInfo

	// 筛选属于该分支的提交
	for _, commit := range commits {
		branchCommits = append(branchCommits, commit)
		contributors[commit.Author.Login]++
		if lastCommit.Commit.Author.Date.Before(commit.Commit.Author.Date) {
			lastCommit = commit
		}
	}

	// 找出分支贡献最多的用户
	mostContributor := ""
	maxCommits := 0
	for contributor, count := range contributors {
		if count > maxCommits {
			mostContributor = contributor
			maxCommits = count
		}
	}

	return lastCommit.Commit.Message, mostContributor
}

// makeGitHubRequest 封装 API 请求
func makeGitHubRequest(url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+Token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data from GitHub API, status code: %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}
